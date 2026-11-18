package authrepository

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
)

type AuthRepository interface {
	GetSessionSnapshot(ctx context.Context, sessionID string) (entity.SessionSnapshot, error)
}
