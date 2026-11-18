package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/azhkadaes/vms/backend/config"
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/handler"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace/noop"
)

// These doubles isolate HTTP decoding/serialization. They do not certify real
// authentication, PostgreSQL lifecycle transactions, rate limits or deployment.
type contractSession struct {
	session      dto.Session
	expired      bool
	fieldError   bool
	wrongCurrent bool
}

func (s *contractSession) result() authusecase.CredentialResult {
	return authusecase.CredentialResult{Result: dto.TokenResult{AccessToken: "synthetic-access-token", TokenType: "Bearer", AccessExpiresAt: time.Now().UTC().Add(5 * time.Minute), Session: s.session}, Refresh: strings.Repeat("r", 43), SessionID: "550e8400-e29b-41d4-a716-446655440000", AbsoluteExpiresAt: time.Now().Add(24 * time.Hour)}
}
func (s *contractSession) Login(context.Context, dto.Login, string) (authusecase.CredentialResult, error) {
	return s.result(), nil
}
func (s *contractSession) Refresh(context.Context, string, string) (authusecase.CredentialResult, error) {
	return s.result(), nil
}
func (s *contractSession) Logout(context.Context, string) error { return nil }
func (s *contractSession) Renew(context.Context, entity.SessionProof) (dto.Session, error) {
	return s.session, nil
}
func (s *contractSession) SwitchRole(context.Context, entity.SessionProof, string) (authusecase.CredentialResult, error) {
	return s.result(), nil
}
func (s *contractSession) ChangePassword(context.Context, entity.SessionProof, dto.PasswordChange) (authusecase.CredentialResult, error) {
	if s.fieldError {
		return authusecase.CredentialResult{}, failure.Validation().Add("newPassword", "Kata sandi baru minimal 15 karakter dan maksimal 72 byte.").Err()
	}
	if s.wrongCurrent {
		return authusecase.CredentialResult{}, failure.Validation().Add("currentPassword", "Kata sandi saat ini tidak sesuai.").Err()
	}
	return s.result(), nil
}
func (s *contractSession) Binding(context.Context, string, bool) (string, time.Time, error) {
	if s.expired {
		return "", time.Time{}, authusecase.ErrUnauthenticated
	}
	return "550e8400-e29b-41d4-a716-446655440000", time.Now().Add(24 * time.Hour), nil
}

type contractCatalog struct{}

func (contractCatalog) GetPublicCatalogs(context.Context) (dto.PublicCatalog, error) {
	return dto.PublicCatalog{Officers: []dto.CatalogOfficer{{ID: "1", Name: "Official Example", Active: true}}, Rooms: []dto.CatalogRoom{{ID: "1", Name: "Meeting room", Active: true}}, AttachmentsEnabled: false}, nil
}

