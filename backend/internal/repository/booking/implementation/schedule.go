package bookingrepositoryimplementation

import (
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
)

const (
	officerLockNamespace int32 = 0x564D5301
	roomLockNamespace    int32 = 0x564D5302
)

// LockScheduleResources gives every visit writer the same resource lock order: officers
// first, then rooms, in ascending ID order. The runtime has SELECT-only catalog access,
// so the locks are transaction-scoped rather than row locks on the catalog tables.
func LockScheduleResources(tx *gorm.DB, officerIDs, roomIDs []int32) error {
	if err := lockResources(tx, officerLockNamespace, officerIDs); err != nil {
		return fmt.Errorf("lock scheduled officers: %w", err)
	}
	if err := lockResources(tx, roomLockNamespace, roomIDs); err != nil {
		return fmt.Errorf("lock scheduled rooms: %w", err)
	}
	return nil
}

func lockResources(tx *gorm.DB, namespace int32, ids []int32) error {
	ordered := slices.Clone(ids)
	slices.Sort(ordered)
	for _, id := range slices.Compact(ordered) {
		if id <= 0 {
			continue
		}
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?::integer, ?::integer)", namespace, id).Error; err != nil {
			return err
		}
	}
	return nil
}

// ScheduleConflict checks approved and checked-in intervals while the writer holds the
// schedule locks. Pass zero for excludeVisitID when creating a new visit. Adjacent
// intervals and two different officers with no shared room do not conflict.
func ScheduleConflict(tx *gorm.DB, excludeVisitID, officerID int32, roomID *int32, start time.Time, duration int) (bool, error) {
	end := start.Add(time.Duration(duration) * time.Minute)
	var found bool
	err := tx.Raw(`SELECT EXISTS (
		SELECT 1 FROM vms.kunjungan v JOIN vms.detail_kunjungan d ON d.id = v.id_detail_kunjungan
		WHERE v.id <> ? AND v.status_penerimaan IN ('approved','checked_in')
		AND (d.id_pejabat = ? OR d.id_ruangan = ?)
		AND ((d.tanggal + d.waktu_mulai) AT TIME ZONE 'Asia/Makassar') < ?::timestamptz
		AND ((d.tanggal + d.waktu_mulai) AT TIME ZONE 'Asia/Makassar') + d.visit_duration * INTERVAL '1 minute' > ?::timestamptz
	)`, excludeVisitID, officerID, roomID, end.UTC(), start.UTC()).Scan(&found).Error
	if err != nil {
		return false, fmt.Errorf("check schedule conflict: %w", err)
	}
	return found, nil
}
