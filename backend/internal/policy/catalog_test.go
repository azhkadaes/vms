package policy

import (
	"testing"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func TestStaffCatalogScopeFollowsTheBookingsAndPeoplePermissions(t *testing.T) {
	ready := func(role string, permissions ...string) dto.Session {
		return dto.Session{Active: true, Role: role, AvailableRoles: []string{role}, Permissions: permissions}
	}
	for _, tc := range []struct {
		name         string
		session      dto.Session
		allowed      bool
		withAssigned bool
	}{
		{"PA with bookings", ready(RolePA, PermBookings, PermCreate), true, false},
		{"Protocol Officer with bookings", ready(RoleProtocol, PermBookings), true, false},
		{"Receptionist with bookings", ready(RoleReceptionist, PermBookings, PermCheckin), true, false},
		{"Super Admin with people only", ready(RoleSuperAdmin, PermPeople), true, true},
		{"bookings and people together", ready(RoleSuperAdmin, PermBookings, PermPeople), true, true},
		// Permission, not role, decides: a PA granted people edits employees too.
		{"PA granted people", ready(RolePA, PermPeople), true, true},
		{"neither permission", ready(RoleReceptionist, PermCheckin, PermCheckout), false, false},
		{"reports alone", ready(RoleSuperAdmin, PermReports, PermRoles), false, false},
		{"no permissions", ready(RolePA), false, false},
		{"role selection pending", dto.Session{Active: true, RoleSelectionRequired: true, AvailableRoles: []string{RolePA}, Permissions: []string{PermBookings, PermPeople}}, false, false},
		{"password change pending", dto.Session{Active: true, Role: RoleSuperAdmin, AvailableRoles: []string{RoleSuperAdmin}, MustChangePassword: true, Permissions: []string{PermPeople}}, false, false},
		{"signed out", dto.Session{Permissions: []string{PermBookings}}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scope, ok := StaffCatalog(tc.session)
			if ok != tc.allowed {
				t.Fatalf("allowed = %v, want %v", ok, tc.allowed)
			}
			if scope.AssignedInactiveOfficers != tc.withAssigned {
				t.Fatalf("assigned inactive officers = %v, want %v", scope.AssignedInactiveOfficers, tc.withAssigned)
			}
		})
	}
}
