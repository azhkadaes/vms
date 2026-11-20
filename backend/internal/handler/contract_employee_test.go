package handler_test

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	"github.com/azhkadaes/vms/backend/internal/usecase/employee"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

type contractEmployees struct{}

var contractEmployee = dto.Employee{
	ID: "7", Name: "Employee Example", Email: "employee@example.test", JobTitle: "Asisten",
	Deputy: "Kedeputian Contoh", Directorate: "Direktorat Contoh", Phone: "0812345678", NIP: "123459900",
	Roles: []string{policy.RolePA}, Active: true, AssignedOfficerIDs: []string{"3"}, Version: 1,
}

func (contractEmployees) GetEmployees(_ context.Context, session dto.Session, query employeeusecase.EmployeeQuery) (dto.EmployeePage, error) {
	if !policy.CanAdministerEmployees(session) {
		return dto.EmployeePage{}, failure.New(403, "forbidden")
	}
	if query.PageSize == "251" {
		return dto.EmployeePage{}, failure.Validation().Add("pageSize", "Jumlah per halaman harus antara 1 dan 250.").Err()
	}
	if query.Q != "Employee" || query.Page != "1" || query.PageSize != "25" {
		return dto.EmployeePage{}, failure.New(500, "internal_error")
	}
	return dto.EmployeePage{Items: []dto.Employee{contractEmployee}, Total: 1, TotalAll: 2, Page: 1, PageSize: 25, PasswordMinLength: policy.PasswordMinLength}, nil
}

func (contractEmployees) GetEmployee(_ context.Context, session dto.Session, id int32) (dto.Employee, error) {
	if !policy.CanAdministerEmployees(session) {
		return dto.Employee{}, failure.New(403, "forbidden")
	}
	if id != 7 {
		return dto.Employee{}, failure.ErrNotFound
	}
	return contractEmployee, nil
}

func (contractEmployees) CreateEmployee(_ context.Context, session dto.Session, proof entity.SessionProof, req dto.EmployeeCreate) (dto.Employee, error) {
	if !policy.CanAdministerEmployees(session) || proof.EmployeeID <= 0 || proof.SessionID == "" {
		return dto.Employee{}, failure.New(403, "forbidden")
	}
	if req.Email == "duplicate@example.test" {
		return dto.Employee{}, failure.Validation().Add("email", "Email sudah digunakan pegawai lain.").Err()
	}
	if req.Email != contractEmployee.Email || req.InitialPassword == "" || req.RequirePasswordChange == nil || !*req.RequirePasswordChange {
		return dto.Employee{}, failure.New(500, "internal_error")
	}
	return contractEmployee, nil
}

func (contractEmployees) UpdateEmployee(_ context.Context, session dto.Session, proof entity.SessionProof, id int32, req dto.EmployeeUpdate) (dto.Employee, error) {
	if !policy.CanAdministerEmployees(session) || proof.EmployeeID <= 0 || proof.SessionID == "" {
		return dto.Employee{}, failure.New(403, "forbidden")
	}
	if id != 7 {
		return dto.Employee{}, failure.ErrNotFound
	}
	if req.Version != 1 {
		return dto.Employee{}, failure.New(409, "version_conflict")
	}
	if !req.Active.Set || req.Active.Null || req.Active.Value {
		return dto.Employee{}, failure.New(500, "internal_error")
	}
	out := contractEmployee
	out.Active, out.Version = false, 2
	return out, nil
}
