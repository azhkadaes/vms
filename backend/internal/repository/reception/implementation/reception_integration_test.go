//go:build integration

package receptionrepositoryimplementation_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authrepoimpl "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
	bookingrepoimpl "github.com/azhkadaes/vms/backend/internal/repository/booking/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	receptionrepository "github.com/azhkadaes/vms/backend/internal/repository/reception"
	implementation "github.com/azhkadaes/vms/backend/internal/repository/reception/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	staffbookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	bookingimpl "github.com/azhkadaes/vms/backend/internal/usecase/booking/implementation"
	receptionusecase "github.com/azhkadaes/vms/backend/internal/usecase/reception"
	receptionimpl "github.com/azhkadaes/vms/backend/internal/usecase/reception/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type clockedRepo struct {
	receptionrepository.ReceptionRepository
	at time.Time
}

func (r clockedRepo) Resolve(ctx context.Context, credential entity.ReceptionCredential) (entity.StaffVisit, entity.ReceptionMatch, time.Time, error) {
	visit, match, _, err := r.ReceptionRepository.Resolve(ctx, credential)
	return visit, match, r.at, err
}

func (r clockedRepo) Transaction(ctx context.Context, fn func(receptionrepository.ReceptionTransaction) error) error {
	return r.ReceptionRepository.Transaction(ctx, func(tx receptionrepository.ReceptionTransaction) error {
		return fn(clockedTx{tx, r.at})
	})
}

type clockedTx struct {
	receptionrepository.ReceptionTransaction
	at time.Time
}

func (t clockedTx) Now() (time.Time, error) { return t.at, nil }

func receptionAt(db *gorm.DB, at time.Time) receptionusecase.ReceptionUsecase {
	return receptionimpl.NewReceptionUsecase(clockedRepo{implementation.NewReceptionRepository(db), at},
		staffbookingserializer.NewStaffBookingSerializer(), authusecase.NewSessionValidator(authserializer.NewAuthSerializer()),
		noop.NewTracerProvider().Tracer("reception-integration"))
}

var scheduled = time.Date(2031, 1, 2, 12, 0, 0, 0, validate.WITA)

type clockedAdmissionRepo struct {
	bookingrepo.BookingAccessRepository
	at time.Time
}

func (r clockedAdmissionRepo) AccessTransaction(ctx context.Context, fn func(bookingrepo.BookingAccessTransaction) error) error {
	return r.BookingAccessRepository.AccessTransaction(ctx, func(tx bookingrepo.BookingAccessTransaction) error {
		return fn(clockedAdmissionTx{tx, r.at})
	})
}

type clockedAdmissionTx struct {
	bookingrepo.BookingAccessTransaction
	at time.Time
}

func (t clockedAdmissionTx) Now() (time.Time, error) { return t.at, nil }

