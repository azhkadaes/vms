package bookingrepositoryimplementation

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	admissionrepo "github.com/azhkadaes/vms/backend/internal/repository/admission/implementation"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
)

type repository struct{ db *gorm.DB }

func NewBookingRepository(db *gorm.DB) bookingrepo.BookingRepository { return &repository{db: db} }

func (r *repository) Transaction(ctx context.Context, fn func(bookingrepo.BookingTransaction) error) error {
	return r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error { return fn(&bookingTx{db: db}) })
}

type bookingTx struct{ db *gorm.DB }

func (t *bookingTx) LockReference(audience, ownerDigest, reference string) error {
	key := "vms.booking_requests:" + audience + ":" + ownerDigest + ":" + reference
	return t.db.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error
}

func (t *bookingTx) Request(audience, ownerDigest, reference string) (entity.BookingRequest, error) {
	var request entity.BookingRequest
	result := t.db.Raw(`SELECT state, coalesce(payload_digest,'') AS payload_digest,
		coalesce(email_digest,'') AS email_digest, coalesce(id_kunjungan,0) AS visit_id
		FROM support.booking_requests WHERE audience = ? AND owner_digest = ? AND reference = ?::uuid`,
		audience, ownerDigest, reference).Scan(&request)
	return request, rowError(result, "read booking request")
}

func (t *bookingTx) StoreRequest(audience, ownerDigest, reference string, request entity.BookingRequest) error {
	return t.db.Exec(`INSERT INTO support.booking_requests
		(audience, owner_digest, reference, state, payload_digest, email_digest, id_kunjungan)
		VALUES (?, ?, ?::uuid, ?, nullif(?,''), nullif(?,''), nullif(?,0))`,
		audience, ownerDigest, reference, request.State, request.PayloadDigest, request.EmailDigest, request.VisitID).Error
}

func (t *bookingTx) Now() (time.Time, error) {
	var now time.Time
	err := t.db.Raw("SELECT clock_timestamp()").Scan(&now).Error
	return now, err
}

const visitProjection = `SELECT v.id, v.kode_kunjungan AS code, v.tipe_kunjungan::text AS kind,
	v.status_penerimaan::text AS status, v.version, v.waktu_cek_in AS check_in, v.waktu_cek_out AS check_out,
	d.id_pejabat AS officer_id, o.nama AS officer_name, d.id_ruangan AS room_id, coalesce(r.nama,'') AS room,
	to_char(d.tanggal,'YYYY-MM-DD') AS date, to_char(d.tanggal + d.waktu_mulai,'HH24:MI') AS time, d.visit_duration AS duration,
	coalesce(d.keperluan,'') AS purpose, coalesce(p.nama,'') AS name, coalesce(p.asal_instansi,'') AS organization,
	coalesce(p.email,'') AS email, coalesce(p.nomor_telpon,'') AS phone, coalesce(p.identity_type::text,'') AS identity_type,
	coalesce(array_to_json(p.nama_rombongan)::text,'[]') AS roster, f.nama AS nd_name, d.catatan AS instructions,
	v.alasan_penolakan AS rejection, v.alasan_pembatalan AS cancellation
	FROM vms.kunjungan v
	JOIN vms.detail_kunjungan d ON d.id = v.id_detail_kunjungan
	JOIN vms.detail_pengunjung p ON p.id = v.id_detail_pengunjung
	JOIN vms.pejabat o ON o.id = d.id_pejabat
	LEFT JOIN vms.ruangan r ON r.id = d.id_ruangan
	LEFT JOIN vms.form_documents f ON f.id = d.id_formdocuments`

func (t *bookingTx) ReadVisit(id int32) (entity.Visit, error) {
	// A command must lock the visit before changing its details or adding history.
	var locked int32
	if err := rowError(t.db.Raw("SELECT id FROM vms.kunjungan WHERE id = ? FOR SHARE", id).Scan(&locked), "lock projected visit"); err != nil {
		return entity.Visit{}, err
	}
	return readVisit(t.db.Raw(visitProjection+" WHERE v.id = ?", id))
}

func readVisit(query *gorm.DB) (entity.Visit, error) {
	var row struct {
		entity.Visit `gorm:"embedded"`
		Roster       string
	}
	// Scan the roster's JSON separately from the projection's Go slice.
	if err := rowError(query.Scan(&row), "read visit"); err != nil {
		return entity.Visit{}, err
	}
	if err := json.Unmarshal([]byte(row.Roster), &row.Visit.Visitors); err != nil {
		return entity.Visit{}, fmt.Errorf("decode stored roster: %w", err)
	}
	return row.Visit, nil
}

