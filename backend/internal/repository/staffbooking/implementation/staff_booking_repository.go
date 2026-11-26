package staffbookingrepositoryimplementation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	staffbookingrepository "github.com/azhkadaes/vms/backend/internal/repository/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type repo struct{ db *gorm.DB }

func NewStaffBookingReader(db *gorm.DB) staffbookingrepository.StaffBookingReader {
	return &repo{db: db}
}

const visitRows = `SELECT v.id, v.kode_kunjungan AS code, v.tipe_kunjungan::text AS kind,
	v.status_penerimaan::text AS status, v.version, v.waktu_cek_in AS check_in, v.waktu_cek_out AS check_out,
	d.id_pejabat AS officer_id, o.nama AS officer_name, d.id_ruangan AS room_id, coalesce(r.nama,'') AS room,
	to_char(d.tanggal,'YYYY-MM-DD') AS date, to_char(d.tanggal + d.waktu_mulai,'HH24:MI') AS time, d.visit_duration AS duration,
	coalesce(d.keperluan,'') AS purpose, coalesce(p.nama,'') AS name, coalesce(p.asal_instansi,'') AS organization,
	coalesce(p.email,'') AS email, coalesce(p.nomor_telpon,'') AS phone, coalesce(p.identity_type::text,'') AS identity_type,
	coalesce(array_to_json(p.nama_rombongan)::text,'[]') AS roster, coalesce(p.jumlah_pengunjung,0) AS visitor_count,
	nd.nama AS nd_name, nd.id IS NOT NULL AS attachment_available, photo.nama AS visitor_photo_name,
	d.no_nd AS service_id, d.catatan AS instructions, d.catatan_internal AS coordination_note,
	v.alasan_penolakan AS rejection, v.catatan_internal_penolakan AS internal_rejection,
	v.alasan_pembatalan AS cancellation, v.catatan_internal_pembatalan AS internal_cancellation,
	coalesce(created.actor_name,'') AS creator_name, coalesce(created.actor_role,'') AS creator_role,
	EXISTS (SELECT 1 FROM vms.audit_log a WHERE a.entity_type='kunjungan' AND a.entity_id=v.id
		AND a.jenis_kegiatan='rescheduled') AS rescheduled,
	(SELECT coalesce(json_agg(json_build_object('nama', e.nama, 'nomorHp', e.nomor_hp) ORDER BY e.nama, e.id),'[]')::text
		FROM vms.pegawai_pejabat pp JOIN vms.pegawai e ON e.id=pp.id_pegawai
		WHERE pp.id_pejabat=d.id_pejabat AND pp.status_aktif AND e.status_aktif) AS officer_pa_json
	FROM vms.kunjungan v
	JOIN vms.detail_kunjungan d ON d.id=v.id_detail_kunjungan
	JOIN vms.detail_pengunjung p ON p.id=v.id_detail_pengunjung
	JOIN vms.pejabat o ON o.id=d.id_pejabat
	LEFT JOIN vms.ruangan r ON r.id=d.id_ruangan
	LEFT JOIN vms.form_documents nd ON nd.id=d.id_formdocuments AND nd.purpose='document'
	LEFT JOIN vms.form_documents photo ON photo.id=p.id_formdocuments AND photo.purpose='visitorPhoto'
	LEFT JOIN LATERAL (
		SELECT coalesce(a.new_values->>'actorName', e.nama) AS actor_name, a.new_values->>'actorRole' AS actor_role
		FROM vms.audit_log a JOIN vms.pegawai e ON e.id=a.id_pegawai
		WHERE a.entity_type='kunjungan' AND a.entity_id=v.id AND a.actor_type='staff' AND a.jenis_kegiatan='created'
		ORDER BY a.tanggal, a.id LIMIT 1
	) created ON true`

type staffVisitRow struct {
	entity.StaffVisit `gorm:"embedded"`
	Roster            string
	OfficerPAJSON     string `gorm:"column:officer_pa_json"`
}

