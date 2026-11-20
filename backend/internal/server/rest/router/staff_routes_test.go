package router_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/config"
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
)

const (
	staffOrigin = "https://staff.example.test"
	sessionID   = "550e8400-e29b-41d4-a716-446655440000"
)

var (
	jwtKey  = []byte(strings.Repeat("j", 32))
	csrfKey = []byte(strings.Repeat("c", 32))
	tokens  = security.Tokens{Key: jwtKey, Issuer: "vms", Audience: "staff"}
)

// The sessions a Bearer token can stand for, keyed by the token's session ID.
var stageSessions = map[string]dto.Session{
	"ready": {Active: true, ID: "7", Name: "Staff Example", Role: policy.RolePA, AvailableRoles: []string{policy.RolePA, policy.RoleProtocol},
		RoleRevision: 1, Permissions: []string{policy.PermBookings}, AssignedOfficerIDs: []string{"1"}, PasswordMinLength: 15, PasswordMaxBytes: 72},
	"role selection": {Active: true, ID: "7", Name: "Staff Example", AvailableRoles: []string{policy.RolePA, policy.RoleProtocol}, RoleSelectionRequired: true,
		RoleRevision: 1, Permissions: []string{}, AssignedOfficerIDs: []string{}, PasswordMinLength: 15, PasswordMaxBytes: 72},
	"password change": {Active: true, ID: "7", Name: "Staff Example", Role: policy.RolePA, AvailableRoles: []string{policy.RolePA}, MustChangePassword: true,
		RoleRevision: 1, Permissions: []string{}, AssignedOfficerIDs: []string{}, PasswordMinLength: 15, PasswordMaxBytes: 72},
}

// authority stands in for the live session read; it serves the session named by the token.
type authority struct{}

func (authority) GetSession(_ context.Context, proof entity.SessionProof) (dto.Session, error) {
	session, ok := stageSessions[proof.SessionID]
	if !ok {
		return dto.Session{}, authusecase.ErrUnauthenticated
	}
	return session, nil
}

type limiter struct{ admitted []string }

func (l *limiter) Admit(_ context.Context, purpose, identity string, _ int, _ time.Duration) error {
	l.admitted = append(l.admitted, purpose+":"+identity)
	return nil
}

// lifecycle records which session usecase methods ran, so blocked requests can prove they reached none.
type lifecycle struct{ calls []string }

func (l *lifecycle) result() authusecase.CredentialResult {
	return authusecase.CredentialResult{Result: dto.TokenResult{AccessToken: "synthetic-access-token", TokenType: "Bearer", AccessExpiresAt: time.Now().UTC().Add(5 * time.Minute), Session: stageSessions["ready"]},
		Refresh: strings.Repeat("r", 43), SessionID: sessionID, AbsoluteExpiresAt: time.Now().Add(24 * time.Hour)}
}
func (l *lifecycle) Login(context.Context, dto.Login, string) (authusecase.CredentialResult, error) {
	l.calls = append(l.calls, "Login")
	return l.result(), nil
}
func (l *lifecycle) Refresh(context.Context, string, string) (authusecase.CredentialResult, error) {
	l.calls = append(l.calls, "Refresh")
	return l.result(), nil
}
func (l *lifecycle) Logout(context.Context, string) error {
	l.calls = append(l.calls, "Logout")
	return nil
}
func (l *lifecycle) Renew(context.Context, entity.SessionProof) (dto.Session, error) {
	l.calls = append(l.calls, "Renew")
	return stageSessions["ready"], nil
}
func (l *lifecycle) SwitchRole(context.Context, entity.SessionProof, string) (authusecase.CredentialResult, error) {
	l.calls = append(l.calls, "SwitchRole")
	return l.result(), nil
}
func (l *lifecycle) ChangePassword(context.Context, entity.SessionProof, dto.PasswordChange) (authusecase.CredentialResult, error) {
	l.calls = append(l.calls, "ChangePassword")
	return l.result(), nil
}
func (l *lifecycle) Binding(context.Context, string, bool) (string, time.Time, error) {
	return sessionID, time.Now().Add(24 * time.Hour), nil
}

