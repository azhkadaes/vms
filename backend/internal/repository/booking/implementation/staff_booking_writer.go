package bookingrepositoryimplementation

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authimplementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
	staffprojection "github.com/azhkadaes/vms/backend/internal/repository/staffbooking/implementation"
)

var _ bookingrepo.StaffBookingWriter = (*repository)(nil)

func NewStaffBookingWriter(db *gorm.DB) bookingrepo.StaffBookingWriter { return &repository{db: db} }

type staffBookingTx struct {
	*bookingTx
	authrepository.AuthorityLock
}

func (r *repository) StaffTransaction(ctx context.Context, fn func(bookingrepo.StaffBookingTransaction) error) error {
	return r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		return fn(&staffBookingTx{bookingTx: &bookingTx{db: db}, AuthorityLock: authimplementation.NewAuthorityLock(db)})
	})
}

func (t *staffBookingTx) ReadStaffVisit(id int32) (entity.StaffVisit, error) {
	return staffprojection.ReadStaffVisit(t.db, id, true)
}

func (t *staffBookingTx) ActiveRoom(id int32) error {
	var found int32
	result := t.db.Raw("SELECT id FROM vms.ruangan WHERE id = ? AND status_aktif", id).Scan(&found)
	return rowError(result, "read active room")
}

func (t *staffBookingTx) LockScheduleResources(officerIDs, roomIDs []int32) error {
	return LockScheduleResources(t.db, officerIDs, roomIDs)
}

func (t *staffBookingTx) ScheduleConflict(excludeVisitID, officerID int32, roomID *int32, start time.Time, duration int) (bool, error) {
	return ScheduleConflict(t.db, excludeVisitID, officerID, roomID, start, duration)
}

func (t *staffBookingTx) CreateStaffVisit(visit entity.StaffVisit, photoID, documentID *int32, actorID int32, now time.Time, ip string) (int32, error) {
	var scheduleID int32
	result := t.db.Raw(`INSERT INTO vms.detail_kunjungan
		(id_pejabat,id_ruangan,id_formdocuments,tanggal,waktu_mulai,visit_duration,keperluan,catatan,catatan_internal,no_nd)
		VALUES (?,?,?,?::date,?::time,?,NULLIF(?,''),?,?,?) RETURNING id`,
		visit.OfficerID, visit.RoomID, documentID, visit.Date, visit.Time, visit.Duration,
		visit.Purpose, visit.Instructions, visit.CoordinationNote, visit.ServiceID).Scan(&scheduleID)
	if err := rowError(result, "insert staff visit schedule"); err != nil {
		return 0, err
	}
	var visitorID int32
	var err error
	if visit.Kind == "terbatas" {
		result := t.db.Raw(`INSERT INTO vms.detail_pengunjung (nama,asal_instansi)
			VALUES (NULLIF(?,''), NULLIF(?,'')) RETURNING id`, visit.Name, visit.Organization).Scan(&visitorID)
		err = rowError(result, "insert restricted visit representative")
	} else {
		visitorID, err = t.insertVisitRoster(visit.Visit, photoID)
	}
	if err != nil {
		return 0, err
	}
	id, err := t.insertVisit(visit.Kind, "approved", visitorID, scheduleID, now)
	if err != nil {
		return 0, err
	}
	err = t.db.Exec(`INSERT INTO vms.audit_log
		(actor_type,id_pegawai,entity_type,entity_id,jenis_kegiatan,tanggal,deskripsi,new_values,ip_address)
		VALUES ('staff',?,'kunjungan',?,'created',?,'Pendaftaran kunjungan oleh petugas',
			json_build_object('actorName',?::text,'actorRole',?::text),?)`,
		actorID, id, now, visit.CreatorName, visit.CreatorRole, ip).Error
	if err != nil {
		return 0, fmt.Errorf("audit staff visit creation: %w", err)
	}
	return id, nil
}