func TestSameQRChecksOutAfterExpiryAndDoesNotResolveAgain(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	ctx := context.Background()
	visit, code, _ := fixtureVisit(t, tx, false)
	receptionist := fixtureReceptionist(t, tx)
	session := dto.Session{Active: true, Role: policy.RoleReceptionist, Permissions: []string{policy.PermCheckin, policy.PermCheckout}}
	accessRepo := bookingrepoimpl.NewBookingAccessRepository(tx)
	qrKey := []byte("synthetic-test-qr-signing-key-000")
	tracer := noop.NewTracerProvider().Tracer("admission-integration")
	arrivalTime, departureTime := scheduled.Add(-30*time.Minute), scheduled.Add(24*time.Hour)
	arrivalQR := bookingimpl.NewAdmissionUsecase(clockedAdmissionRepo{accessRepo, arrivalTime}, qrKey, tracer)
	departureQR := bookingimpl.NewAdmissionUsecase(clockedAdmissionRepo{accessRepo, departureTime}, qrKey, tracer)
	visitorProof := dto.PublicCancel{Code: code, Email: "visitor@example.test", Version: 1}
	firstTicket, err := arrivalQR.Issue(ctx, visit, visitorProof)
	if err != nil {
		t.Fatalf("issue QR before check-in: %v", err)
	}
	qr := firstTicket.Payload
	proof := dto.ReceptionLookup{ReceptionResolve: dto.ReceptionResolve{Credential: qr}}
	arrival := receptionAt(tx, arrivalTime)
	first, err := arrival.Resolve(ctx, session, proof)
	if err != nil || first.Status != "approved" || first.Version != 1 || first.VisitorCount == nil || *first.VisitorCount != 1 || len(first.Visitors) != 1 || first.Purpose == "" || !first.Timing.CanCheckIn {
		t.Fatalf("resolve at the opening boundary: %+v, %v", first, err)
	}
	expired := receptionAt(tx, scheduled.Add(time.Hour+time.Millisecond))
	if _, err = expired.Attendance(ctx, receptionist, visit, dto.ReceptionAttendance{Action: policy.AttendanceCheckIn, Credential: qr, Version: 1}); errorCode(err) != "credential_expired" {
		t.Fatalf("expected credential_expired at the confirm step, got %v", err)
	}
	checkedIn, err := arrival.Attendance(ctx, receptionist, visit, dto.ReceptionAttendance{Action: policy.AttendanceCheckIn, Credential: qr, Version: 1})
	if err != nil || checkedIn.Status != "checked_in" || checkedIn.Version != 2 || !checkedIn.Timing.CanCheckOut {
		t.Fatalf("QR check-in: %+v, %v", checkedIn, err)
	}
	visitorProof.Version = checkedIn.Version
	ticket, err := departureQR.Issue(ctx, visit, visitorProof)
	if err != nil || ticket.Payload != qr || !ticket.ExpiresAt.Equal(firstTicket.ExpiresAt) || ticket.ServerTime <= ticket.ExpiresAt.UnixMilli() {
		t.Fatalf("visitor could not retrieve the same QR after the scheduled end: %v", err)
	}
	depart := receptionAt(tx, departureTime)
	resolved, err := depart.Resolve(ctx, session, proof)
	if err != nil || resolved.Status != "checked_in" || resolved.Version != 2 {
		t.Fatalf("resolve checked-in visit with expired QR: %+v, %v", resolved, err)
	}
	checkedOut, err := depart.Attendance(ctx, receptionist, visit, dto.ReceptionAttendance{Action: policy.AttendanceCheckOut, Credential: qr, Version: 2})
	if err != nil || checkedOut.Status != "checked_out" || checkedOut.Version != 3 || checkedOut.Timing.CanCheckOut {
		t.Fatalf("overnight QR checkout: %+v, %v", checkedOut, err)
	}
	if _, err := depart.Resolve(ctx, session, proof); !errors.Is(err, failure.ErrNotFound) {
		t.Fatalf("completed visit still resolved: %v", err)
	}
	visitorProof.Version = checkedOut.Version
	if _, err := departureQR.Issue(ctx, visit, visitorProof); errorCode(err) != "invalid_state" {
		t.Fatalf("visitor can still retrieve QR after checkout: %v", err)
	}
	if _, err := depart.Attendance(ctx, receptionist, visit, dto.ReceptionAttendance{Action: policy.AttendanceCheckOut, Credential: qr, Version: 3}); errorCode(err) != "invalid_state" {
		t.Fatalf("repeating a completed checkout: %v", err)
	}
	var recorded struct {
		Arrival   time.Time
		Departure time.Time
		Events    int
	}
	if err := tx.Raw(`SELECT waktu_cek_in AS arrival,waktu_cek_out AS departure,
		(SELECT count(*) FROM vms.audit_log WHERE entity_type='kunjungan' AND entity_id=v.id) AS events
		FROM vms.kunjungan v WHERE id=?`, visit).Scan(&recorded).Error; err != nil ||
		!recorded.Arrival.Equal(scheduled.Add(-30*time.Minute)) || !recorded.Departure.Equal(scheduled.Add(24*time.Hour)) || recorded.Events != 2 {
		t.Fatalf("attendance timestamps or audit incorrect: %+v, %v", recorded, err)
	}
}

