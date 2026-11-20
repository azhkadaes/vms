package employeeusecaseimplementation_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func yes() *bool { v := true; return &v }
func no() *bool  { v := false; return &v }

const initialPassword = "initial-synthetic-passphrase"

// validCreate is a PA assigned to the active standard official 3.
func validCreate() dto.EmployeeCreate {
	return dto.EmployeeCreate{Name: "Petugas Contoh", Email: "staff@example.test", Roles: []string{policy.RolePA}, Active: yes(),
		AssignedOfficerIDs: []string{"3"}, InitialPassword: initialPassword, RequirePasswordChange: yes()}
}

// officials are the officials every create and update test can choose from.
func withOfficials(s *store) *store {
	s.officers[3] = entity.EmployeeOfficer{ID: 3, Active: true}
	s.officers[4] = entity.EmployeeOfficer{ID: 4, Active: true}
	s.officers[1] = entity.EmployeeOfficer{ID: 1, Active: true, VIP: true}
	s.officers[2] = entity.EmployeeOfficer{ID: 2}
	return s
}

func TestCreatingAnEmployeeRefusesSessionsWithoutPeopleBeforeHashingOrWriting(t *testing.T) {
	for _, tc := range withoutPeople {
		t.Run(tc.name, func(t *testing.T) {
			s, h := withOfficials(newStore()), &hasher{}
			_, err := newUsecase(s, h).CreateEmployee(context.Background(), tc.session, adminProof, validCreate())
			failed(t, err, 403, "forbidden")
			if len(s.calls) != 0 || len(h.hashed) != 0 {
				t.Fatalf("a refused session reached the repository %v or hashed %v", s.calls, h.hashed)
			}
		})
	}
}

func TestCreatingAnEmployeeRejectsInvalidFieldsBeforeHashingOrWriting(t *testing.T) {
	long := strings.Repeat("a", 256)
	for _, tc := range []struct {
		name   string
		mutate func(*dto.EmployeeCreate)
		keys   []string
	}{
		{"missing name", func(r *dto.EmployeeCreate) { r.Name = "  " }, []string{"name"}},
		{"one-character name", func(r *dto.EmployeeCreate) { r.Name = " A " }, []string{"name"}},
		{"name over 255 characters", func(r *dto.EmployeeCreate) { r.Name = long }, []string{"name"}},
		{"invalid email", func(r *dto.EmployeeCreate) { r.Email = "Staff <staff@example.test>" }, []string{"email"}},
		{"missing roles", func(r *dto.EmployeeCreate) { r.Roles, r.AssignedOfficerIDs = nil, nil }, []string{"roles"}},
		{"empty roles", func(r *dto.EmployeeCreate) { r.Roles, r.AssignedOfficerIDs = []string{}, nil }, []string{"roles"}},
		{"unknown role", func(r *dto.EmployeeCreate) { r.Roles = []string{policy.RolePA, "Admin"} }, []string{"roles"}},
		{"role twice", func(r *dto.EmployeeCreate) { r.Roles = []string{policy.RolePA, policy.RolePA} }, []string{"roles"}},
		{"missing active", func(r *dto.EmployeeCreate) { r.Active = nil }, []string{"active"}},
		{"missing forced change", func(r *dto.EmployeeCreate) { r.RequirePasswordChange = nil }, []string{"requirePasswordChange"}},
		{"short initial password", func(r *dto.EmployeeCreate) { r.InitialPassword = strings.Repeat("p", 14) }, []string{"initialPassword"}},
		{"initial password over 72 bytes", func(r *dto.EmployeeCreate) { r.InitialPassword = strings.Repeat("p", 73) }, []string{"initialPassword"}},
		{"malformed official ID", func(r *dto.EmployeeCreate) { r.AssignedOfficerIDs = []string{"03"} }, []string{"assignedOfficerIds"}},
		{"job title over 255", func(r *dto.EmployeeCreate) { r.JobTitle = long }, []string{"jobTitle"}},
		{"deputy over 255", func(r *dto.EmployeeCreate) { r.Deputy = long }, []string{"deputy"}},
		{"directorate over 255", func(r *dto.EmployeeCreate) { r.Directorate = long }, []string{"directorate"}},
		{"phone over 255", func(r *dto.EmployeeCreate) { r.Phone = long }, []string{"phone"}},
		{"NIP over 255", func(r *dto.EmployeeCreate) { r.NIP = long }, []string{"nip"}},
		{"several at once", func(r *dto.EmployeeCreate) { r.Name, r.Email, r.InitialPassword = "", "", "" }, []string{"email", "initialPassword", "name"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, h := withOfficials(newStore()), &hasher{}
			req := validCreate()
			tc.mutate(&req)
			_, err := newUsecase(s, h).CreateEmployee(context.Background(), superAdmin, adminProof, req)
			if keys := fieldKeys(t, err); !slices.Equal(keys, tc.keys) {
				t.Fatalf("field errors on %v, want %v", keys, tc.keys)
			}
			if len(s.calls) != 0 || len(h.hashed) != 0 {
				t.Fatalf("an invalid request reached the repository %v or hashed %v", s.calls, h.hashed)
			}
		})
	}
}

