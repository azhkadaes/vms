package authrepositoryimplementation_test

import (
	"context"
	"errors"
	"testing"

	implementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	"gorm.io/gorm"
)

func TestNoncanonicalSessionIDsNeverReachDatabase(t *testing.T) {
	// A nil pool deliberately makes any accidental database access fail the test.
	repo := implementation.NewAuthRepository(nil)
	for _, id := range []string{"", "not-a-session", "00000000-0000-0000-0000-000000000000",
		"82D2BB6D-FDCB-4FF5-A914-91E5E9CE9559", "82d2bb6dfdcb4ff5a91491e5e9ce9559",
		"urn:uuid:82d2bb6d-fdcb-4ff5-a914-91e5e9ce9559"} {
		if _, err := repo.GetSessionSnapshot(context.Background(), id); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("noncanonical session identifier accepted: %v", err)
		}
	}
}
