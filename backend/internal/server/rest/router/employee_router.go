package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	employeerepository "github.com/azhkadaes/vms/backend/internal/repository/employee/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	employeeserializer "github.com/azhkadaes/vms/backend/internal/serializer/employee/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	employeecontract "github.com/azhkadaes/vms/backend/internal/usecase/employee"
	employeeusecase "github.com/azhkadaes/vms/backend/internal/usecase/employee/implementation"
)

func registerEmployeeArea(deps *Dependencies) {
	validator := authusecase.NewSessionValidator(authserializer.NewAuthSerializer())
	usecase := employeeusecase.NewEmployeeUsecase(employeerepository.NewEmployeeRepository(deps.DB),
		employeeserializer.NewEmployeeSerializer(), validator, deps.CredentialVerifier, deps.Tracer)
	RegisterEmployeeRoutes(deps, usecase)
}

func RegisterEmployeeRoutes(deps *Dependencies, usecase employeecontract.EmployeeUsecase) {
	h := handler.NewEmployeeHandler(usecase)
	group := deps.App.Group("/v1/staff/employees", deps.StaffAuth, deps.Workspace)
	group.Get("", h.Employees)
	group.Get("/:id", h.Employee)
	group.Post("", h.CreateEmployee)
	group.Patch("/:id", h.UpdateEmployee)
}
