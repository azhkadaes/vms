package reportrepositoryimplementation

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	repo "github.com/azhkadaes/vms/backend/internal/repository/report"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type reportReader struct{ db *gorm.DB }

func NewReportReader(db *gorm.DB) repo.ReportReader { return &reportReader{db: db} }

func (r *reportReader) read(ctx context.Context, run func(*gorm.DB) error) (time.Time, error) {
	var at time.Time
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw("SELECT transaction_timestamp()").Scan(&at).Error; err != nil {
			return fmt.Errorf("read report snapshot time: %w", err)
		}
		return run(tx)
	}, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	return at, err
}

const visitStatus = `v.id,v.version,v.status_penerimaan::text AS status,d.visit_duration AS duration,
	(d.tanggal+d.waktu_mulai) AT TIME ZONE 'Asia/Makassar' AS starts_at`

const joinedSchedule = ` FROM vms.kunjungan v JOIN vms.detail_kunjungan d ON d.id=v.id_detail_kunjungan`

func (r *reportReader) Summary(ctx context.Context, session dto.Session, filter dto.SummaryFilter) ([]entity.ReportStatus, time.Time, error) {
	rows := []entity.ReportStatus{}
	at, err := r.read(ctx, func(tx *gorm.DB) error {
		query := `SELECT ` + visitStatus + joinedSchedule + ` WHERE v.tipe_kunjungan<>'terbatas' AND d.tanggal BETWEEN ?::date AND ?::date`
		args := []any{filter.From, filter.To}
		switch session.Role {
		case policy.RoleSuperAdmin, policy.RoleProtocol:
		case policy.RolePA:
			ids := make([]int32, 0, len(session.AssignedOfficerIDs))
			for _, value := range session.AssignedOfficerIDs {
				if id, err := validate.ID(value); err == nil {
					ids = append(ids, id)
				}
			}
			if len(ids) == 0 {
				return nil
			}
			query += ` AND v.tipe_kunjungan='umum' AND d.id_pejabat IN ?`
			args = append(args, ids)
		case policy.RoleReceptionist:
			query += ` AND v.status_penerimaan IN ('approved','checked_in','checked_out')`
		default:
			return nil
		}
		return tx.Raw(query+` ORDER BY v.id`, args...).Scan(&rows).Error
	})
	return rows, at, err
}

func (r *reportReader) VisitLog(ctx context.Context, filter dto.VisitLogFilter) (entity.ReportSnapshot, error) {
	result := entity.ReportSnapshot{Rows: []entity.ReportVisit{}, Officers: []entity.ReportOfficer{}, Rooms: []string{}}
	at, err := r.read(ctx, func(tx *gorm.DB) error {
		query := `SELECT ` + visitStatus + `,
			CASE WHEN v.tipe_kunjungan='terbatas' THEN 'TRB-' || v.id ELSE v.kode_kunjungan::text END AS reference,
			coalesce(nullif(p.nama,''),'Kunjungan terbatas') AS name,coalesce(p.asal_instansi,'') AS organization,
			d.id_pejabat AS officer_id,o.nama AS host,d.id_ruangan AS room_id,coalesce(r.nama,'') AS room,
			to_char(d.tanggal,'YYYY-MM-DD') AS date,to_char(d.tanggal+d.waktu_mulai,'HH24:MI') AS time,
			CASE v.tipe_kunjungan WHEN 'terbatas' THEN 'restricted' WHEN 'vip' THEN 'vip' ELSE 'standard' END AS kind,
			CASE WHEN v.tipe_kunjungan='terbatas' THEN '' ELSE coalesce(d.keperluan,'') END AS purpose,
			CASE WHEN v.tipe_kunjungan='terbatas' THEN NULL ELSE coalesce(p.jumlah_pengunjung,0) END AS visitors,
			v.waktu_cek_in AS checked_in_at,v.waktu_cek_out AS checked_out_at` + joinedSchedule + `
			JOIN vms.detail_pengunjung p ON p.id=v.id_detail_pengunjung
			JOIN vms.pejabat o ON o.id=d.id_pejabat LEFT JOIN vms.ruangan r ON r.id=d.id_ruangan WHERE TRUE`
		var args []any
		if filter.From != "" {
			query += ` AND d.tanggal>=?::date`
			args = append(args, filter.From)
		}
		if filter.To != "" {
			query += ` AND d.tanggal<=?::date`
			args = append(args, filter.To)
		}
		if err := tx.Raw(query+` ORDER BY v.id`, args...).Scan(&result.Rows).Error; err != nil {
			return fmt.Errorf("read visit log: %w", err)
		}
		if err := tx.Raw(`SELECT DISTINCT o.id,o.nama AS name` + joinedSchedule + `
			JOIN vms.pejabat o ON o.id=d.id_pejabat ORDER BY name,id`).Scan(&result.Officers).Error; err != nil {
			return fmt.Errorf("read visit log officer options: %w", err)
		}
		if err := tx.Raw(`SELECT DISTINCT r.nama` + joinedSchedule + `
			JOIN vms.ruangan r ON r.id=d.id_ruangan WHERE r.nama<>'' ORDER BY r.nama`).Scan(&result.Rooms).Error; err != nil {
			return fmt.Errorf("read visit log room options: %w", err)
		}
		return nil
	})
	result.At = at
	return result, err
}
