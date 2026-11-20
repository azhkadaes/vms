package databasehelper

import (
	"context"
	"crypto/x509"
	"database/sql"
	"errors"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewGormDB creates an owned connection pool. It never creates schemas or migrates.
func NewGormDB(ctx context.Context, dsn string) (*gorm.DB, *sql.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, nil, errors.New("database connection could not be initialized")
	}
	pool, err := db.DB()
	if err != nil {
		return nil, nil, errors.New("database pool unavailable")
	}
	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(30 * time.Minute)
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, nil, connectionFailure(err)
	}
	var version int
	if err := pool.QueryRowContext(ctx, "SELECT current_setting('server_version_num')::integer").Scan(&version); err != nil || version < 180000 || version >= 190000 {
		_ = pool.Close()
		return nil, nil, errors.New("VMS requires the selected PostgreSQL18 database major")
	}
	return db, pool, nil
}

// connectionFailure reports only a category, never driver text or connection secrets.
func connectionFailure(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "28P01", "28000":
			return errors.New("database authentication rejected; check DB_USER and its matching DB_PASSWORD or DB_PASSWORD_FILE")
		case "3D000":
			return errors.New("database does not exist on this server; check DB_NAME and DB_HOST/DB_PORT")
		case "42501":
			return errors.New("database access denied; the configured account needs CONNECT permission")
		case "53300":
			return errors.New("database connection capacity exhausted; retry after connections are released")
		case "57P03":
			return errors.New("database is not accepting connections yet; wait for server readiness")
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("database connection timed out; check DB_HOST/DB_PORT and server availability")
	}
	if errors.Is(err, context.Canceled) {
		return errors.New("database connection canceled")
	}
	var unknownCA x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalidCert x509.CertificateInvalidError
	if errors.As(err, &unknownCA) || errors.As(err, &hostname) || errors.As(err, &invalidCert) {
		return errors.New("database TLS certificate verification failed; check server certificate trust and hostname")
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return errors.New("database hostname could not be resolved; use DB_HOST=127.0.0.1 for the local Docker database from Windows")
	}
	var networkError *net.OpError
	if errors.As(err, &networkError) {
		return errors.New("database network connection failed; check DB_HOST/DB_PORT against the published Docker port")
	}
	return errors.New("database connection failed; check server availability, credentials and DB_SSL_MODE")
}

// readinessTables lists every migrated table in migration order. Each new table migration
// must be added here (readiness_test.go enforces it), so a half-migrated database is not served.
var readinessTables = []string{
	"vms.kedeputian", "vms.direktorat", "vms.jabatan", "vms.pejabat", "vms.ruangan", "vms.pegawai",
	"vms.role", "vms.pegawai_pejabat", "vms.role_pegawai", "vms.feature", "vms.action", "vms.permission",
	"vms.policy", "vms.role_policy", "vms.policy_permission", "vms.audit_log",
	"support.rate_limit_buckets", "support.rate_limit_events", "support.employee_credentials",
	"support.staff_sessions", "support.refresh_lineage", "support.refresh_recovery",
	"vms.form_documents", "support.form_document_uploads",
	"vms.detail_kunjungan", "vms.detail_pengunjung", "vms.kunjungan", "support.booking_requests",
	"vms.qr_kunjungan",
}

// Readiness includes the migrated schema, not merely a working socket.
type Readiness struct{ Pool *sql.DB }

// PingContext requires every migrated business and support table to exist.
func (r Readiness) PingContext(ctx context.Context) error {
	var missing int
	err := r.Pool.QueryRowContext(ctx, "SELECT count(*) FROM unnest($1::text[]) AS required(name) WHERE to_regclass(required.name) IS NULL", readinessTables).Scan(&missing)
	if err != nil || missing != 0 {
		return errors.New("required VMS schema unavailable")
	}
	return nil
}
