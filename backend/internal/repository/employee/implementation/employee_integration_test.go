//go:build integration

package employeerepositoryimplementation_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authrepo "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	employeerepo "github.com/azhkadaes/vms/backend/internal/repository/employee/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	employeeserializer "github.com/azhkadaes/vms/backend/internal/serializer/employee/implementation"
	authcontract "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	auth "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	employeecontract "github.com/azhkadaes/vms/backend/internal/usecase/employee"
	employee "github.com/azhkadaes/vms/backend/internal/usecase/employee/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

func addSession(t *testing.T, db *gorm.DB, employeeID int32, role string) entity.SessionProof {
	t.Helper()
	sessionID := uuid.NewString()
	err := db.Exec(`INSERT INTO support.staff_sessions
		(id,employee_id,active_role_id,credential_generation,role_revision,refresh_hash,created_at,last_activity_at,idle_expires_at,absolute_expires_at)
		SELECT ?,?,id,1,1,?, transaction_timestamp() - INTERVAL '1 minute', transaction_timestamp() - INTERVAL '1 minute',
			transaction_timestamp() + INTERVAL '1 hour', transaction_timestamp() + INTERVAL '2 hours'
		FROM vms.role WHERE nama = ?`, sessionID, employeeID, fmt.Sprintf("%x", sha256.Sum256([]byte(sessionID))), role).Error
	if err != nil {
		t.Fatalf("seed staff session: %v", err)
	}
	return entity.SessionProof{SessionID: sessionID, EmployeeID: employeeID, CredentialGeneration: 1, RoleRevision: 1}
}

func execFixture(t *testing.T, db *gorm.DB, query string, args ...any) {
	t.Helper()
	if err := db.Exec(query, args...).Error; err != nil {
		t.Fatal(err)
	}
}

