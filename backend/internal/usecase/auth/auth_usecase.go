package authusecase

import (
	"context"
	"errors"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

var ErrUnauthenticated = errors.New("unauthenticated")
var ErrRoleChanged = errors.New("role_changed")
var ErrWorkspaceBlocked = errors.New("workspace_blocked")

type AuthUsecase interface {
	GetSession(ctx context.Context, proof entity.SessionProof) (dto.Session, error)
}

// RequireWorkspace is an additional gate; resource usecases still enforce permission and scope.
func RequireWorkspace(session dto.Session) error {
	if !session.Active {
		return ErrUnauthenticated
	}
	if session.RoleSelectionRequired || session.Role == "" || session.MustChangePassword {
		return ErrWorkspaceBlocked
	}
	return nil
}
