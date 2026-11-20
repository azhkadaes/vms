//go:build integration

package bookingrepositoryimplementation_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
	bookingimplementation "github.com/azhkadaes/vms/backend/internal/repository/booking/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking/implementation"
	bookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/booking/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

func accessFixture(t *testing.T, tx *gorm.DB, code, kind, status string, start time.Time) int32 {
	t.Helper()
	var officer, schedule, visitor, visit int32
	if err := tx.Raw(`SELECT id FROM vms.pejabat WHERE status_aktif ORDER BY id LIMIT 1`).Scan(&officer).Error; err != nil || officer == 0 {
		t.Fatalf("load synthetic active officer fixture: %v (%d)", err, officer)
	}
	if err := tx.Raw(`INSERT INTO vms.detail_kunjungan
		(id_pejabat,tanggal,waktu_mulai,visit_duration,catatan,catatan_internal)
		VALUES (?,?::date,?::time,60,'Visitor instructions','Internal note') RETURNING id`,
		officer, start.Format("2006-01-02"), start.Format("15:04")).Scan(&schedule).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Raw(`INSERT INTO vms.detail_pengunjung
		(nama,asal_instansi,email,nomor_telpon,identity_type,nama_rombongan,jumlah_pengunjung)
		VALUES ('Visitor Example','Example','visitor@example.test','08123456789','KTP',ARRAY['Visitor Example'],1)
		RETURNING id`).Scan(&visitor).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Raw(`INSERT INTO vms.kunjungan
		(id_detail_pengunjung,id_detail_kunjungan,kode_kunjungan,tipe_kunjungan,status_penerimaan,
		 tanggal_dibuat,waktu_cek_in,waktu_cek_out,alasan_penolakan,catatan_internal_penolakan)
		VALUES (?,?,?,?::vms.tipe_kunjungan,?::vms.status_penerimaan,now(),
		 CASE WHEN ? = 'checked_out' THEN now() - interval '1 hour' END,
		 CASE WHEN ? = 'checked_out' THEN now() END,
		 CASE WHEN ? = 'rejected' THEN 'Visitor reason' END,'Internal rejection') RETURNING id`,
		visitor, schedule, code, kind, status, status, status, status).Scan(&visit).Error; err != nil {
		t.Fatal(err)
	}
	if visit <= 0 {
		t.Fatal("visit insert returned no ID")
	}
	return visit
}

func TestVisitorLooksUpAndCancelsOnlyTheirVisibleVisit(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	usecase := bookingusecase.NewBookingAccessUsecase(bookingimplementation.NewBookingAccessRepository(tx),
		bookingserializer.NewBookingSerializer(), noop.NewTracerProvider().Tracer("booking-access"))
	ctx := context.Background()
	future := time.Now().In(validate.WITA).AddDate(0, 0, 1)
	past := future.AddDate(0, 0, -2)
	id := accessFixture(t, tx, "09170701", "umum", "approved", future)
	if err := tx.Exec(`INSERT INTO vms.audit_log(actor_type,entity_type,entity_id,jenis_kegiatan,tanggal,deskripsi,new_values) VALUES
		('system','kunjungan',?,'rescheduled',now() - interval '2 minutes','Raw values','{"reason":"New time","secret":"Hidden"}'),
		('visitor','kunjungan',?,'created',now() - interval '3 minutes','Created','{"reason":"Ignore"}'),
		('system','kunjungan',?,'private_change',now() - interval '1 minute','Private','{"reason":"Hidden"}')`, id, id, id).Error; err != nil {
		t.Fatal(err)
	}
	lookup := dto.PublicLookup{Code: "09170701", Email: "Visitor@Example.test"}
	visit, err := usecase.Lookup(ctx, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if !visit.Rescheduled || !visit.Timing.CanCancel || len(visit.History) != 2 || visit.History[0].Action != "created" ||
		visit.History[0].PublicMessage != nil || visit.History[1].PublicMessage == nil || *visit.History[1].PublicMessage != "New time" ||
		visit.VisitorInstructions == nil || *visit.VisitorInstructions != "Visitor instructions" {
		t.Fatalf("incorrect public projection: %+v", visit)
	}
	cancel := dto.PublicCancel{Code: lookup.Code, Email: lookup.Email, Version: visit.Version}
	if _, err := usecase.Cancel(ctx, id+1, cancel, "127.0.0.1"); !errors.Is(err, bookingrepo.ErrNotFound) {
		t.Fatalf("cancel with mismatched ID: %v", err)
	}
	if _, err := usecase.Lookup(ctx, dto.PublicLookup{Code: lookup.Code, Email: "other@example.test"}); !errors.Is(err, bookingrepo.ErrNotFound) {
		t.Fatalf("lookup with mismatched email: %v", err)
	}
	updated, err := usecase.Cancel(ctx, id, cancel, "127.0.0.1")
	if err != nil || updated.Status != "cancelled" || updated.Version != visit.Version+1 || updated.Timing.CanCancel ||
		len(updated.History) != 3 || updated.History[2].Action != "cancelled" {
		t.Fatalf("cancel result %+v, err %v", updated, err)
	}
	assertCode := func(err error, expected string) {
		t.Helper()
		if api, ok := errors.AsType[*failure.Error](err); !ok || api.Code != expected {
			t.Fatalf("expected %s, got %v", expected, err)
		}
	}
	_, err = usecase.Cancel(ctx, id, cancel, "127.0.0.1")
	assertCode(err, "version_conflict")
	cancel.Version = updated.Version
	_, err = usecase.Cancel(ctx, id, cancel, "127.0.0.1")
	assertCode(err, "invalid_state")
	for _, fixture := range []struct{ code, kind, status string }{
		{"09170702", "umum", "approved"}, {"09170703", "terbatas", "approved"},
		{"09170704", "umum", "rejected"}, {"09170705", "umum", "checked_out"},
	} {
		fixtureID := accessFixture(t, tx, fixture.code, fixture.kind, fixture.status, past)
		proof := dto.PublicLookup{Code: fixture.code, Email: lookup.Email}
		found, err := usecase.Lookup(ctx, proof)
		if fixture.kind == "terbatas" {
			if !errors.Is(err, bookingrepo.ErrNotFound) {
				t.Fatalf("restricted visit lookup: %v", err)
			}
			continue
		}
		if err != nil || found.Status != fixture.status {
			t.Fatalf("lookup %s: %+v %v", fixture.status, found, err)
		}
		_, err = usecase.Cancel(ctx, fixtureID, dto.PublicCancel{Code: proof.Code, Email: proof.Email, Version: found.Version}, "127.0.0.1")
		assertCode(err, "invalid_state")
	}
}
