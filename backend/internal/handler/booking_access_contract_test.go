package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image/png"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/piglig/go-qr/v2"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

type contractBookingAccess struct{}
type contractVisitAdmission struct{}

func (contractVisitAdmission) Issue(_ context.Context, id int32, input dto.PublicCancel) (dto.Admission, error) {
	if id != 101 {
		return dto.Admission{}, bookingrepo.ErrNotFound
	}
	if input.Version != 1 {
		return dto.Admission{}, failure.New(409, "version_conflict")
	}
	now := time.Date(2030, 2, 1, 1, 0, 0, 0, time.UTC)
	return dto.Admission{Payload: "vmsqr1." + strings.Repeat("a", 64), ExpiresAt: now.Add(2 * time.Hour), ServerTime: now.UnixMilli()}, nil
}

func contractPublicVisit(status string, version int) dto.PublicBooking {
	now := time.Date(2030, 2, 1, 1, 0, 0, 0, time.UTC)
	return bookingserializer.NewBookingSerializer().ToPublicBooking(entity.Visit{
		ID: 101, Code: "00123456", Kind: "umum", Status: status, Version: version, OfficerID: 2, OfficerName: "Official Example",
		Date: "2030-02-01", Time: "10:00", Duration: 60, Purpose: "Visit Example", Name: "Visitor Example", Organization: "Example",
		Email: "visitor@example.test", Phone: "08123456789", IdentityType: "KTP", Visitors: []string{"Visitor Example"},
	}, now, []entity.VisitHistory{{At: now, Action: "created"}})
}

func (contractBookingAccess) Lookup(_ context.Context, input dto.PublicLookup) (dto.PublicBooking, error) {
	if input.Email != "visitor@example.test" {
		return dto.PublicBooking{}, bookingrepo.ErrNotFound
	}
	return contractPublicVisit("approved", 1), nil
}

func (contractBookingAccess) Cancel(_ context.Context, id int32, input dto.PublicCancel, _ string) (dto.PublicBooking, error) {
	if id != 101 {
		return dto.PublicBooking{}, bookingrepo.ErrNotFound
	}
	if input.Version != 1 {
		return dto.PublicBooking{}, failure.New(409, "version_conflict")
	}
	return contractPublicVisit("cancelled", 2), nil
}

type contractCredentialLimit struct {
	used int
	ip   string
}

func (l *contractCredentialLimit) Admit(_ context.Context, purpose, ip string, limit int, window time.Duration) error {
	if l.ip == "" {
		l.ip = ip
	}
	if _, err := netip.ParseAddr(ip); err != nil || ip != l.ip || purpose != "public-credential" || limit != 30 || window != time.Minute {
		return failure.New(500, "internal_error")
	}
	if l.used >= limit {
		return &failure.Error{Status: 429, Code: "rate_limited", RetryAfter: 48}
	}
	l.used++
	return nil
}

func TestBookingAccessContractHTTP(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: func(c fiber.Ctx, err error) error {
		return restmiddleware.Failure(c, err, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}})
	app.Use(restmiddleware.RequestContext(10*time.Second, noop.NewTracerProvider().Tracer("access-contract")))
	// Lookup, cancel and QR admission together exhaust the remaining slots.
	limiter := &contractCredentialLimit{used: 18}
	router.RegisterPublicBookingAccessRoutes(app, contractAdmission{}, limiter, contractBookingAccess{})
	router.RegisterVisitAdmissionRoutes(app, contractAdmission{}, limiter, contractVisitAdmission{})
	type capture struct {
		Name    string            `json:"name"`
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		Body    any               `json:"body"`
		Bytes   int               `json:"bytes"`
	}
	const lookup = "/v1/public/lookup"
	const cancel = "/v1/public/bookings/101/cancel"
	const cancelTemplate = "/v1/public/bookings/{id}/cancel"
	const admission = "/v1/public/bookings/101/admission"
	const admissionTemplate = "/v1/public/bookings/{id}/admission"
	const proof = `{"code":"00123456","email":"visitor@example.test"}`
	const mutation = `{"code":"00123456","email":"visitor@example.test","version":1}`
	captures := []capture{}
	for _, tt := range []struct {
		name, url, path, body string
		status                int
	}{
		{"public-lookup", lookup, lookup, proof, 200},
		{"public-lookup-unavailable", lookup, lookup, `{"code":"00123456","email":"other@example.test"}`, 404},
		{"public-cancel", cancel, cancelTemplate, mutation, 200},
		{"public-cancel-unavailable", "/v1/public/bookings/102/cancel", cancelTemplate, mutation, 404},
		{"public-cancel-stale-version", cancel, cancelTemplate, `{"code":"00123456","email":"visitor@example.test","version":2}`, 409},
		{"public-cancel-extra-field", cancel, cancelTemplate, `{"code":"00123456","email":"visitor@example.test","version":1,"reason":"unexpected"}`, 400},
		{"public-cancel-invalid-id", "/v1/public/bookings/0101/cancel", cancelTemplate, mutation, 400},
		{"public-lookup-extra-query", lookup + "?debug=true", lookup, proof, 400},
		{"public-admission", admission, admissionTemplate, mutation, 200},
		{"public-admission-unavailable", "/v1/public/bookings/102/admission", admissionTemplate, mutation, 404},
		{"public-admission-stale-version", admission, admissionTemplate, `{"code":"00123456","email":"visitor@example.test","version":2}`, 409},
		{"public-lookup-last-slot", lookup, lookup, proof, 200},
		{"public-cancel-rate-limited", cancel, cancelTemplate, mutation, 429},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", tt.url, strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			response, err := app.Test(request, fiber.TestConfig{Timeout: 10 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			data, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != tt.status {
				t.Fatalf("expected %d, got %d: %s (%v)", tt.status, response.StatusCode, data, err)
			}
			if tt.status == 429 && response.Header.Get("Retry-After") != "48" {
				t.Fatal("429 lost its Retry-After header")
			}
			if tt.name == "public-admission" {
				var admission struct {
					dto.Admission
					ImagePNG []byte `json:"imagePng"`
				}
				if err := json.Unmarshal(data, &admission); err != nil {
					t.Fatal(err)
				}
				image, err := png.Decode(bytes.NewReader(admission.ImagePNG))
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := qr.Decode(image)
				if err != nil || decoded.Text != admission.Payload {
					t.Fatalf("branded PNG does not encode the issued credential: %v", err)
				}
			}
			record := capture{Name: tt.name, Method: "POST", Path: tt.path, Status: response.StatusCode, Bytes: len(data), Headers: map[string]string{}}
			for _, field := range []string{"Content-Type", "Cache-Control", "X-Content-Type-Options", "X-Request-ID", "Retry-After"} {
				record.Headers[field] = response.Header.Get(field)
			}
			if err := json.Unmarshal(data, &record.Body); err != nil {
				t.Fatal(err)
			}
			if offsets := nonUTCTimestamps(record.Body); len(offsets) != 0 {
				t.Fatalf("non-UTC timestamp: %v", offsets)
			}
			captures = append(captures, record)
		})
	}
	if target := os.Getenv("VMS_CONTRACT_CAPTURE"); target != "" {
		data, err := json.Marshal(captures)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