func TestRejectedReceptionProofsAreIndistinguishable(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	visit, code, qr := fixtureVisit(t, tx, true)
	ctx := context.Background()
	session := dto.Session{Active: true, Role: policy.RoleReceptionist, Permissions: []string{policy.PermCheckin, policy.PermCheckout}}
	u := receptionAt(tx, scheduled)
	if err := tx.Exec("UPDATE vms.qr_kunjungan SET dicabut_pada=? WHERE id_kunjungan=?", scheduled, visit).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("UPDATE vms.kunjungan SET status_penerimaan='pending' WHERE id=?", visit).Error; err != nil {
		t.Fatal(err)
	}
	for _, proof := range []string{qr, code, uuid.NewString()} {
		request := dto.ReceptionLookup{
			ReceptionResolve: dto.ReceptionResolve{Credential: proof},
			VisitID:          fmt.Sprint(visit),
		}
		if _, err := u.Resolve(ctx, session, request); !errors.Is(err, failure.ErrNotFound) {
			t.Fatalf("rejected proof returned a different error: %v", err)
		}
	}
}

func TestCodeCheckInCanGetFirstQRForCheckoutAfterScheduledEnd(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	ctx := context.Background()
	visit, code, _ := fixtureVisit(t, tx, false)
	receptionist := fixtureReceptionist(t, tx)
	checkedIn, err := receptionAt(tx, scheduled).Attendance(ctx, receptionist, visit,
		dto.ReceptionAttendance{Action: policy.AttendanceCheckIn, Credential: code, Version: 1})
	if err != nil {
		t.Fatalf("check in by code: %v", err)
	}
	departureTime := scheduled.Add(24 * time.Hour)
	admission := bookingimpl.NewAdmissionUsecase(
		clockedAdmissionRepo{bookingrepoimpl.NewBookingAccessRepository(tx), departureTime},
		[]byte("synthetic-test-qr-signing-key-000"), noop.NewTracerProvider().Tracer("admission-integration"))
	ticket, err := admission.Issue(ctx, visit, dto.PublicCancel{Code: code, Email: "visitor@example.test", Version: checkedIn.Version})
	if err != nil || ticket.Payload == "" {
		t.Fatalf("issue first QR after scheduled end: %v", err)
	}
	checkedOut, err := receptionAt(tx, departureTime.Add(time.Minute)).Attendance(ctx, receptionist, visit,
		dto.ReceptionAttendance{Action: policy.AttendanceCheckOut, Credential: ticket.Payload, Version: checkedIn.Version})
	if err != nil || checkedOut.Status != "checked_out" {
		t.Fatalf("check out with the first QR: %v", err)
	}
}

func TestConcurrentReceptionistsRecordOneCheckIn(t *testing.T) {
	db := integrationtest.Open(t)
	visit, code, _ := fixtureVisit(t, db, false)
	first, second := fixtureReceptionist(t, db), fixtureReceptionist(t, db)
	u := receptionAt(db, scheduled.Add(time.Hour))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, actor := range []entity.SessionProof{first, second} {
		go func() {
			<-start
			_, err := u.Attendance(ctx, actor, visit, dto.ReceptionAttendance{Action: policy.AttendanceCheckIn, Credential: code, Version: 1})
			results <- err
		}()
	}
	close(start)
	var successes, conflicts int
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errorCode(err) == "version_conflict":
			conflicts++
		default:
			t.Fatalf("unexpected concurrent attendance result: %v", err)
		}
	}
	var events int
	err := db.Raw(`SELECT count(*) FROM vms.audit_log WHERE entity_type='kunjungan' AND entity_id=? AND jenis_kegiatan='checked_in'`, visit).Scan(&events).Error
	if err != nil || successes != 1 || conflicts != 1 || events != 1 {
		t.Fatalf("successes=%d conflicts=%d audit=%d err=%v", successes, conflicts, events, err)
	}
}

