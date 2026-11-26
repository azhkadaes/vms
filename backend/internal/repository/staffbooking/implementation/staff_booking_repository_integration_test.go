//go:build integration

package staffbookingrepositoryimplementation_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	staffbookingrepository "github.com/azhkadaes/vms/backend/internal/repository/staffbooking"
	implementation "github.com/azhkadaes/vms/backend/internal/repository/staffbooking/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func TestStaffReaderFiltersAndLoadsCreatorFromAudit(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	var ids []int32
	if err := tx.Raw("SELECT id FROM vms.pejabat ORDER BY id LIMIT 2").Scan(&ids).Error; err != nil || len(ids) != 2 {
		t.Fatalf("seed the synthetic staff catalog in the disposable database: %v", err)
	}
	var staff int32
	scanID(t, tx.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password)
		VALUES ('Renamed staff',?,true,false) RETURNING id`, uuid.NewString()+"@example.test"), &staff)
	if err := tx.Exec(`UPDATE vms.pegawai SET nomor_hp=ARRAY['0811','0812'] WHERE id=?`, staff).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`INSERT INTO vms.pegawai_pejabat(id_pegawai,id_pejabat,status_aktif) VALUES (?,?,true)`, staff, ids[1]).Error; err != nil {
		t.Fatal(err)
	}
	standard := addVisit(t, tx, ids[0], "umum", "pending", "91000001")
	vip := addVisit(t, tx, ids[0], "vip", "approved", "91000002")
	restricted := addVisit(t, tx, ids[0], "terbatas", "approved", "91000003")
	other := addVisit(t, tx, ids[1], "umum", "approved", "91000004")
	fixtureIDs := []int32{standard, vip, restricted, other}
	fixtureRows := func(visits []entity.StaffVisit) []entity.StaffVisit {
		return slices.DeleteFunc(visits, func(v entity.StaffVisit) bool { return !slices.Contains(fixtureIDs, v.ID) })
	}
	result := tx.Exec(`INSERT INTO vms.audit_log(actor_type,id_pegawai,entity_type,entity_id,jenis_kegiatan,tanggal,deskripsi,new_values)
		VALUES ('staff',?,'kunjungan',?,'created',clock_timestamp(),'Dibuat',json_build_object('actorName','Original staff','actorRole','PA'))`, staff, standard)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	reader := implementation.NewStaffBookingReader(tx)
	ctx := context.Background()
	pa := dto.Session{Active: true, Role: policy.RolePA, Permissions: []string{policy.PermBookings}, AssignedOfficerIDs: []string{strconv.Itoa(int(ids[0]))}}
	items, at, err := reader.List(ctx, pa)
	items = fixtureRows(items)
	if err != nil || at.IsZero() || len(items) != 2 || items[0].ID != standard || items[1].ID != restricted {
		t.Fatalf("unexpected scoped rows: %+v, %v, %v", items, at, err)
	}
	item, _, err := reader.Find(ctx, pa, standard)
	if err != nil || item.Version != items[0].Version || item.CreatorName != "Original staff" || item.CreatorRole != policy.RolePA || len(item.History) != 1 {
		t.Fatalf("unexpected staff creator/detail: %+v, %v", item, err)
	}
	for _, id := range []int32{vip, other} {
		if _, _, err := reader.Find(ctx, pa, id); !errors.Is(err, staffbookingrepository.ErrNotFound) {
			t.Fatalf("expected non-disclosing not-found for %d, got %v", id, err)
		}
	}
	reception := dto.Session{Active: true, Role: policy.RoleReceptionist, Permissions: []string{policy.PermBookings}}
	items, _, err = reader.List(ctx, reception)
	items = fixtureRows(items)
	if err != nil || len(items) != 3 || items[0].ID != vip || items[1].ID != restricted || items[2].ID != other {
		t.Fatalf("unexpected reception scope: %+v, %v", items, err)
	}
	if !slices.ContainsFunc(items[2].OfficerPAs, func(pa entity.OfficerPA) bool {
		return pa.Name == "Renamed staff" && slices.Equal(pa.Phones, []string{"0811", "0812"})
	}) {
		t.Fatalf("officer PA missing: %+v", items[2].OfficerPAs)
	}
}

func addVisit(t *testing.T, tx *gorm.DB, officer int32, kind, status, code string) int32 {
	t.Helper()
	var d, p, v int32
	scanID(t, tx.Raw(`INSERT INTO vms.detail_kunjungan(id_pejabat,tanggal,waktu_mulai,visit_duration,no_nd)
		VALUES (?, '2030-01-01', '09:00', 60, 'ND-0123') RETURNING id`, officer), &d)
	scanID(t, tx.Raw(`INSERT INTO vms.detail_pengunjung(nama,asal_instansi,nama_rombongan,jumlah_pengunjung)
		VALUES ('Visitor', 'Organization', ARRAY['Visitor'], 1) RETURNING id`), &p)
	scanID(t, tx.Raw(`INSERT INTO vms.kunjungan(id_detail_pengunjung,id_detail_kunjungan,kode_kunjungan,tipe_kunjungan,status_penerimaan,tanggal_dibuat)
		VALUES (?,?,?,?::vms.tipe_kunjungan,?::vms.status_penerimaan,clock_timestamp()) RETURNING id`, p, d, code, kind, status), &v)
	return v
}

func scanID(t *testing.T, raw *gorm.DB, id *int32) {
	t.Helper()
	if err := raw.Scan(id).Error; err != nil || *id <= 0 {
		t.Fatal(fmt.Errorf("could not create test fixture: id=%d, err=%v", *id, err))
	}
}
