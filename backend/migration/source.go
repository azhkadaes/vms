package vmsmigration

import (
	"embed"

	migrate "github.com/rubenv/sql-migrate"
)

//go:embed files/*.sql
var files embed.FS

// Source contains only new VMS migrations, never the template's monogo migrations.
func Source() migrate.MigrationSource {
	return &migrate.EmbedFileSystemMigrationSource{FileSystem: files, Root: "files"}
}

// Set keeps the migration ledger in support.schema_migrations for every caller.
func Set() migrate.MigrationSet {
	return migrate.MigrationSet{SchemaName: "support", TableName: "schema_migrations"}
}
