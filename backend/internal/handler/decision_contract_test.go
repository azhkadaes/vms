package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	serializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
)

type contractDecisions struct{}

var decisionRole = dto.Session{Active: true, Role: policy.RoleProtocol, Permissions: []string{policy.PermBookings, policy.PermApprove, policy.PermReject, policy.PermCancel, policy.PermReschedule, policy.PermCreate}}

func decisionVisit(status, kind string) dto.StaffBooking {
	v := entity.StaffVisit{Visit: entity.Visit{ID: 1, Code: "00123456", OfficerID: 2, OfficerName: "Pejabat Contoh", Name: "Pengunjung Contoh",
		Organization: "Instansi Contoh", Email: "visitor@example.test", Phone: "081234567890", IdentityType: "KTP", Visitors: []string{"Pengunjung Contoh"},
		Kind: kind, Status: status, Version: 2, Date: "2030-01-03", Time: "10:00", Duration: 60, Purpose: "Pertemuan pengujian"}, VisitorCount: 1}
	return serializer.Project(v, decisionRole, time.Date(2030, 1, 1, 2, 0, 0, 0, time.UTC), false)
}

func (contractDecisions) Approve(_ context.Context, _ entity.SessionProof, _ int32, in dto.ApproveBooking) (dto.StaffBooking, error) {
	if in.Version != 1 {
		return dto.StaffBooking{}, failure.New(409, "version_conflict")
	}
	return decisionVisit("approved", "vip"), nil
}

func (contractDecisions) Reject(_ context.Context, _ entity.SessionProof, _ int32, in dto.RejectBooking) (dto.StaffBooking, error) {
	if in.Version != 1 {
		return dto.StaffBooking{}, failure.New(409, "version_conflict")
	}
	return decisionVisit("rejected", "vip"), nil
}

func (contractDecisions) Cancel(_ context.Context, _ entity.SessionProof, _ int32, in dto.CancelBooking) (dto.StaffBooking, error) {
	if in.Version != 1 {
		return dto.StaffBooking{}, failure.New(409, "version_conflict")
	}
	return decisionVisit("cancelled", "vip"), nil
}

func (contractDecisions) Reschedule(_ context.Context, _ entity.SessionProof, _ int32, in dto.RescheduleBooking) (dto.StaffBooking, error) {
	if in.Version != 1 {
		return dto.StaffBooking{}, failure.New(409, "version_conflict")
	}
	return decisionVisit("approved", "vip"), nil
}

func (contractDecisions) UpdateRestricted(_ context.Context, _ entity.SessionProof, _ int32, in dto.UpdateRestrictedBooking) (dto.StaffBooking, error) {
	if in.Version != 1 {
		return dto.StaffBooking{}, failure.New(409, "version_conflict")
	}
	return decisionVisit("pending", "terbatas"), nil
}

func TestStaffDecisionContract(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{CaseSensitive: true, StrictRouting: true, ErrorHandler: func(c fiber.Ctx, err error) error { return restmiddleware.Failure(c, err, logger) }})
	app.Use(restmiddleware.RequestContext(time.Second, noop.NewTracerProvider().Tracer("decision-test")))
	tokens := security.Tokens{Key: []byte(strings.Repeat("d", 32)), Issuer: "vms", Audience: "staff"}
	deps := &router.Dependencies{App: app, StaffAuth: restmiddleware.StaffAuth(tokens, contractAuthority{"decision": decisionRole}, contractLimiter{}), Workspace: restmiddleware.Workspace}
	router.RegisterDecisionRoutes(deps, contractDecisions{})
	now := time.Now()
	signed, err := tokens.Sign(security.SessionClaims{SessionID: "decision", EmployeeID: 1, CredentialGeneration: 1, RoleRevision: 1}, now, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	type capture struct {
		Name    string            `json:"name"`
		Path    string            `json:"path"`
		Method  string            `json:"method"`
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		Body    json.RawMessage   `json:"body"`
		Bytes   int               `json:"bytes"`
	}
	var records []capture
	for _, tc := range []struct {
		name, suffix, body string
		status             int
	}{
		{"staff-decision-approve", "approve", `{"version":1,"roomId":null}`, 200},
		{"staff-decision-reject", "reject", `{"version":1,"rejectionReason":"Agenda tidak tersedia"}`, 200},
		{"staff-decision-cancel", "cancel", `{"version":1,"reason":"Agenda dibatalkan"}`, 200},
		{"staff-decision-reschedule", "reschedule", `{"version":1,"date":"2030-01-03","time":"10:00","reason":"Agenda berubah"}`, 200},
		{"staff-decision-update", "update", `{"version":1,"officerId":"2","name":"","organization":"","serviceId":""}`, 200},
		{"staff-decision-stale", "approve", `{"version":2}`, 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/staff/bookings/1/"+tc.suffix, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+signed)
			req.Header.Set("Content-Type", "application/json")
			res, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil || !json.Valid(body) || res.StatusCode != tc.status {
				t.Fatalf("status %d, want %d, body %s, error %v", res.StatusCode, tc.status, body, err)
			}
			headers := map[string]string{}
			for _, name := range []string{"Content-Type", "Cache-Control", "X-Content-Type-Options", "X-Request-ID"} {
				headers[name] = res.Header.Get(name)
			}
			records = append(records, capture{tc.name, req.URL.Path, req.Method, res.StatusCode, headers, body, len(body)})
		})
	}
	if target := os.Getenv("VMS_CONTRACT_CAPTURE"); target != "" {
		data, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
