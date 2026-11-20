package handler

import (
	"github.com/gofiber/fiber/v3"

	bookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/booking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type bookingAccessHandler struct {
	usecase bookingusecase.BookingAccessUsecase
}

func NewBookingAccessHandler(usecase bookingusecase.BookingAccessUsecase) *bookingAccessHandler {
	return &bookingAccessHandler{usecase: usecase}
}

func (h *bookingAccessHandler) Lookup(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	var input dto.PublicLookup
	if err := DecodeJSON(c, &input); err != nil {
		return err
	}
	visit, err := h.usecase.Lookup(c.Context(), input)
	if err != nil {
		return err
	}
	return c.JSON(visit)
}

func (h *bookingAccessHandler) Cancel(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	id, err := validate.ID(c.Params("id"))
	if err != nil {
		return err
	}
	var input dto.PublicCancel
	if err := DecodeJSON(c, &input); err != nil {
		return err
	}
	visit, err := h.usecase.Cancel(c.Context(), id, input, c.IP())
	if err != nil {
		return err
	}
	return c.JSON(visit)
}
