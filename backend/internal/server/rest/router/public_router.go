package router

import (
	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/handler"
	admissionrepository "github.com/azhkadaes/vms/backend/internal/repository/admission/implementation"
	bookingrepository "github.com/azhkadaes/vms/backend/internal/repository/booking/implementation"
	catalogrepository "github.com/azhkadaes/vms/backend/internal/repository/catalog/implementation"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking/implementation"
	catalogserializer "github.com/azhkadaes/vms/backend/internal/serializer/catalog/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	admissioncontract "github.com/azhkadaes/vms/backend/internal/usecase/admission"
	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission/implementation"
	bookingcontract "github.com/azhkadaes/vms/backend/internal/usecase/booking"
	bookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/booking/implementation"
	catalogcontract "github.com/azhkadaes/vms/backend/internal/usecase/catalog"
	catalogusecase "github.com/azhkadaes/vms/backend/internal/usecase/catalog/implementation"
	uploadcontract "github.com/azhkadaes/vms/backend/internal/usecase/upload"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// publishedNotice is the configured privacy notice both catalogs carry; nil when none is published.
func publishedNotice(deps *Dependencies) *dto.PublishedNotice {
	if deps.Cfg.Notice == nil {
		return nil
	}
	return &dto.PublishedNotice{Version: deps.Cfg.Notice.Version, ID: deps.Cfg.Notice.ID, EN: deps.Cfg.Notice.EN}
}

func PublicRouter(deps *Dependencies) {
	catalogRepo := catalogrepository.NewCatalogRepository(deps.DB)
	serializer := catalogserializer.NewCatalogSerializer(publishedNotice(deps), deps.UploadsEnabled)
	catalog := catalogusecase.NewCatalogUsecase(catalogRepo, serializer, deps.Tracer)
	admissionRepo := admissionrepository.NewAdmissionRepository(deps.DB)
	admission := admissionusecase.NewAdmissionUsecase(admissionRepo, deps.Cfg.RateKey, deps.Tracer)
	RegisterPublicRoutes(deps.App, catalog, admission, deps.Uploads)
	limiter := admissionusecase.NewRequestLimiter(admissionrepository.NewRequestLimitRepository(deps.DB), deps.Cfg.RateKey)
	version := ""
	if notice := publishedNotice(deps); notice != nil {
		version = notice.Version
	}
	bookings := bookingusecase.NewPublicBookingUsecase(bookingrepository.NewBookingRepository(deps.DB),
		bookingserializer.NewBookingSerializer(), version, deps.Cfg.RateKey, deps.Tracer)
	RegisterPublicBookingRoutes(deps.App, admission, limiter, bookings)
	PublicBookingAccessRouter(deps, admission)
}

// RegisterPublicRoutes shares route and admission behavior between runtime and injected usecases.
// Every public route, uploads included, sits behind the overall public request ceiling (D33).
func RegisterPublicRoutes(app *fiber.App, catalog catalogcontract.CatalogUsecase, admission admissioncontract.AdmissionUsecase, uploads uploadcontract.UploadUsecase) {
	h := handler.NewCatalogHandler(catalog)
	app.Get("/v1/public/catalogs", restmiddleware.PublicAdmission(admission), h.PublicCatalogs)
	u := handler.NewUploadHandler(uploads)
	app.Post("/v1/public/attachments", restmiddleware.PublicAdmission(admission), u.PublicUpload)
}

func RegisterPublicBookingRoutes(app *fiber.App, admission admissioncontract.AdmissionUsecase, limiter admissioncontract.RequestLimiter, bookings bookingcontract.PublicBookingUsecase) {
	h := handler.NewPublicBookingHandler(bookings)
	submissions := restmiddleware.SubmissionAdmission(limiter)
	app.Post("/v1/public/bookings", restmiddleware.PublicAdmission(admission), submissions, h.Create)
	app.Post("/v1/public/submissions/resolve", restmiddleware.PublicAdmission(admission), submissions, h.Resolve)
}
