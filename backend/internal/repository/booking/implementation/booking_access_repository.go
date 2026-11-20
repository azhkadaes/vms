package bookingrepositoryimplementation

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
)

func NewBookingAccessRepository(db *gorm.DB) bookingrepo.BookingAccessRepository {
	return &repository{db: db}
}

func (r *repository) AccessTransaction(ctx context.Context, fn func(bookingrepo.BookingAccessTransaction) error) error {
	return r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error { return fn(&bookingTx{db: db}) })
}

func (t *bookingTx) PublicVisit(code, email string, expectedID *int32) (entity.Visit, error) {
	query := `SELECT v.id FROM vms.kunjungan AS v
		JOIN vms.detail_pengunjung AS p ON p.id = v.id_detail_pengunjung
		WHERE v.kode_kunjungan = ? AND lower(p.email) = ? AND v.tipe_kunjungan <> 'terbatas'`
	args := []any{code, email}
	if expectedID == nil {
		query += ` FOR SHARE OF v`
	} else {
		query += ` AND v.id = ? FOR UPDATE OF v`
		args = append(args, *expectedID)
	}
	var id int32
	if err := rowError(t.db.Raw(query, args...).Scan(&id), "lock public visit"); err != nil {
		return entity.Visit{}, err
	}
	return t.ReadVisit(id)
}

func (t *bookingTx) PublicHistory(id int32) ([]entity.VisitHistory, error) {
	history := make([]entity.VisitHistory, 0)
	err := t.db.Raw(`SELECT tanggal AS at, jenis_kegiatan AS action,
		CASE WHEN jenis_kegiatan = 'rescheduled' AND json_typeof(new_values->'reason') = 'string'
			THEN nullif(new_values->>'reason', '') END AS public_message
		FROM vms.audit_log
		WHERE entity_type = 'kunjungan' AND entity_id = ?
		AND jenis_kegiatan IN ('created', 'approved', 'rejected', 'rescheduled', 'cancelled', 'checked_in', 'checked_out')
		ORDER BY tanggal, id`, id).Scan(&history).Error
	if err != nil {
		return nil, fmt.Errorf("read visitor history: %w", err)
	}
	return history, nil
}

func (t *bookingTx) CancelPublic(id int32, now time.Time, ip string) (int, error) {
	var version int
	result := t.db.Raw(`UPDATE vms.kunjungan SET status_penerimaan = 'cancelled', version = version + 1
		WHERE id = ? RETURNING version`, id).Scan(&version)
	if err := rowError(result, "cancel public visit"); err != nil {
		return 0, err
	}
	if err := RevokeVisitQR(t.db, id, now); err != nil {
		return 0, err
	}
	err := t.db.Exec(`INSERT INTO vms.audit_log
		(actor_type, entity_type, entity_id, jenis_kegiatan, tanggal, deskripsi, new_values, ip_address)
		VALUES ('visitor', 'kunjungan', ?, 'cancelled', ?, 'Pembatalan oleh pengunjung',
			json_build_object('version', ?::integer), ?)`, id, now, version, ip).Error
	if err != nil {
		return 0, fmt.Errorf("record visitor cancellation: %w", err)
	}
	return version, nil
}
