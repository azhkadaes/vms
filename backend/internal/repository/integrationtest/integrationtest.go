//go:build integration

// Package integrationtest opens the disposable vms_stage1 database as the runtime role for repository tests.
package integrationtest

import (
	"net"
	"net/url"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open refuses every target except a loopback (or compose "db") vms_stage1 reached as vms_api.
func Open(t testing.TB) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("VMS_INTEGRATION_DATABASE_URL")
	u, err := url.Parse(dsn)
	if err != nil || u == nil || u.Path != "/vms_stage1" || u.User == nil || u.User.Username() != "vms_api" {
		t.Fatal("explicit vms_api URL for the disposable vms_stage1 database is required")
	}
	if u.Hostname() != "db" && u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback() {
		t.Fatal("integration target must be isolated loopback/db")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot open integration database")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal("integration pool unavailable")
	}
	t.Cleanup(func() { _ = pool.Close() })
	var database, role string
	var version int
	if err := pool.QueryRow("SELECT current_database(), current_user, current_setting('server_version_num')::integer").Scan(&database, &role, &version); err != nil ||
		database != "vms_stage1" || role != "vms_api" || version < 180000 || version >= 190000 {
		t.Fatal("wrong integration database, role or PostgreSQL major")
	}
	return db
}

// RolledBack returns a transaction undone at cleanup. Repository transactions nest inside it
// as savepoints, so fixtures need no DELETE privilege and leave no rows behind.
func RolledBack(t testing.TB, db *gorm.DB) *gorm.DB {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal("cannot begin integration transaction")
	}
	t.Cleanup(func() { tx.Rollback() })
	return tx
}
