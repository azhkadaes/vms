package authusecase

import (
	"context"
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"time"
)

// CredentialResult has transport-only cookie data, excluded from JSON.
type CredentialResult struct {
	Result            dto.TokenResult
	Refresh           string
	SessionID         string
	AbsoluteExpiresAt time.Time
}
type SessionUsecase interface {
	Login(ctx context.Context, request dto.Login, ip string) (CredentialResult, error)
	Refresh(ctx context.Context, secret, requestID string) (CredentialResult, error)
	Logout(ctx context.Context, secret string) error
	Renew(ctx context.Context, proof entity.SessionProof) (dto.Session, error)
	SwitchRole(ctx context.Context, proof entity.SessionProof, role string) (CredentialResult, error)
	ChangePassword(ctx context.Context, proof entity.SessionProof, request dto.PasswordChange) (CredentialResult, error)
	Binding(ctx context.Context, secret string, allowConsumed bool) (string, time.Time, error)
}
