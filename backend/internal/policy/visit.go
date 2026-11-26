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

// ActionUpdate names the restricted-visit edit; every other visit command is named after the
// permission it requires (PermApprove, PermReject, PermReschedule, PermCancel, PermCheckin,
// PermCheckout).
const ActionUpdate = "update"

// RescheduleWindow is how long after a visit's end an approved visit may still be rescheduled.
const RescheduleWindow = 7 * 24 * time.Hour

// VisitFacts is what the visit command rules need to know about a visit.
type VisitFacts struct {
	Kind      string // "standard" or "restricted"
	OfficerID int32
	VIP       bool
	Status    string
	Start     time.Time
	Duration  int
}

// AllowedActions lists, in a fixed order, the visit commands s may run on v at now (ADR-0003).
// It is the single statement of the rules the decision and reception usecases enforce, so the
// frontend never re-derives them. Attendance still needs a matching credential at the desk.
func AllowedActions(s dto.Session, v VisitFacts, now time.Time) []string {
	actions := []string{}
	if !CanRead(s, v.Kind, v.OfficerID, v.VIP, v.Status) {
		return actions
	}
	decide := func(permission string) bool { return CanDecide(s, v.Kind, v.OfficerID, v.VIP, permission) }
	pending := v.Status == "pending"
	open := pending || v.Status == "approved"
	end := v.Start.Add(time.Duration(v.Duration) * time.Minute)
	if pending && decide(PermApprove) {
		actions = append(actions, PermApprove)
	}
	if pending && decide(PermReject) {
		actions = append(actions, PermReject)
	}
	if v.Status == "approved" && decide(PermReschedule) && !now.Truncate(time.Millisecond).After(end.Add(RescheduleWindow)) {
		actions = append(actions, PermReschedule)
	}
	if open && decide(PermCancel) {
		actions = append(actions, PermCancel)
	}
	if open && v.Kind == "restricted" && s.Role == RoleProtocol && Has(s, PermCreate) {
		actions = append(actions, ActionUpdate)
	}
	timing := VisitTiming(v.Status, v.Start, v.Duration, now)
	if timing.CanCheckIn && CanAttend(s, PermCheckin) {
		actions = append(actions, PermCheckin)
	}
	if timing.CanCheckOut && CanAttend(s, PermCheckout) {
		actions = append(actions, PermCheckout)
	}
	return actions
}