func TestCreatingAnEmployeeStoresTheProfileRolesAssignmentAndPasswordVerifier(t *testing.T) {
	s, h := withOfficials(newStore()), &hasher{}
	req := dto.EmployeeCreate{Name: "  Petugas Contoh ", Email: " Staff@Example.TEST", Roles: []string{policy.RoleSuperAdmin, policy.RolePA}, JobTitle: " Asisten ",
		Deputy: "Kedeputian Contoh", Directorate: "Direktorat Contoh", Phone: "+62 811 0000", NIP: "198001012000011001", Active: yes(),
		AssignedOfficerIDs: []string{"3"}, InitialPassword: initialPassword, RequirePasswordChange: yes()}
	out, err := newUsecase(s, h).CreateEmployee(context.Background(), superAdmin, adminProof, req)
	if err != nil {
		t.Fatal(err)
	}
	want := entity.Employee{ID: 101, Name: "Petugas Contoh", Email: "staff@example.test", JobTitle: "Asisten", Deputy: "Kedeputian Contoh", Directorate: "Direktorat Contoh",
		Phone: "+62 811 0000", NIP: "198001012000011001", Active: true, MustChangePassword: true, Version: 1,
		Roles: []string{policy.RolePA, policy.RoleSuperAdmin}, OfficerIDs: []int32{3}}
	if !reflect.DeepEqual(s.employees[101], want) {
		t.Fatalf("stored %+v, want %+v", s.employees[101], want)
	}
	if !slices.Equal(h.hashed, []string{initialPassword}) || s.hashes[101] != "bcrypt:hashed-"+initialPassword {
		t.Fatalf("hashed %v, stored verifier %q", h.hashed, s.hashes[101])
	}
	wantOut := dto.Employee{ID: "101", Name: "Petugas Contoh", Email: "staff@example.test", Roles: []string{"PA", "Super Admin"}, JobTitle: "Asisten", Deputy: "Kedeputian Contoh",
		Directorate: "Direktorat Contoh", Phone: "+62 811 0000", NIP: "198001012000011001", Active: true, AssignedOfficerIDs: []string{"3"}, Version: 1}
	if !reflect.DeepEqual(out, wantOut) {
		t.Fatalf("response %+v, want %+v", out, wantOut)
	}
	if len(s.audits) != 1 || s.audits[0].Actor != admin || s.audits[0].Action != "employee.created" || s.audits[0].Before != nil ||
		!reflect.DeepEqual(s.audits[0].After, want) || !s.audits[0].At.Equal(serverTime) {
		t.Fatalf("audit %+v", s.audits)
	}
	if len(s.ended) != 0 || len(s.reloaded) != 0 {
		t.Fatalf("a new employee's sessions were touched: ended %v reloaded %v", s.ended, s.reloaded)
	}
}

func TestCreatingAnEmployeeWithoutOptionalFieldsOrAssignment(t *testing.T) {
	s := withOfficials(newStore())
	req := dto.EmployeeCreate{Name: "Resepsionis Contoh", Email: "front@example.test", Roles: []string{policy.RoleReceptionist}, Active: no(),
		InitialPassword: initialPassword, RequirePasswordChange: no()}
	out, err := newUsecase(s, &hasher{}).CreateEmployee(context.Background(), superAdmin, adminProof, req)
	if err != nil {
		t.Fatal(err)
	}
	want := dto.Employee{ID: "101", Name: "Resepsionis Contoh", Email: "front@example.test", Roles: []string{"Receptionist"}, Active: false, AssignedOfficerIDs: []string{}, Version: 1}
	if !reflect.DeepEqual(out, want) || s.employees[101].MustChangePassword || slices.Contains(s.calls, "Officers") {
		t.Fatalf("response %+v stored %+v calls %v", out, s.employees[101], s.calls)
	}
}

