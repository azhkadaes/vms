package handler

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v3"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	staff "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

type staffBookingCreateHandler struct {
	usecase staff.StaffBookingCreateUsecase
}

func NewStaffBookingCreateHandler(usecase staff.StaffBookingCreateUsecase) *staffBookingCreateHandler {
	return &staffBookingCreateHandler{usecase: usecase}
}

func (h *staffBookingCreateHandler) Create(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	var input dto.StaffBookingCreate
	if err := DecodeJSON(c, &input); err != nil {
		return err
	}
	if input.VisitKind == "restricted" && containsStandardFields(c.Body()) {
		return failure.New(400, "invalid_request")
	}
	result, created, err := h.usecase.CreateStaffBooking(c.Context(), restmiddleware.Session(c), restmiddleware.Proof(c), c.IP(), input)
	if err != nil {
		return err
	}
	if created {
		c.Status(fiber.StatusCreated)
	}
	return c.JSON(result)
}

// Inspect presence as well as values so a restricted request cannot send an
// ordinary field with a null or an empty string.
func containsStandardFields(body []byte) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return true
	}
	for key := range fields {
		switch strings.ToLower(key) {
		case "purpose", "email", "phone", "identitytype", "visitors", "visitorphotoname", "visitorphototoken", "ndname", "attachmenttoken":
			return true
		}
	}
	return false
}
