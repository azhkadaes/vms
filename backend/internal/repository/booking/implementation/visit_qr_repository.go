package bookingrepositoryimplementation

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
)

func (t *bookingTx) CurrentQR(visitID int32) (entity.VisitQR, error) {
	var qr entity.VisitQR
	result := t.db.Raw(`SELECT id,id_kunjungan AS visit_id,token_hash,tanggal_dibuat AS created_at,tanggal_expire AS expires_at
		FROM vms.qr_kunjungan WHERE id_kunjungan = ? AND dicabut_pada IS NULL`, visitID).Scan(&qr)
	return qr, rowError(result, "read current visit QR")
}

func (t *bookingTx) ReserveQRID() (int32, error) {
	var id int32
	result := t.db.Raw(`SELECT nextval('vms.qr_kunjungan_id_seq')`).Scan(&id)
	return id, rowError(result, "reserve visit QR ID")
}

func (t *bookingTx) InsertQR(qr entity.VisitQR) error {
	if err := t.db.Exec(`INSERT INTO vms.qr_kunjungan (id,id_kunjungan,token_hash,tanggal_dibuat,tanggal_expire)
		OVERRIDING SYSTEM VALUE VALUES (?,?,?,?,?)`, qr.ID, qr.VisitID, qr.TokenHash, qr.CreatedAt, qr.ExpiresAt).Error; err != nil {
		return fmt.Errorf("insert visit QR: %w", err)
	}
	return nil
}

// RevokeVisitQR requires the caller to hold the visit row lock in tx.
func RevokeVisitQR(tx *gorm.DB, visitID int32, now time.Time) error {
	if err := tx.Exec(`UPDATE vms.qr_kunjungan SET dicabut_pada = ? WHERE id_kunjungan = ? AND dicabut_pada IS NULL`, now, visitID).Error; err != nil {
		return fmt.Errorf("revoke visit QR: %w", err)
	}
	return nil
}
