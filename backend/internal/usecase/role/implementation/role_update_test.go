package roleusecaseimplementation_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

func update(permissions []string, version int32) dto.RoleUpdate {
	return dto.RoleUpdate{Permissions: permissions, Version: version}
}

func TestSavingARoleReplacesItsPermissionsBumpsItsVersionAndRevisionAndAuditsTheChange(t *testing.T) {
	store := newStore()
	requested := []string{"reportExport", "bookings", "create", "reject", "reschedule", "cancel", "reports"}
	got, err := newUsecase(store).UpdateRole(context.Background(), store.proof(), 1, update(requested, 1))
	if err != nil {
		t.Fatal(err)
	}
	stored := []string{"bookings", "create", "reject", "reschedule", "cancel", "reports", "reportExport"}
	if got.ID != "1" || got.Name != policy.RolePA || got.Version != 2 || !slices.Equal(got.Permissions, stored) || !got.Editable ||
		!slices.Equal(got.AssignablePermissions, seeded[0].Permissions) || got.Description == "" {
		t.Fatalf("updated definition = %+v", got)
	}
	if want := []string{"LockAuthority", "LockRole", "SavePermissions", "BumpRoleRevision", "AuditPermissionChange"}; !slices.Equal(store.calls, want) {
		t.Fatalf("calls = %v, want %v", store.calls, want)
	}
	if store.transactions != 1 || !slices.Equal(store.saved, requested) || store.savedBy != 9 || !store.savedAt.Equal(now) {
		t.Fatalf("saved %v by %d at %v in %d transactions", store.saved, store.savedBy, store.savedAt, store.transactions)
	}
	if !slices.Equal(store.bumped, []int32{1}) {
		t.Fatalf("revision bumped for roles %v", store.bumped)
	}
	if len(store.audits) != 1 {
		t.Fatalf("audits = %+v", store.audits)
	}
	event := store.audits[0]
	if event.actor != 9 || !event.at.Equal(now) || event.before.Version != 1 || !slices.Equal(event.before.Permissions, seeded[0].Permissions) ||
		event.after.Version != 2 || !slices.Equal(event.after.Permissions, stored) {
		t.Fatalf("audit event = %+v", event)
	}
}

func TestSavingTheCallersOwnRoleBumpsTheirOwnRevisionToo(t *testing.T) {
	store := newStore()
	requested := []string{"bookings", "reports", "reportExport", "people", "roles"}
	got, err := newUsecase(store).UpdateRole(context.Background(), store.proof(), 4, update(requested, 4))
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 5 || !slices.Equal(got.Permissions, requested) || !slices.Equal(store.bumped, []int32{4}) {
		t.Fatalf("got %+v, bumped %v", got, store.bumped)
	}
}

func TestSavingAnUnchangedPermissionSetWritesNothing(t *testing.T) {
	store := newStore()
	got, err := newUsecase(store).UpdateRole(context.Background(), store.proof(), 3, update([]string{"checkout", "bookings", "checkin"}, 1))
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 1 || !slices.Equal(got.Permissions, seeded[2].Permissions) {
		t.Fatalf("got %+v", got)
	}
	if want := []string{"LockAuthority", "LockRole"}; !slices.Equal(store.calls, want) {
		t.Fatalf("calls = %v, want %v", store.calls, want)
	}
}

// writes are the calls that change state; a refused save must make none of them.
var writes = []string{"SavePermissions", "BumpRoleRevision", "AuditPermissionChange"}

func assertNoWrite(t *testing.T, store *roleStore) {
	t.Helper()
	for _, call := range store.calls {
		if slices.Contains(writes, call) {
			t.Fatalf("a refused save wrote: %v", store.calls)
		}
	}
}

