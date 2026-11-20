//go:build integration

package bookingrepositoryimplementation_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authrepo "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	staffserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	auth "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	staff "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

func actor(t *testing.T, db *gorm.DB) (dto.Session, entity.SessionProof) {
	t.Helper()
	var id int32
	email := "booking-test-" + uuid.NewString() + "@example.test"
	if err := db.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password)
		VALUES ('Staff Booking Fixture',?,TRUE,FALSE) RETURNING id`, email).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO vms.role_pegawai(id_karyawan,id_role) SELECT ?,id FROM vms.role WHERE nama=?", id, policy.RoleProtocol).Error; err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.NewString()
	if err := db.Exec(`INSERT INTO support.staff_sessions(id,employee_id,active_role_id,
		credential_generation,role_revision,refresh_hash,created_at,last_activity_at,idle_expires_at,absolute_expires_at)
		SELECT ?,?,id,1,1,?,transaction_timestamp()-INTERVAL '1 minute',transaction_timestamp()-INTERVAL '1 minute',
			transaction_timestamp()+INTERVAL '1 hour',transaction_timestamp()+INTERVAL '2 hours'
		FROM vms.role WHERE nama=?`, sessionID, id, security.Hash(sessionID), policy.RoleProtocol).Error; err != nil {
		t.Fatal(err)
	}
	proof := entity.SessionProof{SessionID: sessionID, EmployeeID: id, CredentialGeneration: 1, RoleRevision: 1}
	session, err := auth.NewAuthUsecase(authrepo.NewAuthRepository(db), authserializer.NewAuthSerializer()).GetSession(context.Background(), proof)
	if err != nil {
		t.Fatal(err)
	}
	return session, proof
}

func deactivateFixture(t *testing.T, db *gorm.DB, id int32) {
	t.Helper()
	t.Cleanup(func() {
		for _, q := range []string{
			"UPDATE support.staff_sessions SET revoked_at=clock_timestamp() WHERE employee_id=? AND revoked_at IS NULL",
			"DELETE FROM vms.role_pegawai WHERE id_karyawan=?",
			"UPDATE vms.pegawai SET status_aktif=FALSE WHERE id=?",
		} {
			if err := db.Exec(q, id).Error; err != nil {
				t.Errorf("deactivate staff booking fixture: %v", err)
			}
		}
	})
}

func vipOfficer(t *testing.T, db *gorm.DB) int32 {
	t.Helper()
	var id int32
	if err := db.Raw(`SELECT p.id FROM vms.pejabat p JOIN vms.jabatan j ON j.id=p.id_jabatan
		WHERE p.status_aktif AND j.pejabat_vip ORDER BY p.id LIMIT 1`).Scan(&id).Error; err != nil || id <= 0 {
		t.Fatalf("missing active VIP officer fixture: %v", err)
	}
	return id
}

func TestConcurrentStaffCreatesAcceptOneAndRejectOneOverlappingVisit(t *testing.T) {
	db := integrationtest.Open(t)
	serializer := authserializer.NewAuthSerializer()
	creator := staff.NewStaffBookingCreator(bookingrepo.NewStaffBookingWriter(db), auth.NewSessionValidator(serializer),
		staffserializer.NewStaffBookingSerializer(), []byte(strings.Repeat("s", 32)), noop.NewTracerProvider().Tracer("test"))
	first, firstProof := actor(t, db)
	second, secondProof := actor(t, db)
	deactivateFixture(t, db, firstProof.EmployeeID)
	deactivateFixture(t, db, secondProof.EmployeeID)
	officer := vipOfficer(t, db)
	date := time.Now().In(validate.WITA).AddDate(0, 0, 20).Format("2006-01-02")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	blocker := db.WithContext(ctx).Begin()
	if blocker.Error != nil {
		t.Fatal(blocker.Error)
	}
	defer blocker.Rollback()
	if err := bookingrepo.LockScheduleResources(blocker, []int32{officer}, nil); err != nil {
		t.Fatal(err)
	}

	type result struct {
		booking dto.StaffBooking
		created bool
		err     error
	}
	results := make(chan result, 2)
	for _, pair := range []struct {
		session dto.Session
		proof   entity.SessionProof
	}{{first, firstProof}, {second, secondProof}} {
		go func() {
			b, created, err := creator.CreateStaffBooking(ctx, pair.session, pair.proof, "127.0.0.1", dto.StaffBookingCreate{
				OfficerID: strconv.Itoa(int(officer)), VisitKind: "restricted", Date: date, Time: "10:30",
				ServiceID: " 0012345 ", IdempotencyKey: uuid.NewString(),
			})
			results <- result{b, created, err}
		}()
	}
	for {
		var n int
		if err := db.WithContext(ctx).Raw(`SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock'
			AND query LIKE 'SELECT pg_advisory_xact_lock(%::integer%'`).Scan(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n >= 2 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("both create transactions did not reach the resource lock")
		case <-time.After(20 * time.Millisecond):
		}
	}
	if err := blocker.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	var saved dto.StaffBooking
	var conflicts int
	for range 2 {
		select {
		case result := <-results:
			var violation *failure.Error
			if errors.As(result.err, &violation) && violation.Status == 409 && violation.Code == "schedule_conflict" {
				conflicts++
			} else if result.err != nil || !result.created {
				t.Fatalf("create must succeed or return the schedule conflict: %v", result.err)
			} else {
				saved = result.booking
				t.Cleanup(func() {
					if err := db.Exec("UPDATE vms.kunjungan SET status_penerimaan='cancelled' WHERE id=?", saved.ID).Error; err != nil {
						t.Errorf("cancel synthetic test booking: %v", err)
					}
				})
			}
		case <-ctx.Done():
			t.Fatal("both create transactions did not return")
		}
	}
	if conflicts != 1 || saved.Status != "approved" || saved.Version != 1 || saved.Duration != 60 || saved.CreatedBy == nil ||
		saved.CreatedBy.Role != policy.RoleProtocol || saved.ServiceID == nil || *saved.ServiceID != "0012345" {
		t.Fatalf("got %d conflicts and %+v", conflicts, saved)
	}
	var restrictedFieldsAbsent bool
	if err := db.Raw(`SELECT p.email IS NULL AND p.nomor_telpon IS NULL AND p.nama_rombongan IS NULL
		AND p.identity_type IS NULL AND p.id_formdocuments IS NULL FROM vms.kunjungan v
		JOIN vms.detail_pengunjung p ON p.id=v.id_detail_pengunjung WHERE v.id=?`, saved.ID).Scan(&restrictedFieldsAbsent).Error; err != nil || !restrictedFieldsAbsent {
		t.Fatalf("restricted visitor row includes ordinary identity fields: %v", err)
	}
}

func TestStandardStaffBookingLinksOnlyItsPhotoAndReplaysTheExistingVisit(t *testing.T) {
	db := integrationtest.RolledBack(t, integrationtest.Open(t))
	ctx := context.Background()
	session, proof := actor(t, db)
	officer := vipOfficer(t, db)
	creator := staff.NewStaffBookingCreator(bookingrepo.NewStaffBookingWriter(db), auth.NewSessionValidator(authserializer.NewAuthSerializer()),
		staffserializer.NewStaffBookingSerializer(), []byte(strings.Repeat("s", 32)), noop.NewTracerProvider().Tracer("test"))
	var photo int32
	if err := db.Raw(`INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose)
		VALUES ('synthetic.png',?,'image/png',10,'visitorPhoto') RETURNING id`, uuid.NewString()).Scan(&photo).Error; err != nil {
		t.Fatal(err)
	}
	token := "staff-test-token-" + uuid.NewString()
	owner := security.Hash("staff:" + strconv.Itoa(int(proof.EmployeeID)))
	if err := db.Exec(`INSERT INTO support.form_document_uploads(id_form_documents,audience,owner_digest,reference,
		content_sha256,token_hash,key_version,created_at,expires_at,uploaded_by)
		VALUES (?,'staff',?,?,?, ?,1,transaction_timestamp(),transaction_timestamp()+INTERVAL '1 hour',?)`,
		photo, owner, uuid.NewString(), security.Hash("fixture"), security.Hash(token), proof.EmployeeID).Error; err != nil {
		t.Fatal(err)
	}
	input := dto.StaffBookingCreate{
		OfficerID: strconv.Itoa(int(officer)), Date: time.Now().In(validate.WITA).AddDate(0, 0, 21).Format("2006-01-02"), Time: "10:30",
		Name: "Visitor Example", Organization: "Organization Example", Phone: "0812345678", Email: "visitor@example.test",
		Purpose: "Meeting Example", IdentityType: "KTP", Visitors: []string{"Visitor Example"}, VisitorPhotoToken: token,
		VisitorPhotoName: "synthetic.png", IdempotencyKey: uuid.NewString(), CoordinationNote: "Coordination Example",
	}
	saved, created, err := creator.CreateStaffBooking(ctx, session, proof, "127.0.0.1", input)
	var validation *failure.Error
	if errors.As(err, &validation) {
		t.Fatalf("create standard booking: %v %v", validation, validation.Fields)
	}
	if err != nil || !created || saved.Status != "approved" || saved.VisitKind != "standard" ||
		saved.VisitorPhotoAvailable == nil || !*saved.VisitorPhotoAvailable || saved.CreatedBy == nil {
		t.Fatalf("create standard booking: %+v %v %v", saved, created, err)
	}
	replay, created, err := creator.CreateStaffBooking(ctx, session, proof, "127.0.0.1", input)
	if err != nil || created || replay.ID != saved.ID {
		t.Fatalf("replay: %+v %v %v", replay, created, err)
	}
	input.Purpose = "Changed Example"
	_, _, err = creator.CreateStaffBooking(ctx, session, proof, "127.0.0.1", input)
	var violation *failure.Error
	if !errors.As(err, &violation) || violation.Code != "idempotency_conflict" {
		t.Fatalf("replay with changed input: %v", err)
	}
}
