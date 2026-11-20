//go:build integration

package catalogrepositoryimplementation_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	implementation "github.com/azhkadaes/vms/backend/internal/repository/catalog/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
)

func TestPublicCatalogReadsOnlyActiveDestinationsFromTheRebuiltSchema(t *testing.T) {
	db := integrationtest.Open(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	catalog, err := implementation.NewCatalogRepository(db).GetPublicCatalogs(ctx)
	if err != nil {
		t.Fatalf("catalog query failed against the rebuilt schema: %v", err)
	}
	for _, officer := range catalog.Officers {
		if !officer.Active || officer.ID <= 0 || officer.Name == "" || officer.JobTitle == "" {
			t.Fatalf("catalog returned an inactive or incomplete official: %+v", officer)
		}
	}
	for _, room := range catalog.Rooms {
		if !room.Active || room.ID <= 0 || room.Name == "" {
			t.Fatalf("catalog returned an inactive or incomplete room: %+v", room)
		}
	}
}

// employee creates an employee whose assignment to officer is active or historical.
func employee(t *testing.T, tx *gorm.DB, officer int32, active bool) {
	t.Helper()
	var id int32
	if err := tx.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password) VALUES ('Catalog Staff',?,TRUE,FALSE) RETURNING id`,
		"catalog-"+uuid.NewString()+"@example.test").Scan(&id).Error; err != nil || id <= 0 {
		t.Fatalf("cannot create employee: %v", err)
	}
	if err := tx.Exec(`INSERT INTO vms.pegawai_pejabat(id_pegawai,id_pejabat,status_aktif) VALUES (?,?,?)`, id, officer, active).Error; err != nil {
		t.Fatalf("cannot assign officer: %v", err)
	}
}

// staffCatalog reads op 08's catalog and checks what holds in every scope: active rooms only,
// complete officers, each listed once. It returns the officers by ID.
func staffCatalog(t *testing.T, tx *gorm.DB, assignedInactive bool) map[int32]entity.CatalogOfficer {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	catalog, err := implementation.NewCatalogRepository(tx).GetStaffCatalogs(ctx, assignedInactive)
	if err != nil {
		t.Fatalf("staff catalog query failed against the rebuilt schema: %v", err)
	}
	if catalog.Officers == nil || catalog.Rooms == nil {
		t.Fatal("staff catalog returned a nil collection")
	}
	for _, room := range catalog.Rooms {
		if !room.Active || room.ID <= 0 || room.Name == "" {
			t.Fatalf("staff catalog returned an inactive or incomplete room: %+v", room)
		}
	}
	officers := map[int32]entity.CatalogOfficer{}
	for _, officer := range catalog.Officers {
		if officer.ID <= 0 || officer.Name == "" || officer.JobTitle == "" {
			t.Fatalf("staff catalog returned an incomplete official: %+v", officer)
		}
		if _, seen := officers[officer.ID]; seen {
			t.Fatalf("official %d listed twice", officer.ID)
		}
		if !officer.Active && !assignedInactive {
			t.Fatalf("an inactive official reached a catalog without the people scope: %+v", officer)
		}
		officers[officer.ID] = officer
	}
	return officers
}

func TestStaffCatalogAddsAnInactiveOfficialOnlyWhileAnEmployeeIsActivelyAssigned(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	var inactive int32
	if err := tx.Raw(`SELECT p.id FROM vms.pejabat p WHERE NOT p.status_aktif
		AND NOT EXISTS (SELECT 1 FROM vms.pegawai_pejabat a WHERE a.id_pejabat = p.id AND a.status_aktif) ORDER BY p.id LIMIT 1`).Scan(&inactive).Error; err != nil {
		t.Fatal(err)
	}
	if inactive == 0 {
		t.Skip("needs an unassigned inactive official; load ops/stage1/seed-test-catalog.sql")
	}
	for _, scope := range []bool{false, true} {
		if _, listed := staffCatalog(t, tx, scope)[inactive]; listed {
			t.Fatalf("unassigned inactive official listed (people scope %v)", scope)
		}
	}
	employee(t, tx, inactive, false)
	if _, listed := staffCatalog(t, tx, true)[inactive]; listed {
		t.Fatal("a historical assignment kept an inactive official in the catalog")
	}
	// Two employees on the same official must not list it twice.
	employee(t, tx, inactive, true)
	employee(t, tx, inactive, true)
	officer, listed := staffCatalog(t, tx, true)[inactive]
	if !listed || officer.Active {
		t.Fatalf("assigned inactive official must be listed as inactive for the people scope: %+v, listed %v", officer, listed)
	}
	if _, listed := staffCatalog(t, tx, false)[inactive]; listed {
		t.Fatal("assigned inactive official listed without the people scope")
	}
}
