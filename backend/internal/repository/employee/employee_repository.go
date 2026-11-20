package employeerepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// ErrNotFound marks an unknown employee ID.
var ErrNotFound = fmt.Errorf("employee: %w", failure.ErrNotFound)

// ErrEmailTaken marks a write refused by the unique email constraint.
var ErrEmailTaken = errors.New("email_taken")

// EmployeeRepository reads employees and runs employee administration writes.
type EmployeeRepository interface {
	// GetEmployees reads one page, ordered by name then ID, and both counts from one snapshot.
	GetEmployees(ctx context.Context, filter entity.EmployeeFilter) (entity.EmployeePage, error)
	// GetEmployee reads one employee; ErrNotFound when absent.
	GetEmployee(ctx context.Context, id int32) (entity.Employee, error)
	// Transaction runs fn in one write transaction. Employee writers run one at a time, so two
	// administrators editing each other cannot deadlock and the last-Super-Admin count holds.
	Transaction(ctx context.Context, fn func(tx EmployeeTransaction) error) error
}

// EmployeeTransaction is one employee write. Lock order: the caller's employee (LockAuthority),
// then the target employee. Times come from the locked session snapshot's ServerTime.
type EmployeeTransaction interface {
	authrepository.AuthorityLock
	// LockEmployee locks the target employee and reads it with its roles and assignment;
	// ErrNotFound when absent.
	LockEmployee(id int32) (entity.Employee, error)
	// EmailInUse reports whether an employee other than except already has email.
	EmailInUse(email string, except int32) (bool, error)
	// Officers reads the officials with these IDs; unknown IDs are left out.
	Officers(ids []int32) ([]entity.EmployeeOfficer, error)
	// OtherActiveSuperAdmins counts active employees other than except holding Super Admin.
	OtherActiveSuperAdmins(except int32) (int, error)
	// CreateEmployee stores the profile, roles, assignment and password verifier at version 1
	// and returns the employee as stored. ErrEmailTaken when the email is already used.
	CreateEmployee(employee entity.Employee, encodedHash string) (entity.Employee, error)
	// SaveEmployee writes after over before: the profile with the version incremented from
	// before.Version, then the changed roles and assignment. ErrEmailTaken when the email is used.
	SaveEmployee(before, after entity.Employee) (entity.Employee, error)
	// EndSessions revokes the employee's unrevoked sessions and bumps their credential generation.
	EndSessions(employeeID int32, at time.Time) error
	// ReloadSessions bumps the role revision of the employee's unrevoked sessions; a session
	// acting in one of removedRoles returns to role selection.
	ReloadSessions(employeeID int32, removedRoles []string, at time.Time) error
	// Audit appends the staff actor's change to the employee target's history.
	Audit(event entity.EmployeeEvent) error
}
