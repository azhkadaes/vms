package restserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/config"
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission"
	catalogusecase "github.com/azhkadaes/vms/backend/internal/usecase/catalog"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type testDependencies struct {
	Catalog   catalogusecase.CatalogUsecase
	Admission admissionusecase.AdmissionUsecase
	Health    Health
	Logger    *slog.Logger
	Tracer    trace.Tracer
}

// Only tests replace usecases; runtime composition stays in PublicRouter.
func newTestApp(cfg config.Config, deps testDependencies) *fiber.App {
	app := newApp(cfg, deps.Health, deps.Logger, deps.Tracer)
	router.RegisterPublicRoutes(app, deps.Catalog, deps.Admission)
	app.Use(func(c fiber.Ctx) error { return fiber.ErrNotFound })
	return app
}

type catalog struct {
	err  error
	wait bool
}

func TestSwaggerProtectsUIContractAndAssets(t *testing.T) {
	deps := testDependencies{Catalog: catalog{}, Admission: &admission{decision: entity.RequestAdmission{Allowed: true}}, Health: health{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Tracer: noop.NewTracerProvider().Tracer("test")}
	cfg := config.Config{JSONMaxBytes: 262144, ExecutionTimeout: time.Second}
	disabled := newTestApp(cfg, deps)
	response, err := disabled.Test(httptest.NewRequest("GET", "/swagger/index.html", nil))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 404 {
		t.Fatal("Swagger enabled without opt-in")
	}
	cfg.SwaggerEnabled, cfg.SwaggerUsername, cfg.SwaggerPassword = true, "docs", "private-password"
	app := newTestApp(cfg, deps)
	for _, path := range []string{"/swagger/index.html", "/swagger/openapi.json", "/swagger/swagger-ui-bundle.js", "/swagger/swagger-ui.css"} {
		request := httptest.NewRequest("GET", path, nil)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 401 {
			t.Fatalf("unprotected docs route: %s", path)
		}
		request.SetBasicAuth("docs", "private-password")
		response, err = app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || response.StatusCode != 200 {
			t.Fatalf("docs route unavailable: %s", path)
		}
		if path == "/swagger/openapi.json" {
			var document struct {
				OpenAPI string         `json:"openapi"`
				Paths   map[string]any `json:"paths"`
			}
			if json.Unmarshal(body, &document) != nil || document.OpenAPI != "3.1.1" || len(document.Paths) != 7 || document.Paths["/public/catalogs"] == nil || document.Paths["/staff/sessions"] == nil || document.Paths["/staff/session"] == nil {
				t.Fatal("runtime docs advertise unimplemented or legacy operations")
			}
		}
	}
}

func (s catalog) GetPublicCatalogs(ctx context.Context) (dto.PublicCatalog, error) {
	if _, ok := ctx.Deadline(); !ok {
		return dto.PublicCatalog{}, errors.New("missing request deadline")
	}
	if s.wait {
		<-ctx.Done()
		return dto.PublicCatalog{}, ctx.Err()
	}
	return dto.PublicCatalog{Officers: []dto.CatalogOfficer{}, Rooms: []dto.CatalogRoom{}}, s.err
}

type admission struct {
	decision entity.RequestAdmission
	err      error
	calls    int
	ip       string
}

func (s *admission) PublicRequest(ctx context.Context, ip string) (entity.RequestAdmission, error) {
	s.calls++
	s.ip = ip
	return s.decision, s.err
}

type health struct{ err error }

func (s health) PingContext(context.Context) error { return s.err }

func TestPublicBoundaryAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name, path                 string
		code                       int
		limiterError, catalogError bool
		denied                     bool
	}{
		{"catalog", "/v1/public/catalogs", 200, false, false, false},
		{"unknown query", "/v1/public/catalogs?role=Super+Admin", 400, false, false, false},
		{"template route not exposed", "/v1/users", 404, false, false, false},
		{"staff endpoint unavailable", "/v1/staff/catalogs", 404, false, false, false},
		{"limiter fails closed", "/v1/public/catalogs", 503, true, false, false},
		{"catalog failure is not empty success", "/v1/public/catalogs", 500, false, true, false},
		{"throttled", "/v1/public/catalogs", 429, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := &admission{decision: entity.RequestAdmission{Allowed: !tc.denied, RetryAfter: 12}}
			var queryError error
			if tc.limiterError {
				limiter.err = errors.New("private-password in SELECT statement")
			}
			if tc.catalogError {
				queryError = errors.New("private-password in SELECT statement")
			}
			cfg := config.Config{JSONMaxBytes: 262144, ExecutionTimeout: 15 * time.Second}
			app := newTestApp(cfg, testDependencies{Catalog: catalog{err: queryError}, Admission: limiter, Health: health{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Tracer: noop.NewTracerProvider().Tracer("test")})
			request := httptest.NewRequest("GET", tc.path, nil)
			request.Header.Set("X-Forwarded-For", "203.0.113.99")
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tc.code {
				t.Fatalf("status %d, want %d: %s", response.StatusCode, tc.code, body)
			}
			if strings.Contains(string(body), "private-password") || strings.Contains(string(body), "SELECT") {
				t.Fatal("internal error leaked")
			}
			if limiter.ip == "203.0.113.99" {
				t.Fatal("untrusted forwarding header changed rate-limit identity")
			}
			if response.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("private/error response cacheable")
			}
			if tc.code == 200 {
				var result map[string]any
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatal(err)
				}
				if result["officers"] == nil || result["rooms"] == nil || result["data"] != nil || result["privacyNotice"] != nil {
					t.Fatal("wrong direct response/absence semantics")
				}
				if _, err := uuid.Parse(response.Header.Get("X-Request-ID")); err != nil {
					t.Fatal("invalid correlation ID")
				}
			}
			if tc.denied && response.Header.Get("Retry-After") != "12" {
				t.Fatal("missing retry cooldown")
			}
		})
	}
}

