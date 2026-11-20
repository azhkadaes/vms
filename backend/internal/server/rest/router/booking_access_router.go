package router

import (
	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/handler"
	admissionrepository "github.com/azhkadaes/vms/backend/internal/repository/admission/implementation"
	bookingrepository "github.com/azhkadaes/vms/backend/internal/repository/booking/implementation"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking/implementation"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	admissioncontract "github.com/azhkadaes/vms/backend/internal/usecase/admission"
	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission/implementation"
	bookingcontract "github.com/azhkadaes/vms/backend/internal/usecase/booking"
	bookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/booking/implementation"
)

func PublicBookingAccessRouter(deps *Dependencies, admission admissioncontract.AdmissionUsecase) {
	limiter := admissionusecase.NewRequestLimiter(admissionrepository.NewRequestLimitRepository(deps.DB), deps.Cfg.RateKey)
	repo := bookingrepository.NewBookingAccessRepository(deps.DB)
	access := bookingusecase.NewBookingAccessUsecase(repo, bookingserializer.NewBookingSerializer(), deps.Tracer)
	RegisterPublicBookingAccessRoutes(deps.App, admission, limiter, access)
	qr := bookingusecase.NewAdmissionUsecase(repo, deps.Cfg.AdmissionQRKey, deps.Tracer)
	RegisterVisitAdmissionRoutes(deps.App, admission, limiter, qr)
}

func RegisterPublicBookingAccessRoutes(app *fiber.App, admission admissioncontract.AdmissionUsecase, limiter admissioncontract.RequestLimiter, access bookingcontract.BookingAccessUsecase) {
	h := handler.NewBookingAccessHandler(access)
	ceiling := restmiddleware.PublicAdmission(admission)
	credentials := restmiddleware.CredentialAdmission(limiter)
	app.Post("/v1/public/lookup", ceiling, credentials, h.Lookup)
	app.Post("/v1/public/bookings/:id/cancel", ceiling, credentials, h.Cancel)
}