// TestContractHTTP captures actual Fiber responses for external JSON Schema
// validation. The source AST gate separately verifies real router registrations.
func TestContractHTTP(t *testing.T) {
	now := time.Now().UTC()
	s := &contractSession{session: dto.Session{Active: true, ID: "1", Name: "Staff Example", Role: "PA", AvailableRoles: []string{"PA"}, RoleRevision: 1, Permissions: []string{"bookings"}, AssignedOfficerIDs: []string{"1"}, PasswordMinLength: 15, PasswordMaxBytes: 72, Timing: dto.SessionTiming{ServerTime: now.UnixMilli(), IdleExpiresAt: now.Add(8 * time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour)}}}
	key := []byte(strings.Repeat("k", 32))
	h := handler.NewAuthHandler(s, config.Auth{Origin: "https://staff.example.test", CSRFKey: key})
	app := fiber.New(fiber.Config{ErrorHandler: func(c fiber.Ctx, err error) error {
		return restmiddleware.Failure(c, err, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}})
	app.Use(restmiddleware.RequestContext(time.Second, noop.NewTracerProvider().Tracer("contract")))
	app.Use(func(c fiber.Ctx) error { c.Locals("vms.session", s.session); return c.Next() })
	app.Post("/v1/staff/sessions", h.Login)
	app.Get("/v1/staff/session", h.Current)
	app.Delete("/v1/staff/session", h.Logout)
	app.Post("/v1/staff/password", h.Password)
	app.Post("/v1/staff/session/role", h.Role)
	app.Post("/v1/staff/session/renew", h.Renew)
	app.Post("/v1/staff/session/refresh", h.Refresh)
	app.Get("/v1/public/catalogs", handler.NewCatalogHandler(contractCatalog{}).PublicCatalogs)
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
	captures := []capture{}
	cases := []struct {
		name, method, path, body  string
		status                    int
		login, lifecycle, expired bool
	}{
		{"catalog", "GET", "/v1/public/catalogs", "", 200, false, false, false},
		{"current", "GET", "/v1/staff/session", "", 200, false, false, false},
		{"login", "POST", "/v1/staff/sessions", `{"email":"staff@example.test","password":"synthetic-password"}`, 200, true, true, false},
		{"refresh", "POST", "/v1/staff/session/refresh", `{"refreshRequestId":"550e8400-e29b-41d4-a716-446655440000"}`, 200, false, true, false},
		{"logout", "DELETE", "/v1/staff/session", "", 204, false, true, false},
		{"password", "POST", "/v1/staff/password", `{"currentPassword":"old-synthetic-password","newPassword":"new-synthetic-password"}`, 200, false, false, false},
		{"role", "POST", "/v1/staff/session/role", `{"role":"PA"}`, 200, false, false, false},
		{"renew", "POST", "/v1/staff/session/renew", `{}`, 200, false, false, false},
		{"unknown-query", "GET", "/v1/public/catalogs?role=PA", "", 400, false, false, false},
		{"unknown-json", "POST", "/v1/staff/session/renew", `{"role":"PA"}`, 400, false, false, false},
		{"wrong-content-type", "POST", "/v1/staff/session/renew", `{}`, 415, false, false, false},
		{"expired-logout", "DELETE", "/v1/staff/session", "", 401, false, true, true},
		{"field-validation", "POST", "/v1/staff/password", `{"currentPassword":"old-synthetic-password","newPassword":"short"}`, 422, false, false, false},
		{"wrong-current-password", "POST", "/v1/staff/password", `{"currentPassword":"wrong-synthetic-password","newPassword":"new-synthetic-password"}`, 422, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s.expired = tc.expired
			s.fieldError = tc.name == "field-validation"
			s.wrongCurrent = tc.name == "wrong-current-password"
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" && tc.name != "wrong-content-type" {
				req.Header.Set("Content-Type", "application/json")
			}
			if tc.lifecycle {
				req.Header.Set("Origin", "https://staff.example.test")
				binding := "session:550e8400-e29b-41d4-a716-446655440000"
				cookie := "__Host-vms-refresh=" + strings.Repeat("r", 43)
				if tc.login {
					body := strings.Repeat("a", 43) + "." + strconv.FormatInt(time.Now().Add(20*time.Minute).Unix(), 10)
					anonymous := body + "." + security.Digest(key, "anonymous", body)
					binding = "anonymous:" + anonymous
					cookie = "__Host-vms-anonymous=" + anonymous
				}
				csrf, err := (security.CSRF{Key: key}).Sign(binding, now, now.Add(time.Hour))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Cookie", cookie+"; __Host-vms-csrf="+csrf)
				req.Header.Set("X-CSRF-Token", csrf)
			}
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != tc.status {
				b, _ := io.ReadAll(res.Body)
				t.Fatalf("status %d expected %d: %s", res.StatusCode, tc.status, b)
			}
			var body any
			raw, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if tc.status != 204 {
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Fatal(err)
				}
			}
			if offsets := nonUTCTimestamps(body); len(offsets) > 0 {
				t.Fatalf("timestamps must be UTC RFC 3339 with Z: %v", offsets)
			}
			captures = append(captures, capture{Name: tc.name, Method: tc.method, Path: strings.Split(tc.path, "?")[0], Status: res.StatusCode, Headers: map[string]string{"Content-Type": res.Header.Get("Content-Type"), "Cache-Control": res.Header.Get("Cache-Control"), "X-Content-Type-Options": res.Header.Get("X-Content-Type-Options"), "X-Request-ID": res.Header.Get("X-Request-ID")}, Body: body, Bytes: len(raw)})
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

var dateTime = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}`)

// nonUTCTimestamps lists date-time strings that are not RFC 3339 UTC with a Z suffix.
func nonUTCTimestamps(value any) []string {
	out := []string{}
	switch v := value.(type) {
	case map[string]any:
		for _, item := range v {
			out = append(out, nonUTCTimestamps(item)...)
		}
	case []any:
		for _, item := range v {
			out = append(out, nonUTCTimestamps(item)...)
		}
	case string:
		if dateTime.MatchString(v) {
			parsed, err := time.Parse(time.RFC3339Nano, v)
			if err != nil || !strings.HasSuffix(v, "Z") || parsed.Location() != time.UTC {
				out = append(out, v)
			}
		}
	}
	return out
}
