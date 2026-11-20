package handler

import (
	"github.com/gofiber/fiber/v3"

	bookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/booking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type admissionHandler struct {
	usecase bookingusecase.AdmissionUsecase
}

func NewAdmissionHandler(usecase bookingusecase.AdmissionUsecase) *admissionHandler {
	return &admissionHandler{usecase: usecase}
}

func (h *admissionHandler) Issue(c fiber.Ctx) error {
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
	result, err := h.usecase.Issue(c.Context(), id, input)
	if err != nil {
		return err
	}
	return c.JSON(result)
}
