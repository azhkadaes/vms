package handler

import (
	"github.com/gofiber/fiber/v3"

	catalogusecase "github.com/azhkadaes/vms/backend/internal/usecase/catalog"
)

type catalogHandler struct{ usecase catalogusecase.CatalogUsecase }

// NewCatalogHandler exposes the public read operation without staff or mutation methods.
func NewCatalogHandler(usecase catalogusecase.CatalogUsecase) *catalogHandler {
	return &catalogHandler{usecase: usecase}
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
