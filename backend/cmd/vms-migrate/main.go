package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	migrate "github.com/rubenv/sql-migrate"

	"github.com/azhkadaes/vms/backend/config"
	vmsmigration "github.com/azhkadaes/vms/backend/migration"
	databasehelper "github.com/azhkadaes/vms/backend/pkg/helper/database"
)

func main() {
	mode := flag.String("mode", "plan", "plan, status or up (no down operation)")
	expected := flag.String("expect-database", "", "required exact target database for status/up")
	flag.Parse()
	if err := run(*mode, *expected); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(mode, expected string) error {
	source := vmsmigration.Source()
	if mode == "plan" {
		items, err := source.FindMigrations()
		if err != nil {
			return errors.New("cannot read embedded VMS migration files")
		}
		for _, item := range items {
			fmt.Printf("%s: %d up statements\n", item.Id, len(item.Up))
		}
		return nil
	}
	if (mode != "up" && mode != "status") || expected == "" {
		return errors.New("use plan, or status/up with --expect-database; down is intentionally unavailable")
	}
	if err := config.LoadEnvironment(); err != nil {
		return err
	}
	dsn, err := config.DatabaseURLFrom(os.Getenv)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	_, pool, err := databasehelper.NewGormDB(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	var database, role string
	var version int
	if err := pool.QueryRowContext(ctx, "SELECT current_database(), current_user, current_setting('server_version_num')::integer").Scan(&database, &role, &version); err != nil {
		return errors.New("cannot verify migration target")
	}
	if database != expected || role != "vms_migrator" || version < 180000 || version >= 190000 {
		return errors.New("target must match --expect-database, use vms_migrator, and run PostgreSQL18")
	}
	conn, err := pool.Conn(ctx)
	if err != nil {
		return errors.New("cannot acquire migration lock connection")
	}
	defer conn.Close()
	var locked bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(864092846132)").Scan(&locked); err != nil || !locked {
		return errors.New("another VMS migration is running or migration lock unavailable")
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.ExecContext(cleanup, "SELECT pg_advisory_unlock(864092846132)")
	}()
	set := vmsmigration.Set()
	if mode == "status" {
		records, err := set.GetMigrationRecords(pool, "postgres")
		if err != nil {
			return errors.New("cannot read VMS migration status")
		}
		for _, record := range records {
			fmt.Println(record.Id)
		}
		return nil
	}
	n, err := set.ExecContext(ctx, pool, "postgres", source, migrate.Up)
	if err != nil {
		return errors.New("VMS migration failed; inspect migration status before retry; database details suppressed")
	}
	fmt.Printf("Applied %d VMS migrations to the explicitly verified target.\n", n)
	return nil
}
