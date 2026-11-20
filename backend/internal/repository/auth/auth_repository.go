package authrepository

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
)

type AuthRepository interface {
	GetSessionSnapshot(ctx context.Context, sessionID string) (entity.SessionSnapshot, error)
}

// AuthorityLock is embedded in every write transaction that rechecks the caller's authority.
type AuthorityLock interface {
	// LockAuthority locks the caller's employee row before any other row, then reads the session
	// snapshot on the same transaction. A missing employee or session, or a session of another
	// employee, yields a zero snapshot, which SessionValidator rejects as signed out.
	LockAuthority(proof entity.SessionProof) (entity.SessionSnapshot, error)
}
