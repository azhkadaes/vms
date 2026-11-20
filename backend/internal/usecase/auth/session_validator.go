package authusecase

import (
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// SessionValidator decides whether a snapshot still carries the proof's authority.
// Current-session reads and write transactions (after LockAuthority) share these rules.
type SessionValidator interface {
	Validate(snapshot entity.SessionSnapshot, proof entity.SessionProof) (dto.Session, error)
}
