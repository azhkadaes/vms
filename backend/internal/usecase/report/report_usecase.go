package reportusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type ReportUsecase interface {
	Summary(context.Context, dto.Session, dto.SummaryFilter) (dto.VisitSummary, error)
	ExportSummary(context.Context, dto.Session, dto.SummaryFilter, string) (dto.SummaryExport, error)
	VisitLog(context.Context, dto.Session, dto.VisitLogFilter, int, int) (dto.VisitLogResponse, error)
	ExportVisitLog(context.Context, dto.Session, dto.VisitLogFilter, string) (dto.VisitLogSnapshot, error)
}
