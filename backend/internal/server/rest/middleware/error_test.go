package restmiddleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace/noop"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

type probe struct {
	status  int
	header  map[string][]string
	body    map[string]any
	raw     string
	logs    string
	request string
}

// send runs one request through the real error handler with a handler that returns err.
func send(t *testing.T, err error) probe {
	t.Helper()
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	app := fiber.New(fiber.Config{ErrorHandler: func(c fiber.Ctx, err error) error { return restmiddleware.Failure(c, err, logger) }})
	app.Use(restmiddleware.RequestContext(time.Second, noop.NewTracerProvider().Tracer("test")))
	app.Post("/v1/probe/:id", func(c fiber.Ctx) error { return err })
	request := httptest.NewRequest("POST", "/v1/probe/7?email=jane@example.test", strings.NewReader(`{"password":"hunter2-private-passphrase"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer secret-access-token")
	request.Header.Set("Cookie", "__Host-vms-refresh=secret-refresh-cookie")
	response, testErr := app.Test(request)
	if testErr != nil {
		t.Fatal(testErr)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	out := probe{status: response.StatusCode, header: response.Header, raw: string(raw), logs: logs.String(), request: response.Header.Get("X-Request-ID")}
	if err := json.Unmarshal(raw, &out.body); err != nil {
		t.Fatalf("error body is not JSON: %s", raw)
	}
	if out.body["requestId"] != out.request || out.request == "" {
		t.Fatalf("requestId %v does not match header %q", out.body["requestId"], out.request)
	}
	return out
}

func catalogueMessage(t *testing.T, code string) string {
	t.Helper()
	message, ok := failure.Message(code)
	if !ok {
		t.Fatalf("%s missing from catalogue", code)
	}
	return message
}

func TestFailureBodyCarriesFieldErrorArraysKeyedByFormKey(t *testing.T) {
	got := send(t, failure.Validation().
		Add("visitors", "Nama pengunjung wajib diisi.").
		Add("visitors", "Maksimal 10 pengunjung.").
		Add("time", "Pilih waktu mulai antara 08.00–16.00 WITA.").Err())
	if got.status != 422 {
		t.Fatalf("status %d", got.status)
	}
	keys := []string{}
	for key := range got.body {
		keys = append(keys, key)
	}
	if len(keys) != 4 || got.body["code"] != "validation_failed" || got.body["message"] != "Periksa isian yang ditandai." {
		t.Fatalf("body %s", got.raw)
	}
	want := map[string]any{"visitors": []any{"Nama pengunjung wajib diisi.", "Maksimal 10 pengunjung."}, "time": []any{"Pilih waktu mulai antara 08.00–16.00 WITA."}}
	if !reflect.DeepEqual(got.body["fieldErrors"], want) {
		t.Fatalf("fieldErrors %v", got.body["fieldErrors"])
	}
}

func TestHandBuiltFieldErrorsNeverEmitEmptyArraysOrMessages(t *testing.T) {
	got := send(t, &failure.Error{Status: 422, Code: "validation_failed", Fields: map[string][]string{"empty": {}, "blank": {""}, "note": {"", "Catatan terlalu panjang."}}})
	if !reflect.DeepEqual(got.body["fieldErrors"], map[string]any{"note": []any{"Catatan terlalu panjang."}}) {
		t.Fatalf("fieldErrors %v", got.body["fieldErrors"])
	}
	got = send(t, &failure.Error{Status: 422, Code: "validation_failed", Fields: map[string][]string{"empty": {}}})
	if _, present := got.body["fieldErrors"]; present || got.status != 422 {
		t.Fatalf("got %d %s", got.status, got.raw)
	}
}

func TestEachCatalogueCodeIsAnsweredWithItsOwnMessage(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{
		{400, "invalid_request"}, {400, "invalid_query_parameter"},
		{401, "unauthenticated"}, {401, "invalid_credentials"},
		{403, "forbidden"}, {403, "csrf_expired"},
		{404, "not_found"},
		{409, "version_conflict"}, {409, "invalid_state"}, {409, "schedule_conflict"}, {409, "reschedule_window_closed"},
		{409, "idempotency_conflict"}, {409, "notice_outdated"}, {409, "credential_expired"}, {409, "revision_conflict"}, {409, "role_changed"},
		{413, "payload_too_large"}, {415, "unsupported_media_type"},
		{422, "validation_failed"}, {422, "export_too_large"},
		{429, "rate_limited"},
		{503, "service_unavailable"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			got := send(t, failure.New(tc.status, tc.code))
			if got.status != tc.status || got.body["code"] != tc.code || got.body["message"] != catalogueMessage(t, tc.code) {
				t.Fatalf("got %d %s", got.status, got.raw)
			}
			if _, present := got.body["fieldErrors"]; present {
				t.Fatal("fieldErrors without fields")
			}
		})
	}
}

func TestConflictAndValidationStatusesNeverSurfaceAs500(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"bare fiber 409", fiber.NewError(409), 409, "invalid_state"},
		{"fiber 409 naming a catalogue code", fiber.NewError(409, "schedule_conflict"), 409, "schedule_conflict"},
		{"bare fiber 422", fiber.NewError(422), 422, "validation_failed"},
		{"fiber 409 with free text", fiber.NewError(409, "duplicate key value violates unique constraint"), 409, "invalid_state"},
		{"wrapped failure 409", fmt.Errorf("approve visit: %w", failure.New(409, "version_conflict")), 409, "version_conflict"},
		{"failure code outside catalogue", failure.New(409, "made_up_conflict"), 409, "invalid_state"},
		{"unknown client status", fiber.NewError(418), 400, "invalid_request"},
		{"invalid failure status", failure.New(200, "validation_failed"), 500, "internal_error"},
		{"unauthenticated sentinel", fmt.Errorf("session: %w", authusecase.ErrUnauthenticated), 401, "unauthenticated"},
		{"role changed sentinel", authusecase.ErrRoleChanged, 409, "role_changed"},
		{"workspace sentinel", authusecase.ErrWorkspaceBlocked, 403, "forbidden"},
		{"deadline", fmt.Errorf("catalog: %w", context.DeadlineExceeded), 503, "service_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := send(t, tc.err)
			if got.status != tc.status || got.body["code"] != tc.code {
				t.Fatalf("got %d %s", got.status, got.raw)
			}
			if message, _ := got.body["message"].(string); message == "" || strings.Contains(message, "duplicate") {
				t.Fatalf("message %q", message)
			}
		})
	}
}

func TestRepositoryNotFoundIsANonDisclosing404(t *testing.T) {
	notFound := fmt.Errorf("booking: %w", failure.ErrNotFound)
	got := send(t, fmt.Errorf("lock visit 7 for jane@example.test: %w", notFound))
	if got.status != 404 || got.body["code"] != "not_found" || got.body["message"] != catalogueMessage(t, "not_found") {
		t.Fatalf("got %d %s", got.status, got.raw)
	}
	if strings.Contains(got.raw, "jane") || strings.Contains(got.raw, "visit 7") {
		t.Fatalf("cause leaked: %s", got.raw)
	}
	if got.logs != "" {
		t.Fatalf("4xx logged as failure: %s", got.logs)
	}
}

func TestRetryAfterIsWholeSeconds(t *testing.T) {
	got := send(t, &failure.Error{Status: 429, Code: "rate_limited", RetryAfter: 12})
	if got.status != 429 || got.header["Retry-After"][0] != "12" {
		t.Fatalf("got %d %v", got.status, got.header["Retry-After"])
	}
}

// pgError mimics a driver error: SQLSTATE plus a message that quotes SQL and data.
type pgError struct{ code string }

func (e *pgError) Error() string {
	return "ERROR: duplicate key in INSERT INTO vms.pengunjung (email) VALUES ('jane@example.test')"
}
func (e *pgError) SQLState() string { return e.code }

func TestServerFailuresLogRedactedCauseClassOnly(t *testing.T) {
	secrets := []string{"jane@example.test", "INSERT INTO", "secret-access-token", "secret-refresh-cookie", "hunter2", "token=abc", "10.0.0.5"}
	for _, tc := range []struct {
		name, code, cause, sqlState string
		status                      int
		err                         error
	}{
		{"driver error", "internal_error", "pgError", "23505", 500, fmt.Errorf("insert visitor jane@example.test token=abc: %w", &pgError{code: "23505"})},
		{"unavailable with cause", "service_unavailable", "context.deadlineExceededError", "", 503, failure.New(503, "service_unavailable").WithCause(fmt.Errorf("dial 10.0.0.5: %w", context.DeadlineExceeded))},
		{"plain error", "internal_error", "errors.errorString", "", 500, errors.New("token=abc for jane@example.test")},
		{"hostile sqlstate", "internal_error", "pgError", "", 500, &pgError{code: "x' OR jane@example.test"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := send(t, tc.err)
			if got.status != tc.status || got.body["code"] != tc.code || got.body["message"] != catalogueMessage(t, tc.code) {
				t.Fatalf("got %d %s", got.status, got.raw)
			}
			lines := strings.Split(strings.TrimSpace(got.logs), "\n")
			if len(lines) != 1 {
				t.Fatalf("want one log line, got %q", got.logs)
			}
			var line map[string]any
			if err := json.Unmarshal([]byte(lines[0]), &line); err != nil {
				t.Fatal(err)
			}
			if line["request_id"] != got.request || line["status"] != float64(tc.status) || line["code"] != tc.code {
				t.Fatalf("log line %s", lines[0])
			}
			if cause, _ := line["cause"].(string); !strings.Contains(cause, tc.cause) {
				t.Fatalf("cause class %q, want %q", cause, tc.cause)
			}
			if sqlState, _ := line["sqlstate"].(string); sqlState != tc.sqlState {
				t.Fatalf("sqlstate %q", sqlState)
			}
			for _, secret := range secrets {
				if strings.Contains(got.logs, secret) || strings.Contains(got.raw, secret) {
					t.Fatalf("%q leaked: log %s body %s", secret, got.logs, got.raw)
				}
			}
		})
	}
}
