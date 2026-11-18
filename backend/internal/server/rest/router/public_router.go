package router

import (
	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/handler"
	admissionrepository "github.com/azhkadaes/vms/backend/internal/repository/admission/implementation"
	catalogrepository "github.com/azhkadaes/vms/backend/internal/repository/catalog/implementation"
	catalogserializer "github.com/azhkadaes/vms/backend/internal/serializer/catalog/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	admissioncontract "github.com/azhkadaes/vms/backend/internal/usecase/admission"
	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission/implementation"
	catalogcontract "github.com/azhkadaes/vms/backend/internal/usecase/catalog"
	catalogusecase "github.com/azhkadaes/vms/backend/internal/usecase/catalog/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func PublicRouter(deps *Dependencies) {
	var notice *dto.PublishedNotice
	if deps.Cfg.Notice != nil {
		notice = &dto.PublishedNotice{Version: deps.Cfg.Notice.Version, ID: deps.Cfg.Notice.ID, EN: deps.Cfg.Notice.EN}
	}
	catalogRepo := catalogrepository.NewCatalogRepository(deps.DB)
	serializer := catalogserializer.NewCatalogSerializer(notice)
	catalog := catalogusecase.NewCatalogUsecase(catalogRepo, serializer, deps.Tracer)
	admissionRepo := admissionrepository.NewAdmissionRepository(deps.DB)
	admission := admissionusecase.NewAdmissionUsecase(admissionRepo, deps.Cfg.RateKey, deps.Tracer)
	RegisterPublicRoutes(deps.App, catalog, admission)
}

// RegisterPublicRoutes shares route and admission behavior between runtime and injected usecases.
func RegisterPublicRoutes(app *fiber.App, catalog catalogcontract.CatalogUsecase, admission admissioncontract.AdmissionUsecase) {
	group := app.Group("/v1/public", restmiddleware.PublicAdmission(admission))
	h := handler.NewCatalogHandler(catalog)
	group.Get("/catalogs", h.PublicCatalogs)
}
