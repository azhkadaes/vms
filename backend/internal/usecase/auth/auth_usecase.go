package authusecase

import (
	"context"
	"errors"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
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
	switch policy.Stage(session) {
	case policy.StageSignedOut:
		return ErrUnauthenticated
	case policy.StageWorkspace:
		return nil
	}
	return ErrWorkspaceBlocked
}

// RequireSessionOperation admits a session to op only in the stages A-18 allows.
func RequireSessionOperation(session dto.Session, op policy.SessionOperation) error {
	if policy.Stage(session) == policy.StageSignedOut {
		return ErrUnauthenticated
	}
	if !policy.AllowsSessionOperation(session, op) {
		return ErrWorkspaceBlocked
	}
	return nil
}
