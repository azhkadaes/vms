package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
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
	"github.com/azhkadaes/vms/backend/internal/policy"
	catalogserializer "github.com/azhkadaes/vms/backend/internal/serializer/catalog/implementation"
	roleserializer "github.com/azhkadaes/vms/backend/internal/serializer/role/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	catalogusecase "github.com/azhkadaes/vms/backend/internal/usecase/catalog/implementation"
	uploadusecase "github.com/azhkadaes/vms/backend/internal/usecase/upload"
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

// contractAuthority stands in for the live session read behind StaffAuth: the Bearer
// token's session ID names the A-18 stage whose session it returns.
type contractAuthority map[string]dto.Session

func (a contractAuthority) GetSession(_ context.Context, proof entity.SessionProof) (dto.Session, error) {
	session, ok := a[proof.SessionID]
	if !ok {
		return dto.Session{}, authusecase.ErrUnauthenticated
	}
	return session, nil
}

type contractLimiter struct{}

func (contractLimiter) Admit(context.Context, string, string, int, time.Duration) error { return nil }

type contractAdmission struct{}

func (contractAdmission) PublicRequest(context.Context, string) (entity.RequestAdmission, error) {
	return entity.RequestAdmission{Allowed: true}, nil
}

type contractCatalog struct{}

func (contractCatalog) GetPublicCatalogs(context.Context) (dto.PublicCatalog, error) {
	return dto.PublicCatalog{Officers: []dto.CatalogOfficer{{ID: "1", Name: "Official Example", Active: true}}, Rooms: []dto.CatalogRoom{{ID: "1", Name: "Meeting room", Active: true}}, AttachmentsEnabled: false}, nil
}

// contractUploads answers by reference, so each capture exercises one upload outcome.
type contractUploads struct{}

const (
	uploadCreated   = "0f8fad5b-d9cb-469f-a165-70867728950e"
	uploadReplayed  = "7c9e6679-7425-40de-944b-e07fc1f90ae7"
	uploadConflict  = "9b2d6f1e-3c4a-4d5e-8f60-718293a4b5c6"
	uploadWrongType = "1e2f3a4b-5c6d-4e7f-8a9b-0c1d2e3f4a5b"
	uploadLimited   = "2a3b4c5d-6e7f-4a8b-9c0d-1e2f3a4b5c6d"
	uploadDenied    = "3b4c5d6e-7f8a-4b9c-8d1e-2f3a4b5c6d7e"
)

func (contractUploads) issue(upload entity.UploadRequest) (uploadusecase.Issued, error) {
	switch upload.Reference {
	case uploadConflict:
		return uploadusecase.Issued{}, failure.New(409, "idempotency_conflict")
	case uploadWrongType:
		return uploadusecase.Issued{}, failure.Validation().Add("file", "Gunakan berkas PDF yang valid.").Err()
	case uploadLimited:
		return uploadusecase.Issued{}, &failure.Error{Status: 429, Code: "rate_limited", RetryAfter: 1800}
	case uploadDenied:
		return uploadusecase.Issued{}, failure.New(403, "forbidden")
	}
	now := time.Now().UTC()
	return uploadusecase.Issued{Created: upload.Reference != uploadReplayed, Result: dto.UploadResult{Token: "vmsu1." + strings.Repeat("b", 43), ExpiresAt: now.Add(time.Hour), ServerTime: now.UnixMilli()}}, nil
}
func (u contractUploads) UploadPublic(_ context.Context, _ string, upload entity.UploadRequest) (uploadusecase.Issued, error) {
	return u.issue(upload)
}
func (u contractUploads) UploadStaff(_ context.Context, _ entity.SessionProof, _ string, upload entity.UploadRequest) (uploadusecase.Issued, error) {
	return u.issue(upload)
}

// multipartUpload is a browser-shaped upload body: one file part plus reference and purpose.
func multipartUpload(t *testing.T, reference string, file []byte) (string, *bytes.Buffer) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "nota dinas.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(file); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("reference", reference); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("purpose", "document"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return writer.FormDataContentType(), body
}

