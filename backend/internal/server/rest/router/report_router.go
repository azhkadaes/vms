package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	reportrepository "github.com/azhkadaes/vms/backend/internal/repository/report/implementation"
	reportserializer "github.com/azhkadaes/vms/backend/internal/serializer/report/implementation"
	reportcontract "github.com/azhkadaes/vms/backend/internal/usecase/report"
	reportusecase "github.com/azhkadaes/vms/backend/internal/usecase/report/implementation"
)

func ReportRouter(deps *Dependencies) {
	reader := reportrepository.NewReportReader(deps.DB)
	RegisterReportRoutes(deps, reportusecase.NewReportUsecase(reader, reportserializer.NewReportSerializer(), deps.Tracer))
}

func RegisterReportRoutes(deps *Dependencies, usecase reportcontract.ReportUsecase) {
	h := handler.NewReportHandler(usecase)
	group := deps.App.Group("/v1/staff/reports", deps.StaffAuth, deps.Workspace)
	group.Get("/summary", h.Summary)
	group.Get("/export", h.ExportSummary)
	group.Get("/visits", h.VisitLog)
	group.Get("/visits/export", h.ExportVisitLog)
}
