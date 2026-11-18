package authusecaseimplementation_test

import (
	"context"
	implementation "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	"strings"
	"testing"
)

func TestCredentialsPreservePasswordBytesAndEnforceUTF8Boundary(t *testing.T) {
	v, err := implementation.NewCredentialVerifier(4, 1)
	if err != nil {
		t.Fatal(err)
	}
	password := "  fifteen characters with spaces  "
	hash, err := v.Hash(context.Background(), password)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		hash, password string
		want           bool
	}{{hash, password, true}, {hash, strings.TrimSpace(password), false}, {"", password, false}, {"bcrypt:invalid", password, false}, {hash, strings.Repeat("x", 73), false}} {
		got, err := v.Verify(context.Background(), tc.hash, tc.password)
		if err != nil || got != tc.want {
			t.Fatal("wrong credential result")
		}
	}
	if _, err := v.Hash(context.Background(), strings.Repeat("é", 37)); err == nil {
		t.Fatal("over 72 UTF-8 bytes accepted")
	}
	if _, err := v.Hash(context.Background(), "short"); err == nil {
		t.Fatal("short new password accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := v.Verify(ctx, hash, password); err == nil {
		t.Fatal("canceled work accepted")
	}
}