func newApp() *fiber.App {
	app := fiber.New(fiber.Config{CaseSensitive: true, StrictRouting: true, ErrorHandler: func(c fiber.Ctx, err error) error {
		return restmiddleware.Failure(c, err, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}})
	app.Use(restmiddleware.RequestContext(time.Second, noop.NewTracerProvider().Tracer("router-test")))
	return app
}

func authConfig() *config.Config {
	return &config.Config{Auth: &config.Auth{JWTKey: jwtKey, Issuer: "vms", Audience: "staff", CSRFKey: csrfKey, Origin: staffOrigin}}
}

// probe mounts a business group the way every staff router must: from the router dependencies.
func probe(deps *router.Dependencies, prefix string) {
	group := deps.App.Group(prefix, deps.StaffAuth, deps.Workspace)
	group.Get("/items", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"session": restmiddleware.Session(c).ID}) })
}

// staffApp registers business groups (a probe and the real staff routes) before the session
// routes and one probe after.
func staffApp() (*fiber.App, *lifecycle, *limiter) {
	app, sessions, limits := newApp(), &lifecycle{}, &limiter{}
	deps := &router.Dependencies{App: app, Cfg: authConfig(), StaffAuth: restmiddleware.StaffAuth(tokens, authority{}, limits), Workspace: restmiddleware.Workspace}
	probe(deps, "/v1/staff/before")
	router.RegisterStaffRoutes(deps, &catalogs{})
	router.RegisterAuthRoutes(deps, sessions)
	probe(deps, "/v1/staff/after")
	app.Use(func(c fiber.Ctx) error { return fiber.ErrNotFound })
	return app, sessions, limits
}

func bearer(t *testing.T, session string) string {
	t.Helper()
	now := time.Now()
	token, err := tokens.Sign(security.SessionClaims{SessionID: session, EmployeeID: 7, CredentialGeneration: 1, RoleRevision: 1}, now, now.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + token
}

type reply struct {
	status int
	code   string
}

func send(t *testing.T, app *fiber.App, request *http.Request) reply {
	t.Helper()
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var body struct {
		Code string `json:"code"`
	}
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode >= 400 {
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("error without JSON body: %s", raw)
		}
	}
	return reply{response.StatusCode, body.Code}
}

func TestStaffGroupsFromTheRouterDependenciesAdmitOnlyAReadyWorkspace(t *testing.T) {
	for _, prefix := range []string{"/v1/staff/before", "/v1/staff/after"} {
		for _, tc := range []struct {
			name, authorization string
			want                reply
		}{
			{"unauthenticated", "", reply{401, "unauthenticated"}},
			{"malformed bearer", "Bearer not-a-token", reply{401, "unauthenticated"}},
			{"unknown session", "unknown", reply{401, "unauthenticated"}},
			{"role selection pending", "role selection", reply{403, "forbidden"}},
			{"password change pending", "password change", reply{403, "forbidden"}},
			{"ready", "ready", reply{200, ""}},
		} {
			t.Run(prefix+"/"+tc.name, func(t *testing.T) {
				app, _, limits := staffApp()
				request := httptest.NewRequest("GET", prefix+"/items", nil)
				switch {
				case strings.HasPrefix(tc.authorization, "Bearer "):
					request.Header.Set("Authorization", tc.authorization)
				case tc.authorization != "":
					request.Header.Set("Authorization", bearer(t, tc.authorization))
				}
				if got := send(t, app, request); got != tc.want {
					t.Fatalf("got %+v, want %+v", got, tc.want)
				}
				if tc.want.status != 401 && !slices.Equal(limits.admitted, []string{"staff:7"}) {
					t.Fatalf("per-employee limit not applied once: %v", limits.admitted)
				}
			})
		}
	}
}

