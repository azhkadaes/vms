package handler

import (
	"github.com/gofiber/fiber/v3"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	staffbooking "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type decisionHandler struct{ usecase staffbooking.DecisionUsecase }

func NewDecisionHandler(usecase staffbooking.DecisionUsecase) *decisionHandler {
	return &decisionHandler{usecase: usecase}
}

func decodeDecision(c fiber.Ctx, target any) (int32, error) {
	if err := NoQuery(c); err != nil {
		return 0, err
	}
	id, err := validate.ID(c.Params("id"))
	if err != nil {
		return 0, err
	}
	return id, DecodeJSON(c, target)
}

func (h *decisionHandler) Approve(c fiber.Ctx) error {
	var input dto.ApproveBooking
	id, err := decodeDecision(c, &input)
	if err != nil {
		return err
	}
	visit, err := h.usecase.Approve(c.Context(), restmiddleware.Proof(c), id, input)
	if err != nil {
		return err
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(visit)
}

func (h *decisionHandler) Reject(c fiber.Ctx) error {
	var input dto.RejectBooking
	id, err := decodeDecision(c, &input)
	if err != nil {
		return err
	}
	visit, err := h.usecase.Reject(c.Context(), restmiddleware.Proof(c), id, input)
	if err != nil {
		return err
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(visit)
}

func (h *decisionHandler) Cancel(c fiber.Ctx) error {
	var input dto.CancelBooking
	id, err := decodeDecision(c, &input)
	if err != nil {
		return err
	}
	visit, err := h.usecase.Cancel(c.Context(), restmiddleware.Proof(c), id, input)
	if err != nil {
		return err
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(visit)
}

func (h *decisionHandler) Reschedule(c fiber.Ctx) error {
	var input dto.RescheduleBooking
	id, err := decodeDecision(c, &input)
	if err != nil {
		return err
	}
	visit, err := h.usecase.Reschedule(c.Context(), restmiddleware.Proof(c), id, input)
	if err != nil {
		return err
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(visit)
}

func (h *decisionHandler) UpdateRestricted(c fiber.Ctx) error {
	var input dto.UpdateRestrictedBooking
	id, err := decodeDecision(c, &input)
	if err != nil {
		return err
	}
	visit, err := h.usecase.UpdateRestricted(c.Context(), restmiddleware.Proof(c), id, input)
	if err != nil {
		return err
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(visit)
}