func TestCreatingAnEmployeeRefusesStateRulesWithoutWriting(t *testing.T) {
	existing := entity.Employee{ID: 8, Name: "Existing Staff", Email: "staff@example.test", Active: true, Version: 1, Roles: []string{policy.RoleReceptionist}}
	for _, tc := range []struct {
		name       string
		mutate     func(*dto.EmployeeCreate)
		emailTaken bool
		keys       []string
	}{
		{"an email already used", func(r *dto.EmployeeCreate) { r.Email = "STAFF@example.test" }, false, []string{"email"}},
		{"an email taken concurrently", func(r *dto.EmployeeCreate) { r.Email = "new@example.test" }, true, []string{"email"}},
		{"an assignment without the PA role", func(r *dto.EmployeeCreate) {
			r.Email, r.Roles = "new@example.test", []string{policy.RoleProtocol}
		}, false, []string{"assignedOfficerIds"}},
		{"two officials", func(r *dto.EmployeeCreate) { r.Email, r.AssignedOfficerIDs = "new@example.test", []string{"3", "4"} }, false, []string{"assignedOfficerIds"}},
		{"a VIP official", func(r *dto.EmployeeCreate) { r.Email, r.AssignedOfficerIDs = "new@example.test", []string{"1"} }, false, []string{"assignedOfficerIds"}},
		{"an inactive official", func(r *dto.EmployeeCreate) { r.Email, r.AssignedOfficerIDs = "new@example.test", []string{"2"} }, false, []string{"assignedOfficerIds"}},
		{"an unknown official", func(r *dto.EmployeeCreate) { r.Email, r.AssignedOfficerIDs = "new@example.test", []string{"99"} }, false, []string{"assignedOfficerIds"}},
		{"a used email and a VIP official", func(r *dto.EmployeeCreate) { r.AssignedOfficerIDs = []string{"1"} }, false, []string{"assignedOfficerIds", "email"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := withOfficials(newStore(existing))
			s.emailTaken = tc.emailTaken
			req := validCreate()
			tc.mutate(&req)
			_, err := newUsecase(s, &hasher{}).CreateEmployee(context.Background(), superAdmin, adminProof, req)
			if keys := fieldKeys(t, err); !slices.Equal(keys, tc.keys) {
				t.Fatalf("field errors on %v, want %v", keys, tc.keys)
			}
			if len(s.employees) != 1 || len(s.audits) != 0 {
				t.Fatalf("a refused create committed: %v", s.calls)
			}
		})
	}
}

func TestCreatingAnEmployeeRechecksTheCallersAuthorityBeforeWriting(t *testing.T) {
	signedOut := liveAuthority(policy.RoleSuperAdmin, policy.PermPeople)
	signedOut.AccountActive = false
	staleRole := liveAuthority(policy.RoleSuperAdmin, policy.PermPeople)
	staleRole.RoleRevision = 2
	for _, tc := range []struct {
		name      string
		authority entity.SessionSnapshot
		check     func(*testing.T, error)
	}{
		{"people withdrawn since the request began", liveAuthority(policy.RoleSuperAdmin, policy.PermRoles), func(t *testing.T, err error) { failed(t, err, 403, "forbidden") }},
		{"account deactivated since", signedOut, func(t *testing.T, err error) {
			if !errors.Is(err, authusecase.ErrUnauthenticated) {
				t.Fatalf("want unauthenticated, got %v", err)
			}
		}},
		{"role revision moved on", staleRole, func(t *testing.T, err error) {
			if !errors.Is(err, authusecase.ErrRoleChanged) {
				t.Fatalf("want role_changed, got %v", err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := withOfficials(newStore())
			s.authority = tc.authority
			_, err := newUsecase(s, &hasher{}).CreateEmployee(context.Background(), superAdmin, adminProof, validCreate())
			tc.check(t, err)
			if !slices.Equal(s.calls, []string{"Transaction", "LockAuthority"}) {
				t.Fatalf("calls after a failed recheck: %v", s.calls)
			}
		})
	}
}
