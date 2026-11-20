//go:build integration

package rolerepositoryimplementation_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authimplementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	rolerepository "github.com/azhkadaes/vms/backend/internal/repository/role"
	implementation "github.com/azhkadaes/vms/backend/internal/repository/role/implementation"
	roleserializer "github.com/azhkadaes/vms/backend/internal/serializer/role/implementation"
	roleusecase "github.com/azhkadaes/vms/backend/internal/usecase/role"
	roleusecaseimplementation "github.com/azhkadaes/vms/backend/internal/usecase/role/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// These tests need several connections, so their employees and sessions are committed; the
// runtime role cannot delete employees, so cleanup revokes the sessions, removes the role
// assignments and deactivates the employees. Role changes are never committed: every save runs
// its whole write path and is then rolled back, because another package's suite may be reading
// the seeded grants at the same time.

var errUndone = errors.New("role save undone by the test")

// undoing rolls back each role save that got through its whole write path.
type undoing struct{ rolerepository.RoleRepository }

func (u undoing) Transaction(ctx context.Context, fn func(rolerepository.RoleTransaction) error) error {
	return u.RoleRepository.Transaction(ctx, func(tx rolerepository.RoleTransaction) error {
		if err := fn(tx); err != nil {
			return err
		}
		return errUndone
	})
}

func undoingUsecaseOn(db *gorm.DB) roleusecase.RoleUsecase {
	return roleusecaseimplementation.NewRoleUsecase(undoing{implementation.NewRoleRepository(db)}, roleserializer.NewRoleSerializer(), validator,
		noop.NewTracerProvider().Tracer("role-concurrency"))
}

// committedSignIn is signIn on committed rows, undone as far as vms_api can at cleanup.
func committedSignIn(t *testing.T, db *gorm.DB, role string) entity.SessionProof {
	t.Helper()
	proof := signIn(t, db, role)
	t.Cleanup(func() {
		for _, statement := range []string{
			"UPDATE support.staff_sessions SET revoked_at = clock_timestamp() WHERE employee_id = ? AND revoked_at IS NULL",
			"DELETE FROM vms.role_pegawai WHERE id_karyawan = ?",
			"UPDATE vms.pegawai SET status_aktif = FALSE WHERE id = ?",
		} {
			if err := db.Exec(statement, proof.EmployeeID).Error; err != nil {
				t.Errorf("cleanup of employee %d failed: %v", proof.EmployeeID, err)
			}
		}
	})
	return proof
}

// waitForBlockedRoleWriters waits until n role-writer statements are waiting for a lock.
func waitForBlockedRoleWriters(t *testing.T, db *gorm.DB, n int) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		var blocked int
		if err := db.Raw(`SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'
			AND (query LIKE '%pg_advisory_xact_lock%' OR query LIKE '%FOR UPDATE OF r%' OR query LIKE '%FOR UPDATE OF e%')`).Scan(&blocked).Error; err != nil {
			t.Fatal(err)
		}
		if blocked >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d role writers are waiting for a lock", blocked, n)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func receive(t *testing.T, results <-chan error) error {
	t.Helper()
	select {
	case err := <-results:
		return err
	case <-time.After(30 * time.Second):
		t.Fatal("role save did not finish")
		return nil
	}
}

