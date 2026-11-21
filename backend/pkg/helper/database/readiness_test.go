package databasehelper

import (
	"regexp"
	"slices"
	"testing"

	vmsmigration "github.com/azhkadaes/vms/backend/migration"
)

var migratedTable = regexp.MustCompile(`^\d{12}_create_(vms|support)_([a-z_]+)\.sql$`)

// A table added by a migration must also gate readiness, so a half-migrated database is not served.
func TestReadinessRequiresEveryMigratedTable(t *testing.T) {
	items, err := vmsmigration.Source().FindMigrations()
	if err != nil {
		t.Fatal(err)
	}
	var migrated []string
	for _, item := range items {
		if match := migratedTable.FindStringSubmatch(item.Id); match != nil {
			migrated = append(migrated, match[1]+"."+match[2])
		}
	}
	if !slices.Equal(readinessTables, migrated) {
		t.Fatalf("readiness tables differ from migrated tables:\n got %v\nwant %v", readinessTables, migrated)
	}
}
