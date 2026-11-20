//go:build integration

package authrepositoryimplementation_test

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	implementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	authusecaseimplementation "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
)

// statements records every SQL statement a GORM session sends, in order.
type statements struct{ sql []string }

func (s *statements) LogMode(logger.LogLevel) logger.Interface { return s }
func (s *statements) Info(context.Context, string, ...any)     {}
func (s *statements) Warn(context.Context, string, ...any)     {}
func (s *statements) Error(context.Context, string, ...any)    {}
func (s *statements) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	s.sql = append(s.sql, sql)
}

var rowLock = regexp.MustCompile(`(?i)\bFOR\s+(UPDATE|NO\s+KEY\s+UPDATE|SHARE|KEY\s+SHARE)\b`)

// signedIn creates an employee with the PA role selected and returns the proof of a live session.
func signedIn(t *testing.T, tx *gorm.DB) entity.SessionProof {
	t.Helper()
	var employee, role int32
	if err := tx.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password) VALUES ('Authority Staff',?,TRUE,FALSE) RETURNING id`,
		"authority-"+uuid.NewString()+"@example.test").Scan(&employee).Error; err != nil || employee <= 0 {
		t.Fatalf("cannot create employee: %v", err)
	}
	if err := tx.Raw(`INSERT INTO vms.role_pegawai(id_karyawan,id_role) SELECT ?, id FROM vms.role WHERE nama = ? RETURNING id_role`, employee, policy.RolePA).Scan(&role).Error; err != nil || role <= 0 {
		t.Fatalf("cannot assign role: %v", err)
	}
	sessionID := uuid.NewString()
	if err := implementation.NewSessionRepository(tx).Transaction(context.Background(), employee, func(t authrepository.SessionTransaction) error {
		now := t.Now()
		return t.Create(entity.StaffSession{ID: sessionID, EmployeeID: employee, ActiveRoleID: &role, CredentialGeneration: 1, RoleRevision: 1,
			RefreshHash: digest("authority"), CreatedAt: now, LastActivityAt: now, IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(2 * time.Hour)})
	}); err != nil {
		t.Fatalf("cannot create session: %v", err)
	}
	return entity.SessionProof{SessionID: sessionID, EmployeeID: employee, CredentialGeneration: 1, RoleRevision: 1}
}

func TestLockAuthorityLocksTheEmployeeFirstAndOnlyTheEmployee(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	proof := signedIn(t, tx)
	recorder := &statements{}
	if _, err := implementation.NewAuthorityLock(tx.Session(&gorm.Session{Logger: recorder})).LockAuthority(proof); err != nil {
		t.Fatal(err)
	}
	if len(recorder.sql) == 0 || !rowLock.MatchString(recorder.sql[0]) || !regexp.MustCompile(`(?i)\bFROM\s+vms\.pegawai\b`).MatchString(recorder.sql[0]) {
		t.Fatalf("first statement must lock the employee row: %q", recorder.sql)
	}
	for _, sql := range recorder.sql[1:] {
		if rowLock.MatchString(sql) {
			t.Fatalf("authority lock takes a second row lock: %q", sql)
		}
	}
}

func TestLockAuthorityReturnsTheSessionSnapshotThatTheValidatorAccepts(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	ctx := context.Background()
	proof := signedIn(t, tx)
	var beforeLock time.Time
	if err := tx.Raw("SELECT clock_timestamp()").Scan(&beforeLock).Error; err != nil {
		t.Fatal(err)
	}
	locked, err := implementation.NewAuthorityLock(tx).LockAuthority(proof)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture rows are uncommitted, so finding them proves the read ran on the write transaction.
	read, err := implementation.NewAuthRepository(tx).GetSessionSnapshot(ctx, proof.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if locked.ServerTime.Before(beforeLock) {
		t.Fatalf("authority time %v predates the lock (%v): it must be read after locking, not at transaction start", locked.ServerTime, beforeLock)
	}
	locked.ServerTime, read.ServerTime = time.Time{}, time.Time{}
	if locked.EmployeeID != proof.EmployeeID || locked.ActiveRole != policy.RolePA || !slices.Equal(locked.AvailableRoles, read.AvailableRoles) ||
		!slices.Equal(locked.Permissions, read.Permissions) || !slices.Equal(locked.AssignedOfficerIDs, read.AssignedOfficerIDs) || len(locked.Permissions) == 0 {
		t.Fatalf("locked snapshot %+v differs from the current-session read %+v", locked, read)
	}
	locked.AvailableRoles, read.AvailableRoles, locked.Permissions, read.Permissions, locked.AssignedOfficerIDs, read.AssignedOfficerIDs = nil, nil, nil, nil, nil, nil
	if !equalSnapshots(locked, read) {
		t.Fatalf("locked snapshot %+v differs from the current-session read %+v", locked, read)
	}

	validator := authusecaseimplementation.NewSessionValidator(authserializer.NewAuthSerializer())
	locked, err = implementation.NewAuthorityLock(tx).LockAuthority(proof)
	if err != nil {
		t.Fatal(err)
	}
	session, err := validator.Validate(locked, proof)
	if err != nil || !session.Active || session.Role != policy.RolePA {
		t.Fatalf("validator rejected a live locked session: %+v %v", session, err)
	}

	// A revocation earlier in the same transaction is what the recheck sees.
	if err := tx.Exec("UPDATE support.staff_sessions SET revoked_at = clock_timestamp() WHERE id = ?", proof.SessionID).Error; err != nil {
		t.Fatal(err)
	}
	locked, err = implementation.NewAuthorityLock(tx).LockAuthority(proof)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validator.Validate(locked, proof); !errors.Is(err, authusecase.ErrUnauthenticated) {
		t.Fatalf("revoked session passed the recheck: %v", err)
	}
}

func equalSnapshots(a, b entity.SessionSnapshot) bool {
	sameRevocation := (a.RevokedAt == nil) == (b.RevokedAt == nil) && (a.RevokedAt == nil || a.RevokedAt.Equal(*b.RevokedAt))
	a.RevokedAt, b.RevokedAt = nil, nil
	return sameRevocation && a.CreatedAt.Equal(b.CreatedAt) && a.LastActivityAt.Equal(b.LastActivityAt) && a.IdleExpiresAt.Equal(b.IdleExpiresAt) &&
		a.AbsoluteExpiresAt.Equal(b.AbsoluteExpiresAt) && a.EmployeeID == b.EmployeeID && a.Name == b.Name && a.AccountActive == b.AccountActive &&
		a.MustChangePassword == b.MustChangePassword && a.ActiveRole == b.ActiveRole && a.CredentialGeneration == b.CredentialGeneration && a.RoleRevision == b.RoleRevision
}

func TestLockAuthoritySignsOutVanishedSessionsAndEmployees(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	live, other := signedIn(t, tx), signedIn(t, tx)
	validator := authusecaseimplementation.NewSessionValidator(authserializer.NewAuthSerializer())
	for name, proof := range map[string]entity.SessionProof{
		"unknown session":                    {SessionID: uuid.NewString(), EmployeeID: live.EmployeeID, CredentialGeneration: 1, RoleRevision: 1},
		"noncanonical session":               {SessionID: "not-a-session", EmployeeID: live.EmployeeID, CredentialGeneration: 1, RoleRevision: 1},
		"unknown employee":                   {SessionID: live.SessionID, EmployeeID: 2147483647, CredentialGeneration: 1, RoleRevision: 1},
		"session of another locked employee": {SessionID: live.SessionID, EmployeeID: other.EmployeeID, CredentialGeneration: 1, RoleRevision: 1},
	} {
		t.Run(name, func(t *testing.T) {
			snapshot, err := implementation.NewAuthorityLock(tx).LockAuthority(proof)
			if err != nil {
				t.Fatalf("a vanished session is a sign-out, not a failure: %v", err)
			}
			if snapshot.EmployeeID != 0 || len(snapshot.Permissions) != 0 {
				t.Fatalf("authority returned without its employee lock: %+v", snapshot)
			}
			if _, err := validator.Validate(snapshot, proof); !errors.Is(err, authusecase.ErrUnauthenticated) {
				t.Fatalf("vanished authority accepted: %v", err)
			}
		})
	}
}

func TestPolicyConstantsSpellTheSeededRolesAndPermissions(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	var roles, permissions []string
	if err := tx.Raw("SELECT nama FROM vms.role ORDER BY id").Scan(&roles).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Raw("SELECT permission_key FROM vms.permission ORDER BY id").Scan(&permissions).Error; err != nil {
		t.Fatal(err)
	}
	if want := []string{policy.RolePA, policy.RoleProtocol, policy.RoleReceptionist, policy.RoleSuperAdmin}; !slices.Equal(roles, want) {
		t.Fatalf("seeded roles %v, policy constants %v", roles, want)
	}
	want := []string{policy.PermBookings, policy.PermCreate, policy.PermApprove, policy.PermReject, policy.PermReschedule, policy.PermCancel,
		policy.PermCheckin, policy.PermCheckout, policy.PermReports, policy.PermReportExport, policy.PermPeople, policy.PermRoles,
		policy.PermIntegrations, policy.PermAudit}
	if !slices.Equal(permissions, want) {
		t.Fatalf("seeded permissions %v, policy constants %v", permissions, want)
	}
}
