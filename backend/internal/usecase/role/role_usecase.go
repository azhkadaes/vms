package roleusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// RoleUsecase serves role administration (ops 30, 31); both need the roles permission.
type RoleUsecase interface {
	// GetRoles lists the four fixed roles for a session StaffAuth has just read.
	GetRoles(ctx context.Context, session dto.Session) ([]dto.RoleDefinition, error)
	// UpdateRole replaces role id's assignable permissions, rechecking the caller's authority in
	// the write transaction. Live sessions on that role then answer role_changed until refreshed.
	UpdateRole(ctx context.Context, proof entity.SessionProof, id int32, req dto.RoleUpdate) (dto.RoleDefinition, error)
}