func rowError(result *gorm.DB, action string) error {
	if result.Error != nil {
		return fmt.Errorf("%s: %w", action, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%s: %w", action, bookingrepo.ErrNotFound)
	}
	return nil
}

func (t *bookingTx) ActiveOfficer(id int32) (string, bool, error) {
	var officer struct {
		Name string
		VIP  bool `gorm:"column:vip"`
	}
	result := t.db.Raw(`SELECT p.nama AS name, j.pejabat_vip AS vip
		FROM vms.pejabat p JOIN vms.jabatan j ON j.id = p.id_jabatan
		WHERE p.id = ? AND p.status_aktif`, id).Scan(&officer)
	return officer.Name, officer.VIP, rowError(result, "read active officer")
}

func (t *bookingTx) LinkUpload(tokenHash, purpose, audience, ownerDigest string, now time.Time) (int32, error) {
	var id int32
	result := t.db.Raw(`UPDATE support.form_document_uploads AS u
		SET linked_at = ?
		FROM vms.form_documents AS d
		WHERE d.id = u.id_form_documents AND d.purpose = ? AND u.token_hash = ?
		AND u.audience = ? AND u.owner_digest = ? AND u.linked_at IS NULL
		AND u.created_at <= ? AND u.expires_at > clock_timestamp()
		RETURNING u.id_form_documents`, now, purpose, tokenHash, audience, ownerDigest, now).Scan(&id)
	return id, rowError(result, "link upload")
}

func (t *bookingTx) AdmitBooking(bucket string) (entity.RequestAdmission, error) {
	return admissionrepo.Admit(t.db, bucket, 30, time.Hour)
}

func (t *bookingTx) CreatePublicVisit(visit entity.Visit, photoID int32, documentID *int32, now time.Time, privacyVersion, ip string) (int32, error) {
	var scheduleID int32
	result := t.db.Raw(`INSERT INTO vms.detail_kunjungan (id_pejabat,id_formdocuments,tanggal,waktu_mulai,visit_duration,keperluan)
		VALUES (?,?,?::date,?::time,?,?) RETURNING id`, visit.OfficerID, documentID, visit.Date, visit.Time, visit.Duration, visit.Purpose).Scan(&scheduleID)
	if err := rowError(result, "insert visit schedule"); err != nil {
		return 0, err
	}
	visitorID, err := t.insertVisitRoster(visit, &photoID)
	if err != nil {
		return 0, err
	}
	id, err := t.insertVisit(visit.Kind, "pending", visitorID, scheduleID, now)
	if err != nil {
		return 0, err
	}
	err = t.db.Exec(`INSERT INTO vms.audit_log
		(actor_type,entity_type,entity_id,jenis_kegiatan,tanggal,deskripsi,new_values,ip_address)
		VALUES ('visitor','kunjungan',?,'created',?,'Pendaftaran kunjungan',json_build_object('privacyVersion',?::text),?)`,
		id, now, privacyVersion, ip).Error
	return id, err
}

func (t *bookingTx) insertVisitRoster(visit entity.Visit, photoID *int32) (int32, error) {
	roster, err := json.Marshal(visit.Visitors)
	if err != nil {
		return 0, err
	}
	var visitorID int32
	result := t.db.Raw(`INSERT INTO vms.detail_pengunjung
		(nama,asal_instansi,email,nomor_telpon,identity_type,nama_rombongan,jumlah_pengunjung,id_formdocuments)
		VALUES (?,?,?,?,?::vms.identity_type,ARRAY(SELECT json_array_elements_text(?::json)),?,?) RETURNING id`,
		visit.Name, visit.Organization, visit.Email, visit.Phone, visit.IdentityType, string(roster), len(visit.Visitors), photoID).Scan(&visitorID)
	if err := rowError(result, "insert visit roster"); err != nil {
		return 0, err
	}
	return visitorID, nil
}

func (t *bookingTx) insertVisit(kind, status string, visitorID, scheduleID int32, now time.Time) (int32, error) {
	for range 20 {
		code, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
		if err != nil {
			return 0, err
		}
		var id int32
		result := t.db.Raw(`INSERT INTO vms.kunjungan
			(id_detail_pengunjung,id_detail_kunjungan,kode_kunjungan,tipe_kunjungan,status_penerimaan,tanggal_dibuat)
			VALUES (?,?,?,?::vms.tipe_kunjungan,?::vms.status_penerimaan,?)
			ON CONFLICT (kode_kunjungan) DO NOTHING RETURNING id`,
			visitorID, scheduleID, fmt.Sprintf("%08d", code.Int64()), kind, status, now).Scan(&id)
		if result.Error != nil {
			return 0, fmt.Errorf("insert visit: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			continue
		}
		return id, nil
	}
	return 0, fmt.Errorf("could not allocate a unique visit code")
}
