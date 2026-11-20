package policy

import (
	"slices"
	"strconv"
	"time"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// Has reports whether a ready session's active role holds the permission key.
func Has(s dto.Session, key string) bool {
	return WorkspaceReady(s) && slices.Contains(s.Permissions, key)
}
func CanRead(s dto.Session, kind string, officerID int32, vip bool, status string) bool {
	if !Has(s, PermBookings) {
		return false
	}
	switch s.Role {
	case RoleProtocol:
		return true
	case RoleSuperAdmin:
		return kind == "restricted"
	case RoleReceptionist:
		return status == "approved" || status == "checked_in" || status == "checked_out"
	case RolePA:
		return slices.Contains(s.AssignedOfficerIDs, strconv.FormatInt(int64(officerID), 10)) && (kind == "restricted" || !vip)
	}
	return false
}
func CanDecide(s dto.Session, kind string, officerID int32, vip bool, permission string) bool {
	if !Has(s, permission) {
		return false
	}
	if s.Role == RoleProtocol {
		return kind == "restricted" || vip
	}
	return s.Role == RolePA && kind == "standard" && !vip && slices.Contains(s.AssignedOfficerIDs, strconv.FormatInt(int64(officerID), 10))
}
func CanCreate(s dto.Session, kind string, officerID int32, vip bool) bool {
	if !Has(s, PermCreate) {
		return false
	}
	if s.Role == RoleProtocol {
		return kind == "restricted" || (kind == "standard" && vip)
	}
	return kind == "standard" && s.Role == RolePA && !vip && slices.Contains(s.AssignedOfficerIDs, strconv.FormatInt(int64(officerID), 10))
}
func CanAttend(s dto.Session, permission string) bool {
	return Has(s, permission) && s.Role == RoleReceptionist
}
func DisplayStatus(status string, start time.Time, duration int, now time.Time) string {
	if status == "approved" && now.After(start.Add(time.Duration(duration)*time.Minute)) {
		return "missed"
	}
	return status
}
