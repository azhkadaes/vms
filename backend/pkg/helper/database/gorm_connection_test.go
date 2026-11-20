package databasehelper

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestConnectionFailureRedactsDriverDetails(t *testing.T) {
	const secret = "private-driver-connection-details"
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"password", &pgconn.PgError{Code: "28P01", Message: secret}, "authentication rejected"},
		{"authorization", &pgconn.PgError{Code: "28000", Message: secret}, "authentication rejected"},
		{"database", &pgconn.PgError{Code: "3D000", Message: secret}, "database does not exist"},
		{"permission", &pgconn.PgError{Code: "42501", Message: secret}, "access denied"},
		{"capacity", &pgconn.PgError{Code: "53300", Message: secret}, "capacity exhausted"},
		{"starting", &pgconn.PgError{Code: "57P03", Message: secret}, "not accepting connections"},
		{"timeout", context.DeadlineExceeded, "timed out"},
		{"canceled", context.Canceled, "canceled"},
		{"tls", x509.UnknownAuthorityError{}, "certificate verification failed"},
		{"dns", &net.DNSError{Name: secret, Err: secret}, "hostname could not be resolved"},
		{"network", &net.OpError{Op: "dial", Err: errors.New(secret)}, "network connection failed"},
		{"unknown postgres", &pgconn.PgError{Code: "XX000", Message: secret}, "database connection failed"},
		{"unknown", errors.New(secret), "database connection failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// pgx wraps server and network failures; classification must survive wrapping.
			result := connectionFailure(fmt.Errorf("%s: %w", secret, tc.err))
			if !strings.Contains(result.Error(), tc.want) || strings.Contains(result.Error(), secret) || errors.Unwrap(result) != nil {
				t.Fatalf("incorrect or unsafe error: %s", result)
			}
		})
	}
}
