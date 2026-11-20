//go:build integration

package rolerepositoryimplementation_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authimplementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	implementation "github.com/azhkadaes/vms/backend/internal/repository/role/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	roleserializer "github.com/azhkadaes/vms/backend/internal/serializer/role/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	authusecaseimplementation "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	roleusecase "github.com/azhkadaes/vms/backend/internal/usecase/role"
	roleusecaseimplementation "github.com/azhkadaes/vms/backend/internal/usecase/role/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// seededPermissions are the migration's default grants, in catalogue order.
var seededPermissions = map[string][]string{
	policy.RolePA:           {"bookings", "create", "approve", "reject", "reschedule", "cancel", "reports", "reportExport"},
	policy.RoleProtocol:     {"bookings", "create", "approve", "reject", "reschedule", "cancel", "reports", "reportExport"},
	policy.RoleReceptionist: {"bookings", "checkin", "checkout"},
	policy.RoleSuperAdmin:   {"bookings", "reports", "reportExport", "people", "roles", "integrations", "audit"},
}

var validator = authusecaseimplementation.NewSessionValidator(authserializer.NewAuthSerializer())

func usecaseOn(db *gorm.DB) roleusecase.RoleUsecase {
	return roleusecaseimplementation.NewRoleUsecase(implementation.NewRoleRepository(db), roleserializer.NewRoleSerializer(), validator, noop.NewTracerProvider().Tracer("role-integration"))
}

func digest(label string) string {
	sum := sha256.Sum256([]byte(label + uuid.NewString()))
	return hex.EncodeToString(sum[:])
}

func roleID(t *testing.T, db *gorm.DB, name string) int32 {
	t.Helper()
	var id int32
	if err := db.Raw("SELECT id FROM vms.role WHERE nama = ?", name).Scan(&id).Error; err != nil || id <= 0 {
		t.Fatalf("role %s not seeded: %v", name, err)
	}
	return id
}

func roleVersion(t *testing.T, db *gorm.DB, id int32) int32 {
	t.Helper()
	var version int32
	if err := db.Raw("SELECT version FROM vms.role WHERE id = ?", id).Scan(&version).Error; err != nil || version <= 0 {
		t.Fatalf("role %d version unreadable: %v", id, err)
	}
	return version
}

