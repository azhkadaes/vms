package handler_test

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/handler"
)

// testFileCap stands in for the 5 MiB production cap so oversize cases stay small.
const testFileCap = 64

type part struct{ name, filename, value string }

func form(t *testing.T, parts ...part) (string, *bytes.Buffer) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, p := range parts {
		var w io.Writer
		var err error
		if p.filename != "" {
			w, err = writer.CreateFormFile(p.name, p.filename)
		} else {
			w, err = writer.CreateFormField(p.name)
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, p.value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return writer.FormDataContentType(), body
}

func uploadRoute(app *fiber.App) {
	app.Post("/v1/probe", func(c fiber.Ctx) error {
		upload, err := handler.Multipart(c, testFileCap)
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"size": len(upload.File), "content": string(upload.File), "filename": upload.Filename, "reference": upload.Reference, "purpose": upload.Purpose})
	})
}

func upload(t *testing.T, contentType string, body io.Reader) reply {
	t.Helper()
	request := httptest.NewRequest("POST", "/v1/probe", body)
	request.Header.Set("Content-Type", contentType)
	return do(t, transportApp(fiber.Config{}, uploadRoute), request)
}

const reference = "550e8400-e29b-41d4-a716-446655440000"

func TestMultipartReturnsTheFileAndTextParts(t *testing.T) {
	contentType, body := form(t, part{"reference", "", reference}, part{"file", "nota dinas.pdf", "%PDF-1.7 synthetic"}, part{"purpose", "", "document"})
	got := upload(t, contentType, body)
	if got.status != 200 || got.body["content"] != "%PDF-1.7 synthetic" || got.body["filename"] != "nota dinas.pdf" || got.body["reference"] != reference || got.body["purpose"] != "document" {
		t.Fatalf("got %d %s", got.status, got.raw)
	}
}

func TestMultipartReportsEveryMissingOrEmptyPartAsFieldErrors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		parts []part
		keys  []string
	}{
		{"only file", []part{{"file", "a.pdf", "%PDF"}}, []string{"purpose", "reference"}},
		{"no file", []part{{"reference", "", reference}, {"purpose", "", "document"}}, []string{"file"}},
		{"empty file and blank purpose", []part{{"file", "a.pdf", ""}, {"reference", "", reference}, {"purpose", "", ""}}, []string{"file", "purpose"}},
		{"nothing", nil, []string{"file", "purpose", "reference"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contentType, body := form(t, tc.parts...)
			got := upload(t, contentType, body)
			expectError(t, got, 422, "validation_failed")
			fields, _ := got.body["fieldErrors"].(map[string]any)
			keys := []string{}
			for _, key := range []string{"file", "purpose", "reference"} {
				if messages, ok := fields[key].([]any); ok && len(messages) > 0 {
					keys = append(keys, key)
				}
			}
			if !reflect.DeepEqual(keys, tc.keys) || len(fields) != len(tc.keys) {
				t.Fatalf("fieldErrors %v, want keys %v", fields, tc.keys)
			}
		})
	}
}

func TestMultipartRejectsAFileOverTheCapWith413(t *testing.T) {
	contentType, body := form(t, part{"file", "a.pdf", strings.Repeat("x", testFileCap+1)}, part{"reference", "", reference}, part{"purpose", "", "document"})
	expectError(t, upload(t, contentType, body), 413, "payload_too_large")
	contentType, body = form(t, part{"file", "a.pdf", strings.Repeat("x", testFileCap)}, part{"reference", "", reference}, part{"purpose", "", "document"})
	if got := upload(t, contentType, body); got.status != 200 || got.body["size"] != float64(testFileCap) {
		t.Fatalf("file at the cap: %d %s", got.status, got.raw)
	}
}

func TestMultipartRejectsWrongMediaUnknownDuplicateAndMalformedParts(t *testing.T) {
	expectError(t, upload(t, "application/json", strings.NewReader(`{"file":"x"}`)), 415, "unsupported_media_type")
	expectError(t, upload(t, "multipart/form-data", strings.NewReader("--x\r\n")), 415, "unsupported_media_type")
	contentType, body := form(t, part{"file", "a.pdf", "%PDF"}, part{"reference", "", reference}, part{"purpose", "", "document"}, part{"owner", "", "1"})
	expectError(t, upload(t, contentType, body), 400, "invalid_request")
	contentType, body = form(t, part{"file", "a.pdf", "%PDF"}, part{"reference", "", reference}, part{"purpose", "", "document"}, part{"purpose", "", "visitorPhoto"})
	expectError(t, upload(t, contentType, body), 400, "invalid_request")
	expectError(t, upload(t, "multipart/form-data; boundary=x", strings.NewReader("--x\r\nContent-Disposition: form-data; name=\"file\"\r\n\r\nunterminated")), 400, "invalid_request")
}

func TestMultipartRejectsAnOversizedDeclaredBodyBeforeReadingIt(t *testing.T) {
	// Streaming hands the handler the body before it is read; the server pre-reads at most BodyLimit bytes.
	app := transportApp(fiber.Config{StreamRequestBody: true, BodyLimit: 1024}, uploadRoute)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.ShutdownWithContext(ctx)
		_ = listener.Close()
		<-stopped
	})
	connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	// Declare 10 MiB but send only the 1 KiB prefix; reading any further would block until the deadline.
	head := "POST /v1/probe HTTP/1.1\r\nHost: localhost\r\nContent-Type: multipart/form-data; boundary=x\r\nContent-Length: 10485760\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(connection, head+strings.Repeat("x", 1024)); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), nil)
	if err != nil {
		t.Fatalf("no early rejection: %v", err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != 413 || !strings.Contains(string(raw), `"code":"payload_too_large"`) {
		t.Fatalf("got %d %s", response.StatusCode, raw)
	}
}
