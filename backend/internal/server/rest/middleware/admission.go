package restmiddleware

import (
	"github.com/gofiber/fiber/v3"

	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// PublicAdmission applies the shared public request limit before endpoint handlers.
func PublicAdmission(admission admissionusecase.AdmissionUsecase) fiber.Handler {
	return func(c fiber.Ctx) error {
		decision, err := admission.PublicRequest(c.Context(), c.IP())
		if err != nil {
			return failure.New(503, "service_unavailable").WithCause(err)
		}
		if !decision.Allowed {
			return &failure.Error{Status: 429, Code: "rate_limited", RetryAfter: max(1, decision.RetryAfter)}
		}
		return c.Next()
	}
}
