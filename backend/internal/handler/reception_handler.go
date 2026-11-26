package handler

import (
	"github.com/gofiber/fiber/v3"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	receptionusecase "github.com/azhkadaes/vms/backend/internal/usecase/reception"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type receptionHandler struct {
	usecase receptionusecase.ReceptionUsecase
}

func NewReceptionHandler(usecase receptionusecase.ReceptionUsecase) *receptionHandler {
	return &receptionHandler{usecase: usecase}
}

func (h *receptionHandler) Resolve(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	var req dto.ReceptionLookup
	if err := DecodeJSON(c, &req); err != nil {
		return err
	}
	visit, err := h.usecase.Resolve(c.Context(), restmiddleware.Session(c), req)
	if err != nil {
		return err
	}
	return c.JSON(visit)
}

func (h *receptionHandler) Attendance(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	id, err := validate.ID(c.Params("id"))
	if err != nil {
		return err
	}
	var req dto.ReceptionAttendance
	if err := DecodeJSON(c, &req); err != nil {
		return err
	}
	visit, err := h.usecase.Attendance(c.Context(), restmiddleware.Proof(c), id, req)
	if err != nil {
		return err
	}
	return c.JSON(visit)
}
