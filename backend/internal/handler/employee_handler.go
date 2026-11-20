package handler

import (
	"github.com/gofiber/fiber/v3"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	employeeusecase "github.com/azhkadaes/vms/backend/internal/usecase/employee"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type employeeHandler struct {
	usecase employeeusecase.EmployeeUsecase
}

func NewEmployeeHandler(usecase employeeusecase.EmployeeUsecase) *employeeHandler {
	return &employeeHandler{usecase: usecase}
}

func (h *employeeHandler) Employees(c fiber.Ctx) error {
	query, err := Query(c, "q", "page", "pageSize")
	if err != nil {
		return err
	}
	q, err := query.String("q")
	if err != nil {
		return err
	}
	page, err := query.String("page")
	if err != nil {
		return err
	}
	pageSize, err := query.String("pageSize")
	if err != nil {
		return err
	}
	result, err := h.usecase.GetEmployees(c.Context(), restmiddleware.Session(c), employeeusecase.EmployeeQuery{Q: q, Page: page, PageSize: pageSize})
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func (h *employeeHandler) Employee(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	id, err := validate.ID(c.Params("id"))
	if err != nil {
		return err
	}
	result, err := h.usecase.GetEmployee(c.Context(), restmiddleware.Session(c), id)
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func (h *employeeHandler) CreateEmployee(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	var req dto.EmployeeCreate
	if err := DecodeJSON(c, &req); err != nil {
		return err
	}
	result, err := h.usecase.CreateEmployee(c.Context(), restmiddleware.Session(c), restmiddleware.Proof(c), req)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *employeeHandler) UpdateEmployee(c fiber.Ctx) error {
	if err := NoQuery(c); err != nil {
		return err
	}
	id, err := validate.ID(c.Params("id"))
	if err != nil {
		return err
	}
	var req dto.EmployeeUpdate
	if err := DecodeJSON(c, &req); err != nil {
		return err
	}
	result, err := h.usecase.UpdateEmployee(c.Context(), restmiddleware.Session(c), restmiddleware.Proof(c), id, req)
	if err != nil {
		return err
	}
	return c.JSON(result)
}
