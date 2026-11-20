//go:build integration

package authrepositoryimplementation_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	implementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
)

func digest(label string) string {
	sum := sha256.Sum256([]byte(label + uuid.NewString()))
	return hex.EncodeToString(sum[:])
}

type auditRow struct {
	ActorType     string
	IDPegawai     *int32 `gorm:"column:id_pegawai"`
	EntityType    string
	EntityID      int32 `gorm:"column:entity_id"`
	JenisKegiatan string
}

// The test drives the repository calls that login, role switch, refresh, password change
// and logout make, as vms_api, inside one rolled-back transaction.
func TestSessionLifecycleWorksAgainstTheRebuiltSchema(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	email := "integration-" + uuid.NewString() + "@example.test"
	var employee int32
	if err := tx.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password)
		VALUES ('Integration Staff',?,TRUE,TRUE) RETURNING id`, email).Scan(&employee).Error; err != nil || employee <= 0 {
		t.Fatalf("runtime role cannot create an employee: %v", err)
	}
	if err := tx.Exec(`INSERT INTO support.employee_credentials(employee_id,encoded_hash) VALUES (?,'bcrypt:initial')`, employee).Error; err != nil {
		t.Fatalf("runtime role cannot store an initial credential: %v", err)
	}
	if err := tx.Exec(`INSERT INTO vms.role_pegawai(id_karyawan,id_role) SELECT ?, id FROM vms.role WHERE nama IN ('PA','Receptionist')`, employee).Error; err != nil {
		t.Fatalf("runtime role cannot assign roles: %v", err)
	}
	sessions := implementation.NewSessionRepository(tx)
	reader := implementation.NewAuthRepository(tx)

	// Login.
	reservation, retry, err := sessions.ReserveLogin(ctx, digest("ip"), digest("account"))
	if err != nil || retry != 0 || reservation == "" {
		t.Fatalf("login admission failed: %v", err)
	}
	credential, err := sessions.GetCredential(ctx, email)
	if err != nil || credential.EmployeeID != employee || !credential.Active || credential.EncodedHash != "bcrypt:initial" {
		t.Fatalf("credential lookup failed: %+v %v", credential, err)
	}
	sessionID := uuid.NewString()
	firstRefresh := digest("refresh-1")
	var roles map[string]int32
	if err := sessions.Transaction(ctx, employee, func(t authrepository.SessionTransaction) error {
		if _, err := t.Credential(employee); err != nil {
			return err
		}
		if roles, err = t.Roles(employee); err != nil {
			return err
		}
		now := t.Now()
		if err := t.Create(entity.StaffSession{ID: sessionID, EmployeeID: employee, CredentialGeneration: 1, RoleRevision: 1, RefreshHash: firstRefresh,
			CreatedAt: now, LastActivityAt: now, IdleExpiresAt: now.Add(8 * time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour)}); err != nil {
			return err
		}
		return t.Audit(employee, "session.login")
	}); err != nil {
		t.Fatalf("login transaction failed: %v", err)
	}
	if len(roles) != 2 || roles["PA"] == 0 || roles["Receptionist"] == 0 {
		t.Fatalf("roles must come from role_pegawai.id_karyawan, got %v", roles)
	}
	if err := sessions.FinishLogin(ctx, reservation, true); err != nil {
		t.Fatalf("finish login failed: %v", err)
	}
	snapshot, err := reader.GetSessionSnapshot(ctx, sessionID)
	if err != nil || snapshot.EmployeeID != employee || snapshot.ActiveRole != "" || !slices.Equal(snapshot.AvailableRoles, []string{"PA", "Receptionist"}) || len(snapshot.Permissions) != 0 {
		t.Fatalf("role-selection snapshot wrong: %+v %v", snapshot, err)
	}

	// Role switch: permissions come from the role's seeded default policy.
	if err := sessions.Transaction(ctx, employee, func(t authrepository.SessionTransaction) error {
		session, err := t.Session(sessionID)
		if err != nil {
			return err
		}
		pa := roles["PA"]
		session.ActiveRoleID = &pa
		session.RoleRevision++
		if err := t.Save(session); err != nil {
			return err
		}
		return t.Audit(employee, "session.role_changed")
	}); err != nil {
		t.Fatalf("role switch failed: %v", err)
	}
	snapshot, err = reader.GetSessionSnapshot(ctx, sessionID)
	permissions := slices.Sorted(slices.Values(snapshot.Permissions))
	if err != nil || snapshot.ActiveRole != "PA" || snapshot.RoleRevision != 2 ||
		!slices.Equal(permissions, []string{"approve", "bookings", "cancel", "create", "reject", "reportExport", "reports", "reschedule"}) {
		t.Fatalf("PA snapshot wrong: %+v %v", snapshot, err)
	}

	// Refresh with recovery.
	secondRefresh, requestID := digest("refresh-2"), uuid.NewString()
	if id, err := sessions.FindRefresh(ctx, firstRefresh); err != nil || id != sessionID {
		t.Fatalf("refresh lookup failed: %v", err)
	}
	if err := sessions.Transaction(ctx, employee, func(t authrepository.SessionTransaction) error {
		session, err := t.Session(sessionID)
		if err != nil {
			return err
		}
		if _, err := t.Recovery(sessionID, requestID); !errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("unexpected recovery row")
		}
		if err := t.RememberRefresh(sessionID, session.RefreshHash); err != nil {
			return err
		}
		session.RefreshHash = secondRefresh
		if err := t.Save(session); err != nil {
			return err
		}
		return t.SaveRecovery(entity.SessionRecovery{SessionID: sessionID, RequestID: requestID, PredecessorHash: firstRefresh,
			SuccessorHash: secondRefresh, SuccessorCipher: []byte("sealed"), ExpiresAt: t.Now().Add(10 * time.Second)})
	}); err != nil {
		t.Fatalf("refresh rotation failed: %v", err)
	}
	for _, hash := range []string{firstRefresh, secondRefresh} {
		if id, err := sessions.FindRefresh(ctx, hash); err != nil || id != sessionID {
			t.Fatalf("rotated or consumed refresh hash not found: %v", err)
		}
	}
	if err := sessions.Transaction(ctx, employee, func(t authrepository.SessionTransaction) error {
		recovery, err := t.Recovery(sessionID, requestID)
		if err != nil || recovery.SuccessorHash != secondRefresh {
			return errors.New("recovery row not readable")
		}
		return t.ClearRecoveries(sessionID)
	}); err != nil {
		t.Fatalf("refresh recovery failed: %v", err)
	}

	// Password change.
	thirdRefresh := digest("refresh-3")
	if err := sessions.Transaction(ctx, employee, func(t authrepository.SessionTransaction) error {
		session, err := t.Session(sessionID)
		if err != nil {
			return err
		}
		if err := t.SetPassword(employee, "bcrypt:changed", sessionID); err != nil {
			return err
		}
		if err := t.RememberRefresh(sessionID, session.RefreshHash); err != nil {
			return err
		}
		session.CredentialGeneration++
		session.RefreshHash = thirdRefresh
		if err := t.Save(session); err != nil {
			return err
		}
		return t.Audit(employee, "session.password_changed")
	}); err != nil {
		t.Fatalf("password change failed: %v", err)
	}
	if credential, err = sessions.GetCredential(ctx, email); err != nil || credential.EncodedHash != "bcrypt:changed" {
		t.Fatalf("password not stored: %v", err)
	}
	if snapshot, err = reader.GetSessionSnapshot(ctx, sessionID); err != nil || snapshot.MustChangePassword || snapshot.CredentialGeneration != 2 {
		t.Fatalf("password change not reflected: %+v %v", snapshot, err)
	}

	// Logout.
	if err := sessions.Transaction(ctx, employee, func(t authrepository.SessionTransaction) error {
		session, err := t.Session(sessionID)
		if err != nil {
			return err
		}
		now := t.Now()
		session.RevokedAt = &now
		if err := t.Save(session); err != nil {
			return err
		}
		if err := t.ClearRecoveries(sessionID); err != nil {
			return err
		}
		return t.Audit(employee, "session.logout")
	}); err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	if snapshot, err = reader.GetSessionSnapshot(ctx, sessionID); err != nil || snapshot.RevokedAt == nil {
		t.Fatalf("logout not reflected: %v", err)
	}

	var audits []auditRow
	if err := tx.Raw(`SELECT actor_type, id_pegawai, entity_type, entity_id, jenis_kegiatan FROM vms.audit_log
		WHERE entity_type = 'pegawai' AND entity_id = ? ORDER BY id`, employee).Scan(&audits).Error; err != nil {
		t.Fatalf("runtime role cannot read audit events: %v", err)
	}
	var actions []string
	for _, audit := range audits {
		if audit.ActorType != "staff" || audit.IDPegawai == nil || *audit.IDPegawai != employee || audit.EntityType != "pegawai" || audit.EntityID != employee {
			t.Fatalf("session audit must record the staff actor and the employee target: %+v", audit)
		}
		actions = append(actions, audit.JenisKegiatan)
	}
	if !slices.Equal(actions, []string{"session.login", "session.role_changed", "session.password_changed", "session.logout"}) {
		t.Fatalf("session audits missing: %v", actions)
	}
}

func TestDefaultPoliciesGrantEachRoleTheFrontendPermissionSet(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	ctx := context.Background()
	want := map[string][]string{
		"PA":               {"approve", "bookings", "cancel", "create", "reject", "reportExport", "reports", "reschedule"},
		"Protocol Officer": {"approve", "bookings", "cancel", "create", "reject", "reportExport", "reports", "reschedule"},
		"Receptionist":     {"bookings", "checkin", "checkout"},
		"Super Admin":      {"audit", "bookings", "integrations", "people", "reportExport", "reports", "roles"},
	}
	for role, permissions := range want {
		var employee int32
		if err := tx.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password) VALUES ('Policy Staff',?,TRUE,FALSE) RETURNING id`,
			"policy-"+uuid.NewString()+"@example.test").Scan(&employee).Error; err != nil {
			t.Fatal(err)
		}
		var roleID int32
		if err := tx.Raw(`INSERT INTO vms.role_pegawai(id_karyawan,id_role) SELECT ?, id FROM vms.role WHERE nama = ? RETURNING id_role`, employee, role).Scan(&roleID).Error; err != nil || roleID == 0 {
			t.Fatalf("cannot assign %s: %v", role, err)
		}
		sessionID := uuid.NewString()
		if err := implementation.NewSessionRepository(tx).Transaction(ctx, employee, func(t authrepository.SessionTransaction) error {
			now := t.Now()
			return t.Create(entity.StaffSession{ID: sessionID, EmployeeID: employee, ActiveRoleID: &roleID, CredentialGeneration: 1, RoleRevision: 1,
				RefreshHash: digest("policy"), CreatedAt: now, LastActivityAt: now, IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(2 * time.Hour)})
		}); err != nil {
			t.Fatal(err)
		}
		snapshot, err := implementation.NewAuthRepository(tx).GetSessionSnapshot(ctx, sessionID)
		if got := slices.Sorted(slices.Values(snapshot.Permissions)); err != nil || snapshot.ActiveRole != role || !slices.Equal(got, permissions) {
			t.Fatalf("%s permissions = %v, want %v (%v)", role, got, permissions, err)
		}
	}
}
