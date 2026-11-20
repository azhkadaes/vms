package policy

import "slices"

// permissionCatalogue is the A-17 catalogue in vms.permission seed order.
var permissionCatalogue = []string{PermBookings, PermCreate, PermApprove, PermReject, PermReschedule, PermCancel,
	PermCheckin, PermCheckout, PermReports, PermReportExport, PermPeople, PermRoles, PermIntegrations, PermAudit}

// roleDefinition is the display text and the A-17 assignable set of one fixed role.
type roleDefinition struct {
	description string
	assignable  []string
}

// roleDefinitions follow the frontend role defaults (domain.ts sessionFor). Super Admin's
// role administration is left out of its assignable set: it is a fixed permission, because
// withdrawing it would lock every administrator out of this screen for good.
var roleDefinitions = map[string]roleDefinition{
	RolePA: {"Asisten pejabat. Mengelola kunjungan reguler pejabat yang ditugaskan.",
		[]string{PermBookings, PermCreate, PermApprove, PermReject, PermReschedule, PermCancel, PermReports, PermReportExport}},
	RoleProtocol: {"Petugas protokol. Memutuskan kunjungan VIP dan kunjungan terbatas.",
		[]string{PermBookings, PermCreate, PermApprove, PermReject, PermReschedule, PermCancel, PermReports, PermReportExport}},
	RoleReceptionist: {"Resepsionis. Mencatat kedatangan dan kepulangan tamu.",
		[]string{PermBookings, PermCheckin, PermCheckout}},
	RoleSuperAdmin: {"Administrator. Mengelola pegawai, akses peran, dan laporan organisasi.",
		[]string{PermBookings, PermReports, PermReportExport, PermPeople, PermIntegrations, PermAudit}},
}

// PermissionCatalogue returns a copy of the 14 permission keys in seed order.
func PermissionCatalogue() []string { return slices.Clone(permissionCatalogue) }

// KnownPermission reports whether key is one of the 14 catalogue keys, spelled exactly.
func KnownPermission(key string) bool { return slices.Contains(permissionCatalogue, key) }

// AssignablePermissions returns a copy of the permissions a Super Admin may grant to or withdraw
// from role, in catalogue order; empty, never nil, for anything but the four fixed roles.
func AssignablePermissions(role string) []string {
	return append([]string{}, roleDefinitions[role].assignable...)
}

// RoleDescription is the fixed display text of role; empty for anything but the four fixed roles.
func RoleDescription(role string) string { return roleDefinitions[role].description }

// RoleEditable reports whether role's permissions can be changed: it needs an active policy to
// hold them and at least one assignable permission.
func RoleEditable(role string, activePolicy bool) bool {
	return activePolicy && len(roleDefinitions[role].assignable) > 0
}

// KeepsFixedPermissions reports whether requested holds exactly the permissions of current that
// are not assignable to role. Only assignable permissions may be granted or withdrawn.
func KeepsFixedPermissions(role string, current, requested []string) bool {
	assignable := roleDefinitions[role].assignable
	fixed := func(values []string) []string {
		out := []string{}
		for _, value := range values {
			if !slices.Contains(assignable, value) && !slices.Contains(out, value) {
				out = append(out, value)
			}
		}
		slices.Sort(out)
		return out
	}
	return slices.Equal(fixed(current), fixed(requested))
}
