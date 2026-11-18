//go:build integration

package admissionrepositoryimplementation_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/azhkadaes/vms/backend/internal/entity"
	admission "github.com/azhkadaes/vms/backend/internal/repository/admission/implementation"
	databasehelper "github.com/azhkadaes/vms/backend/pkg/helper/database"
)

// This suite requires explicit -tags integration and the reviewed disposable target.
func TestSharedPublicLimitOnPostgres18(t *testing.T) {
	dsn := os.Getenv("VMS_INTEGRATION_DATABASE_URL")
	u, err := url.Parse(dsn)
	if err != nil || u == nil || u.Path != "/vms_stage1" || u.User == nil || u.User.Username() != "vms_api" {
		t.Fatal("explicit vms_api URL for the disposable vms_stage1 database is required")
	}
	if u.Hostname() != "db" && u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback() {
		t.Fatal("integration target must be isolated loopback/db")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, pool, err := databasehelper.NewGormDB(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var database, role string
	if err := pool.QueryRowContext(ctx, "SELECT current_database(),current_user").Scan(&database, &role); err != nil || database != "vms_stage1" || role != "vms_api" {
		t.Fatal("wrong integration database/role")
	}
	digest := sha256.Sum256([]byte("integration-only:" + uuid.NewString()))
	key := hex.EncodeToString(digest[:])
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := db.WithContext(cleanup).Exec("DELETE FROM support.rate_limit_buckets WHERE bucket_key = ?", key).Error; err != nil {
			t.Error("could not remove this test's private rate bucket")
		}
	}()
	instances := []interface {
		PublicRequest(context.Context, string) (entity.RequestAdmission, error)
	}{admission.NewAdmissionRepository(db), admission.NewAdmissionRepository(db)}
	type outcome struct {
		decision entity.RequestAdmission
		err      error
	}
	results := make(chan outcome, 130)
	var workers sync.WaitGroup
	for i := 0; i < 130; i++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			decision, err := instances[index%2].PublicRequest(ctx, key)
			results <- outcome{decision, err}
		}(i)
	}
	workers.Wait()
	close(results)
	allowed, denied := 0, 0
	for result := range results {
		if result.err != nil {
			t.Fatal("concurrent admission database operation failed")
		}
		if result.decision.Allowed {
			allowed++
		} else {
			denied++
			if result.decision.RetryAfter < 1 || result.decision.RetryAfter > 60 {
				t.Fatal("invalid retry deadline")
			}
		}
	}
	if allowed != 120 || denied != 10 {
		t.Fatalf("got %d admitted/%d denied; expected120/10", allowed, denied)
	}
	decision, err := admission.NewAdmissionRepository(db).PublicRequest(ctx, key)
	if err != nil || decision.Allowed {
		t.Fatal("new repository instance reset durable limit")
	}
	var stored int
	if err := pool.QueryRowContext(ctx, "SELECT count(*) FROM support.rate_limit_events WHERE bucket_key = $1", key).Scan(&stored); err != nil || stored != 120 {
		t.Fatal("rejected requests changed the admitted-event count")
	}
	if err := db.WithContext(ctx).Exec("UPDATE support.rate_limit_events SET occurred_at = clock_timestamp() - INTERVAL '61 seconds' WHERE bucket_key = ?", key).Error; err != nil {
		t.Fatal("cannot age this test's synthetic events")
	}
	decision, err = admission.NewAdmissionRepository(db).PublicRequest(ctx, key)
	if err != nil || !decision.Allowed {
		t.Fatal("expired allowance did not recover")
	}
	if err := pool.QueryRowContext(ctx, "SELECT count(*) FROM support.rate_limit_events WHERE bucket_key = $1", key).Scan(&stored); err != nil || stored != 1 {
		t.Fatal("expired events were retained or rejection extended the window")
	}
	var canCreate bool
	if err := pool.QueryRowContext(ctx, "SELECT has_schema_privilege(current_user,'vms','CREATE')").Scan(&canCreate); err != nil || canCreate {
		t.Fatal("runtime account can create business-schema objects")
	}
}
