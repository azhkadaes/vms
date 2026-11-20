package authusecaseimplementation

import (
	"slices"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type sessionValidator struct {
	serializer authserializer.AuthSerializer
}

var _ authusecase.SessionValidator = (*sessionValidator)(nil)

func NewSessionValidator(serializer authserializer.AuthSerializer) authusecase.SessionValidator {
	return &sessionValidator{serializer: serializer}
}

// validProof rejects token claims that can never match a session, before any database read.
func validProof(proof entity.SessionProof) bool {
	return proof.EmployeeID > 0 && proof.CredentialGeneration > 0 && proof.RoleRevision > 0
}

// Validate accepts only an active, unrevoked, unexpired session whose credential generation and
// role revision still match the proof. A zero snapshot (no session row) signs the caller out.
func (u *sessionValidator) Validate(in entity.SessionSnapshot, proof entity.SessionProof) (dto.Session, error) {
	if !validProof(proof) {
		return dto.Session{}, authusecase.ErrUnauthenticated
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
