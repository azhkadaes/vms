package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/azhkadaes/vms/backend/internal/infrastructure/storage"
	"github.com/azhkadaes/vms/backend/internal/policy"
	filerepo "github.com/azhkadaes/vms/backend/internal/repository/bookingfile"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	fileusecase "github.com/azhkadaes/vms/backend/internal/usecase/bookingfile/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

var contractFileBytes = map[string][]byte{
	"application/pdf": []byte("%PDF-1.4\n%%EOF"),
	"image/jpeg":      {0xff, 0xd8, 0xff, 0xd9},
	"image/png":       {0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
}

type contractFileRepo struct{ matched int }

func (r *contractFileRepo) Lookup(_ context.Context, id int32, purpose string, credential *entity.ReceptionCredential) (entity.BookingFile, error) {
	if credential != nil {
		r.matched++
		if credential.Code != "12345678" && credential.TokenHash != security.Hash("synthetic-qr-for-file") {
			return entity.BookingFile{}, filerepo.ErrNotFound
		}
	}
	if id == 99 {
		return entity.BookingFile{}, filerepo.ErrNotFound
	}
	start := time.Date(2026, 11, 17, 10, 0, 0, 0, validate.WITA)
	mime := "application/pdf"
	if purpose == policy.PurposeVisitorPhoto {
		mime = "image/jpeg"
		if id == 8 {
			mime = "image/png"
		}
	}
	file := entity.BookingFile{
		Key: mime, MIME: mime, Size: len(contractFileBytes[mime]), OfficerID: 3, Kind: "umum",
		Status: "approved", Date: "2026-11-17", Time: "10:00", Duration: 60, ServerTime: start,
	}
	if id == 4 {
		file.ServerTime = start.Add(-time.Hour)
	}
	if id == 6 {
		file.Kind = "terbatas"
	}
	if id == 8 {
		file.Status = "checked_in"
		file.ServerTime = start.Add(20 * time.Hour)
	}
	if id == 9 {
		file.Key = "missing object"
	}
	if credential != nil {
		file.Match.VisitID = id
		if credential.TokenHash != "" {
			expiry := start.Add(60 * time.Minute)
			file.Match.QRExpiresAt = &expiry
		}
	}
	return file, nil
}

type contractFileStorage struct{}

func (contractFileStorage) Put(context.Context, []byte) (string, error) {
	return "", errors.New("cannot write through a read route")
}

func (contractFileStorage) Delete(context.Context, string) error {
	return errors.New("cannot delete through a read route")
}

func (contractFileStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if value, ok := contractFileBytes[key]; ok {
		return io.NopCloser(bytes.NewReader(value)), nil
	}
	return nil, storage.ErrNotFound
}

func TestContractBookingFiles(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("contract-files")
	app := fiber.New(fiber.Config{CaseSensitive: true, StrictRouting: true, ErrorHandler: func(c fiber.Ctx, err error) error {
		return restmiddleware.Failure(c, err, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}})
	app.Use(restmiddleware.RequestContext(time.Second, tracer))
	tokens := security.Tokens{Key: []byte(strings.Repeat("k", 32)), Issuer: "vms", Audience: "staff"}
	protocol := dto.Session{ID: "1", Active: true, Role: policy.RoleProtocol, Permissions: []string{policy.PermBookings}}
	pa := dto.Session{ID: "1", Active: true, Role: policy.RolePA, Permissions: []string{policy.PermBookings}, AssignedOfficerIDs: []string{"5"}}
	reception := dto.Session{ID: "1", Active: true, Role: policy.RoleReceptionist, Permissions: []string{policy.PermCheckin, policy.PermCheckout}}
	deps := &router.Dependencies{App: app, StaffAuth: restmiddleware.StaffAuth(tokens,
		contractAuthority{"protocol": protocol, "narrow pa": pa, "reception": reception}, contractLimiter{}), Workspace: restmiddleware.Workspace}
	repo := &contractFileRepo{}
	router.RegisterBookingFileRoutes(deps, fileusecase.NewBookingFileUsecase(repo, contractFileStorage{}, tracer))
	app.Use(func(c fiber.Ctx) error { return fiber.ErrNotFound })

	type capture struct {
		Name    string            `json:"name"`
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		Body    any               `json:"body"`
		Bytes   int               `json:"bytes"`
	}
	const base = "/v1/staff/bookings"
	const code = `{"credential":"12345678"}`
	const qr = `{"credential":"synthetic-qr-for-file"}`
	var captures []capture
	for _, test := range []struct {
		name, method, path, session, request, mime, code string
		status                                           int
	}{
		{"staff-attachment-binary", "GET", "/7/attachment", "protocol", "", "application/pdf", "", 200},
		{"staff-photo-jpeg-binary", "GET", "/7/visitor-photo", "protocol", "", "image/jpeg", "", 200},
		{"staff-file-outside-scope", "GET", "/7/attachment", "narrow pa", "", "", "not_found", 404},
		{"staff-file-missing-object", "GET", "/9/attachment", "protocol", "", "", "not_found", 404},
		{"reception-file-wrong-method", "GET", "/7/attachment", "reception", "", "", "forbidden", 403},
		{"reception-attachment-binary", "POST", "/7/attachment", "reception", code, "application/pdf", "", 200},
		{"reception-attachment-repeat", "POST", "/7/attachment", "reception", code, "application/pdf", "", 200},
		{"reception-checkout-photo-png", "POST", "/8/visitor-photo", "reception", qr, "image/png", "", 200},
		{"reception-file-early", "POST", "/4/attachment", "reception", code, "", "not_found", 404},
		{"reception-file-restricted", "POST", "/6/attachment", "reception", code, "", "not_found", 404},
		{"reception-file-wrong-credential", "POST", "/7/attachment", "reception", `{"credential":"unmatched credential"}`, "", "not_found", 404},
		{"reception-file-missing", "POST", "/99/attachment", "reception", code, "", "not_found", 404},
		{"reception-file-wrong-role", "POST", "/7/attachment", "protocol", code, "", "not_found", 404},
		{"reception-file-query-denied", "POST", "/7/attachment?credential=synthetic-query-proof", "reception", code, "", "invalid_query_parameter", 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			now := time.Now().UTC()
			token, err := tokens.Sign(security.SessionClaims{SessionID: test.session, EmployeeID: 1, CredentialGeneration: 1, RoleRevision: 1}, now, now.Add(time.Minute))
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(test.method, base+test.path, strings.NewReader(test.request))
			req.Header.Set("Authorization", "Bearer "+token)
			if test.request != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			raw, err := io.ReadAll(res.Body)
			if err != nil || res.StatusCode != test.status {
				t.Fatalf("status %d, body %q, error %v", res.StatusCode, raw, err)
			}
			var body any
			if test.status == 200 {
				if res.Header.Get("Content-Type") != test.mime || !bytes.Equal(raw, contractFileBytes[test.mime]) {
					t.Fatal("unexpected file bytes or Content-Type")
				}
			} else {
				var got struct {
					Code string `json:"code"`
				}
				if json.Unmarshal(raw, &got) != nil || json.Unmarshal(raw, &body) != nil || got.Code != test.code {
					t.Fatalf("error body: %s", raw)
				}
				for _, value := range []string{"12345678", "unmatched credential", "synthetic-query-proof", "synthetic-qr-for-file"} {
					if bytes.Contains(raw, []byte(value)) {
						t.Fatal("the response returned a credential")
					}
				}
			}
			headers := map[string]string{}
			for _, name := range []string{"Content-Type", "Cache-Control", "X-Content-Type-Options", "X-Request-ID"} {
				headers[name] = res.Header.Get(name)
			}
			if headers["Cache-Control"] != "no-store" || headers["X-Content-Type-Options"] != "nosniff" {
				t.Fatal("missing private file response headers")
			}
			captures = append(captures, capture{test.name, test.method, base + strings.Split(test.path, "?")[0], res.StatusCode, headers, body, len(raw)})
		})
	}
	if repo.matched != 7 {
		t.Fatalf("got %d proof checks, want 7 including replay", repo.matched)
	}
	if out := os.Getenv("VMS_CONTRACT_CAPTURE"); out != "" {
		raw, err := json.Marshal(captures)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(out, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
