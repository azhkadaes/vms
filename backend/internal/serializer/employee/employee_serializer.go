package employeeserializer

import (
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// EmployeeSerializer projects employees for the administration screens without credentials.
type EmployeeSerializer interface {
	ToEmployee(in entity.Employee) dto.Employee
	// ToEmployeePage echoes the effective page and page size and the password policy minimum.
	ToEmployeePage(in entity.EmployeePage, filter entity.EmployeeFilter) dto.EmployeePage
}
