package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	catalogrepository "github.com/azhkadaes/vms/backend/internal/repository/catalog/implementation"
	catalogserializer "github.com/azhkadaes/vms/backend/internal/serializer/catalog/implementation"
	catalogcontract "github.com/azhkadaes/vms/backend/internal/usecase/catalog"
	catalogusecase "github.com/azhkadaes/vms/backend/internal/usecase/catalog/implementation"
)

// StaffRouter wires staff business areas after authentication has been configured.
func StaffRouter(deps *Dependencies) {
	serializer := catalogserializer.NewCatalogSerializer(publishedNotice(deps), deps.UploadsEnabled)
	catalog := catalogusecase.NewCatalogUsecase(catalogrepository.NewCatalogRepository(deps.DB), serializer, deps.Tracer)
	RegisterStaffRoutes(deps, catalog)
	registerEmployeeArea(deps)
	StaffBookingRouter(deps)
	staffBookingCreateRouter(deps)
	DecisionRouter(deps)
	ReceptionRouter(deps)
	BookingFileRouter(deps)
	ReportRouter(deps)
}

// RegisterStaffRoutes shares the staff routes between runtime and injected usecases. Each area is
// its own literal group behind StaffAuth and the workspace gate, never a group on /v1/staff.
func RegisterStaffRoutes(deps *Dependencies, catalog catalogcontract.CatalogUsecase) {
	catalogs := deps.App.Group("/v1/staff/catalogs", deps.StaffAuth, deps.Workspace)
	h := handler.NewCatalogHandler(catalog)
	catalogs.Get("", h.StaffCatalogs)
}
