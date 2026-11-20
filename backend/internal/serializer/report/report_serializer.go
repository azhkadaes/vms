package reportserializer

import (
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type ReportSerializer interface {
	ToVisitLogRow(entity.ReportVisit, string) dto.VisitLogRow
	ToSummaryExport(dto.VisitSummary, []string) dto.SummaryExport
}