func fixtureVisit(t *testing.T, db *gorm.DB, seedQR bool) (int32, string, string) {
	t.Helper()
	var officer int32
	if err := db.Raw("SELECT id FROM vms.pejabat ORDER BY id LIMIT 1").Scan(&officer).Error; err != nil || officer <= 0 {
		t.Fatalf("seed the synthetic test catalog first: %v", err)
	}
	codeNumber, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
	if err != nil {
		t.Fatal(err)
	}
	code := fmt.Sprintf("%08d", codeNumber.Int64())
	var scheduleID, visitorID, visitID int32
	scanID(t, db.Raw(`INSERT INTO vms.detail_kunjungan(id_pejabat,tanggal,waktu_mulai,visit_duration,keperluan) VALUES
		(?,?::date,?::time,60,'Rapat') RETURNING id`, officer, scheduled.Format(time.DateOnly), scheduled.Format("15:04")), &scheduleID)
	scanID(t, db.Raw(`INSERT INTO vms.detail_pengunjung(nama,asal_instansi,email,identity_type,nama_rombongan,jumlah_pengunjung)
		VALUES ('Visitor Example','Organization Example','visitor@example.test','KTP',ARRAY['Visitor Example'],1) RETURNING id`), &visitorID)
	scanID(t, db.Raw(`INSERT INTO vms.kunjungan(id_detail_pengunjung,id_detail_kunjungan,kode_kunjungan,tipe_kunjungan,status_penerimaan,tanggal_dibuat)
		VALUES (?,?,?,'umum','approved',clock_timestamp()) RETURNING id`, visitorID, scheduleID, code), &visitID)
	if !seedQR {
		return visitID, code, ""
	}
	qr := "test-reception-" + uuid.NewString()
	if err := db.Exec(`INSERT INTO vms.qr_kunjungan(id_kunjungan,token_hash,tanggal_dibuat,tanggal_expire)
		VALUES (?,?,?,?)`, visitID, security.Hash(qr), scheduled.Add(-time.Hour), scheduled.Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	return visitID, code, qr
}

func fixtureReceptionist(t *testing.T, db *gorm.DB) entity.SessionProof {
	t.Helper()
	var staffID, roleID int32
	scanID(t, db.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password) VALUES
		('Reception Example',?,true,false) RETURNING id`, uuid.NewString()+"@example.test"), &staffID)
	scanID(t, db.Raw(`INSERT INTO vms.role_pegawai(id_karyawan,id_role) SELECT ?,id FROM vms.role
		WHERE nama=? RETURNING id_role`, staffID, policy.RoleReceptionist), &roleID)
	id := uuid.NewString()
	if err := authrepoimpl.NewSessionRepository(db).Transaction(context.Background(), staffID, func(tx authrepository.SessionTransaction) error {
		at := tx.Now()
		return tx.Create(entity.StaffSession{ID: id, EmployeeID: staffID, ActiveRoleID: &roleID, CredentialGeneration: 1,
			RoleRevision: 1, RefreshHash: security.Hash(uuid.NewString()), CreatedAt: at, LastActivityAt: at,
			IdleExpiresAt: at.Add(time.Hour), AbsoluteExpiresAt: at.Add(time.Hour)})
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Exec("UPDATE support.staff_sessions SET revoked_at=clock_timestamp() WHERE id=?", id).Error
		_ = db.Exec("UPDATE vms.pegawai SET status_aktif=false WHERE id=?", staffID).Error
	})

	return entity.SessionProof{SessionID: id, EmployeeID: staffID, CredentialGeneration: 1, RoleRevision: 1}
}

func scanID(t *testing.T, raw *gorm.DB, id *int32) {
	t.Helper()
	if err := raw.Scan(id).Error; err != nil || *id <= 0 {
		t.Fatalf("could not create test fixture: id=%d, err=%v", *id, err)
	}
}

func errorCode(err error) string {
	var api *failure.Error
	if errors.As(err, &api) {
		return api.Code
	}
	return ""
}
