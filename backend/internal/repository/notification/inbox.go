package notification

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const pageSize = 25

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) repository { return repository{db: db} }

type Item struct {
	ID      string     `json:"id"`
	VisitID string     `json:"visitId"`
	Event   string     `json:"event"`
	At      time.Time  `json:"at"`
	ReadAt  *time.Time `json:"readAt"`
}

type Page struct {
	Items       []Item `json:"items"`
	UnreadCount int    `json:"unreadCount"`
	NextBefore  string `json:"nextBefore,omitempty"`
}

const eligibility = `
	a.id = n.event_id AND k.id = a.entity_id AND d.id = k.id_detail_kunjungan
	AND n.employee_id = ? AND n.recipient_role = ?
	AND (n.recipient_role = 'Protocol Officer' AND k.tipe_kunjungan IN ('vip','terbatas') OR
		n.recipient_role = 'PA' AND k.tipe_kunjungan = 'umum' AND EXISTS (
			SELECT 1 FROM vms.pegawai_pejabat p
			WHERE p.id_pegawai = n.employee_id AND p.id_pejabat = d.id_pejabat AND p.status_aktif
		))
`

const inboxTables = `vms.audit_log a, vms.kunjungan k, vms.detail_kunjungan d`

func (r repository) List(ctx context.Context, employee int32, role string, before int32) (Page, error) {
	result := Page{Items: []Item{}}
	db := r.db.WithContext(ctx)
	from := " FROM support.staff_notifications n, " + inboxTables + " WHERE " + eligibility
	if err := db.Raw("SELECT count(*)"+from+" AND n.read_at IS NULL", employee, role).
		Scan(&result.UnreadCount).Error; err != nil {
		return Page{}, fmt.Errorf("count unread notifications: %w", err)
	}
	err := db.Raw(`SELECT n.event_id::text AS id, k.id::text AS visit_id, a.jenis_kegiatan AS event,
		a.tanggal AS at, n.read_at`+from+
		" AND (? = 0 OR n.event_id < ?) ORDER BY n.event_id DESC LIMIT ?",
		employee, role, before, before, pageSize+1).Scan(&result.Items).Error
	if err != nil {
		return Page{}, fmt.Errorf("list notifications: %w", err)
	}
	if len(result.Items) > pageSize {
		result.Items = result.Items[:pageSize]
		result.NextBefore = result.Items[pageSize-1].ID
	}
	return result, nil
}

func (r repository) MarkRead(ctx context.Context, employee int32, role string, event int32) (bool, error) {
	result := r.db.WithContext(ctx).Exec(`UPDATE support.staff_notifications n
		SET read_at = COALESCE(n.read_at, transaction_timestamp()) FROM `+inboxTables+
		" WHERE "+eligibility+" AND n.event_id = ?", employee, role, event)
	if result.Error != nil {
		return false, fmt.Errorf("mark notification %d as read: %w", event, result.Error)
	}
	return result.RowsAffected > 0, nil
}
