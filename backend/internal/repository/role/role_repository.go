package rolerepository

import (
	"context"
	"fmt"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// ErrNotFound marks an unknown role ID.
var ErrNotFound = fmt.Errorf("role: %w", failure.ErrNotFound)

// RoleRepository reads the four fixed roles and runs role permission changes.
type RoleRepository interface {
	// GetRoles reads every role with its active policy's permissions from one snapshot, in ID order.
	GetRoles(ctx context.Context) ([]entity.Role, error)
	// Transaction runs fn in one write transaction. Role permission writers run one at a time,
	// so each takes the employee locks it needs without waiting on another one.
	Transaction(ctx context.Context, fn func(tx RoleTransaction) error) error
}

// RoleTransaction is a role permission change. Lock order: the caller's employee
// (LockAuthority), the role, then the employees whose live sessions use the role.
type RoleTransaction interface {
	authrepository.AuthorityLock
	// LockRole locks the role row and reads its active policy's permissions; ErrNotFound when absent.
	LockRole(id int32) (entity.Role, error)
	// SavePermissions makes permissions the active policy's grants, increments the version from
	// role.Version and stamps updated_at/updated_by. It returns the role as now stored.
	SavePermissions(role entity.Role, permissions []string, actor int32, at time.Time) (entity.Role, error)
	// BumpRoleRevision locks every employee with a live session on the role, in ascending ID order,
	// then increments those sessions' role revision so their next request answers role_changed.
	BumpRoleRevision(roleID int32, at time.Time) error
	// AuditPermissionChange appends the staff actor's change to the role's audit history.
	AuditPermissionChange(actor int32, before, after entity.Role, at time.Time) error
}
