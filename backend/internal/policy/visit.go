package policy

import (
	"slices"
	"strconv"
	"time"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func Has(s dto.Session, key string) bool {
	return s.Active && !s.MustChangePassword && !s.RoleSelectionRequired && slices.Contains(s.Permissions, key)
}
func CanRead(s dto.Session, kind string, officerID int32, vip bool, status string) bool {
	if !Has(s, "bookings") {
		return false
	}
	switch s.Role {
	case "Protocol Officer":
		return true
	case "Super Admin":
		return kind == "restricted"
	case "Receptionist":
		return status == "approved" || status == "checked_in" || status == "checked_out"
	case "PA":
		return slices.Contains(s.AssignedOfficerIDs, strconv.FormatInt(int64(officerID), 10)) && (kind == "restricted" || !vip)
	}
	return false
}
func CanDecide(s dto.Session, kind string, officerID int32, vip bool, permission string) bool {
	if !Has(s, permission) {
		return false
	}
	if s.Role == "Protocol Officer" {
		return kind == "restricted" || vip
	}
	return s.Role == "PA" && kind == "standard" && !vip && slices.Contains(s.AssignedOfficerIDs, strconv.FormatInt(int64(officerID), 10))
}
func CanCreate(s dto.Session, kind string, officerID int32, vip bool) bool {
	if !Has(s, "create") {
		return false
	}
	if s.Role == "Protocol Officer" {
		return kind == "restricted" || (kind == "standard" && vip)
	}
	return kind == "standard" && s.Role == "PA" && !vip && slices.Contains(s.AssignedOfficerIDs, strconv.FormatInt(int64(officerID), 10))
}
func CanAttend(s dto.Session, permission string) bool {
	return Has(s, permission) && s.Role == "Receptionist"
}
func DisplayStatus(status string, start time.Time, duration int, now time.Time) string {
	if status == "approved" && now.After(start.Add(time.Duration(duration)*time.Minute)) {
		return "missed"
	}
	return status
}
