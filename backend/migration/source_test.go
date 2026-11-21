package vmsmigration_test

import (
	"regexp"
	"strings"
	"testing"

	vmsmigration "github.com/azhkadaes/vms/backend/migration"
)

// migrationOrder is the physical schema's order: the schemas prerequisite, vms tables 1–16
// (17 integration_keys is on hold), support tables 23–28, then ticket 05's 18 and 29.
var migrationOrder = []string{
	"202611180001_create_schemas.sql",
	"202611180002_create_vms_kedeputian.sql",
	"202611180003_create_vms_direktorat.sql",
	"202611180004_create_vms_jabatan.sql",
	"202611180005_create_vms_pejabat.sql",
	"202611180006_create_vms_ruangan.sql",
	"202611180007_create_vms_pegawai.sql",
	"202611180008_create_vms_role.sql",
	"202611180009_create_vms_pegawai_pejabat.sql",
	"202611180010_create_vms_role_pegawai.sql",
	"202611180011_create_vms_feature.sql",
	"202611180012_create_vms_action.sql",
	"202611180013_create_vms_permission.sql",
	"202611180014_create_vms_policy.sql",
	"202611180015_create_vms_role_policy.sql",
	"202611180016_create_vms_policy_permission.sql",
	"202611180017_create_vms_audit_log.sql",
	"202611180018_create_support_rate_limit_buckets.sql",
	"202611180019_create_support_rate_limit_events.sql",
	"202611180020_create_support_employee_credentials.sql",
	"202611180021_create_support_staff_sessions.sql",
	"202611180022_create_support_refresh_lineage.sql",
	"202611180023_create_support_refresh_recovery.sql",
	"202611180024_create_vms_form_documents.sql",
	"202611180025_create_support_form_document_uploads.sql",
	"202611180026_create_vms_detail_kunjungan.sql",
	"202611180027_create_vms_detail_pengunjung.sql",
	"202611180028_create_vms_kunjungan.sql",
	"202611180029_create_support_booking_requests.sql",
	"202611180030_create_vms_qr_kunjungan.sql",
	"202611210001_expand_pegawai_phone_numbers.sql",
}

var (
	createTable = regexp.MustCompile(`(?i)\bCREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-z_]+\.[a-z_]+)`)
	dropTable   = regexp.MustCompile(`(?i)\bDROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?([a-z_]+\.[a-z_]+)`)
	tableFile   = regexp.MustCompile(`^\d{12}_create_(vms|support)_([a-z_]+)\.sql$`)
)

func TestEmbeddedSourceHoldsOnePrerequisiteThenOneMigrationPerTableInSchemaOrder(t *testing.T) {
	items, err := vmsmigration.Source().FindMigrations()
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.Id
	}
	if strings.Join(ids, "\n") != strings.Join(migrationOrder, "\n") {
		t.Fatalf("embedded migrations differ from the physical schema order:\n%s", strings.Join(ids, "\n"))
	}
	for _, item := range items {
		if len(item.Up) == 0 || len(item.Down) == 0 {
			t.Fatalf("%s needs both Up and Down statements", item.Id)
		}
	}
}

func TestPrerequisiteCreatesOnlySchemasAndTheirUsageGrants(t *testing.T) {
	items, err := vmsmigration.Source().FindMigrations()
	if err != nil {
		t.Fatal(err)
	}
	up := strings.Join(items[0].Up, "\n")
	for _, want := range []string{"CREATE SCHEMA IF NOT EXISTS vms", "CREATE SCHEMA IF NOT EXISTS support", "GRANT USAGE ON SCHEMA vms, support TO vms_api"} {
		if !strings.Contains(up, want) {
			t.Fatalf("prerequisite Up lacks %q", want)
		}
	}
	if createTable.MatchString(up) {
		t.Fatal("prerequisite must not create tables")
	}
	down := strings.Join(items[0].Down, "\n")
	// The migration ledger lives in support, so only vms is dropped.
	if !strings.Contains(down, "REVOKE USAGE ON SCHEMA vms, support FROM vms_api") || !strings.Contains(down, "DROP SCHEMA vms;") || strings.Contains(down, "DROP SCHEMA support") {
		t.Fatal("prerequisite Down must revoke usage and drop only the vms schema")
	}
}

func TestEveryTableMigrationCreatesAndDropsOnlyItsOwnTable(t *testing.T) {
	items, err := vmsmigration.Source().FindMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items[1:] {
		if item.Id == "202611210001_expand_pegawai_phone_numbers.sql" {
			continue
		}
		match := tableFile.FindStringSubmatch(item.Id)
		if match == nil {
			t.Fatalf("%s does not follow <timestamp>_create_<schema>_<table>.sql", item.Id)
		}
		table := match[1] + "." + match[2]
		up, down := strings.Join(item.Up, "\n"), strings.Join(item.Down, "\n")
		if created := createTable.FindAllStringSubmatch(up, -1); len(created) != 1 || created[0][1] != table {
			t.Fatalf("%s must create exactly %s, got %v", item.Id, table, created)
		}
		if dropped := dropTable.FindAllStringSubmatch(down, -1); len(dropped) != 1 || dropped[0][1] != table {
			t.Fatalf("%s Down must drop exactly %s, got %v", item.Id, table, dropped)
		}
		if strings.Contains(strings.ToUpper(down), "CASCADE") {
			t.Fatalf("%s Down must not cascade into other tables", item.Id)
		}
		if !strings.Contains(up, "TO vms_api") || !strings.Contains(down, "FROM vms_api") {
			t.Fatalf("%s must grant its runtime privileges in Up and revoke them in Down", item.Id)
		}
	}
}