func TestSavingRefusesInvalidPermissionsWithAFieldError(t *testing.T) {
	for _, tc := range []struct {
		name        string
		role        int32
		permissions []string
		version     int32
		field       string
	}{
		{"unknown permission", 1, []string{"bookings", "superpower"}, 1, "permissions"},
		{"permission in another spelling", 1, []string{"Bookings"}, 1, "permissions"},
		{"repeated permission", 3, []string{"bookings", "checkin", "checkin"}, 1, "permissions"},
		{"permissions omitted or null", 3, nil, 1, "permissions"},
		{"grant a permission the role may not hold", 3, []string{"bookings", "checkin", "checkout", "approve"}, 1, "permissions"},
		{"grant role administration to a PA", 1, append(slices.Clone(seeded[0].Permissions), "roles"), 1, "permissions"},
		{"withdraw the fixed role administration from Super Admin", 4, []string{"bookings", "people", "integrations", "reports", "reportExport", "audit"}, 4, "permissions"},
		{"version omitted", 1, seeded[0].Permissions, 0, "version"},
		{"negative version", 1, seeded[0].Permissions, -1, "version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newStore()
			_, err := newUsecase(store).UpdateRole(context.Background(), store.proof(), tc.role, update(tc.permissions, tc.version))
			var business *failure.Error
			if !errors.As(err, &business) || business.Status != 422 || business.Code != "validation_failed" || len(business.Fields[tc.field]) == 0 {
				t.Fatalf("got %v, want 422 with fieldErrors.%s", err, tc.field)
			}
			for field, messages := range business.Fields {
				for _, message := range messages {
					if message == "" {
						t.Fatalf("blank message for %s", field)
					}
				}
			}
			assertNoWrite(t, store)
		})
	}
}

func TestSavingAStaleVersionIsAVersionConflictWithoutWriting(t *testing.T) {
	store := newStore()
	_, err := newUsecase(store).UpdateRole(context.Background(), store.proof(), 4, update([]string{"bookings", "roles"}, 3))
	if status, code := statusOf(err); status != 409 || code != "version_conflict" {
		t.Fatalf("got %v, want 409 version_conflict", err)
	}
	assertNoWrite(t, store)
}

func TestSavingAnUnknownRoleIsNotFound(t *testing.T) {
	store := newStore()
	_, err := newUsecase(store).UpdateRole(context.Background(), store.proof(), 5, update([]string{"bookings"}, 1))
	if !errors.Is(err, failure.ErrNotFound) {
		t.Fatalf("got %v, want not found", err)
	}
	assertNoWrite(t, store)
}

func TestSavingARoleWithoutAnActivePolicyIsRefusedForItsState(t *testing.T) {
	store := newStore()
	store.roles[2].PolicyID, store.roles[2].Permissions = 0, []string{}
	_, err := newUsecase(store).UpdateRole(context.Background(), store.proof(), 3, update([]string{"bookings"}, 1))
	if status, code := statusOf(err); status != 409 || code != "invalid_state" {
		t.Fatalf("got %v, want 409 invalid_state", err)
	}
	assertNoWrite(t, store)
}

// The write transaction rechecks authority: the snapshot read under the employee lock, not the
// session StaffAuth read earlier, decides. A refused caller locks no role and writes nothing.
func TestSavingRechecksTheCallersAuthorityBeforeTouchingTheRole(t *testing.T) {
	forbidden := func(err error) bool { status, code := statusOf(err); return status == 403 && code == "forbidden" }
	for _, tc := range []struct {
		name   string
		change func(*entity.SessionSnapshot)
		want   func(error) bool
	}{
		{"roles permission withdrawn meanwhile", func(s *entity.SessionSnapshot) { s.Permissions = []string{"bookings", "people"} }, forbidden},
		{"another active role", func(s *entity.SessionSnapshot) {
			s.ActiveRole, s.AvailableRoles, s.Permissions = policy.RolePA, []string{policy.RolePA}, []string{"bookings", "approve"}
		}, forbidden},
		{"password change now required", func(s *entity.SessionSnapshot) { s.MustChangePassword = true }, forbidden},
		{"session revoked", func(s *entity.SessionSnapshot) { revoked := now; s.RevokedAt = &revoked },
			func(err error) bool { return errors.Is(err, authusecase.ErrUnauthenticated) }},
		{"session gone", func(s *entity.SessionSnapshot) { *s = entity.SessionSnapshot{} },
			func(err error) bool { return errors.Is(err, authusecase.ErrUnauthenticated) }},
		{"role revision moved on", func(s *entity.SessionSnapshot) { s.RoleRevision++ },
			func(err error) bool { return errors.Is(err, authusecase.ErrRoleChanged) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newStore()
			tc.change(&store.snapshot)
			_, err := newUsecase(store).UpdateRole(context.Background(), store.proof(), 1, update([]string{"bookings"}, 1))
			if !tc.want(err) {
				t.Fatalf("got %v", err)
			}
			if !slices.Equal(store.calls, []string{"LockAuthority"}) {
				t.Fatalf("a refused caller went on to %v", store.calls)
			}
		})
	}
}
