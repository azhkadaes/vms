package handler

import (
	"github.com/gofiber/fiber/v3"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	staffbookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type staffBookingHandler struct {
	usecase staffbookingusecase.StaffBookingUsecase
}

func NewStaffBookingHandler(usecase staffbookingusecase.StaffBookingUsecase) *staffBookingHandler {
	return &staffBookingHandler{usecase: usecase}
}

func (h *staffBookingHandler) List(c fiber.Ctx) error {
	keys := []string{"view", "q", "status", "kind", "officer", "date", "vip", "sort", "dir", "page", "pageSize", "mode"}
	values, err := Query(c, keys...)
	if err != nil {
		return err
	}
	var query staffbookingusecase.Query
	fields := []*string{&query.View, &query.Search, &query.Status, &query.Kind, &query.Officer, &query.Date, &query.VIP, &query.Sort, &query.Direction, &query.Page, &query.PageSize, &query.Mode}
	for i, key := range keys {
		*fields[i], err = values.String(key)
		if err != nil {
			return err
		}
	}
	out, err := h.usecase.List(c.Context(), restmiddleware.Session(c), query)
	if err != nil {
		return err
	}
	return c.JSON(out)
}

func (h *staffBookingHandler) Get(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	id, err := validate.ID(c.Params("id"))
	if err != nil {
		return err
	}
	out, err := h.usecase.Get(c.Context(), restmiddleware.Session(c), id)
	if err != nil {
		return err
	}
	return c.JSON(out)
}
