package restmiddleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/gofiber/fiber/v3"
)

// Per-employee budget for Bearer-authenticated staff requests.
const (
	staffRequestLimit  = 120
	staffRequestWindow = time.Minute
)

// StaffAuth verifies the Bearer access token, reads live session authority on every request
// and applies the per-employee limit. Mount it per staff group or route, never on /v1/staff.
func StaffAuth(tokens security.Tokens, reader authusecase.AuthUsecase, limiter admissionusecase.RequestLimiter) fiber.Handler {
	return func(c fiber.Ctx) error {
		parts := strings.Fields(c.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return authusecase.ErrUnauthenticated
		}
		claims, err := tokens.Verify(parts[1], time.Now())
		if err != nil {
			return authusecase.ErrUnauthenticated
		}
		proof := entity.SessionProof(claims)
		session, err := reader.GetSession(c.Context(), proof)
		if err != nil {
			return err
		}
		if err := limiter.Admit(c.Context(), "staff", strconv.FormatInt(int64(proof.EmployeeID), 10), staffRequestLimit, staffRequestWindow); err != nil {
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

// Workspace follows StaffAuth on business groups: a pending role selection or password change is 403.
func Workspace(c fiber.Ctx) error {
	if err := authusecase.RequireWorkspace(Session(c)); err != nil {
		return err
	}
	return c.Next()
}

// SessionOperation follows StaffAuth on a session route: it admits only the stages A-18 allows for op.
func SessionOperation(op policy.SessionOperation) fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := authusecase.RequireSessionOperation(Session(c), op); err != nil {
			return err
		}
		return c.Next()
	}
}