func TestEmployeeAdministrationStoresProfilesAndRefreshesLiveAuthority(t *testing.T) {
	ctx := context.Background()
	db := integrationtest.RolledBack(t, integrationtest.Open(t))
	var admin int32
	adminEmail := "employee-admin-" + uuid.NewString() + "@example.test"
	if err := db.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password)
		VALUES ('Employee Administration Test',?,TRUE,FALSE) RETURNING id`, adminEmail).Scan(&admin).Error; err != nil {
		t.Fatal(err)
	}
	execFixture(t, db, "INSERT INTO vms.role_pegawai(id_karyawan,id_role) SELECT ?,id FROM vms.role WHERE nama = ?", admin, policy.RoleSuperAdmin)
	adminProof := addSession(t, db, admin, policy.RoleSuperAdmin)
	reader := auth.NewAuthUsecase(authrepo.NewAuthRepository(db), authserializer.NewAuthSerializer())
	adminSession, err := reader.GetSession(ctx, adminProof)
	if err != nil {
		t.Fatal(err)
	}
	hasher, err := auth.NewCredentialVerifier(4, 1)
	if err != nil {
		t.Fatal(err)
	}
	usecase := employee.NewEmployeeUsecase(employeerepo.NewEmployeeRepository(db), employeeserializer.NewEmployeeSerializer(),
		auth.NewSessionValidator(authserializer.NewAuthSerializer()), hasher, noop.NewTracerProvider().Tracer("employee-test"))
	var officer int32
	if err := db.Raw(`SELECT p.id FROM vms.pejabat p JOIN vms.jabatan j ON p.id_jabatan = j.id
		WHERE p.status_aktif AND NOT j.pejabat_vip ORDER BY p.id LIMIT 1`).Scan(&officer).Error; err != nil || officer == 0 {
		t.Fatalf("missing standard official fixture: %v", err)
	}
	enabled := true
	email := "employee-" + uuid.NewString() + "@example.test"
	const password = "synthetic-initial-password"
	req := dto.EmployeeCreate{
		Name: "Employee Write Fixture", Email: strings.ToUpper(email), Roles: []string{policy.RoleProtocol, policy.RolePA},
		Active: &enabled, RequirePasswordChange: &enabled, InitialPassword: password, AssignedOfficerIDs: []string{strconv.Itoa(int(officer))},
		JobTitle: "Asisten", Deputy: "Deputy", Directorate: "Directorate", Phone: "081234", NIP: "9991",
	}
	var vip int32
	if err := db.Raw(`SELECT p.id FROM vms.pejabat p JOIN vms.jabatan j ON p.id_jabatan = j.id
		WHERE p.status_aktif AND j.pejabat_vip ORDER BY p.id LIMIT 1`).Scan(&vip).Error; err != nil || vip == 0 {
		t.Fatalf("missing VIP official fixture: %v", err)
	}
	forbiddenAssignment := req
	forbiddenAssignment.AssignedOfficerIDs = []string{strconv.Itoa(int(vip))}
	_, err = usecase.CreateEmployee(ctx, adminSession, adminProof, forbiddenAssignment)
	var assignmentError *failure.Error
	if !errors.As(err, &assignmentError) || assignmentError.Status != 422 || len(assignmentError.Fields["assignedOfficerIds"]) == 0 {
		t.Fatalf("VIP official assignment accepted: %v", err)
	}
	created, err := usecase.CreateEmployee(ctx, adminSession, adminProof, req)
	if err != nil || created.Version != 1 || created.Email != email || !slices.Equal(created.Roles, []string{policy.RolePA, policy.RoleProtocol}) {
		t.Fatalf("create employee: %+v %v", created, err)
	}
	id, err := strconv.ParseInt(created.ID, 10, 32)
	if err != nil || id <= 0 {
		t.Fatalf("bad employee id: %s", created.ID)
	}
	employeeID := int32(id)
	detail, err := usecase.GetEmployee(ctx, adminSession, employeeID)
	if err != nil || detail.JobTitle != req.JobTitle || detail.Deputy != req.Deputy || detail.Phone != req.Phone || detail.Directorate != req.Directorate ||
		!slices.Equal(detail.Roles, created.Roles) || !slices.Equal(detail.AssignedOfficerIDs, created.AssignedOfficerIDs) {
		t.Fatalf("read employee: %+v %v", detail, err)
	}
	for _, query := range []string{email, "write fixture", req.NIP} {
		page, err := usecase.GetEmployees(ctx, adminSession, employeecontract.EmployeeQuery{Q: query, PageSize: "1"})
		if err != nil || len(page.Items) != 1 || page.Items[0].ID != created.ID || page.Total != 1 || page.TotalAll < 2 || page.PasswordMinLength != policy.PasswordMinLength {
			t.Fatalf("search %q: %+v %v", query, page, err)
		}
	}
	past, err := usecase.GetEmployees(ctx, adminSession, employeecontract.EmployeeQuery{Q: email, Page: "2"})
	if err != nil || past.Items == nil || len(past.Items) != 0 || past.Total != 1 {
		t.Fatalf("page after matching records: %+v %v", past, err)
	}
	var credential struct {
		EncodedHash        string
		MustChangePassword bool
	}
	if err := db.Raw(`SELECT c.encoded_hash,e.must_change_password FROM support.employee_credentials c
		JOIN vms.pegawai e ON c.employee_id=e.id WHERE c.employee_id=?`, employeeID).Scan(&credential).Error; err != nil {
		t.Fatal(err)
	}
	if ok, err := hasher.Verify(ctx, credential.EncodedHash, password); err != nil || !ok || !credential.MustChangePassword {
		t.Fatal("credential or forced password change missing")
	}
	targetProof := addSession(t, db, employeeID, policy.RolePA)
	if _, err := reader.GetSession(ctx, targetProof); err != nil {
		t.Fatalf("session should be live before update: %v", err)
	}
	updated, err := usecase.UpdateEmployee(ctx, adminSession, adminProof, employeeID, dto.EmployeeUpdate{
		JobTitle: dto.Optional[string]{Set: true}, Roles: []string{policy.RoleProtocol}, AssignedOfficerIDs: []string{}, Version: 1,
	})
	if err != nil || updated.Version != 2 || updated.JobTitle != "" || len(updated.AssignedOfficerIDs) != 0 {
		t.Fatalf("remove PA role and assignment: %+v %v", updated, err)
	}
	if _, err := reader.GetSession(ctx, targetProof); !errors.Is(err, authcontract.ErrRoleChanged) {
		t.Fatalf("old proof still works after role removal: %v", err)
	}
	targetProof.RoleRevision++
	after, err := reader.GetSession(ctx, targetProof)
	if err != nil || !after.RoleSelectionRequired {
		t.Fatalf("session in removed role must choose a current role: %+v %v", after, err)
	}
	_, err = usecase.UpdateEmployee(ctx, adminSession, adminProof, employeeID, dto.EmployeeUpdate{Active: dto.Optional[bool]{Set: true}, Version: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.GetSession(ctx, targetProof); !errors.Is(err, authcontract.ErrUnauthenticated) {
		t.Fatalf("old proof still works after deactivation: %v", err)
	}
	var events []string
	if err := db.Raw(`SELECT jenis_kegiatan FROM vms.audit_log WHERE actor_type='staff' AND id_pegawai=?
		AND entity_type='pegawai' AND entity_id=? ORDER BY id`, admin, employeeID).Scan(&events).Error; err != nil ||
		!slices.Equal(events, []string{"employee.created", "employee.updated", "employee.updated"}) {
		t.Fatalf("employee audit: %v %v", events, err)
	}
}
