package employeeusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// EmployeeQuery is op 26's query string as received; the usecase validates paging (422).
type EmployeeQuery struct {
	Q, Page, PageSize string
}

// EmployeeUsecase is employee administration (ops 26-29). Every operation needs the people
// permission of the live session's active role; writes recheck it inside their transaction.
type EmployeeUsecase interface {
	GetEmployees(ctx context.Context, session dto.Session, query EmployeeQuery) (dto.EmployeePage, error)
	GetEmployee(ctx context.Context, session dto.Session, id int32) (dto.Employee, error)
	CreateEmployee(ctx context.Context, session dto.Session, proof entity.SessionProof, req dto.EmployeeCreate) (dto.Employee, error)
	UpdateEmployee(ctx context.Context, session dto.Session, proof entity.SessionProof, id int32, req dto.EmployeeUpdate) (dto.Employee, error)
}
