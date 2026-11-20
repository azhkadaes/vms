package authrepositoryimplementation

import (
	"fmt"

	"gorm.io/gorm"
)

// LockAccessWriters serializes employee and role administration before either acquires
// employee row locks. The two operations can each hold the caller's row while needing
// another employee's row, so separate locks would allow a cycle across them.
func LockAccessWriters(tx *gorm.DB) error {
	const accessWriters int64 = 0x564D53524F4C45
	if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", accessWriters).Error; err != nil {
		return fmt.Errorf("wait for access writers: %w", err)
	}
	return nil
}
