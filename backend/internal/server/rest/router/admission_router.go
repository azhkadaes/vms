package router

import (
	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/handler"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	admissioncontract "github.com/azhkadaes/vms/backend/internal/usecase/admission"
	bookingcontract "github.com/azhkadaes/vms/backend/internal/usecase/booking"
)

func RegisterVisitAdmissionRoutes(app *fiber.App, admission admissioncontract.AdmissionUsecase, limiter admissioncontract.RequestLimiter, usecase bookingcontract.AdmissionUsecase) {
	h := handler.NewAdmissionHandler(usecase)
	app.Post("/v1/public/bookings/:id/admission", restmiddleware.PublicAdmission(admission), restmiddleware.CredentialAdmission(limiter), h.Issue)
}
