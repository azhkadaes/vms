package policy

import (
	"slices"
	"strings"
	"testing"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func TestOnlyAReadySessionHoldingPeopleAdministersEmployees(t *testing.T) {
	for _, tc := range []struct {
		name    string
		session dto.Session
		want    bool
	}{
		{"super admin with people", dto.Session{Active: true, Role: RoleSuperAdmin, Permissions: []string{PermPeople, PermRoles}}, true},
		// Permission, not role, decides.
		{"PA granted people", dto.Session{Active: true, Role: RolePA, Permissions: []string{PermBookings, PermPeople}}, true},
		{"super admin without people", dto.Session{Active: true, Role: RoleSuperAdmin, Permissions: []string{PermRoles, PermReports}}, false},
		{"role selection pending", dto.Session{Active: true, RoleSelectionRequired: true, Permissions: []string{PermPeople}}, false},
		{"password change pending", dto.Session{Active: true, Role: RoleSuperAdmin, MustChangePassword: true, Permissions: []string{PermPeople}}, false},
		{"signed out", dto.Session{Role: RoleSuperAdmin, Permissions: []string{PermPeople}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanAdministerEmployees(tc.session); got != tc.want {
				t.Fatalf("administers = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestThePasswordPolicyIsFifteenCodePointsAndSeventyTwoBytes(t *testing.T) {
	if PasswordMinLength != 15 || PasswordMaxBytes != 72 {
		t.Fatalf("policy = %d code points / %d bytes, want 15 / 72", PasswordMinLength, PasswordMaxBytes)
	}
	for _, tc := range []struct {
		name     string
		password string
		want     bool
	}{
		{"fifteen ASCII characters", strings.Repeat("a", 15), true},
		{"fourteen ASCII characters", strings.Repeat("a", 14), false},
		// Code points, not bytes, count towards the minimum: 15 two-byte characters are 30 bytes.
		{"fifteen multi-byte characters", strings.Repeat("ā", 15), true},
		{"seventy-two bytes", strings.Repeat("a", 72), true},
		{"seventy-three bytes", strings.Repeat("a", 73), false},
		{"twenty-five three-byte characters are 75 bytes", strings.Repeat("あ", 25), false},
		{"invalid UTF-8", strings.Repeat("a", 15) + "\xff", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := AcceptablePassword(tc.password); got != tc.want {
				t.Fatalf("acceptable = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOnlyTheFourSeededRolesAreKnown(t *testing.T) {
	for _, role := range []string{"PA", "Protocol Officer", "Receptionist", "Super Admin"} {
		if !KnownRole(role) {
			t.Errorf("%q is not known", role)
		}
	}
	for _, role := range []string{"", "pa", "Admin", "Super Admin ", "Visitor"} {
		if KnownRole(role) {
			t.Errorf("%q is known", role)
		}
	}
}

func TestAnAdministratorCannotChangeTheirOwnRolesOrActiveFlag(t *testing.T) {
	before := EmployeeAccess{Active: true, Roles: []string{RolePA, RoleSuperAdmin}, OfficerIDs: []int32{3}}
	for _, tc := range []struct {
		name          string
		actor, target int32
		after         EmployeeAccess
		want          bool
	}{
		{"own record unchanged", 7, 7, before, true},
		{"own roles in another order", 7, 7, EmployeeAccess{Active: true, Roles: []string{RoleSuperAdmin, RolePA}, OfficerIDs: []int32{3}}, true},
		// The assignment is not access the administrator holds over themselves.
		{"own assignment changed", 7, 7, EmployeeAccess{Active: true, Roles: []string{RolePA, RoleSuperAdmin}}, true},
		{"own role removed", 7, 7, EmployeeAccess{Active: true, Roles: []string{RoleSuperAdmin}}, false},
		{"own role added", 7, 7, EmployeeAccess{Active: true, Roles: []string{RolePA, RoleProtocol, RoleSuperAdmin}, OfficerIDs: []int32{3}}, false},
		{"own deactivation", 7, 7, EmployeeAccess{Active: false, Roles: []string{RolePA, RoleSuperAdmin}, OfficerIDs: []int32{3}}, false},
		{"another employee's roles and flag", 7, 8, EmployeeAccess{Active: false, Roles: []string{RoleReceptionist}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := MayChangeAccess(tc.actor, tc.target, before, tc.after); got != tc.want {
				t.Fatalf("may change = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTheLastActiveSuperAdminKeepsTheRoleAndStaysActive(t *testing.T) {
	admin := EmployeeAccess{Active: true, Roles: []string{RoleSuperAdmin}}
	for _, tc := range []struct {
		name          string
		before, after EmployeeAccess
		others        int
		want          bool
	}{
		{"deactivating the last one", admin, EmployeeAccess{Active: false, Roles: []string{RoleSuperAdmin}}, 0, true},
		{"removing the role from the last one", admin, EmployeeAccess{Active: true, Roles: []string{RolePA}}, 0, true},
		{"deactivating one of two", admin, EmployeeAccess{Active: false, Roles: []string{RoleSuperAdmin}}, 1, false},
		{"removing the role from one of two", admin, EmployeeAccess{Active: true, Roles: []string{RoleReceptionist}}, 1, false},
		{"editing the last one's other roles", admin, EmployeeAccess{Active: true, Roles: []string{RolePA, RoleSuperAdmin}}, 0, false},
		{"an inactive super admin", EmployeeAccess{Active: false, Roles: []string{RoleSuperAdmin}}, EmployeeAccess{Active: false, Roles: []string{RolePA}}, 0, false},
		{"not a super admin", EmployeeAccess{Active: true, Roles: []string{RolePA}}, EmployeeAccess{Active: false, Roles: []string{RolePA}}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RemovesLastSuperAdmin(tc.before, tc.after, tc.others); got != tc.want {
				t.Fatalf("removes last = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAPAMayBeAssignedOneActiveNonVIPOfficial(t *testing.T) {
	standard := AssignableOfficer{ID: 3, Found: true, Active: true}
	for _, tc := range []struct {
		name      string
		roles     []string
		requested []AssignableOfficer
		current   []int32
		ok        bool
	}{
		{"no assignment", []string{RoleReceptionist}, nil, nil, true},
		{"a PA and one active standard official", []string{RolePA}, []AssignableOfficer{standard}, nil, true},
		{"a PA among several roles", []string{RoleProtocol, RolePA}, []AssignableOfficer{standard}, nil, true},
		{"an assignment without the PA role", []string{RoleProtocol}, []AssignableOfficer{standard}, nil, false},
		{"two officials", []string{RolePA}, []AssignableOfficer{standard, {ID: 4, Found: true, Active: true}}, nil, false},
		{"the same official twice", []string{RolePA}, []AssignableOfficer{standard, standard}, nil, false},
		{"an unknown official", []string{RolePA}, []AssignableOfficer{{ID: 9}}, nil, false},
		{"a VIP official", []string{RolePA}, []AssignableOfficer{{ID: 1, Found: true, Active: true, VIP: true}}, nil, false},
		{"a newly chosen inactive official", []string{RolePA}, []AssignableOfficer{{ID: 2, Found: true}}, []int32{3}, false},
		// The editor offers an inactive official only while the employee still holds it.
		{"keeping an official that became inactive", []string{RolePA}, []AssignableOfficer{{ID: 2, Found: true}}, []int32{2}, true},
		{"keeping an official that became VIP", []string{RolePA}, []AssignableOfficer{{ID: 1, Found: true, Active: true, VIP: true}}, []int32{1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			message := AssignmentProblem(tc.roles, tc.requested, tc.current)
			if (message == "") != tc.ok {
				t.Fatalf("problem = %q, want ok %v", message, tc.ok)
			}
		})
	}
}

func TestAccessChangesEndOrRefreshTheEmployeeSessions(t *testing.T) {
	pa := EmployeeAccess{Active: true, Roles: []string{RolePA, RoleProtocol}, OfficerIDs: []int32{3}}
	for _, tc := range []struct {
		name  string
		after EmployeeAccess
		want  SessionEffect
	}{
		{"nothing access-bearing changed", EmployeeAccess{Active: true, Roles: []string{RoleProtocol, RolePA}, OfficerIDs: []int32{3}}, SessionEffect{}},
		{"deactivation", EmployeeAccess{Active: false, Roles: []string{RolePA, RoleProtocol}, OfficerIDs: []int32{3}}, SessionEffect{End: true}},
		{"role removal", EmployeeAccess{Active: true, Roles: []string{RolePA}, OfficerIDs: []int32{3}}, SessionEffect{RoleRevision: true, RemovedRoles: []string{RoleProtocol}}},
		{"role added", EmployeeAccess{Active: true, Roles: []string{RolePA, RoleProtocol, RoleReceptionist}, OfficerIDs: []int32{3}}, SessionEffect{RoleRevision: true, RemovedRoles: []string{}}},
		{"assignment changed", EmployeeAccess{Active: true, Roles: []string{RolePA, RoleProtocol}, OfficerIDs: []int32{4}}, SessionEffect{RoleRevision: true, RemovedRoles: []string{}}},
		{"assignment cleared", EmployeeAccess{Active: true, Roles: []string{RolePA, RoleProtocol}}, SessionEffect{RoleRevision: true, RemovedRoles: []string{}}},
		{"deactivation with a role removal ends the sessions", EmployeeAccess{Active: false, Roles: []string{RoleProtocol}}, SessionEffect{End: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := EmployeeSessionEffect(pa, tc.after)
			if got.End != tc.want.End || got.RoleRevision != tc.want.RoleRevision || !slices.Equal(got.RemovedRoles, tc.want.RemovedRoles) {
				t.Fatalf("effect = %+v, want %+v", got, tc.want)
			}
		})
	}
	reactivated := EmployeeSessionEffect(EmployeeAccess{Roles: []string{RolePA}}, EmployeeAccess{Active: true, Roles: []string{RolePA}})
	if reactivated.End || reactivated.RoleRevision {
		t.Fatalf("reactivation touched sessions: %+v", reactivated)
	}
}
