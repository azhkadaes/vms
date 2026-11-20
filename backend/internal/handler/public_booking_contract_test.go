package handler_test

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/entity"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

type contractPublicBooking struct{}

func contractReceipt() dto.PublicBooking {
	v := entity.Visit{
		ID: 1, Code: "00123456", OfficerID: 1, OfficerName: "Official Example", Kind: "umum",
		Status: "pending", Version: 1, Date: "2030-01-02", Time: "10:00", Duration: 60,
		Purpose: "Pertemuan pengujian", Name: "Visitor Example", Organization: "Example Office",
		Email: "visitor@example.test", Phone: "081234567890", IdentityType: "KTP", Visitors: []string{"Visitor Example"},
	}
	return bookingserializer.NewBookingSerializer().ToPublicBooking(v, time.Date(2030, 1, 1, 2, 0, 0, 0, time.UTC), nil)
}

func (contractPublicBooking) Create(_ context.Context, _ string, in dto.PublicBookingCreate) (dto.PublicBooking, bool, error) {
	if in.VisitorPhotoToken == "" {
		return dto.PublicBooking{}, false, failure.Validation().Add("visitorPhotoToken", "Unggah foto pengunjung.").Err()
	}
	return contractReceipt(), in.Purpose != "Repeated submission", nil
}

func (contractPublicBooking) Resolve(_ context.Context, in dto.PublicSubmissionProof) (dto.PublicSubmissionResolution, error) {
	if in.Email == "unknown@example.test" {
		return dto.PublicSubmissionResolution{Outcome: "not_committed"}, nil
	}
	visit := contractReceipt()
	return dto.PublicSubmissionResolution{Outcome: "committed", Booking: &visit}, nil
}

type submissionBudget struct {
	byKey map[string]int
}

func (b submissionBudget) Admit(_ context.Context, purpose, ip string, limit int, window time.Duration) error {
	if limit != 30 || window != time.Minute {
		return failure.New(500, "internal_error")
	}
	key := purpose + ":" + ip
	b.byKey[key]++
	if b.byKey[key] > 1 {
		return &failure.Error{Status: 429, Code: "rate_limited", RetryAfter: 60}
	}
	return nil
}

func TestPublicSubmissionRoutesShareBudgetBeforeDecoding(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: func(c fiber.Ctx, err error) error {
		return restmiddleware.Failure(c, err, logger)
	}})
	router.RegisterPublicBookingRoutes(app, contractAdmission{}, submissionBudget{byKey: map[string]int{}}, contractPublicBooking{})
	for _, tc := range []struct {
		path string
		code int
	}{
		{"/v1/public/bookings", 400}, // Malformed submissions still spend the shared budget.
		{"/v1/public/submissions/resolve", 429},
	} {
		req := httptest.NewRequest("POST", tc.path, strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		res, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != tc.code || tc.code == 429 && res.Header.Get("Retry-After") != "60" {
			t.Fatalf("%s: status=%d retry=%q", tc.path, res.StatusCode, res.Header.Get("Retry-After"))
		}
	}
}
