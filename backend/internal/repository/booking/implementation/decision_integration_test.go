//go:build integration

package bookingrepositoryimplementation_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authrepo "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authimpl "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	staffserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	staffusecase "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

func TestConcurrentVisitDecisions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db := integrationtest.Open(t)
	a, b := protocolSession(t, db, ctx), protocolSession(t, db, ctx)
	u := staffusecase.NewDecisionUsecase(bookingrepo.NewDecisionRepository(db), authusecase.NewSessionValidator(authserializer.NewAuthSerializer()),
		staffserializer.NewStaffBookingSerializer(), noop.NewTracerProvider().Tracer("test"))

	// Tests use a different day on each run so previous fixture visits cannot consume this slot.
	today := time.Now().In(validate.WITA)
	slot := today.AddDate(0, 0, 90+int(today.UnixNano()%5000))
	date, nextDate := slot.Format("2006-01-02"), slot.AddDate(0, 0, 1).Format("2006-01-02")
	first := pendingVisit(t, db, date)
	second := pendingVisit(t, db, date)
	var errs [2]error
	input := dto.ApproveBooking{Version: 1}
	race(
		func() { _, errs[0] = u.Approve(ctx, a, first, input) },
		func() { _, errs[1] = u.Approve(ctx, b, second, input) },
	)
	winner, loserErr := first, errs[1]
	if errs[0] != nil {
		winner, loserErr = second, errs[0]
	}
	if (errs[0] == nil) == (errs[1] == nil) || decisionErrorCode(loserErr) != "schedule_conflict" {
		t.Fatalf("overlap: %v / %v", errs[0], errs[1])
	}
	if _, err := u.Approve(ctx, b, winner, input); decisionErrorCode(err) != "version_conflict" {
		t.Fatalf("double approve: %v", err)
	}

	target := pendingVisit(t, db, nextDate)
	race(
		func() {
			_, errs[0] = u.Reschedule(ctx, a, winner, dto.RescheduleBooking{Version: 2, Date: nextDate, Time: "10:00", Reason: "Perubahan agenda"})
		},
		func() { _, errs[1] = u.Approve(ctx, b, target, input) },
	)
	if (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("reschedule and approval: %v / %v", errs[0], errs[1])
	}
	for _, err := range errs {
		if err != nil && decisionErrorCode(err) != "schedule_conflict" {
			t.Fatal(err)
		}
	}
	version := 2
	if errs[0] == nil {
		version = 3
	}
	if err := db.Exec(`INSERT INTO vms.qr_kunjungan
		(id_kunjungan,token_hash,tanggal_dibuat,tanggal_expire)
		VALUES (?, ?, clock_timestamp(),clock_timestamp()+interval '1 hour')`, winner, security.Hash(uuid.NewString())).Error; err != nil {
		t.Fatal(err)
	}
	out, err := u.Cancel(ctx, a, winner, dto.CancelBooking{Version: version, Reason: "Agenda dibatalkan"})
	if err != nil || out.Status != "cancelled" || out.Version != version+1 || out.CancellationReason == nil || *out.CancellationReason != "Agenda dibatalkan" {
		t.Fatalf("cancel: %+v, %v", out, err)
	}
	var revoked bool
	if err := db.Raw("SELECT dicabut_pada IS NOT NULL FROM vms.qr_kunjungan WHERE id_kunjungan = ?", winner).Scan(&revoked).Error; err != nil || !revoked {
		t.Fatalf("cancel must revoke the existing QR: %v", err)
	}

	restricted := pendingVisit(t, db, date)
	if err := db.Exec("UPDATE vms.kunjungan SET tipe_kunjungan = 'terbatas' WHERE id = ?", restricted).Error; err != nil {
		t.Fatal(err)
	}
	var officer int32
	if err := db.Raw("SELECT id_pejabat FROM vms.detail_kunjungan d JOIN vms.kunjungan v ON v.id_detail_kunjungan = d.id WHERE v.id = ?", restricted).Scan(&officer).Error; err != nil {
		t.Fatal(err)
	}
	edit := dto.UpdateRestrictedBooking{Version: 1, OfficerID: fmt.Sprint(officer), Date: dto.Optional[string]{Set: true, Value: nextDate}}
	if _, err := u.UpdateRestricted(ctx, a, restricted, edit); decisionErrorCode(err) != "validation_failed" {
		t.Fatalf("restricted date change: %v", err)
	}
	edit.Date, edit.Name = dto.Optional[string]{Set: true, Value: date}, dto.Optional[string]{Set: true}
	edit.ServiceID = dto.Optional[string]{Set: true, Value: "00123"}
	saved, err := u.UpdateRestricted(ctx, a, restricted, edit)
	if err != nil || saved.Name != "" || saved.Date != date || saved.ServiceID == nil || *saved.ServiceID != "00123" {
		t.Fatalf("restricted identity update: %+v, %v", saved, err)
	}
}

