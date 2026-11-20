package bookingrepositoryimplementation

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepo "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authimplementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	staffrepo "github.com/azhkadaes/vms/backend/internal/repository/staffbooking"
	staffprojection "github.com/azhkadaes/vms/backend/internal/repository/staffbooking/implementation"
)

type decisions struct{ db *gorm.DB }

func NewDecisionRepository(db *gorm.DB) staffrepo.DecisionRepository { return decisions{db} }

func (r decisions) DecisionTransaction(ctx context.Context, fn func(staffrepo.DecisionTransaction) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(decisionTx{db: tx, AuthorityLock: authimplementation.NewAuthorityLock(tx)})
	})
}

type decisionTx struct {
	authrepo.AuthorityLock
	db *gorm.DB
}

func (t decisionTx) LockVisit(id int32) (entity.StaffVisit, error) {
	var locked int32
	result := t.db.Raw("SELECT id FROM vms.kunjungan WHERE id = ? FOR UPDATE", id).Scan(&locked)
	if result.Error != nil {
		return entity.StaffVisit{}, fmt.Errorf("lock visit: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return entity.StaffVisit{}, staffrepo.ErrNotFound
	}
	return staffprojection.ReadStaffVisit(t.db, locked, false)
}

func (t decisionTx) Now() (time.Time, error) {
	var now time.Time
	err := t.db.Raw("SELECT clock_timestamp()").Scan(&now).Error
	return now, err
}

func (t decisionTx) ActiveOfficer(id int32) (bool, error) {
	var active bool
	err := t.db.Raw("SELECT EXISTS (SELECT FROM vms.pejabat WHERE id = ? AND status_aktif)", id).Scan(&active).Error
	return active, err
}

func (t decisionTx) ActiveRoom(id int32) (bool, error) {
	var active bool
	err := t.db.Raw("SELECT EXISTS (SELECT FROM vms.ruangan WHERE id = ? AND status_aktif)", id).Scan(&active).Error
	return active, err
}

func (t decisionTx) LockResources(officers, rooms []int32) error {
	return LockScheduleResources(t.db, officers, rooms)
}

func (t decisionTx) HasConflict(visitID int32, s entity.VisitSchedule, start time.Time) (bool, error) {
	return ScheduleConflict(t.db, visitID, s.OfficerID, s.RoomID, start, s.Duration)
}

func (t decisionTx) ApplyChange(id int32, version int, actor int32, role string, now time.Time, change entity.VisitChange) (entity.StaffVisit, error) {
	if s := change.Schedule; s != nil {
		if err := t.db.Exec(`UPDATE vms.detail_kunjungan d SET id_pejabat=?, id_ruangan=?, tanggal=?::date, waktu_mulai=?::time,
			visit_duration=?, catatan=?, catatan_internal=?, no_nd=?
			FROM vms.kunjungan v WHERE v.id=? AND d.id=v.id_detail_kunjungan`,
			s.OfficerID, s.RoomID, s.Date, s.Time, s.Duration, s.Instructions, s.Coordination, s.ServiceID, id).Error; err != nil {
			return entity.StaffVisit{}, fmt.Errorf("update visit schedule: %w", err)
		}
	}
	if x := change.Identity; x != nil {
		if err := t.db.Exec(`UPDATE vms.detail_pengunjung p SET nama=?, asal_instansi=?
			FROM vms.kunjungan v WHERE v.id=? AND p.id=v.id_detail_pengunjung`, x.Name, x.Organization, id).Error; err != nil {
			return entity.StaffVisit{}, fmt.Errorf("update restricted visitor: %w", err)
		}
	}
	result := t.db.Exec(`UPDATE vms.kunjungan SET status_penerimaan=?::vms.status_penerimaan, version=version+1,
		alasan_penolakan=CASE WHEN ?='rejected' THEN ? ELSE alasan_penolakan END,
		catatan_internal_penolakan=CASE WHEN ?='rejected' THEN ? ELSE catatan_internal_penolakan END,
		alasan_pembatalan=CASE WHEN ?='cancelled' THEN ? ELSE alasan_pembatalan END,
		catatan_internal_pembatalan=CASE WHEN ?='cancelled' THEN ? ELSE catatan_internal_pembatalan END
		WHERE id=? AND version=?`, change.Status, change.AuditType, change.Reason, change.AuditType, change.Internal,
		change.AuditType, change.Reason, change.AuditType, change.Internal, id, version)
	if result.Error != nil || result.RowsAffected != 1 {
		return entity.StaffVisit{}, fmt.Errorf("write locked visit transition (rows %d): %v", result.RowsAffected, result.Error)
	}
	if change.RevokeQR {
		if err := RevokeVisitQR(t.db, id, now); err != nil {
			return entity.StaffVisit{}, err
		}
	}
	err := t.db.Exec(`INSERT INTO vms.audit_log
		(actor_type,id_pegawai,entity_type,entity_id,jenis_kegiatan,tanggal,deskripsi,old_values,new_values)
		VALUES ('staff',?,'kunjungan',?,?,?,?,
			json_build_object('version',?::integer),
			json_build_object('version',?::integer,'actorRole',?::text,'reason',nullif(?::text,'')))`,
		actor, id, change.AuditType, now, change.Message, version, version+1, role, change.Reason).Error
	if err != nil {
		return entity.StaffVisit{}, fmt.Errorf("audit visit transition: %w", err)
	}
	return staffprojection.ReadStaffVisit(t.db, id, true)
}
