package handler

import (
	"github.com/gofiber/fiber/v3"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	roleusecase "github.com/azhkadaes/vms/backend/internal/usecase/role"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type roleHandler struct{ usecase roleusecase.RoleUsecase }

// NewRoleHandler serves role administration behind StaffAuth and the workspace gate.
func NewRoleHandler(usecase roleusecase.RoleUsecase) *roleHandler {
	return &roleHandler{usecase: usecase}
}

// Roles lists the four fixed roles as a bare JSON array (op 30).
func (h *roleHandler) Roles(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	result, err := h.usecase.GetRoles(c.Context(), restmiddleware.Session(c))
	if err != nil {
		return err
	}
	return c.JSON(result)
}

// UpdateRole replaces a role's permissions (op 31); the usecase rechecks authority itself.
func (h *roleHandler) UpdateRole(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	id, err := validate.ID(c.Params("id"))
	if err != nil {
		return err
	}
	var req dto.RoleUpdate
	if err := DecodeJSON(c, &req); err != nil {
		return err
	}
	result, err := h.usecase.UpdateRole(c.Context(), restmiddleware.Proof(c), id, req)
	if err != nil {
		return err
	}
	return c.JSON(result)
}
