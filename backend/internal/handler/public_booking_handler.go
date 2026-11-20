package handler

import (
	"github.com/gofiber/fiber/v3"

	booking "github.com/azhkadaes/vms/backend/internal/usecase/booking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type publicBookingHandler struct{ usecase booking.PublicBookingUsecase }

func NewPublicBookingHandler(usecase booking.PublicBookingUsecase) *publicBookingHandler {
	return &publicBookingHandler{usecase: usecase}
}

func (h *publicBookingHandler) Create(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	var input dto.PublicBookingCreate
	if err := DecodeJSON(c, &input); err != nil {
		return err
	}
	visit, created, err := h.usecase.Create(c.Context(), c.IP(), input)
	if err != nil {
		return err
	}
	if created {
		c.Status(fiber.StatusCreated)
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(visit)
}

func (h *publicBookingHandler) Resolve(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	var input dto.PublicSubmissionProof
	if err := DecodeJSON(c, &input); err != nil {
		return err
	}
	result, err := h.usecase.Resolve(c.Context(), input)
	if err != nil {
		return err
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(result)
}
