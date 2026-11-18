package restmiddleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/gofiber/fiber/v3"
)

func BearerRequired(tokens security.Tokens, reader authusecase.AuthUsecase, limiter admissionusecase.RequestLimiter) fiber.Handler {
	return func(c fiber.Ctx) error {
		parts := strings.Fields(c.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return authusecase.ErrUnauthenticated
		}
		proof, err := tokens.Verify(parts[1], time.Now())
		if err != nil {
			return authusecase.ErrUnauthenticated
		}
		session, err := reader.GetSession(c.Context(), proof)
		if err != nil {
			return err
		}
		if err := limiter.Admit(c.Context(), "staff", strconv.FormatInt(int64(proof.EmployeeID), 10), 120, time.Minute); err != nil {
			return err
		}
		c.Locals("vms.proof", proof)
		c.Locals("vms.session", session)
		return c.Next()
	}
}
func Proof(c fiber.Ctx) entity.SessionProof {
	proof, _ := c.Locals("vms.proof").(entity.SessionProof)
	return proof
}
func Session(c fiber.Ctx) dto.Session {
	session, _ := c.Locals("vms.session").(dto.Session)
	return session
}
func WorkspaceRequired(c fiber.Ctx) error {
	if err := authusecase.RequireWorkspace(Session(c)); err != nil {
		return err
	}
	return c.Next()
}