// contractRoles serves the four seeded roles at version 2 through the real role serializer. It
// refuses like the role usecase for the captured cases; the rules themselves are tested there.
type contractRoles struct{}

var contractRoleRows = []entity.Role{
	{ID: 1, Name: policy.RolePA, Version: 2, PolicyID: 1, Permissions: []string{"bookings", "create", "approve", "reject", "reschedule", "cancel", "reports", "reportExport"}},
	{ID: 2, Name: policy.RoleProtocol, Version: 2, PolicyID: 2, Permissions: []string{"bookings", "create", "approve", "reject", "reschedule", "cancel", "reports", "reportExport"}},
	{ID: 3, Name: policy.RoleReceptionist, Version: 2, PolicyID: 3, Permissions: []string{"bookings", "checkin", "checkout"}},
	{ID: 4, Name: policy.RoleSuperAdmin, Version: 2, PolicyID: 4, Permissions: []string{"bookings", "reports", "reportExport", "people", "roles", "integrations", "audit"}},
}

func (contractRoles) GetRoles(_ context.Context, session dto.Session) ([]dto.RoleDefinition, error) {
	if !policy.Has(session, policy.PermRoles) {
		return nil, failure.New(403, "forbidden")
	}
	return roleserializer.NewRoleSerializer().ToRoleDefinitions(contractRoleRows), nil
}
func (contractRoles) UpdateRole(_ context.Context, _ entity.SessionProof, id int32, req dto.RoleUpdate) (dto.RoleDefinition, error) {
	role := contractRoleRows[id-1]
	switch {
	case req.Version != role.Version:
		return dto.RoleDefinition{}, failure.New(409, "version_conflict")
	case !policy.KeepsFixedPermissions(role.Name, role.Permissions, req.Permissions):
		return dto.RoleDefinition{}, failure.Validation().Add("permissions", "Izin tetap peran ini tidak dapat diubah.").Err()
	}
	role.Permissions, role.Version = req.Permissions, role.Version+1
	return roleserializer.NewRoleSerializer().ToRoleDefinition(role), nil
}

func (contractCatalog) GetStaffCatalogs(context.Context, dto.Session) (dto.StaffCatalog, error) {
	return dto.StaffCatalog{}, errors.New("op 08 captures run through the real catalog usecase")
}

// contractCatalogRows stands in for the catalog repository, so op 08 captures carry the real
// usecase scope and serializer output: an inactive official appears only when asked for.
type contractCatalogRows struct{}

func (contractCatalogRows) GetPublicCatalogs(context.Context) (entity.PublicCatalog, error) {
	return entity.PublicCatalog{}, errors.New("public catalog captures use contractCatalog")
}
func (contractCatalogRows) GetStaffCatalogs(_ context.Context, assignedInactiveOfficers bool) (entity.StaffCatalog, error) {
	officers := []entity.CatalogOfficer{
		{ID: 1, Name: "Official Example", Active: true, JobTitle: "Direktur Contoh", Directorate: "Direktorat Contoh", Deputy: "Kedeputian Contoh"},
		{ID: 2, Name: "VIP Official Example", VIP: true, Active: true, JobTitle: "Deputi Contoh", Deputy: "Kedeputian Contoh"},
	}
	if assignedInactiveOfficers {
		officers = append(officers, entity.CatalogOfficer{ID: 3, Name: "Former Official Example", Active: false, JobTitle: "Direktur Contoh", Directorate: "Direktorat Contoh", Deputy: "Kedeputian Contoh"})
	}
	return entity.StaffCatalog{Officers: officers, Rooms: []entity.CatalogRoom{{ID: 1, Name: "Meeting room", Active: true}}}, nil
}

