package policy

import (
	"slices"
	"testing"
)

func TestTheCatalogueIsTheFourteenSeededPermissionKeysInSeedOrder(t *testing.T) {
	want := []string{"bookings", "create", "approve", "reject", "reschedule", "cancel", "checkin", "checkout",
		"reports", "reportExport", "people", "roles", "integrations", "audit"}
	if !slices.Equal(PermissionCatalogue(), want) {
		t.Fatalf("catalogue = %v, want %v", PermissionCatalogue(), want)
	}
	for _, key := range want {
		if !KnownPermission(key) {
			t.Errorf("%s is not known", key)
		}
	}
	for _, key := range []string{"", "Bookings", "admin", "roles ", "reportexport"} {
		if KnownPermission(key) {
			t.Errorf("%q is known", key)
		}
	}
}

// A-17: each role may be given or denied its frontend default permissions. Super Admin keeps
// role administration as a fixed permission, so no save can lock every admin out of this screen.
func TestEachRoleMayBeAssignedItsDefaultPermissionsOnly(t *testing.T) {
	for _, tc := range []struct {
		role string
		want []string
	}{
		{RolePA, []string{"bookings", "create", "approve", "reject", "reschedule", "cancel", "reports", "reportExport"}},
		{RoleProtocol, []string{"bookings", "create", "approve", "reject", "reschedule", "cancel", "reports", "reportExport"}},
		{RoleReceptionist, []string{"bookings", "checkin", "checkout"}},
		{RoleSuperAdmin, []string{"bookings", "reports", "reportExport", "people", "integrations", "audit"}},
		{"Visitor", []string{}},
	} {
		t.Run(tc.role, func(t *testing.T) {
			got := AssignablePermissions(tc.role)
			if got == nil || !slices.Equal(got, tc.want) {
				t.Fatalf("assignable = %v, want %v", got, tc.want)
			}
			got = append(got, "mutated")
			if slices.Contains(AssignablePermissions(tc.role), "mutated") {
				t.Fatal("callers can change the shared assignable list")
			}
		})
	}
}

func TestEveryFixedRoleIsDescribedAndEditableOnlyThroughAnActivePolicy(t *testing.T) {
	for _, role := range []string{RolePA, RoleProtocol, RoleReceptionist, RoleSuperAdmin} {
		if RoleDescription(role) == "" {
			t.Errorf("%s has no description", role)
		}
		if !RoleEditable(role, true) {
			t.Errorf("%s with an active policy is not editable", role)
		}
		if RoleEditable(role, false) {
			t.Errorf("%s without an active policy is editable", role)
		}
	}
	if RoleEditable("Visitor", true) || RoleDescription("Visitor") != "" {
		t.Error("an unknown role is described or editable")
	}
}

func reversed(values []string) []string {
	out := slices.Clone(values)
	slices.Reverse(out)
	return out
}

func TestAChangeMustLeaveTheFixedPermissionSubsetAsItWas(t *testing.T) {
	superAdmin := []string{"bookings", "people", "roles", "integrations", "reports", "reportExport", "audit"}
	for _, tc := range []struct {
		name               string
		role               string
		current, requested []string
		want               bool
	}{
		{"withdraw an assignable permission", RolePA, []string{"bookings", "create", "approve"}, []string{"bookings", "approve"}, true},
		{"grant an assignable permission", RoleReceptionist, []string{"bookings"}, []string{"bookings", "checkin"}, true},
		{"withdraw every assignable permission", RolePA, []string{"bookings", "create"}, []string{}, true},
		{"order does not matter", RoleSuperAdmin, superAdmin, reversed(superAdmin), true},
		{"grant a permission the role may not hold", RoleReceptionist, []string{"bookings"}, []string{"bookings", "approve"}, false},
		{"grant role administration to a PA", RolePA, []string{"bookings"}, []string{"bookings", "roles"}, false},
		{"withdraw the fixed role administration from Super Admin", RoleSuperAdmin, superAdmin, []string{"bookings", "people", "integrations", "reports", "reportExport", "audit"}, false},
		{"keep a fixed permission while changing the rest", RoleSuperAdmin, superAdmin, []string{"roles"}, true},
		{"keep a non-assignable permission the role already holds", RoleReceptionist, []string{"bookings", "approve"}, []string{"approve"}, true},
		{"withdraw a non-assignable permission the role already holds", RoleReceptionist, []string{"bookings", "approve"}, []string{"bookings"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := KeepsFixedPermissions(tc.role, tc.current, tc.requested); got != tc.want {
				t.Fatalf("keeps fixed = %v, want %v", got, tc.want)
			}
		})
	}
}