// signIn creates an active employee holding role, with a live session on it, and returns its proof.
func signIn(t *testing.T, db *gorm.DB, role string) entity.SessionProof {
	t.Helper()
	var employee, active int32
	if err := db.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password) VALUES ('Role Staff',?,TRUE,FALSE) RETURNING id`,
		"role-"+uuid.NewString()+"@example.test").Scan(&employee).Error; err != nil || employee <= 0 {
		t.Fatalf("cannot create employee: %v", err)
	}
	if err := db.Raw(`INSERT INTO vms.role_pegawai(id_karyawan,id_role) SELECT ?, id FROM vms.role WHERE nama = ? RETURNING id_role`, employee, role).Scan(&active).Error; err != nil || active <= 0 {
		t.Fatalf("cannot assign %s: %v", role, err)
	}
	sessionID := uuid.NewString()
	if err := authimplementation.NewSessionRepository(db).Transaction(context.Background(), employee, func(tx authrepository.SessionTransaction) error {
		now := tx.Now()
		return tx.Create(entity.StaffSession{ID: sessionID, EmployeeID: employee, ActiveRoleID: &active, CredentialGeneration: 1, RoleRevision: 1,
			RefreshHash: digest("role"), CreatedAt: now, LastActivityAt: now, IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(2 * time.Hour)})
	}); err != nil {
		t.Fatalf("cannot create session: %v", err)
	}
	return entity.SessionProof{SessionID: sessionID, EmployeeID: employee, CredentialGeneration: 1, RoleRevision: 1}
}

// sessionOf is what the next request of proof's holder gets from the session validator. It reads
// the authority at clock time, as a write recheck does: a rolled-back fixture transaction began
// before its sessions were created, so its transaction time would predate them.
func sessionOf(t *testing.T, db *gorm.DB, proof entity.SessionProof) (dto.Session, error) {
	t.Helper()
	snapshot, err := authimplementation.NewAuthorityLock(db).LockAuthority(proof)
	if err != nil {
		t.Fatal(err)
	}
	return validator.Validate(snapshot, proof)
}

func without(values []string, removed string) []string {
	return slices.DeleteFunc(slices.Clone(values), func(value string) bool { return value == removed })
}

func TestTheRoleListIsTheFourSeededRolesWithTheirActivePolicyPermissions(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	roles, err := implementation.NewRoleRepository(tx).GetRoles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, role := range roles {
		names = append(names, role.Name)
		if role.ID <= 0 || role.Version <= 0 || role.PolicyID <= 0 || !slices.Equal(role.Permissions, seededPermissions[role.Name]) {
			t.Errorf("role %+v, want permissions %v", role, seededPermissions[role.Name])
		}
	}
	if !slices.Equal(names, []string{policy.RolePA, policy.RoleProtocol, policy.RoleReceptionist, policy.RoleSuperAdmin}) {
		t.Fatalf("roles in ID order = %v", names)
	}
}

func TestSavingARoleChangesItsPolicyAndMakesItsLiveSessionsReReadTheirSession(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	ctx := context.Background()
	admin, pa, otherPA, receptionist := signIn(t, tx, policy.RoleSuperAdmin), signIn(t, tx, policy.RolePA), signIn(t, tx, policy.RolePA), signIn(t, tx, policy.RoleReceptionist)
	if err := tx.Exec("UPDATE support.staff_sessions SET revoked_at = clock_timestamp() WHERE id = ?", otherPA.SessionID).Error; err != nil {
		t.Fatal(err)
	}
	id := roleID(t, tx, policy.RolePA)
	version := roleVersion(t, tx, id)
	requested := without(seededPermissions[policy.RolePA], policy.PermApprove)

	got, err := usecaseOn(tx).UpdateRole(ctx, admin, id, dto.RoleUpdate{Permissions: requested, Version: version})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != strconv.Itoa(int(id)) || got.Version != version+1 || !slices.Equal(got.Permissions, requested) {
		t.Fatalf("saved definition %+v", got)
	}

	// The role row carries the new version and who changed it, as employee ID text.
	var row struct {
		Version   int32
		UpdatedBy string
		UpdatedAt time.Time
	}
	if err := tx.Raw("SELECT version, updated_by, updated_at FROM vms.role WHERE id = ?", id).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Version != version+1 || row.UpdatedBy != strconv.Itoa(int(admin.EmployeeID)) || row.UpdatedAt.IsZero() {
		t.Fatalf("role row %+v", row)
	}
	roles, err := implementation.NewRoleRepository(tx).GetRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(roles[id-1].Permissions, requested) {
		t.Fatalf("listed permissions %v, want %v", roles[id-1].Permissions, requested)
	}

	// The PA's next request is refused until it re-reads its session (refresh), which then lacks approve.
	if _, err := sessionOf(t, tx, pa); !errors.Is(err, authusecase.ErrRoleChanged) {
		t.Fatalf("live PA session after the change: %v, want role_changed", err)
	}
	pa.RoleRevision++
	session, err := sessionOf(t, tx, pa)
	if err != nil || slices.Contains(session.Permissions, policy.PermApprove) || !slices.Contains(session.Permissions, policy.PermCreate) {
		t.Fatalf("re-read PA session %+v (%v)", session, err)
	}
	// Sessions on other roles, and revoked ones, are left alone.
	for name, proof := range map[string]entity.SessionProof{"receptionist": receptionist, "admin": admin} {
		if _, err := sessionOf(t, tx, proof); err != nil {
			t.Fatalf("%s session disturbed: %v", name, err)
		}
	}
	var revision int64
	if err := tx.Raw("SELECT role_revision FROM support.staff_sessions WHERE id = ?", otherPA.SessionID).Scan(&revision).Error; err != nil || revision != 1 {
		t.Fatalf("revoked session revision = %d (%v)", revision, err)
	}

	// One audit event names the staff actor and the role target, with the before and after values.
	var events []struct {
		ActorType, EntityType, JenisKegiatan string
		IDPegawai                            int32 `gorm:"column:id_pegawai"`
		EntityID                             int32
		OldValues, NewValues                 string
	}
	if err := tx.Raw(`SELECT actor_type, id_pegawai, entity_type, entity_id, jenis_kegiatan, old_values::text AS old_values, new_values::text AS new_values
		FROM vms.audit_log WHERE entity_type = 'role' AND entity_id = ? AND id_pegawai = ?`, id, admin.EmployeeID).Scan(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ActorType != "staff" || events[0].JenisKegiatan != "role.permissions_changed" {
		t.Fatalf("audit events %+v", events)
	}
	var before, after struct {
		Permissions []string `json:"permissions"`
		Version     int32    `json:"version"`
	}
	if json.Unmarshal([]byte(events[0].OldValues), &before) != nil || json.Unmarshal([]byte(events[0].NewValues), &after) != nil ||
		before.Version != version || !slices.Equal(before.Permissions, seededPermissions[policy.RolePA]) || after.Version != version+1 || !slices.Equal(after.Permissions, requested) {
		t.Fatalf("audit values %s -> %s", events[0].OldValues, events[0].NewValues)
	}
}

func TestSavingTheCallersOwnActiveRoleMakesTheCallerReReadToo(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	admin := signIn(t, tx, policy.RoleSuperAdmin)
	id := roleID(t, tx, policy.RoleSuperAdmin)
	requested := without(seededPermissions[policy.RoleSuperAdmin], policy.PermAudit)
	if _, err := usecaseOn(tx).UpdateRole(context.Background(), admin, id, dto.RoleUpdate{Permissions: requested, Version: roleVersion(t, tx, id)}); err != nil {
		t.Fatal(err)
	}
	if _, err := sessionOf(t, tx, admin); !errors.Is(err, authusecase.ErrRoleChanged) {
		t.Fatalf("caller's own session: %v, want role_changed", err)
	}
}

func TestSavingRefusalsLeaveTheRoleUntouched(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	ctx := context.Background()
	admin, pa := signIn(t, tx, policy.RoleSuperAdmin), signIn(t, tx, policy.RolePA)
	id := roleID(t, tx, policy.RoleReceptionist)
	version := roleVersion(t, tx, id)
	for _, tc := range []struct {
		name   string
		proof  entity.SessionProof
		role   int32
		update dto.RoleUpdate
		want   func(error) bool
	}{
		{"caller without the roles permission", pa, id, dto.RoleUpdate{Permissions: []string{"bookings"}, Version: version}, func(err error) bool {
			var business *failure.Error
			return errors.As(err, &business) && business.Status == 403
		}},
		{"unknown role", admin, 2147483647, dto.RoleUpdate{Permissions: []string{"bookings"}, Version: 1}, func(err error) bool { return errors.Is(err, failure.ErrNotFound) }},
		{"stale version", admin, id, dto.RoleUpdate{Permissions: []string{"bookings"}, Version: version + 1}, func(err error) bool {
			var business *failure.Error
			return errors.As(err, &business) && business.Code == "version_conflict"
		}},
		{"permission the role may not hold", admin, id, dto.RoleUpdate{Permissions: []string{"bookings", "checkin", "checkout", "approve"}, Version: version}, func(err error) bool {
			var business *failure.Error
			return errors.As(err, &business) && business.Status == 422
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := usecaseOn(tx).UpdateRole(ctx, tc.proof, tc.role, tc.update); !tc.want(err) {
				t.Fatalf("got %v", err)
			}
			roles, err := implementation.NewRoleRepository(tx).GetRoles(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if roles[id-1].Version != version || !slices.Equal(roles[id-1].Permissions, seededPermissions[policy.RoleReceptionist]) {
				t.Fatalf("refused save changed the role: %+v", roles[id-1])
			}
			if _, err := sessionOf(t, tx, admin); err != nil {
				t.Fatalf("refused save disturbed a session: %v", err)
			}
		})
	}
}

func TestEveryRoleCanBeGrantedAndDeniedEachOfItsAssignablePermissions(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	ctx := context.Background()
	admin := signIn(t, tx, policy.RoleSuperAdmin)
	for _, name := range []string{policy.RolePA, policy.RoleProtocol, policy.RoleReceptionist, policy.RoleSuperAdmin} {
		id := roleID(t, tx, name)
		fixed := []string{}
		for _, key := range seededPermissions[name] {
			if !slices.Contains(policy.AssignablePermissions(name), key) {
				fixed = append(fixed, key)
			}
		}
		// Deny every assignable permission, then grant them all back.
		for _, permissions := range [][]string{fixed, seededPermissions[name]} {
			got, err := usecaseOn(tx).UpdateRole(ctx, admin, id, dto.RoleUpdate{Permissions: permissions, Version: roleVersion(t, tx, id)})
			if err != nil || !slices.Equal(got.Permissions, permissions) {
				t.Fatalf("%s saved %v (%v), want %v", name, got.Permissions, err, permissions)
			}
			// The admin's own role revision moves when the Super Admin role changes.
			if name == policy.RoleSuperAdmin {
				admin.RoleRevision++
			}
		}
	}
}
