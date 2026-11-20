package roleusecaseimplementation_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	rolerepository "github.com/azhkadaes/vms/backend/internal/repository/role"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	roleserializer "github.com/azhkadaes/vms/backend/internal/serializer/role/implementation"
	authusecaseimplementation "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	roleusecase "github.com/azhkadaes/vms/backend/internal/usecase/role"
	implementation "github.com/azhkadaes/vms/backend/internal/usecase/role/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

var seeded = []entity.Role{
	{ID: 1, Name: policy.RolePA, Version: 1, PolicyID: 1, Permissions: []string{"bookings", "create", "approve", "reject", "reschedule", "cancel", "reports", "reportExport"}},
	{ID: 2, Name: policy.RoleProtocol, Version: 1, PolicyID: 2, Permissions: []string{"bookings", "create", "approve", "reject", "reschedule", "cancel", "reports", "reportExport"}},
	{ID: 3, Name: policy.RoleReceptionist, Version: 1, PolicyID: 3, Permissions: []string{"bookings", "checkin", "checkout"}},
	{ID: 4, Name: policy.RoleSuperAdmin, Version: 4, PolicyID: 4, Permissions: []string{"bookings", "reports", "reportExport", "people", "roles", "integrations", "audit"}},
}

// roleStore is the role repository and its transaction. It records every lock and write so a
// test can prove what a refused change did not touch.
type roleStore struct {
	roles        []entity.Role
	snapshot     entity.SessionSnapshot
	rolesErr     error
	transactions int
	calls        []string
	saved        []string
	savedBy      int32
	savedAt      time.Time
	bumped       []int32
	audits       []audit
}

type audit struct {
	actor         int32
	before, after entity.Role
	at            time.Time
}

var now = time.Date(2026, 11, 19, 2, 0, 0, 0, time.UTC)

// newStore holds the seeded roles and a live Super Admin session (employee 9) with the roles permission.
func newStore() *roleStore {
	roles := make([]entity.Role, len(seeded))
	for i, role := range seeded {
		role.Permissions = slices.Clone(role.Permissions)
		roles[i] = role
	}
	return &roleStore{roles: roles, snapshot: entity.SessionSnapshot{EmployeeID: 9, Name: "Admin Example", AccountActive: true,
		ActiveRole: policy.RoleSuperAdmin, AvailableRoles: []string{policy.RoleSuperAdmin}, Permissions: slices.Clone(seeded[3].Permissions),
		AssignedOfficerIDs: []int32{}, CredentialGeneration: 1, RoleRevision: 2, CreatedAt: now.Add(-time.Hour), LastActivityAt: now.Add(-time.Minute),
		IdleExpiresAt: now.Add(7 * time.Hour), AbsoluteExpiresAt: now.Add(23 * time.Hour), ServerTime: now}}
}

func (s *roleStore) proof() entity.SessionProof {
	return entity.SessionProof{SessionID: "82d2bb6d-fdcb-4ff5-a914-91e5e9ce9559", EmployeeID: 9, CredentialGeneration: 1, RoleRevision: 2}
}

func (s *roleStore) GetRoles(context.Context) ([]entity.Role, error) {
	s.calls = append(s.calls, "GetRoles")
	return s.roles, s.rolesErr
}
func (s *roleStore) Transaction(_ context.Context, fn func(rolerepository.RoleTransaction) error) error {
	s.transactions++
	return fn(s)
}
func (s *roleStore) LockAuthority(entity.SessionProof) (entity.SessionSnapshot, error) {
	s.calls = append(s.calls, "LockAuthority")
	return s.snapshot, nil
}
func (s *roleStore) LockRole(id int32) (entity.Role, error) {
	s.calls = append(s.calls, "LockRole")
	for _, role := range s.roles {
		if role.ID == id {
			role.Permissions = slices.Clone(role.Permissions)
			return role, nil
		}
	}
	return entity.Role{}, rolerepository.ErrNotFound
}
func (s *roleStore) SavePermissions(role entity.Role, permissions []string, actor int32, at time.Time) (entity.Role, error) {
	s.calls = append(s.calls, "SavePermissions")
	s.saved, s.savedBy, s.savedAt = slices.Clone(permissions), actor, at
	stored := entity.Role{ID: role.ID, Name: role.Name, Version: role.Version + 1, PolicyID: role.PolicyID, Permissions: []string{}}
	for _, key := range policy.PermissionCatalogue() {
		if slices.Contains(permissions, key) {
			stored.Permissions = append(stored.Permissions, key)
		}
	}
	return stored, nil
}
func (s *roleStore) BumpRoleRevision(roleID int32, at time.Time) error {
	s.calls = append(s.calls, "BumpRoleRevision")
	s.bumped = append(s.bumped, roleID)
	return nil
}
func (s *roleStore) AuditPermissionChange(actor int32, before, after entity.Role, at time.Time) error {
	s.calls = append(s.calls, "AuditPermissionChange")
	s.audits = append(s.audits, audit{actor, before, after, at})
	return nil
}