// Two administrators save their own role at the same moment. Each holds its own employee lock
// and, as a holder of that role, needs the other's: unless role writers run one at a time,
// one of them dies in a deadlock.
func TestTwoAdminsSavingTheirOwnRoleAtOnceDoNotDeadlock(t *testing.T) {
	db := integrationtest.Open(t)
	ctx := context.Background()
	id := roleID(t, db, policy.RoleSuperAdmin)
	first, second := committedSignIn(t, db, policy.RoleSuperAdmin), committedSignIn(t, db, policy.RoleSuperAdmin)
	request := dto.RoleUpdate{Permissions: without(seededPermissions[policy.RoleSuperAdmin], policy.PermAudit), Version: roleVersion(t, db, id)}

	// Hold the role row so that both saves are under way before either can lock it.
	blocker := db.Begin()
	defer blocker.Rollback()
	var locked int32
	if err := blocker.Raw("SELECT id FROM vms.role WHERE id = ? FOR UPDATE", id).Scan(&locked).Error; err != nil || locked != id {
		t.Fatalf("cannot hold the role row: %v", err)
	}
	results := make(chan error, 2)
	for _, proof := range []entity.SessionProof{first, second} {
		go func(proof entity.SessionProof) {
			_, err := undoingUsecaseOn(db).UpdateRole(ctx, proof, id, request)
			results <- err
		}(proof)
	}
	waitForBlockedRoleWriters(t, db, 2)
	blocker.Rollback()

	for _, err := range []error{receive(t, results), receive(t, results)} {
		if !errors.Is(err, errUndone) {
			t.Fatalf("a save did not get through its write path: %v", err)
		}
	}
}

// A PA has authorized a write under the old permissions and not committed it yet. The role
// change waits for that write, so the write can never commit after a change that forbids it.
func TestARoleSaveWaitsForAWriteItsHolderAlreadyAuthorized(t *testing.T) {
	db := integrationtest.Open(t)
	ctx := context.Background()
	id := roleID(t, db, policy.RolePA)
	admin, pa := committedSignIn(t, db, policy.RoleSuperAdmin), committedSignIn(t, db, policy.RolePA)
	request := dto.RoleUpdate{Permissions: without(seededPermissions[policy.RolePA], policy.PermApprove), Version: roleVersion(t, db, id)}

	inFlight := db.Begin()
	defer inFlight.Rollback()
	snapshot, err := authimplementation.NewAuthorityLock(inFlight).LockAuthority(pa)
	if err != nil {
		t.Fatal(err)
	}
	if session, err := validator.Validate(snapshot, pa); err != nil || !slices.Contains(session.Permissions, policy.PermApprove) {
		t.Fatalf("in-flight PA authority %+v (%v)", session, err)
	}
	results := make(chan error, 1)
	go func() {
		_, err := undoingUsecaseOn(db).UpdateRole(ctx, admin, id, request)
		results <- err
	}()
	waitForBlockedRoleWriters(t, db, 1)
	select {
	case err := <-results:
		t.Fatalf("role saved while its holder's authorized write was in flight: %v", err)
	default:
	}
	if err := inFlight.Commit().Error; err != nil {
		t.Fatal(err)
	}
	if err := receive(t, results); !errors.Is(err, errUndone) {
		t.Fatalf("role save after the in-flight write: %v", err)
	}
}

func TestARoleSaveWaitsForAnAssignedEmployeeToActivateTheRole(t *testing.T) {
	db := integrationtest.Open(t)
	ctx := context.Background()
	id := roleID(t, db, policy.RolePA)
	admin, pa := committedSignIn(t, db, policy.RoleSuperAdmin), committedSignIn(t, db, policy.RolePA)
	request := dto.RoleUpdate{Permissions: without(seededPermissions[policy.RolePA], policy.PermApprove), Version: roleVersion(t, db, id)}
	if err := db.Exec("UPDATE support.staff_sessions SET active_role_id = NULL WHERE id = ?", pa.SessionID).Error; err != nil {
		t.Fatal(err)
	}
	activation := db.Begin()
	defer activation.Rollback()
	if _, err := authimplementation.NewAuthorityLock(activation).LockAuthority(pa); err != nil {
		t.Fatal(err)
	}
	if err := activation.Exec("UPDATE support.staff_sessions SET active_role_id = ?, role_revision = role_revision + 1 WHERE id = ?", id, pa.SessionID).Error; err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 1)
	go func() {
		_, err := undoingUsecaseOn(db).UpdateRole(ctx, admin, id, request)
		results <- err
	}()
	waitForBlockedRoleWriters(t, db, 1)
	if err := activation.Commit().Error; err != nil {
		t.Fatal(err)
	}
	if err := receive(t, results); !errors.Is(err, errUndone) {
		t.Fatalf("role save after activation: %v", err)
	}
}
