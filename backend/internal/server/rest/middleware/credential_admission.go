package restmiddleware

import (
	"errors"
	"net/netip"
	"time"

	"github.com/gofiber/fiber/v3"

	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// CredentialAdmission applies the shared D29 budget to lookup, cancel and admission.
func CredentialAdmission(limiter admissionusecase.RequestLimiter) fiber.Handler {
	return proofBudget(limiter, "public-credential")
}

// SubmissionAdmission applies the shared D32 attempt budget before decoding either submission route.
func SubmissionAdmission(limiter admissionusecase.RequestLimiter) fiber.Handler {
	return proofBudget(limiter, "public-submission")
}

func proofBudget(limiter admissionusecase.RequestLimiter, purpose string) fiber.Handler {
	return func(c fiber.Ctx) error {
		address, err := netip.ParseAddr(c.IP())
		if err != nil {
			return failure.New(503, "service_unavailable").WithCause(errors.New("trusted request address unavailable"))
		}
		if err := limiter.Admit(c.Context(), purpose, address.Unmap().String(), 30, time.Minute); err != nil {
			return err
		}
		return c.Next()
	}
}
