package handler

import (
	"github.com/gofiber/fiber/v3"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	catalogusecase "github.com/azhkadaes/vms/backend/internal/usecase/catalog"
)

type catalogHandler struct{ usecase catalogusecase.CatalogUsecase }

// NewCatalogHandler exposes the public and staff catalog reads; it has no mutation methods.
func NewCatalogHandler(usecase catalogusecase.CatalogUsecase) *catalogHandler {
	return &catalogHandler{usecase: usecase}
}

// StaffCatalogs is op 08. It runs behind StaffAuth and the workspace gate and passes the live
// session, so the usecase decides scope from current authority, never from the request.
func (h *catalogHandler) StaffCatalogs(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	result, err := h.usecase.GetStaffCatalogs(c.Context(), restmiddleware.Session(c))
	if err != nil {
		return err
	}
	return c.JSON(result)
}

// PublicCatalogs reads active public destinations and the configured published notice.
func (h *catalogHandler) PublicCatalogs(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	result, err := h.usecase.GetPublicCatalogs(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(result)
}
