package receptionrepositoryimplementation

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authrepositoryimplementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	receptionrepository "github.com/azhkadaes/vms/backend/internal/repository/reception"
	staffbookingrepository "github.com/azhkadaes/vms/backend/internal/repository/staffbooking/implementation"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

type repo struct{ db *gorm.DB }

func NewReceptionRepository(db *gorm.DB) receptionrepository.ReceptionRepository {
	return &repo{db: db}
}

// MatchCredential checks an unrevoked QR or manual booking code. A nonzero visit ID binds the proof to that visit.
// Eligibility and authorization are checked separately by each caller.
func MatchCredential(tx *gorm.DB, credential entity.ReceptionCredential, visitID int32) (entity.ReceptionMatch, error) {
	var match entity.ReceptionMatch
	var query *gorm.DB
	tx = tx.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	switch {
	case credential.Code != "" && credential.TokenHash == "":
		query = tx.Raw(`SELECT id AS visit_id FROM vms.kunjungan WHERE kode_kunjungan=? AND (?=0 OR id=?)`, credential.Code, visitID, visitID)
	case credential.TokenHash != "" && credential.Code == "":
		query = tx.Raw(`SELECT id_kunjungan AS visit_id, tanggal_expire AS qr_expires_at
			FROM vms.qr_kunjungan WHERE token_hash=? AND dicabut_pada IS NULL AND (?=0 OR id_kunjungan=?)`, credential.TokenHash, visitID, visitID)
	default:
		return entity.ReceptionMatch{}, receptionrepository.ErrNotFound
	}
	if err := query.Scan(&match).Error; err != nil {
		return entity.ReceptionMatch{}, fmt.Errorf("match reception credential: %w", err)
	}
	if match.VisitID <= 0 {
		return entity.ReceptionMatch{}, receptionrepository.ErrNotFound
	}
	return match, nil
}

func (r *repo) Resolve(ctx context.Context, credential entity.ReceptionCredential) (entity.StaffVisit, entity.ReceptionMatch, time.Time, error) {
	var visit entity.StaffVisit
	var match entity.ReceptionMatch
	var at time.Time
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw("SELECT transaction_timestamp()").Scan(&at).Error; err != nil {
			return fmt.Errorf("get reception time: %w", err)
		}
		var err error
		if match, err = MatchCredential(tx, credential, 0); err != nil {
			return err
		}
		visit, err = staffbookingrepository.ReadStaffVisit(tx, match.VisitID, false)
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return visit, match, at, err
}

type attendanceTx struct {
	authrepository.AuthorityLock
	db *gorm.DB
}

func (r *repo) Transaction(ctx context.Context, fn func(receptionrepository.ReceptionTransaction) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&attendanceTx{AuthorityLock: authrepositoryimplementation.NewAuthorityLock(tx), db: tx})
	})
}

func (t *attendanceTx) LockVisit(id int32) (entity.StaffVisit, error) {
	var locked int32
	if err := t.db.Raw("SELECT id FROM vms.kunjungan WHERE id=? FOR UPDATE", id).Scan(&locked).Error; err != nil {
		return entity.StaffVisit{}, fmt.Errorf("lock reception visit: %w", err)
	}
	if locked <= 0 {
		return entity.StaffVisit{}, receptionrepository.ErrNotFound
	}
	return staffbookingrepository.ReadStaffVisit(t.db, locked, false)
}

func (t *attendanceTx) MatchCredential(credential entity.ReceptionCredential, id int32) (entity.ReceptionMatch, error) {
	return MatchCredential(t.db, credential, id)
}

func (t *attendanceTx) Now() (time.Time, error) {
	var at time.Time
	if err := t.db.Raw("SELECT clock_timestamp()").Scan(&at).Error; err != nil {
		return time.Time{}, fmt.Errorf("get attendance time: %w", err)
	}
	return at, nil
}

func (t *attendanceTx) Transition(id int32, version int, actor int32, action string, at time.Time) (entity.StaffVisit, error) {
	var result *gorm.DB
	var event, message string
	switch action {
	case policy.AttendanceCheckIn:
		event, message = "checked_in", "Kedatangan dicatat"
		result = t.db.Exec(`UPDATE vms.kunjungan SET status_penerimaan='checked_in',waktu_cek_in=?,version=version+1
			WHERE id=? AND version=? AND version<2147483647 AND status_penerimaan='approved'`, at, id, version)
	case policy.AttendanceCheckOut:
		event, message = "checked_out", "Kepulangan dicatat"
		result = t.db.Exec(`UPDATE vms.kunjungan SET status_penerimaan='checked_out',waktu_cek_out=?,version=version+1
			WHERE id=? AND version=? AND version<2147483647 AND status_penerimaan='checked_in'`, at, id, version)
	default:
		return entity.StaffVisit{}, fmt.Errorf("unsupported attendance transition")
	}
	if result.Error != nil {
		return entity.StaffVisit{}, fmt.Errorf("record attendance: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return entity.StaffVisit{}, failure.New(409, "version_conflict")
	}
	if err := t.db.Exec(`INSERT INTO vms.audit_log(actor_type,id_pegawai,entity_type,entity_id,jenis_kegiatan,tanggal,deskripsi,new_values)
		VALUES ('staff',?,'kunjungan',?,?,?,?,json_build_object('version',?::integer))`, actor, id, event, at, message, version+1).Error; err != nil {
		return entity.StaffVisit{}, fmt.Errorf("audit attendance: %w", err)
	}
	return staffbookingrepository.ReadStaffVisit(t.db, id, false)
}