// A staff group registered before the session routes must neither lose its protection nor
// put the cookie-bound login, refresh and logout behind a Bearer token.
func TestSessionLifecycleRoutesStayOpenAroundProtectedStaffGroups(t *testing.T) {
	now := time.Now()
	anonymousBody := strings.Repeat("a", 43) + "." + strconv.FormatInt(now.Add(20*time.Minute).Unix(), 10)
	anonymous := anonymousBody + "." + security.Digest(csrfKey, "anonymous", anonymousBody)
	for _, tc := range []struct {
		name, method, path, body, binding, cookie, call string
		status                                          int
	}{
		{"login", "POST", "/v1/staff/sessions", `{"email":"staff@example.test","password":"synthetic-password"}`, "anonymous:" + anonymous, "__Host-vms-anonymous=" + anonymous, "Login", 200},
		{"refresh", "POST", "/v1/staff/session/refresh", `{"refreshRequestId":"` + sessionID + `"}`, "session:" + sessionID, "__Host-vms-refresh=" + strings.Repeat("r", 43), "Refresh", 200},
		{"logout", "DELETE", "/v1/staff/session", "", "session:" + sessionID, "__Host-vms-refresh=" + strings.Repeat("r", 43), "Logout", 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, sessions, _ := staffApp()
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			csrf, err := (security.CSRF{Key: csrfKey}).Sign(tc.binding, now, now.Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Origin", staffOrigin)
			request.Header.Set("Cookie", tc.cookie+"; __Host-vms-csrf="+csrf)
			request.Header.Set("X-CSRF-Token", csrf)
			if got := send(t, app, request); got.status != tc.status {
				t.Fatalf("got %+v, want %d", got, tc.status)
			}
			if !slices.Equal(sessions.calls, []string{tc.call}) {
				t.Fatalf("usecase calls %v", sessions.calls)
			}
		})
	}
}

func TestPendingSessionsCallOnlyTheirA18SessionOperations(t *testing.T) {
	type operation struct {
		number             int
		method, path, body string
		call               string
	}
	operations := []operation{
		{33, "GET", "/v1/staff/session", "", ""},
		{34, "POST", "/v1/staff/session/role", `{"role":"PA"}`, "SwitchRole"},
		{36, "POST", "/v1/staff/password", `{"currentPassword":"old-synthetic-password","newPassword":"new-synthetic-password"}`, "ChangePassword"},
		{40, "POST", "/v1/staff/session/renew", `{}`, "Renew"},
	}
	// A-18: 35 (logout) and 39 (refresh) are cookie-bound and open to every stage.
	allowed := map[string][]int{
		"role selection":  {33, 34, 40},
		"password change": {33, 36, 40},
		"ready":           {33, 34, 36, 40},
	}
	for stage, numbers := range allowed {
		for _, op := range operations {
			t.Run(stage+"/"+strconv.Itoa(op.number), func(t *testing.T) {
				app, sessions, _ := staffApp()
				request := httptest.NewRequest(op.method, op.path, strings.NewReader(op.body))
				if op.body != "" {
					request.Header.Set("Content-Type", "application/json")
				}
				request.Header.Set("Authorization", bearer(t, stage))
				got := send(t, app, request)
				if slices.Contains(numbers, op.number) {
					if got.status != 200 {
						t.Fatalf("allowed operation refused: %+v", got)
					}
					if op.call != "" && !slices.Equal(sessions.calls, []string{op.call}) {
						t.Fatalf("usecase calls %v", sessions.calls)
					}
					return
				}
				if got != (reply{403, "forbidden"}) || len(sessions.calls) != 0 {
					t.Fatalf("blocked operation got %+v and reached %v", got, sessions.calls)
				}
			})
		}
	}
	for _, op := range operations {
		t.Run("unauthenticated/"+strconv.Itoa(op.number), func(t *testing.T) {
			app, sessions, _ := staffApp()
			request := httptest.NewRequest(op.method, op.path, strings.NewReader(op.body))
			if op.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			if got := send(t, app, request); got != (reply{401, "unauthenticated"}) || len(sessions.calls) != 0 {
				t.Fatalf("got %+v, calls %v", got, sessions.calls)
			}
		})
	}
}

func TestRuntimeDependenciesFailClosedWithoutStaffAuthentication(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("router-test")
	for _, tc := range []struct {
		name string
		cfg  *config.Config
		want reply
	}{
		{"not configured", &config.Config{}, reply{503, "service_unavailable"}},
		// Token verification fails before any database read, so no pool is needed here.
		{"configured", authConfig(), reply{401, "unauthenticated"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newApp()
			deps := router.NewDependencies(app, nil, tc.cfg, tracer)
			probe(deps, "/v1/staff/probe")
			app.Use(func(c fiber.Ctx) error { return fiber.ErrNotFound })
			if got := send(t, app, httptest.NewRequest("GET", "/v1/staff/probe/items", nil)); got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
