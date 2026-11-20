package policy

import (
	"slices"
	"unicode/utf8"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// The staff password policy (ops 28 and 36): bcrypt reads at most 72 bytes, so the maximum is in
// UTF-8 bytes while the minimum counts Unicode code points.
const (
	PasswordMinLength = 15
	PasswordMaxBytes  = 72
)

// AcceptablePassword reports whether password meets the staff password policy.
func AcceptablePassword(password string) bool {
	return utf8.ValidString(password) && utf8.RuneCountInString(password) >= PasswordMinLength && len(password) <= PasswordMaxBytes
}

// fixedRoles are the four roles seeded in vms.role, in seed order.
var fixedRoles = []string{RolePA, RoleProtocol, RoleReceptionist, RoleSuperAdmin}

// KnownRole reports whether name is one of the four fixed roles, spelled exactly.
func KnownRole(name string) bool { return slices.Contains(fixedRoles, name) }

// CanAdministerEmployees grants ops 26-29 to a ready session whose active role holds people.
func CanAdministerEmployees(s dto.Session) bool { return Has(s, PermPeople) }

// EmployeeAccess is the access-bearing part of an employee: the active flag, the assigned roles
// and the officials a PA is assigned to.
type EmployeeAccess struct {
	Active     bool
	Roles      []string
	OfficerIDs []int32
}

// sameMembers compares two lists as sets.
func sameMembers[T comparable](a, b []T) bool {
	for _, value := range a {
		if !slices.Contains(b, value) {
			return false
		}
	}
	for _, value := range b {
		if !slices.Contains(a, value) {
			return false
		}
	}
	return true
}

// MayChangeAccess refuses an administrator changing their own roles or active flag (A-16), so no
// one can lock themselves out. Another employee's access is never refused here.
func MayChangeAccess(actor, target int32, before, after EmployeeAccess) bool {
	return actor != target || (before.Active == after.Active && sameMembers(before.Roles, after.Roles))
}

// RemovesLastSuperAdmin reports whether the change leaves no active Super Admin: before is an
// active Super Admin, after is not, and others active employees hold the role besides this one.
func RemovesLastSuperAdmin(before, after EmployeeAccess, others int) bool {
	held := func(a EmployeeAccess) bool { return a.Active && slices.Contains(a.Roles, RoleSuperAdmin) }
	return others == 0 && held(before) && !held(after)
}

// AssignableOfficer is an official named by a PA assignment, as currently stored; Found is false
// when no such official exists.
type AssignableOfficer struct {
	ID     int32
	Found  bool
	VIP    bool
	Active bool
}

// AssignmentProblem returns the user-facing reason the requested assignment breaks A-16, or ""
// when it is allowed: only a PA, at most one official, which exists and is not VIP. A newly chosen
// official must be active; one the employee already holds may be kept after it became inactive.
func AssignmentProblem(roles []string, requested []AssignableOfficer, current []int32) string {
	switch {
	case len(requested) == 0:
		return ""
	case !slices.Contains(roles, RolePA):
		return "Hanya PA yang dapat ditugaskan ke pejabat."
	case len(requested) > 1:
		return "Pegawai hanya dapat ditugaskan ke satu pejabat."
	}
	officer := requested[0]
	switch {
	case !officer.Found:
		return "Pilih pejabat dari daftar yang tersedia."
	case officer.VIP:
		return "PA tidak dapat ditugaskan ke pejabat VIP."
	case !officer.Active && !slices.Contains(current, officer.ID):
		return "Pejabat yang dipilih tidak aktif."
	}
	return ""
}

// SessionEffect is what an employee change does to that employee's live sessions.
type SessionEffect struct {
	// End revokes the sessions and bumps their credential generation: the next request is 401.
	End bool
	// RoleRevision bumps the sessions' role revision: the next request is 409 role_changed.
	RoleRevision bool
	// RemovedRoles are roles the employee no longer holds; sessions acting in one of them go
	// back to role selection.
	RemovedRoles []string
}

// EmployeeSessionEffect derives the session effect of a change (A-16): deactivation ends the
// sessions; a changed role set or PA assignment makes them reload their authority.
func EmployeeSessionEffect(before, after EmployeeAccess) SessionEffect {
	if before.Active && !after.Active {
		return SessionEffect{End: true}
	}
	if sameMembers(before.Roles, after.Roles) && sameMembers(before.OfficerIDs, after.OfficerIDs) {
		return SessionEffect{}
	}
	removed := []string{}
	for _, role := range before.Roles {
		if !slices.Contains(after.Roles, role) {
			removed = append(removed, role)
		}
	}
	return SessionEffect{RoleRevision: true, RemovedRoles: removed}
}