func race(a, b func()) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, fn := range []func(){a, b} {
		wg.Go(func() { <-start; fn() })
	}
	close(start)
	wg.Wait()
}

func decisionErrorCode(err error) string {
	var business *failure.Error
	if errors.As(err, &business) {
		return business.Code
	}
	return ""
}

func protocolSession(t *testing.T, db *gorm.DB, ctx context.Context) entity.SessionProof {
	t.Helper()
	var employee, role int32
	if err := db.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password)
		VALUES ('Decision test',?,TRUE,FALSE) RETURNING id`, "decision-"+uuid.NewString()+"@example.test").Scan(&employee).Error; err != nil || employee == 0 {
		t.Fatalf("insert staff: %v", err)
	}
	if err := db.Raw(`INSERT INTO vms.role_pegawai(id_karyawan,id_role)
		SELECT ?, id FROM vms.role WHERE nama = ? RETURNING id_role`, employee, policy.RoleProtocol).Scan(&role).Error; err != nil || role == 0 {
		t.Fatalf("assign staff role: %v", err)
	}
	sessionID := uuid.NewString()
	if err := authimpl.NewSessionRepository(db).Transaction(ctx, employee, func(tx authrepo.SessionTransaction) error {
		now := tx.Now()
		return tx.Create(entity.StaffSession{ID: sessionID, EmployeeID: employee, ActiveRoleID: &role, CredentialGeneration: 1, RoleRevision: 1,
			RefreshHash: security.Hash(sessionID), CreatedAt: now, LastActivityAt: now, IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(2 * time.Hour)})
	}); err != nil {
		t.Fatal(err)
	}
	return entity.SessionProof{SessionID: sessionID, EmployeeID: employee, CredentialGeneration: 1, RoleRevision: 1}
}

func pendingVisit(t *testing.T, db *gorm.DB, date string) int32 {
	t.Helper()
	var visit int32
	for i := 0; i < 20; i++ {
		code := fmt.Sprintf("%08d", (time.Now().UnixNano()+int64(i))%100_000_000)
		err := db.Transaction(func(tx *gorm.DB) error {
			return tx.Raw(`WITH detail AS (
				INSERT INTO vms.detail_kunjungan(id_pejabat,tanggal,waktu_mulai,visit_duration,keperluan)
				SELECT p.id,?::date,'10:00',60,'Fixture agenda' FROM vms.pejabat p
				JOIN vms.jabatan j ON j.id=p.id_jabatan WHERE p.status_aktif AND j.pejabat_vip ORDER BY p.id LIMIT 1 RETURNING id
			), visitor AS (
				INSERT INTO vms.detail_pengunjung(nama) VALUES ('Pengunjung Contoh') RETURNING id
			)
			INSERT INTO vms.kunjungan(id_detail_kunjungan,id_detail_pengunjung,kode_kunjungan,tipe_kunjungan,status_penerimaan,tanggal_dibuat)
			SELECT detail.id,visitor.id,?,'vip','pending',clock_timestamp() FROM detail,visitor RETURNING id`, date, code).Scan(&visit).Error
		})
		if err == nil && visit > 0 {
			return visit
		}
		var coded interface{ SQLState() string }
		if !errors.As(err, &coded) || coded.SQLState() != "23505" {
			t.Fatalf("insert booking decision fixture: %v", err)
		}
	}
	t.Fatal("could not allocate fixture visit")
	return 0
}
