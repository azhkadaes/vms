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
	reportserializer "github.com/azhkadaes/vms/backend/internal/serializer/report/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	"github.com/azhkadaes/vms/backend/internal/server/rest/router"
	reportusecase "github.com/azhkadaes/vms/backend/internal/usecase/report/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
)

type contractReportReader struct{ source entity.ReportSnapshot }

func (r contractReportReader) Summary(context.Context, dto.Session, dto.SummaryFilter) ([]entity.ReportStatus, time.Time, error) {
	rows := make([]entity.ReportStatus, 0, len(r.source.Rows))
	for _, row := range r.source.Rows {
		if row.Kind != "restricted" {
			rows = append(rows, row.ReportStatus)
		}
	}
	return rows, r.source.At, nil
}

func (r contractReportReader) VisitLog(context.Context, dto.VisitLogFilter) (entity.ReportSnapshot, error) {
	return r.source, nil
}

func TestReportContract(t *testing.T) {
	wita := time.FixedZone("WITA", 8*3600)
	at := time.Date(2030, 1, 2, 11, 1, 0, 0, wita)
	room, visitorCount := int32(2), 1
	rows := []entity.ReportVisit{
		{ReportStatus: entity.ReportStatus{ID: 1, Version: 1, Status: "approved", StartsAt: at.Add(-61 * time.Minute), Duration: 60},
			Reference: "00123456", Name: "Visitor Example", Organization: "Example", OfficerID: 1, Host: "Official Example",
			Date: "2030-01-02", Time: "10:00", RoomID: &room, Room: "Meeting room", Purpose: "Meeting", Kind: "standard", Visitors: &visitorCount},
		{ReportStatus: entity.ReportStatus{ID: 2, Version: 2, Status: "checked_out", StartsAt: at.Add(-3 * time.Hour), Duration: 30},
			Reference: "TRB-2", Name: "Kunjungan terbatas", OfficerID: 1, Host: "Official Example",
			Date: "2030-01-02", Time: "08:01", RoomID: &room, Room: "Meeting room", Kind: "restricted", CheckedInAt: &at, CheckedOutAt: &at},
	}
	tracer := noop.NewTracerProvider().Tracer("report-contract")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{CaseSensitive: true, StrictRouting: true, ErrorHandler: func(c fiber.Ctx, err error) error {
		return restmiddleware.Failure(c, err, logger)
	}})
	app.Use(restmiddleware.RequestContext(10*time.Second, tracer))
	admin := dto.Session{Active: true, ID: "1", Role: policy.RoleSuperAdmin, Permissions: []string{policy.PermReports, policy.PermReportExport}}
	noExport := admin
	noExport.Permissions = []string{policy.PermReports}
	protocol := admin
	protocol.Role = policy.RoleProtocol
	tokens := security.Tokens{Key: []byte(strings.Repeat("j", 32)), Issuer: "vms", Audience: "staff"}
	authority := contractAuthority{"admin": admin, "no-export": noExport, "protocol": protocol}
	deps := &router.Dependencies{App: app, StaffAuth: restmiddleware.StaffAuth(tokens, authority, contractLimiter{}), Workspace: restmiddleware.Workspace}
	reader := contractReportReader{entity.ReportSnapshot{Rows: rows, Officers: []entity.ReportOfficer{{ID: 1, Name: "Official Example"}}, Rooms: []string{"Meeting room"}, At: at}}
	router.RegisterReportRoutes(deps, reportusecase.NewReportUsecase(reader, reportserializer.NewReportSerializer(), tracer))
	type capture struct {
		Name    string            `json:"name"`
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		Body    json.RawMessage   `json:"body"`
		Bytes   int               `json:"bytes"`
	}
	const root = "/v1/staff/reports"
	const filters = "?from=2030-01-01&to=2030-01-31&status=missed&status=pending"
	const logFilters = "?from=&to=&q=&status=missed&status=checked_out&kind=all&officer=&room=2&room=1&sort=date&dir=desc"
	wrong := strings.Repeat("f", 64)
	var summaryRevision, logRevision string
	var captures []capture
	for _, tc := range []struct {
		name, path, actor string
		status            int
	}{
		{"report-summary", root + "/summary" + filters, "admin", 200},
		{"report-summary-export", root + "/export" + filters + "&revision=$summary", "admin", 200},
		{"report-summary-stale", root + "/export" + filters + "&revision=" + wrong, "admin", 409},
		{"report-summary-denied-export", root + "/export" + filters + "&revision=" + wrong, "no-export", 403},
		{"report-summary-invalid-date", root + "/summary?from=2030-02-30&to=2030-03-01", "admin", 422},
		{"visit-log", root + "/visits" + logFilters + "&page=100&pageSize=10", "admin", 200},
		{"visit-log-export", root + "/visits/export" + logFilters + "&revision=$log", "admin", 200},
		{"visit-log-stale", root + "/visits/export" + logFilters + "&revision=" + wrong, "admin", 409},
		{"visit-log-denied", root + "/visits" + logFilters, "protocol", 403},
		{"visit-log-invalid-page-size", root + "/visits" + logFilters + "&pageSize=11", "admin", 422},
		{"visit-log-unknown-query", root + "/visits?scope=all", "admin", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := strings.NewReplacer("$summary", summaryRevision, "$log", logRevision).Replace(tc.path)
			request := httptest.NewRequest("GET", path, nil)
			now := time.Now()
			token, err := tokens.Sign(security.SessionClaims{SessionID: tc.actor, EmployeeID: 1, CredentialGeneration: 1, RoleRevision: 1}, now, now.Add(time.Minute))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request, fiber.TestConfig{Timeout: 10 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != tc.status || !json.Valid(body) {
				t.Fatalf("status %d, want %d; body %q (%v)", response.StatusCode, tc.status, body, err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatal(err)
			}
			if offsets := nonUTCTimestamps(decoded); len(offsets) > 0 {
				t.Fatalf("non-UTC timestamps: %v", offsets)
			}
			switch tc.name {
			case "report-summary":
				summaryRevision, _ = decoded["revision"].(string)
				if !strings.Contains(string(body), `"status":["missed","pending"]`) || decoded["total"] != float64(1) {
					t.Fatal("summary changed the status filter order or counts")
				}
			case "report-summary-export":
				csv, _ := decoded["csv"].(string)
				if !strings.HasPrefix(csv, "\ufeff\"Tanggal awal\"") || !strings.Contains(csv, "\r\n") || !strings.Contains(csv, `"Terlewat","1"`) {
					t.Fatal("export CSV no longer matches the frontend format")
				}
			case "visit-log":
				logRevision, _ = decoded["revision"].(string)
				if !strings.Contains(string(body), `"room":["2","1"]`) || strings.Contains(string(body), `"roomId"`) ||
					!strings.Contains(string(body), `"visitors":null`) || decoded["page"] != float64(1) || decoded["total"] != float64(2) {
					t.Fatal("log filter echo, paging or row projection does not match the frontend shape")
				}
			}
			headers := map[string]string{}
			for _, key := range []string{"Content-Type", "Cache-Control", "X-Content-Type-Options", "X-Request-ID"} {
				headers[key] = response.Header.Get(key)
			}
			captures = append(captures, capture{tc.name, "GET", request.URL.Path, response.StatusCode, headers, body, len(body)})
		})
	}
	if target := os.Getenv("VMS_CONTRACT_CAPTURE"); target != "" {
		data, err := json.Marshal(captures)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