func scope(s dto.Session) (string, []any) {
	if !policy.Has(s, policy.PermBookings) {
		return "FALSE", nil
	}
	switch s.Role {
	case policy.RoleProtocol:
		return "TRUE", nil
	case policy.RoleSuperAdmin:
		return "v.tipe_kunjungan='terbatas'", nil
	case policy.RoleReceptionist:
		return "v.status_penerimaan IN ('approved','checked_in','checked_out')", nil
	case policy.RolePA:
		ids := make([]int32, 0, len(s.AssignedOfficerIDs))
		for _, raw := range s.AssignedOfficerIDs {
			if id, err := validate.ID(raw); err == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) != 0 {
			return "d.id_pejabat IN ? AND v.tipe_kunjungan IN ('umum','terbatas')", []any{ids}
		}
	}
	return "FALSE", nil
}

func scanVisits(tx *gorm.DB, condition string, args []any) ([]entity.StaffVisit, error) {
	var rows []staffVisitRow
	if err := tx.Raw(visitRows+" WHERE "+condition+" ORDER BY v.id", args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("read staff visits: %w", err)
	}
	visits := make([]entity.StaffVisit, len(rows))
	for i, row := range rows {
		if err := json.Unmarshal([]byte(row.Roster), &row.Visitors); err != nil {
			return nil, fmt.Errorf("read staff visit roster: %w", err)
		}
		if err := json.Unmarshal([]byte(row.OfficerPAJSON), &row.OfficerPAs); err != nil {
			return nil, fmt.Errorf("read staff visit officer PAs: %w", err)
		}
		visits[i] = row.StaffVisit
	}
	return visits, nil
}

func (r *repo) read(ctx context.Context, run func(*gorm.DB) error) (time.Time, error) {
	var at time.Time
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw("SELECT transaction_timestamp()").Scan(&at).Error; err != nil {
			return fmt.Errorf("get staff visits time: %w", err)
		}
		return run(tx)
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return at, err
}

func (r *repo) List(ctx context.Context, session dto.Session) ([]entity.StaffVisit, time.Time, error) {
	var visits []entity.StaffVisit
	at, err := r.read(ctx, func(tx *gorm.DB) error {
		condition, args := scope(session)
		var err error
		visits, err = scanVisits(tx, condition, args)
		return err
	})
	return visits, at, err
}

func (r *repo) Find(ctx context.Context, session dto.Session, id int32) (entity.StaffVisit, time.Time, error) {
	var visit entity.StaffVisit
	at, err := r.read(ctx, func(tx *gorm.DB) error {
		condition, args := scope(session)
		var err error
		visit, err = readOne(tx, "("+condition+") AND v.id=?", append(args, id), true, session.Role == policy.RolePA)
		return err
	})
	return visit, at, err
}

// ReadStaffVisit reads a visit inside the caller's transaction. The caller must enforce visit scope.
func ReadStaffVisit(tx *gorm.DB, id int32, includeHistory bool) (entity.StaffVisit, error) {
	return readOne(tx, "v.id=?", []any{id}, includeHistory, false)
}

func readOne(tx *gorm.DB, condition string, args []any, includeHistory, hideRestrictedHistory bool) (entity.StaffVisit, error) {
	visits, err := scanVisits(tx, condition, args)
	if err != nil {
		return entity.StaffVisit{}, err
	}
	if len(visits) != 1 {
		return entity.StaffVisit{}, staffbookingrepository.ErrNotFound
	}
	visit := visits[0]
	if includeHistory && !(hideRestrictedHistory && visit.Kind == "terbatas") {
		visit.History = []entity.StaffVisitEvent{}
		if err := tx.Raw(`SELECT tanggal AS at, deskripsi AS message FROM vms.audit_log
			WHERE entity_type='kunjungan' AND entity_id=? AND jenis_kegiatan IN
			('created','approved','rejected','rescheduled','cancelled','checked_in','checked_out','updated')
			ORDER BY tanggal,id`, visit.ID).Scan(&visit.History).Error; err != nil {
			return entity.StaffVisit{}, fmt.Errorf("read staff visit history: %w", err)
		}
	}
	return visit, nil
}
