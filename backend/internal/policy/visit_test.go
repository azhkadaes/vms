package policy

import (
	"testing"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func TestProtocolVisitScope(t *testing.T) {
	protocol := dto.Session{
		Active: true, Role: "Protocol Officer", AssignedOfficerIDs: []string{"2"},
		Permissions: []string{"bookings", "create", "approve", "reject", "reschedule", "cancel", "checkin", "checkout"},
	}
	for _, visit := range []struct {
		name, kind  string
		vip, manage bool
	}{
		{"general non-VIP", "standard", false, false},
		{"VIP", "standard", true, true},
		{"restricted non-VIP", "restricted", false, true},
		{"restricted VIP", "restricted", true, true},
	} {
		t.Run(visit.name, func(t *testing.T) {
			for _, status := range []string{"pending", "approved", "rejected", "cancelled", "checked_in", "checked_out"} {
				if !CanRead(protocol, visit.kind, 2, visit.vip, status) {
					t.Errorf("Protocol must retain detail access for %s", status)
				}
			}
			if got := CanCreate(protocol, visit.kind, 2, visit.vip); got != visit.manage {
				t.Errorf("CanCreate = %v, want %v", got, visit.manage)
			}
			for _, action := range []string{"approve", "reject", "reschedule", "cancel"} {
				if got := CanDecide(protocol, visit.kind, 2, visit.vip, action); got != visit.manage {
					t.Errorf("CanDecide(%s) = %v, want %v", action, got, visit.manage)
				}
			}
		})
	}
	for _, action := range []string{"checkin", "checkout"} {
		if CanAttend(protocol, action) {
			t.Errorf("Protocol cannot %s even with an extra permission", action)
		}
		reception := protocol
		reception.Role = "Receptionist"
		if !CanAttend(reception, action) {
			t.Errorf("Receptionist must retain %s", action)
		}
	}
}

func TestVisitCreationRetainsPAAndSessionRestrictions(t *testing.T) {
	pa := dto.Session{Active: true, Role: "PA", Permissions: []string{"create"}, AssignedOfficerIDs: []string{"2"}}
	if !CanCreate(pa, "standard", 2, false) {
		t.Fatal("assigned PA must be able to create an ordinary non-VIP visit")
	}
	if CanCreate(pa, "standard", 3, false) || CanCreate(pa, "standard", 2, true) || CanCreate(pa, "restricted", 2, false) {
		t.Fatal("PA creation exceeded its assigned ordinary non-VIP scope")
	}
	protocol := dto.Session{Active: true, Role: "Protocol Officer", Permissions: []string{"create"}}
	for _, session := range []dto.Session{
		{Role: protocol.Role, Permissions: protocol.Permissions},
		{Active: true, Role: protocol.Role},
		{Active: true, Role: protocol.Role, Permissions: protocol.Permissions, MustChangePassword: true},
		{Active: true, Role: protocol.Role, Permissions: protocol.Permissions, RoleSelectionRequired: true},
	} {
		if CanCreate(session, "restricted", 0, false) || CanCreate(session, "standard", 1, true) {
			t.Fatal("creation must require an active authorized session")
		}
	}
	if CanCreate(protocol, "unknown", 1, true) {
		t.Fatal("unknown visit kinds must not grant creation")
	}
}
