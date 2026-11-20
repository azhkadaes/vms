package reportusecaseimplementation_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	reportserializer "github.com/azhkadaes/vms/backend/internal/serializer/report/implementation"
	reportusecase "github.com/azhkadaes/vms/backend/internal/usecase/report/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"go.opentelemetry.io/otel/trace/noop"
)

type source struct{ snapshot entity.ReportSnapshot }

func (s source) Summary(context.Context, dto.Session, dto.SummaryFilter) ([]entity.ReportStatus, time.Time, error) {
	return nil, s.snapshot.At, nil
}

func (s source) VisitLog(context.Context, dto.VisitLogFilter) (entity.ReportSnapshot, error) {
	return s.snapshot, nil
}

func TestVisitLogAllowsPagingPastTheExportCap(t *testing.T) {
	rows := make([]entity.ReportVisit, 10_001)
	for i := range rows {
		rows[i] = entity.ReportVisit{ReportStatus: entity.ReportStatus{ID: int32(i + 1), Version: 1, Status: "pending"}, Kind: "standard"}
	}
	reader := source{entity.ReportSnapshot{At: time.Now(), Rows: rows}}
	u := reportusecase.NewReportUsecase(reader, reportserializer.NewReportSerializer(), noop.NewTracerProvider().Tracer("test"))
	admin := dto.Session{Active: true, Role: policy.RoleSuperAdmin, Permissions: []string{policy.PermReports, policy.PermReportExport}}
	filter := dto.VisitLogFilter{Status: dto.ReportFilterValues{"all"}, Kind: dto.ReportFilterValues{"all"}, Sort: "date", Dir: "desc"}
	ctx := context.Background()
	list, err := u.VisitLog(ctx, admin, filter, 1001, 10)
	if err != nil || list.Total != len(rows) || len(list.Items) != 1 {
		t.Fatalf("list failed at the export limit: total=%d len=%d err=%v", list.Total, len(list.Items), err)
	}
	_, err = u.ExportVisitLog(ctx, admin, filter, list.Revision)
	if api, ok := errors.AsType[*failure.Error](err); !ok || api.Status != 422 || api.Code != "export_too_large" {
		t.Fatalf("large export did not return export_too_large: %v", err)
	}
}