func TestHealthDoesNotDependOnLimiter(t *testing.T) {
	limiter := &admission{err: errors.New("unavailable")}
	app := newTestApp(config.Config{JSONMaxBytes: 262144, ExecutionTimeout: 15 * time.Second}, testDependencies{Catalog: catalog{}, Admission: limiter, Health: health{err: errors.New("database offline")}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Tracer: noop.NewTracerProvider().Tracer("test")})
	for path, want := range map[string]int{"/health/live": 200, "/health/ready": 503} {
		response, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != want {
			t.Errorf("%s returned %d", path, response.StatusCode)
		}
	}
	if limiter.calls != 0 {
		t.Fatal("health consumed limiter allowance")
	}
}

func TestCatalogReceivesCancellableDeadline(t *testing.T) {
	app := newTestApp(config.Config{JSONMaxBytes: 262144, ExecutionTimeout: 5 * time.Millisecond}, testDependencies{Catalog: catalog{wait: true}, Admission: &admission{decision: entity.RequestAdmission{Allowed: true}}, Health: health{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Tracer: noop.NewTracerProvider().Tracer("test")})
	response, err := app.Test(httptest.NewRequest("GET", "/v1/public/catalogs", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 503 {
		t.Fatal("deadline not propagated to use case")
	}
}

func TestOversizedRequestStillHasSafeCorrelatedError(t *testing.T) {
	app := newTestApp(config.Config{JSONMaxBytes: 262144, ExecutionTimeout: 15 * time.Second}, testDependencies{Catalog: catalog{}, Admission: &admission{decision: entity.RequestAdmission{Allowed: true}}, Health: health{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Tracer: noop.NewTracerProvider().Tracer("test")})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- app.Listener(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.ShutdownWithContext(ctx)
		_ = listener.Close()
		select {
		case <-stopped:
		case <-ctx.Done():
			t.Error("test listener did not stop")
		}
	})
	// Send headers only: racing a body write against early rejection can reset Windows TCP.
	connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(connection, "GET /v1/public/catalogs HTTP/1.1\r\nHost: localhost\r\nContent-Length: 262145\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 413 {
		t.Fatalf("oversize status %d", response.StatusCode)
	}
	if _, err := uuid.Parse(response.Header.Get("X-Request-ID")); err != nil {
		t.Fatal("early transport failure lost correlation ID")
	}
}
