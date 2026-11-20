package reportrepository

import (
	"context"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type ReportReader interface {
	Summary(context.Context, dto.Session, dto.SummaryFilter) ([]entity.ReportStatus, time.Time, error)
	VisitLog(context.Context, dto.VisitLogFilter) (entity.ReportSnapshot, error)
}
