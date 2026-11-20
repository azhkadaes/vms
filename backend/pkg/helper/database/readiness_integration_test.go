//go:build integration

package databasehelper

import (
	"context"
	"net"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestReadinessPassesOnlyWhenEveryRequiredTableExists(t *testing.T) {
	dsn := os.Getenv("VMS_INTEGRATION_DATABASE_URL")
	u, err := url.Parse(dsn)
	if err != nil || u == nil || u.Path != "/vms_stage1" || u.User == nil || u.User.Username() != "vms_api" {
		t.Fatal("explicit vms_api URL for the disposable vms_stage1 database is required")
	}
	if u.Hostname() != "db" && u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback() {
		t.Fatal("integration target must be isolated loopback/db")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, pool, err := NewGormDB(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := (Readiness{Pool: pool}).PingContext(ctx); err != nil {
		t.Fatalf("migrated database reported not ready: %v", err)
	}
	original := readinessTables
	defer func() { readinessTables = original }()
	readinessTables = append(append([]string{}, original...), "vms.integration_keys")
	if err := (Readiness{Pool: pool}).PingContext(ctx); err == nil {
		t.Fatal("readiness passed although a required table is missing")
	}
}
