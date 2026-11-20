package bookingfilerepositoryimplementation

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	bookingfiles "github.com/azhkadaes/vms/backend/internal/repository/bookingfile"
	reception "github.com/azhkadaes/vms/backend/internal/repository/reception/implementation"
)

type repository struct{ db *gorm.DB }

func NewBookingFileRepository(db *gorm.DB) bookingfiles.BookingFileRepository {
	return &repository{db: db}
}

func (r *repository) Lookup(ctx context.Context, id int32, purpose string, credential *entity.ReceptionCredential) (entity.BookingFile, error) {
	var output entity.BookingFile
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var match entity.ReceptionMatch
		var err error
		if credential != nil {
			match, err = reception.MatchCredential(tx, *credential, id)
			if err != nil {
				return err
			}
		}
		output, err = readMetadata(tx, id, purpose)
		output.Match = match
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return entity.BookingFile{}, fmt.Errorf("read booking file: %w", err)
	}
	return output, nil
}

func readMetadata(tx *gorm.DB, id int32, purpose string) (entity.BookingFile, error) {
	var column string
	switch purpose {
	case policy.PurposeDocument:
		column = "d.id_formdocuments"
	case policy.PurposeVisitorPhoto:
		column = "p.id_formdocuments"
	default:
		return entity.BookingFile{}, bookingfiles.ErrNotFound
	}
	var row entity.BookingFile
	result := tx.Raw(`SELECT f.storage_path AS key, f.mime_type AS mime, f.ukuran_file_bytes AS size,
		d.id_pejabat AS officer_id, v.tipe_kunjungan::text AS kind, v.status_penerimaan::text AS status,
		to_char(d.tanggal,'YYYY-MM-DD') AS date, to_char(d.tanggal+d.waktu_mulai,'HH24:MI') AS time,
		d.visit_duration AS duration, transaction_timestamp() AS server_time
		FROM vms.kunjungan v
		JOIN vms.detail_kunjungan d ON d.id=v.id_detail_kunjungan
		JOIN vms.detail_pengunjung p ON p.id=v.id_detail_pengunjung
		JOIN vms.form_documents f ON f.id=`+column+` AND f.purpose=?
		JOIN support.form_document_uploads u ON u.id_form_documents=f.id AND u.linked_at IS NOT NULL
		WHERE v.id=? AND v.tipe_kunjungan IN ('umum','vip')`, purpose, id).Scan(&row)
	if result.Error != nil {
		return entity.BookingFile{}, result.Error
	}
	if result.RowsAffected != 1 {
		return entity.BookingFile{}, bookingfiles.ErrNotFound
	}
	return row, nil
}