// TestContractHTTP captures actual Fiber responses for external JSON Schema validation
// (check.py --runtime). Requests go through the exported route registrations with their
// real middleware (public admission, StaffAuth, the A-18 stage gate) and injected usecases.
func TestContractHTTP(t *testing.T) {
	now := time.Now().UTC()
	ready := dto.Session{Active: true, ID: "1", Name: "Staff Example", Role: policy.RolePA, AvailableRoles: []string{policy.RolePA}, RoleRevision: 1, Permissions: []string{policy.PermBookings}, AssignedOfficerIDs: []string{"1"}, PasswordMinLength: 15, PasswordMaxBytes: 72, Timing: dto.SessionTiming{ServerTime: now.UnixMilli(), IdleExpiresAt: now.Add(8 * time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour)}}
	roleSelection := ready
	roleSelection.Role, roleSelection.RoleSelectionRequired, roleSelection.AvailableRoles = "", true, []string{policy.RolePA, policy.RoleProtocol}
	roleSelection.Permissions, roleSelection.AssignedOfficerIDs = []string{}, []string{}
	passwordChange := ready
	passwordChange.MustChangePassword, passwordChange.Permissions, passwordChange.AssignedOfficerIDs = true, []string{}, []string{}
	admin := ready
	admin.Role, admin.AvailableRoles, admin.AssignedOfficerIDs = policy.RoleSuperAdmin, []string{policy.RoleSuperAdmin}, []string{}
	admin.Permissions = []string{policy.PermBookings, policy.PermReports, policy.PermReportExport, policy.PermPeople, policy.PermRoles, policy.PermIntegrations, policy.PermAudit}
	people := ready
	people.Role, people.AvailableRoles, people.Permissions, people.AssignedOfficerIDs = policy.RoleSuperAdmin, []string{policy.RoleSuperAdmin}, []string{policy.PermPeople, policy.PermRoles}, []string{}
	reception := ready
	reception.Role, reception.AvailableRoles, reception.Permissions, reception.AssignedOfficerIDs = policy.RoleReceptionist, []string{policy.RoleReceptionist}, []string{policy.PermCheckin, policy.PermCheckout}, []string{}
	staffCreator := ready
	staffCreator.Role, staffCreator.AvailableRoles, staffCreator.Permissions = policy.RoleProtocol, []string{policy.RoleProtocol}, []string{policy.PermBookings, policy.PermCreate}
	s := &contractSession{session: ready}
	key := []byte(strings.Repeat("k", 32))
	tokens := security.Tokens{Key: []byte(strings.Repeat("j", 32)), Issuer: "vms", Audience: "staff"}
	cfg := &config.Config{Auth: &config.Auth{Origin: "https://staff.example.test", CSRFKey: key, JWTKey: tokens.Key, Issuer: tokens.Issuer, Audience: tokens.Audience}}
	// The production server lifts the body limit only for upload paths; here every route takes the upload size.
	app := fiber.New(fiber.Config{CaseSensitive: true, StrictRouting: true, BodyLimit: handler.MultipartBodyLimit(uploadusecase.MaxFileBytes), DisablePreParseMultipartForm: true, ErrorHandler: func(c fiber.Ctx, err error) error {
		return restmiddleware.Failure(c, err, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}})
	app.Use(restmiddleware.RequestContext(time.Second, noop.NewTracerProvider().Tracer("contract")))
	authority := contractAuthority{"ready": ready, "role selection": roleSelection, "password change": passwordChange, "people": people, "reception without bookings": reception, "admin": admin}
	authority["staff create protocol"] = staffCreator
	deps := &router.Dependencies{App: app, Cfg: cfg, StaffAuth: restmiddleware.StaffAuth(tokens, authority, contractLimiter{}), Workspace: restmiddleware.Workspace}
	router.RegisterPublicRoutes(app, contractCatalog{}, contractAdmission{}, contractUploads{})
	router.RegisterPublicBookingRoutes(app, contractAdmission{}, contractLimiter{}, contractPublicBooking{})
	router.RegisterAuthRoutes(deps, s)
	notice := &dto.PublishedNotice{Version: "2026-11-12", ID: "Pemberitahuan organisasi (contoh).", EN: "Organization notice (example)."}
	router.RegisterStaffRoutes(deps, catalogusecase.NewCatalogUsecase(contractCatalogRows{}, catalogserializer.NewCatalogSerializer(notice, false), noop.NewTracerProvider().Tracer("contract")))
	router.RegisterStaffUploadRoutes(deps, contractUploads{})
	router.RegisterRoleRoutes(deps, contractRoles{})
	router.RegisterEmployeeRoutes(deps, contractEmployees{})
	router.RegisterStaffBookingCreateRoute(deps, contractStaffCreator{})
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
	// stage names the Bearer session for StaffAuth routes; lifecycle routes use cookies instead.
	cases := []struct {
		name, method, path, body, stage string
		status                          int
		login, lifecycle, expired       bool
	}{
		{"catalog", "GET", "/v1/public/catalogs", "", "", 200, false, false, false},
		{"public-booking-created", "POST", "/v1/public/bookings", `{"visitorPhotoToken":"synthetic-fixture"}`, "", 201, false, false, false},
		{"public-booking-replay", "POST", "/v1/public/bookings", `{"visitorPhotoToken":"synthetic-fixture","purpose":"Repeated submission"}`, "", 200, false, false, false},
		{"public-booking-invalid-token", "POST", "/v1/public/bookings", `{}`, "", 422, false, false, false},
		{"public-submission-committed", "POST", "/v1/public/submissions/resolve", `{"email":"visitor@example.test"}`, "", 200, false, false, false},
		{"public-submission-closed", "POST", "/v1/public/submissions/resolve", `{"email":"unknown@example.test"}`, "", 200, false, false, false},
		{"current", "GET", "/v1/staff/session", "", "ready", 200, false, false, false},
		{"login", "POST", "/v1/staff/sessions", `{"email":"staff@example.test","password":"synthetic-password"}`, "", 200, true, true, false},
		{"refresh", "POST", "/v1/staff/session/refresh", `{"refreshRequestId":"550e8400-e29b-41d4-a716-446655440000"}`, "", 200, false, true, false},
		{"logout", "DELETE", "/v1/staff/session", "", "", 204, false, true, false},
		{"password", "POST", "/v1/staff/password", `{"currentPassword":"old-synthetic-password","newPassword":"new-synthetic-password"}`, "ready", 200, false, false, false},
		{"role", "POST", "/v1/staff/session/role", `{"role":"PA"}`, "ready", 200, false, false, false},
		{"renew", "POST", "/v1/staff/session/renew", `{}`, "ready", 200, false, false, false},
		{"unknown-query", "GET", "/v1/public/catalogs?role=PA", "", "", 400, false, false, false},
		{"unknown-json", "POST", "/v1/staff/session/renew", `{"role":"PA"}`, "ready", 400, false, false, false},
		{"wrong-content-type", "POST", "/v1/staff/session/renew", `{}`, "ready", 415, false, false, false},
		{"expired-logout", "DELETE", "/v1/staff/session", "", "", 401, false, true, true},
		{"field-validation", "POST", "/v1/staff/password", `{"currentPassword":"old-synthetic-password","newPassword":"short"}`, "ready", 422, false, false, false},
		{"wrong-current-password", "POST", "/v1/staff/password", `{"currentPassword":"wrong-synthetic-password","newPassword":"new-synthetic-password"}`, "ready", 422, false, false, false},
		{"password-role-selection-pending", "POST", "/v1/staff/password", `{"currentPassword":"old-synthetic-password","newPassword":"new-synthetic-password"}`, "role selection", 403, false, false, false},
		{"role-password-change-pending", "POST", "/v1/staff/session/role", `{"role":"PA"}`, "password change", 403, false, false, false},
		{"staff-catalogs", "GET", "/v1/staff/catalogs", "", "ready", 200, false, false, false},
		{"staff-catalogs-people", "GET", "/v1/staff/catalogs", "", "people", 200, false, false, false},
		{"staff-catalogs-unauthenticated", "GET", "/v1/staff/catalogs", "", "", 401, false, false, false},
		{"staff-catalogs-without-bookings-or-people", "GET", "/v1/staff/catalogs", "", "reception without bookings", 403, false, false, false},
		{"staff-catalogs-password-change-pending", "GET", "/v1/staff/catalogs", "", "password change", 403, false, false, false},
		{"staff-catalogs-unknown-query", "GET", "/v1/staff/catalogs?officer=1", "", "ready", 400, false, false, false},
		{"roles", "GET", "/v1/staff/roles", "", "admin", 200, false, false, false},
		{"roles-without-permission", "GET", "/v1/staff/roles", "", "ready", 403, false, false, false},
		{"update-role", "PATCH", "/v1/staff/roles/1", `{"permissions":["bookings","create","reject","reschedule","cancel","reports"],"version":2}`, "admin", 200, false, false, false},
		{"update-role-stale-version", "PATCH", "/v1/staff/roles/1", `{"permissions":["bookings"],"version":1}`, "admin", 409, false, false, false},
		{"update-role-fixed-permission", "PATCH", "/v1/staff/roles/4", `{"permissions":["bookings"],"version":2}`, "admin", 422, false, false, false},
		{"employees-page", "GET", "/v1/staff/employees?q=Employee&page=1&pageSize=25", "", "people", 200, false, false, false},
		{"employees-no-people", "GET", "/v1/staff/employees", "", "ready", 403, false, false, false},
		{"employees-invalid-page-size", "GET", "/v1/staff/employees?pageSize=251", "", "people", 422, false, false, false},
		{"employee-detail", "GET", "/v1/staff/employees/7", "", "people", 200, false, false, false},
		{"employee-self-profile", "GET", "/v1/staff/employees/1", "", "ready", 200, false, false, false},
		{"employee-other-profile-denied", "GET", "/v1/staff/employees/7", "", "ready", 403, false, false, false},
		{"employee-not-found", "GET", "/v1/staff/employees/8", "", "people", 404, false, false, false},
		{"employee-create", "POST", "/v1/staff/employees", `{"name":"Employee Example","email":"employee@example.test","roles":["PA"],"active":true,"assignedOfficerIds":["3"],"initialPassword":"synthetic-initial-password","requirePasswordChange":true}`, "people", 201, false, false, false},
		{"employee-duplicate-email", "POST", "/v1/staff/employees", `{"email":"duplicate@example.test"}`, "people", 422, false, false, false},
		{"employee-update", "PATCH", "/v1/staff/employees/7", `{"active":false,"version":1}`, "people", 200, false, false, false},
		{"employee-stale-version", "PATCH", "/v1/staff/employees/7", `{"active":false,"version":2}`, "people", 409, false, false, false},
		{"staff-booking-created", "POST", "/v1/staff/bookings", `{"visitKind":"restricted","officerId":"3","date":"2026-11-17","time":"10:00","idempotencyKey":"12345678-1234-4234-8234-123456789abc"}`, "staff create protocol", 201, false, false, false},
		{"staff-booking-create-replay", "POST", "/v1/staff/bookings", `{"visitKind":"restricted","officerId":"3","date":"2026-11-17","time":"10:00","idempotencyKey":"12345678-1234-4234-8234-123456789abb"}`, "staff create protocol", 200, false, false, false},
		{"staff-booking-create-conflict", "POST", "/v1/staff/bookings", `{"visitKind":"restricted","officerId":"3","date":"2026-11-17","time":"10:05","idempotencyKey":"12345678-1234-4234-8234-123456789abc"}`, "staff create protocol", 409, false, false, false},
		{"staff-booking-create-restricted-fields", "POST", "/v1/staff/bookings", `{"visitKind":"restricted","officerId":"3","date":"2026-11-17","time":"10:00","email":null,"idempotencyKey":"12345678-1234-4234-8234-123456789abc"}`, "staff create protocol", 400, false, false, false},
	}
	// record runs req and captures its response for check.py --runtime.
	record := func(t *testing.T, name, method, path string, status int, req *http.Request) {
		t.Helper()
		res, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != status {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("status %d expected %d: %s", res.StatusCode, status, b)
		}
		var body any
		raw, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if status != 204 {
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatal(err)
			}
		}
		if offsets := nonUTCTimestamps(body); len(offsets) > 0 {
			t.Fatalf("timestamps must be UTC RFC 3339 with Z: %v", offsets)
		}
		captures = append(captures, capture{Name: name, Method: method, Path: strings.Split(path, "?")[0], Status: res.StatusCode, Headers: map[string]string{"Content-Type": res.Header.Get("Content-Type"), "Cache-Control": res.Header.Get("Cache-Control"), "X-Content-Type-Options": res.Header.Get("X-Content-Type-Options"), "X-Request-ID": res.Header.Get("X-Request-ID")}, Body: body, Bytes: len(raw)})
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
			if tc.stage != "" {
				token, err := tokens.Sign(security.SessionClaims{SessionID: tc.stage, EmployeeID: 1, CredentialGeneration: 1, RoleRevision: 1}, now, now.Add(5*time.Minute))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", "Bearer "+token)
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
			record(t, tc.name, tc.method, tc.path, tc.status, req)
		})
	}
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	uploadCases := []struct {
		name, path, stage, reference, contentType string
		file                                      []byte
		status                                    int
	}{
		{"public-upload", "/v1/public/attachments", "", uploadCreated, "", []byte("%PDF-1.7 synthetic"), 201},
		{"public-upload-replay", "/v1/public/attachments", "", uploadReplayed, "", []byte("%PDF-1.7 synthetic"), 200},
		{"staff-upload", "/v1/staff/attachments", "ready", uploadCreated, "", []byte("%PDF-1.7 synthetic"), 201},
		{"upload-reference-conflict", "/v1/public/attachments", "", uploadConflict, "", []byte("%PDF-1.7 synthetic"), 409},
		{"upload-file-too-large", "/v1/public/attachments", "", uploadCreated, "", bytes.Repeat([]byte("x"), uploadusecase.MaxFileBytes+1), 413},
		{"upload-not-multipart", "/v1/public/attachments", "", uploadCreated, "application/pdf", nil, 415},
		{"upload-empty-file", "/v1/public/attachments", "", uploadCreated, "", []byte{}, 422},
		{"upload-content-mismatch", "/v1/public/attachments", "", uploadWrongType, "", png, 422},
		{"upload-rate-limited", "/v1/public/attachments", "", uploadLimited, "", []byte("%PDF-1.7 synthetic"), 429},
		{"staff-upload-without-create", "/v1/staff/attachments", "ready", uploadDenied, "", []byte("%PDF-1.7 synthetic"), 403},
		{"staff-upload-role-selection-pending", "/v1/staff/attachments", "role selection", uploadCreated, "", []byte("%PDF-1.7 synthetic"), 403},
	}
	for _, tc := range uploadCases {
		t.Run(tc.name, func(t *testing.T) {
			contentType, body := multipartUpload(t, tc.reference, tc.file)
			if tc.contentType != "" {
				contentType, body = tc.contentType, bytes.NewBufferString("%PDF-1.7 synthetic")
			}
			req := httptest.NewRequest("POST", tc.path, body)
			req.Header.Set("Content-Type", contentType)
			if tc.stage != "" {
				token, err := tokens.Sign(security.SessionClaims{SessionID: tc.stage, EmployeeID: 1, CredentialGeneration: 1, RoleRevision: 1}, now, now.Add(5*time.Minute))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", "Bearer "+token)
			}
			record(t, tc.name, "POST", tc.path, tc.status, req)
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
