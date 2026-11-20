package authrepositoryimplementation

import (
	"fmt"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	"gorm.io/gorm"
)

type authorityLock struct{ db *gorm.DB }

var _ authrepository.AuthorityLock = (*authorityLock)(nil)

// NewAuthorityLock binds the authority recheck to a write transaction; module transactions embed it.
func NewAuthorityLock(tx *gorm.DB) authrepository.AuthorityLock { return &authorityLock{db: tx} }

// LockAuthority takes the transaction's first row lock on the caller's employee. Every session,
// credential, role and policy writer locks that employee too, so the snapshot read next on the
// same transaction cannot change until it commits.
func (t *authorityLock) LockAuthority(proof entity.SessionProof) (entity.SessionSnapshot, error) {
	var locked int32
	if err := t.db.Raw("SELECT id FROM vms.pegawai WHERE id = ? FOR UPDATE", proof.EmployeeID).Scan(&locked).Error; err != nil {
		return entity.SessionSnapshot{}, fmt.Errorf("lock employee authority: %w", err)
	}
	if locked <= 0 || locked != proof.EmployeeID || !canonicalSession(proof.SessionID) {
		return entity.SessionSnapshot{}, nil
	}
	out, found, err := readSnapshot(t.db, lockedSnapshotSQL, proof.SessionID)
	if err != nil {
		return entity.SessionSnapshot{}, fmt.Errorf("read locked session authority: %w", err)
	}
	if !found || out.EmployeeID != proof.EmployeeID {
		return entity.SessionSnapshot{}, nil
	}
	return out, nil
}