func newUsecase(store *roleStore) roleusecase.RoleUsecase {
	return implementation.NewRoleUsecase(store, roleserializer.NewRoleSerializer(),
		authusecaseimplementation.NewSessionValidator(authserializer.NewAuthSerializer()), noop.NewTracerProvider().Tracer("role-test"))
}

func adminSession() dto.Session {
	return dto.Session{Active: true, ID: "9", Name: "Admin Example", Role: policy.RoleSuperAdmin, AvailableRoles: []string{policy.RoleSuperAdmin},
		RoleRevision: 2, Permissions: slices.Clone(seeded[3].Permissions), AssignedOfficerIDs: []string{}}
}

func statusOf(err error) (int, string) {
	var business *failure.Error
	if errors.As(err, &business) {
		return business.Status, business.Code
	}
	return 0, ""
}

func TestListingRolesShowsTheFourFixedRolesWithTheirAssignablePermissions(t *testing.T) {
	store := newStore()
	got, err := newUsecase(store).GetRoles(context.Background(), adminSession())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d roles", len(got))
	}
	want := []dto.RoleDefinition{
		{ID: "1", Name: policy.RolePA, Permissions: seeded[0].Permissions, AssignablePermissions: seeded[0].Permissions, Editable: true, Version: 1},
		{ID: "2", Name: policy.RoleProtocol, Permissions: seeded[1].Permissions, AssignablePermissions: seeded[1].Permissions, Editable: true, Version: 1},
		{ID: "3", Name: policy.RoleReceptionist, Permissions: seeded[2].Permissions, AssignablePermissions: seeded[2].Permissions, Editable: true, Version: 1},
		{ID: "4", Name: policy.RoleSuperAdmin, Permissions: seeded[3].Permissions,
			AssignablePermissions: []string{"bookings", "reports", "reportExport", "people", "integrations", "audit"}, Editable: true, Version: 4},
	}
	for i, role := range got {
		if role.ID != want[i].ID || role.Name != want[i].Name || role.Version != want[i].Version || role.Editable != want[i].Editable ||
			!slices.Equal(role.Permissions, want[i].Permissions) || !slices.Equal(role.AssignablePermissions, want[i].AssignablePermissions) || role.Description == "" {
			t.Errorf("role %d = %+v, want %+v with a description", i, role, want[i])
		}
	}
}

func TestListingRolesNeedsTheRolesPermissionOfAReadySession(t *testing.T) {
	pa := dto.Session{Active: true, ID: "7", Role: policy.RolePA, AvailableRoles: []string{policy.RolePA, policy.RoleSuperAdmin}, Permissions: []string{"bookings", "approve"}}
	pending := adminSession()
	pending.MustChangePassword = true
	for name, session := range map[string]dto.Session{"another role's permissions": pa, "password change pending": pending} {
		t.Run(name, func(t *testing.T) {
			store := newStore()
			_, err := newUsecase(store).GetRoles(context.Background(), session)
			if status, code := statusOf(err); status != 403 || code != "forbidden" {
				t.Fatalf("got %v, want 403 forbidden", err)
			}
			if len(store.calls) != 0 {
				t.Fatalf("a refused caller read %v", store.calls)
			}
		})
	}
}

func TestListingRolesReportsARepositoryFailure(t *testing.T) {
	store := newStore()
	store.rolesErr = errors.New("database unavailable")
	if _, err := newUsecase(store).GetRoles(context.Background(), adminSession()); !errors.Is(err, store.rolesErr) {
		t.Fatalf("got %v", err)
	}
}
