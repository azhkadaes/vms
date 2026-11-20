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
	staffbookingrepository "github.com/azhkadaes/vms/backend/internal/repository/staffbooking"
	staffbookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	staffbookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
)

type contractStaffReader struct {
	at    time.Time
	visit entity.StaffVisit
}

func (r contractStaffReader) List(context.Context, dto.Session) ([]entity.StaffVisit, time.Time, error) {
	return []entity.StaffVisit{r.visit}, r.at, nil
}

func (r contractStaffReader) Find(_ context.Context, _ dto.Session, id int32) (entity.StaffVisit, time.Time, error) {
	if id != r.visit.ID {
		return entity.StaffVisit{}, r.at, staffbookingrepository.ErrNotFound
	}
	return r.visit, r.at, nil
}

func TestStaffBookingContract(t *testing.T) {
	at := time.Date(2026, 11, 19, 1, 30, 0, 0, time.UTC)
	fixture := entity.StaffVisit{Visit: entity.Visit{ID: 1, Kind: "umum", Code: "00123456", Status: "approved", Version: 5,
		Date: "2026-11-19", Time: "10:00", Duration: 60, OfficerID: 2, OfficerName: "Pejabat Contoh",
		Name: "Pengunjung Contoh", Email: "visitor@example.test", Phone: "08123456789", Organization: "Instansi Contoh",
		Purpose: "Pertemuan", Visitors: []string{"Pengunjung Contoh"}, IdentityType: "KTP"}, VisitorCount: 1,
		History: []entity.StaffVisitEvent{{At: at, Message: "Kunjungan disetujui"}}}
	tracer := noop.NewTracerProvider().Tracer("staff-bookings-contract")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{CaseSensitive: true, StrictRouting: true, ErrorHandler: func(c fiber.Ctx, err error) error {
		return restmiddleware.Failure(c, err, logger)
	}})
	app.Use(restmiddleware.RequestContext(time.Second, tracer))
	session := func(role string) dto.Session {
		return dto.Session{Active: true, ID: "1", Role: role, Permissions: []string{policy.PermBookings}, AssignedOfficerIDs: []string{"2"}}
	}
	tokens := security.Tokens{Key: []byte(strings.Repeat("j", 32)), Issuer: "vms", Audience: "staff"}
	authority := contractAuthority{"pa": session(policy.RolePA), "reception": session(policy.RoleReceptionist)}
	deps := &router.Dependencies{App: app, StaffAuth: restmiddleware.StaffAuth(tokens, authority, contractLimiter{}), Workspace: restmiddleware.Workspace}
	router.RegisterStaffBookingRoutes(deps, staffbookingusecase.NewStaffBookingUsecase(contractStaffReader{at, fixture}, staffbookingserializer.NewStaffBookingSerializer(), tracer))
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
	const list = "/v1/staff/bookings"
	for _, tc := range []struct {
		name, path, stage string
		status            int
	}{
		{"staff-bookings-list", list + "?view=list", "pa", 200},
		{"staff-bookings-dashboard", list + "?view=dashboard&date=2026-11-19", "pa", 200},
		{"staff-bookings-reception", list + "?view=reception&mode=checkin", "reception", 200},
		{"staff-bookings-unknown-query", list + "?role=PA", "pa", 400},
		{"staff-bookings-invalid-page", list + "?pageSize=101", "pa", 422},
		{"staff-booking-detail", list + "/1", "pa", 200},
		{"staff-booking-missing", list + "/9", "pa", 404},
		{"staff-booking-reception-forbidden", list + "/1", "reception", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", tc.path, nil)
			now := time.Now()
			token, err := tokens.Sign(security.SessionClaims{SessionID: tc.stage, EmployeeID: 1, CredentialGeneration: 1, RoleRevision: 1}, now, now.Add(time.Minute))
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
			headers := map[string]string{}
			for _, key := range []string{"Content-Type", "Cache-Control", "X-Content-Type-Options", "X-Request-ID"} {
				headers[key] = res.Header.Get(key)
			}
			captures = append(captures, record{tc.name, "GET", request.URL.Path, res.StatusCode, headers, body, len(body)})
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
