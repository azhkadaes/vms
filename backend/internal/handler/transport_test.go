package handler_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace/noop"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
)

// transportApp mounts probe routes behind the real request context and error handler,
// with multipart pre-parsing disabled as in the production server.
func transportApp(config fiber.Config, register func(app *fiber.App)) *fiber.App {
	config.DisablePreParseMultipartForm = true
	config.ErrorHandler = func(c fiber.Ctx, err error) error {
		return restmiddleware.Failure(c, err, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
	app := fiber.New(config)
	app.Use(restmiddleware.RequestContext(time.Second, noop.NewTracerProvider().Tracer("transport")))
	register(app)
	return app
}

type reply struct {
	status int
	header http.Header
	body   map[string]any
	raw    string
}

func do(t *testing.T, app *fiber.App, request *http.Request) reply {
	t.Helper()
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	out := reply{status: response.StatusCode, header: response.Header, raw: string(raw)}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out.body); err != nil {
			t.Fatalf("not JSON: %s", raw)
		}
	}
	return out
}

func expectError(t *testing.T, got reply, status int, code string) {
	t.Helper()
	if got.status != status || got.body["code"] != code {
		t.Fatalf("got %d %s, want %d %s", got.status, got.raw, status, code)
	}
}
