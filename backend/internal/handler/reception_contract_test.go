package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	receptionrepository "github.com/azhkadaes/vms/backend/internal/repository/reception"
	staffbookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	receptionimpl "github.com/azhkadaes/vms/backend/internal/usecase/reception/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
)

type receptionCaptureRepo struct {
	visit entity.StaffVisit
	at    time.Time
}

func (r receptionCaptureRepo) Resolve(_ context.Context, credential entity.ReceptionCredential) (entity.StaffVisit, entity.ReceptionMatch, time.Time, error) {
	match, err := r.MatchCredential(credential, r.visit.ID)
	return r.visit, match, r.at, err
}

func (r receptionCaptureRepo) Transaction(_ context.Context, fn func(receptionrepository.ReceptionTransaction) error) error {
	return fn(r)
}
func (r receptionCaptureRepo) LockAuthority(entity.SessionProof) (entity.SessionSnapshot, error) {
	return entity.SessionSnapshot{EmployeeID: 1}, nil
}
func (r receptionCaptureRepo) LockVisit(id int32) (entity.StaffVisit, error) {
	if r.visit.ID != id {
		return entity.StaffVisit{}, receptionrepository.ErrNotFound
	}
	return r.visit, nil
}
func (r receptionCaptureRepo) MatchCredential(credential entity.ReceptionCredential, id int32) (entity.ReceptionMatch, error) {
	if id != r.visit.ID || (credential.Code != r.visit.Code && credential.TokenHash != security.Hash("synthetic-expired-qr")) {
		return entity.ReceptionMatch{}, receptionrepository.ErrNotFound
	}
	result := entity.ReceptionMatch{VisitID: id}
	if credential.TokenHash != "" {
		past := r.at.Add(-time.Second)
		result.QRExpiresAt = &past
	}
	return result, nil
}
func (r receptionCaptureRepo) Now() (time.Time, error) { return r.at, nil }
func (r receptionCaptureRepo) Transition(_ int32, _ int, _ int32, action string, _ time.Time) (entity.StaffVisit, error) {
	visit := r.visit
	visit.Version++
	if action == policy.AttendanceCheckIn {
		visit.Status = "checked_in"
	} else {
		visit.Status = "checked_out"
	}
	return visit, nil
}

type receptionCaptureValidator struct{ session dto.Session }

func (v receptionCaptureValidator) Validate(entity.SessionSnapshot, entity.SessionProof) (dto.Session, error) {
	return v.session, nil
}

func TestReceptionContract(t *testing.T) {
	at := time.Date(2026, 11, 19, 2, 0, 0, 0, time.UTC)
	name := "Nota Dinas Contoh.pdf"
	photo := "Foto Contoh.png"
	visit := entity.StaffVisit{Visit: entity.Visit{ID: 1, Version: 5, Kind: "umum", Code: "00123456", Status: "approved",
		Date: "2026-11-19", Time: "10:00", Duration: 60, OfficerID: 2, OfficerName: "Pejabat Contoh",
		Name: "Pengunjung Contoh", Organization: "Instansi Contoh", Purpose: "Pertemuan", Visitors: []string{"Pengunjung Contoh"},
		IdentityType: "KTP", NDName: &name}, VisitorCount: 1, AttachmentAvailable: true, VisitorPhotoName: &photo}
	tracer := noop.NewTracerProvider().Tracer("reception-contract")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{CaseSensitive: true, StrictRouting: true, ErrorHandler: func(c fiber.Ctx, err error) error {
		return restmiddleware.Failure(c, err, logger)
	}})
	app.Use(restmiddleware.RequestContext(time.Second, tracer))
	session := dto.Session{Active: true, ID: "1", Role: policy.RoleReceptionist, Permissions: []string{policy.PermCheckin, policy.PermCheckout}}
	tokens := security.Tokens{Key: []byte(strings.Repeat("j", 32)), Issuer: "vms", Audience: "staff"}
	authority := contractAuthority{"reception": session}
	deps := &router.Dependencies{App: app, StaffAuth: restmiddleware.StaffAuth(tokens, authority, contractLimiter{}), Workspace: restmiddleware.Workspace}
	router.RegisterReceptionRoutes(deps, receptionimpl.NewReceptionUsecase(receptionCaptureRepo{visit, at}, staffbookingserializer.NewStaffBookingSerializer(), receptionCaptureValidator{session}, tracer))
	type record struct {
		Name    string            `json:"name"`
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		Body    json.RawMessage   `json:"body"`
		Bytes   int               `json:"bytes"`
	}
	var captures []record
	const resolve = "/v1/staff/resolve"
	const attendance = "/v1/staff/bookings/1/attendance"
	for _, tc := range []struct {
		name, path, payload string
		status              int
	}{
		{"reception-resolve", resolve, `{"credential":"00123456","visitId":"1"}`, 200},
		{"reception-resolve-missing", resolve, `{"credential":"12345678","visitId":"1"}`, 404},
		{"reception-resolve-pin-without-visitor", resolve, `{"credential":"00123456"}`, 422},
		{"reception-resolve-pin-with-other-visitor", resolve, `{"credential":"00123456","visitId":"2"}`, 404},
		{"reception-resolve-expired", resolve, `{"credential":"synthetic-expired-qr"}`, 404},
		{"reception-check-in", attendance, `{"action":"check_in","credential":"00123456","version":5}`, 200},
		{"reception-credential-expired", attendance, `{"action":"check_in","credential":"synthetic-expired-qr","version":5}`, 409},
		{"reception-stale-version", attendance, `{"action":"check_in","credential":"00123456","version":4}`, 409},
		{"reception-action-required", attendance, `{"credential":"00123456","version":5}`, 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.payload))
			request.Header.Set("Content-Type", "application/json")
			now := time.Now()
			token, err := tokens.Sign(security.SessionClaims{SessionID: "reception", EmployeeID: 1, CredentialGeneration: 1, RoleRevision: 1}, now, now.Add(time.Minute))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Authorization", "Bearer "+token)
			res, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil || res.StatusCode != tc.status || !json.Valid(body) {
				t.Fatalf("status %d, want %d; body %q (%v)", res.StatusCode, tc.status, body, err)
			}
			if strings.Contains(string(body), "synthetic-expired-qr") || strings.Contains(string(body), "00123456") || strings.Contains(string(body), "12345678") {
				t.Fatalf("credential leaked in response: %s", body)
			}
			headers := map[string]string{}
			for _, key := range []string{"Content-Type", "Cache-Control", "X-Content-Type-Options", "X-Request-ID"} {
				headers[key] = res.Header.Get(key)
			}
			captures = append(captures, record{tc.name, "POST", request.URL.Path, res.StatusCode, headers, body, len(body)})
		})
	}
	if target := os.Getenv("VMS_CONTRACT_CAPTURE"); target != "" {
		data, err := json.MarshalIndent(captures, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
