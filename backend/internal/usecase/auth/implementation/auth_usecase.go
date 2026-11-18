package authusecaseimplementation

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type usecase struct {
	repo       authrepository.AuthRepository
	serializer authserializer.AuthSerializer
}

var _ authusecase.AuthUsecase = (*usecase)(nil)

func NewAuthUsecase(repo authrepository.AuthRepository, serializer authserializer.AuthSerializer) authusecase.AuthUsecase {
	return &usecase{repo: repo, serializer: serializer}
}

// GetSession requires verified token proof plus current database authority on every call.
// Mutations must recheck authority inside their own transaction before committing.
func (u *usecase) GetSession(ctx context.Context, proof entity.SessionProof) (dto.Session, error) {
	if proof.EmployeeID <= 0 || proof.CredentialGeneration <= 0 || proof.RoleRevision <= 0 {
		return dto.Session{}, authusecase.ErrUnauthenticated
	}
	in, err := u.repo.GetSessionSnapshot(ctx, proof.SessionID)
	if missing(err) {
		return dto.Session{}, authusecase.ErrUnauthenticated
	}
	if err != nil {
		return dto.Session{}, fmt.Errorf("current session: %w", err)
	}
	if !in.AccountActive || in.EmployeeID != proof.EmployeeID || in.CredentialGeneration != proof.CredentialGeneration || in.RevokedAt != nil ||
		in.ServerTime.IsZero() || in.ServerTime.Before(in.CreatedAt) || !in.ServerTime.Before(in.IdleExpiresAt) || !in.ServerTime.Before(in.AbsoluteExpiresAt) ||
		in.IdleExpiresAt.After(in.AbsoluteExpiresAt) || in.AbsoluteExpiresAt.After(in.CreatedAt.Add(24*time.Hour)) || len(in.AvailableRoles) == 0 {
		return dto.Session{}, authusecase.ErrUnauthenticated
	}
	if in.LastActivityAt.Before(in.CreatedAt) || in.LastActivityAt.After(in.ServerTime) || in.IdleExpiresAt.After(in.LastActivityAt.Add(8*time.Hour)) {
		return dto.Session{}, authusecase.ErrUnauthenticated
	}
	if in.RoleRevision != proof.RoleRevision || (in.ActiveRole != "" && !slices.Contains(in.AvailableRoles, in.ActiveRole)) {
		return dto.Session{}, authusecase.ErrRoleChanged
	}
	return u.serializer.ToSession(in), nil
}
