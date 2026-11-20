//go:build integration

package reportrepositoryimplementation_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/policy"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	reportrepo "github.com/azhkadaes/vms/backend/internal/repository/report/implementation"
	reportserializer "github.com/azhkadaes/vms/backend/internal/serializer/report/implementation"
	reportusecase "github.com/azhkadaes/vms/backend/internal/usecase/report/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

func reportID(t *testing.T, tx *gorm.DB) int32 {
	t.Helper()
	var id int32
	if err := tx.Scan(&id).Error; err != nil || id <= 0 {
		t.Fatalf("create report fixture: %d (%v)", id, err)
	}
	return id
}

func reportVisit(t *testing.T, tx *gorm.DB, officer, room int32, kind, status, code string) int32 {
	t.Helper()
	schedule := reportID(t, tx.Raw(`INSERT INTO vms.detail_kunjungan(id_pejabat,id_ruangan,tanggal,waktu_mulai,visit_duration,keperluan,catatan_internal,no_nd)
		VALUES (?,?,'2000-01-02','09:00',60,'Test purpose','Private note','Private service ID') RETURNING id`, officer, room))
	var party int32
	if kind == "terbatas" {
		party = reportID(t, tx.Raw(`INSERT INTO vms.detail_pengunjung DEFAULT VALUES RETURNING id`))
	} else {
		party = reportID(t, tx.Raw(`INSERT INTO vms.detail_pengunjung(nama,asal_instansi,email,nama_rombongan,jumlah_pengunjung)
			VALUES ('Visitor Example','Example','private@example.test',ARRAY['Visitor Example'],1) RETURNING id`))
	}
	return reportID(t, tx.Raw(`INSERT INTO vms.kunjungan(id_detail_pengunjung,id_detail_kunjungan,kode_kunjungan,tipe_kunjungan,
		status_penerimaan,tanggal_dibuat,waktu_cek_in,waktu_cek_out) VALUES (?,?,?,?::vms.tipe_kunjungan,?::vms.status_penerimaan,now(),
		CASE WHEN ? = 'checked_out' THEN now() - interval '2 hours' END,CASE WHEN ? = 'checked_out' THEN now() END) RETURNING id`,
		party, schedule, code, kind, status, status, status))
}

func TestReportsReadAConsistentScopedProjection(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	var officers, rooms []int32
	if err := tx.Raw(`SELECT id FROM vms.pejabat ORDER BY status_aktif DESC,id LIMIT 2`).Scan(&officers).Error; err != nil || len(officers) != 2 {
		t.Fatalf("load the synthetic catalog fixture: %v", err)
	}
	if err := tx.Raw(`SELECT id FROM vms.ruangan ORDER BY status_aktif DESC,id LIMIT 2`).Scan(&rooms).Error; err != nil || len(rooms) != 2 {
		t.Fatalf("load the synthetic room fixture: %v", err)
	}
	missed := reportVisit(t, tx, officers[0], rooms[0], "umum", "approved", "51150001")
	reportVisit(t, tx, officers[0], rooms[1], "vip", "checked_out", "51150002")
	restricted := reportVisit(t, tx, officers[1], rooms[1], "terbatas", "pending", "51150003")
	reportVisit(t, tx, officers[1], rooms[1], "umum", "pending", "51150004")
	tracer := noop.NewTracerProvider().Tracer("reports-integration")
	u := reportusecase.NewReportUsecase(reportrepo.NewReportReader(tx), reportserializer.NewReportSerializer(), tracer)
	ctx := context.Background()
	officer := strconv.Itoa(int(officers[0]))
	session := func(role string) dto.Session {
		return dto.Session{Active: true, Role: role, Permissions: []string{policy.PermReports, policy.PermReportExport}, AssignedOfficerIDs: []string{officer}}
	}
	filter := dto.SummaryFilter{From: "2000-01-02", To: "2000-01-02", Status: dto.ReportFilterValues{"all"}}
	for _, tc := range []struct {
		role, scope string
		count       int
	}{
		{policy.RoleSuperAdmin, "organization", 3}, {policy.RoleProtocol, "authorized", 3},
		{policy.RolePA, "assigned", 1}, {policy.RoleReceptionist, "attendance", 2},
	} {
		got, err := u.Summary(ctx, session(tc.role), filter)
		if err != nil || got.Total != tc.count || got.Scope != tc.scope || got.Counts["missed"] != 1 ||
			got.Counts["approved"] != 0 || got.GeneratedAt.Location() != time.UTC {
			t.Fatalf("%s summary: %+v (%v)", tc.role, got, err)
		}
	}
	admin := session(policy.RoleSuperAdmin)
	first, err := u.Summary(ctx, admin, filter)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`UPDATE vms.kunjungan SET version=version+1 WHERE id=?`, missed).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := u.ExportSummary(ctx, admin, filter, first.Revision); !hasReportError(err, "revision_conflict") {
		t.Fatalf("a counted visit version change must invalidate the summary revision: %v", err)
	}
	log := dto.VisitLogFilter{From: "2000-01-02", To: "2000-01-02", Status: dto.ReportFilterValues{"all"},
		Kind: dto.ReportFilterValues{"all"}, Room: dto.ReportFilterValues{strconv.Itoa(int(rooms[1]))}, Sort: "visitors", Dir: "asc"}
	page, err := u.VisitLog(ctx, admin, log, 30, 10)
	if err != nil || page.Page != 1 || page.Total != 3 || page.Summary.Visitors != 2 || len(page.Items) != 3 ||
		page.Items[0].Reference != "TRB-"+strconv.Itoa(int(restricted)) || page.Items[0].Name != "Kunjungan terbatas" ||
		page.Items[0].Purpose != "" || page.Items[0].Visitors != nil || len(page.Officers) < 2 ||
		!slices.ContainsFunc(page.Rooms, func(name string) bool { return strings.Contains(name, "Inactive") }) {
		t.Fatalf("unexpected log projection, room options or paging: %+v (%v)", page, err)
	}
	for _, row := range page.Items {
		if row.Status == "checked_out" && (row.CheckedInAt == nil || row.CheckedOutAt == nil || row.CheckedOutAt.Location() != time.UTC) {
			t.Fatalf("checked-out timestamp projection: %+v", row)
		}
	}
	snapshot, err := u.ExportVisitLog(ctx, admin, log, page.Revision)
	if err != nil || len(snapshot.Items) != snapshot.Summary.Total || snapshot.Revision != page.Revision {
		t.Fatalf("exported snapshot differs from the reviewed log: %+v (%v)", snapshot, err)
	}
	log.Q = "TEST PURPOSE"
	search, err := u.VisitLog(ctx, admin, log, 1, 25)
	if err != nil || search.Total != 2 {
		t.Fatalf("search used a restricted row's concealed purpose: %+v (%v)", search, err)
	}
}

func hasReportError(err error, code string) bool {
	api, ok := errors.AsType[*failure.Error](err)
	return ok && api.Code == code
}
