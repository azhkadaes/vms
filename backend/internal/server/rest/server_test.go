package restserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
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
	uploadusecase "github.com/azhkadaes/vms/backend/internal/usecase/upload"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type testDependencies struct {
	Catalog   catalogusecase.CatalogUsecase
	Admission admissionusecase.AdmissionUsecase
	Uploads   uploadusecase.UploadUsecase
	Health    Health
	Logger    *slog.Logger
	Tracer    trace.Tracer
}

// Only tests replace usecases; runtime composition stays in PublicRouter.
func newTestApp(cfg config.Config, deps testDependencies) *fiber.App {
	app := newApp(cfg, deps.Health, deps.Logger, deps.Tracer)
	if deps.Uploads == nil {
		deps.Uploads = &uploads{}
	}
	router.RegisterPublicRoutes(app, deps.Catalog, deps.Admission, deps.Uploads)
	app.Use(func(c fiber.Ctx) error { return fiber.ErrNotFound })
	return app
}

type catalog struct {
	err  error
	wait bool
}

// servedOperations are the implemented operation numbers of the 40-operation contract
// (docs/contracts/manifest.json); servedPaths is how many distinct paths they use.
// Update both when a slice registers operations and regenerates docs/openapi.json.
var servedOperations = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}

const servedPaths = 34

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
		response, err := app.Test(request, fiber.TestConfig{Timeout: 10 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 401 {
			t.Fatalf("unprotected docs route: %s", path)
		}
		request.SetBasicAuth("docs", "private-password")
		response, err = app.Test(request, fiber.TestConfig{Timeout: 10 * time.Second})
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
				OpenAPI string `json:"openapi"`
				Paths   map[string]map[string]struct {
					Number int `json:"x-operation-number"`
				} `json:"paths"`
			}
			if json.Unmarshal(body, &document) != nil || document.OpenAPI != "3.1.1" || len(document.Paths) != servedPaths {
				t.Fatalf("runtime docs serve %d paths, want %d", len(document.Paths), servedPaths)
			}
			served := []int{}
			for _, item := range document.Paths {
				for _, operation := range item {
					served = append(served, operation.Number)
				}
			}
			slices.Sort(served)
			if !slices.Equal(served, servedOperations) {
				t.Fatalf("runtime docs serve contract operations %v, want %v", served, servedOperations)
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

func (s catalog) GetStaffCatalogs(context.Context, dto.Session) (dto.StaffCatalog, error) {
	return dto.StaffCatalog{}, errors.New("the public boundary must not read the staff catalog")
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

// The runtime composition must mount op 08 behind StaffAuth: an unauthenticated request is
// refused before any database read, while an unregistered staff path stays not found.
func TestRuntimeRoutesServeTheStaffCatalogBehindStaffAuthentication(t *testing.T) {
	cfg := config.Config{JSONMaxBytes: 262144, ExecutionTimeout: time.Second, Auth: &config.Auth{JWTKey: []byte(strings.Repeat("j", 32)), Issuer: "vms", Audience: "staff",
		CSRFKey: []byte(strings.Repeat("c", 32)), RecoveryKey: []byte(strings.Repeat("r", 32)), Origin: "https://staff.example.test", BcryptCost: 4, HashConcurrency: 1}}
	logger, tracer := slog.New(slog.NewTextHandler(io.Discard, nil)), noop.NewTracerProvider().Tracer("test")
	s := &RestServer{cfg: cfg, logger: logger, tracer: tracer}
	s.app = newApp(cfg, health{}, logger, tracer)
	s.RegisterRoutes()
	for path, want := range map[string]int{"/v1/staff/catalogs": 401, "/v1/staff/unregistered": 404} {
		response, err := s.app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != want {
			t.Errorf("%s returned %d, want %d", path, response.StatusCode, want)
		}
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

// uploads accepts every upload and remembers how many file bytes reached it.
type uploads struct{ received int }

func (u *uploads) UploadPublic(_ context.Context, _ string, upload entity.UploadRequest) (uploadusecase.Issued, error) {
	u.received = len(upload.Content)
	return uploadusecase.Issued{Created: true, Result: dto.UploadResult{Token: "vmsu1.synthetic-upload-token", ExpiresAt: time.Now().UTC().Add(time.Hour), ServerTime: time.Now().UnixMilli()}}, nil
}
func (u *uploads) UploadStaff(ctx context.Context, _ entity.SessionProof, ip string, upload entity.UploadRequest) (uploadusecase.Issued, error) {
	return u.UploadPublic(ctx, ip, upload)
}

func testDeps(u uploadusecase.UploadUsecase) testDependencies {
	return testDependencies{Catalog: catalog{}, Admission: &admission{decision: entity.RequestAdmission{Allowed: true}}, Uploads: u, Health: health{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Tracer: noop.NewTracerProvider().Tracer("test")}
}

func multipartBody(t *testing.T, size int) (string, *bytes.Buffer) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, err := writer.CreateFormFile("file", "nota dinas.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(bytes.Repeat([]byte("x"), size)); err != nil {
		t.Fatal(err)
	}
	_ = writer.WriteField("reference", "0f8fad5b-d9cb-469f-a165-70867728950e")
	_ = writer.WriteField("purpose", "document")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return writer.FormDataContentType(), body
}

// D66/D67: upload routes take a 5 MiB file while every other route keeps the 256 KiB JSON cap.
func TestOnlyUploadRoutesAcceptBodiesBeyondTheJSONCap(t *testing.T) {
	u := &uploads{}
	app := newTestApp(config.Config{JSONMaxBytes: 262144, ExecutionTimeout: 15 * time.Second}, testDeps(u))
	contentType, body := multipartBody(t, 5<<20)
	request := httptest.NewRequest("POST", "/v1/public/attachments", body)
	request.Header.Set("Content-Type", contentType)
	response, err := app.Test(request, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 201 || u.received != 5<<20 {
		t.Fatalf("a 5 MiB upload got %d with %d bytes received", response.StatusCode, u.received)
	}
	// Headers only: the JSON cap refuses the declared length before any body byte is read.
	if status, _ := headersOnly(t, app, "/v1/public/catalogs", 262145); status != 413 {
		t.Fatalf("a body over the JSON cap to a non-upload route got %d", status)
	}
}

// headersOnly sends a POST declaring length body bytes without sending any, and returns the
// status and request ID of the answer, which must come before the body is read.
func headersOnly(t *testing.T, app *fiber.App, path string, length int) (int, string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- app.Listener(listener) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.ShutdownWithContext(ctx)
		_ = listener.Close()
		<-stopped
	}()
	connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	head := fmt.Sprintf("POST %s HTTP/1.1\r\nHost: localhost\r\nContent-Type: multipart/form-data; boundary=x\r\nContent-Length: %d\r\nConnection: close\r\n\r\n", path, length)
	if _, err := io.WriteString(connection, head); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), nil)
	if err != nil {
		t.Fatalf("no early rejection: %v", err)
	}
	defer response.Body.Close()
	return response.StatusCode, response.Header.Get("X-Request-ID")
}

func TestAnUploadDeclaredOverItsCapIsRefusedBeforeItsBodyIsRead(t *testing.T) {
	app := newTestApp(config.Config{JSONMaxBytes: 262144, ExecutionTimeout: 15 * time.Second}, testDeps(&uploads{}))
	// One byte over the 6 MiB total upload cap.
	status, requestID := headersOnly(t, app, "/v1/public/attachments", 6<<20+1)
	if status != 413 {
		t.Fatalf("oversize upload status %d", status)
	}
	if _, err := uuid.Parse(requestID); err != nil {
		t.Fatal("early upload rejection lost its correlation ID")
	}
}
