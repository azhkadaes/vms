package policy

import (
	"slices"
	"testing"
	"time"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func TestAllowedActions(t *testing.T) {
	start := time.Date(2099, 8, 19, 10, 0, 0, 0, time.UTC)
	staff := []string{PermBookings, PermCreate, PermApprove, PermReject, PermReschedule, PermCancel}
	pa := dto.Session{Active: true, Role: RolePA, AssignedOfficerIDs: []string{"2"}, Permissions: staff}
	protocol := dto.Session{Active: true, Role: RoleProtocol, Permissions: staff}
	reception := dto.Session{Active: true, Role: RoleReceptionist, Permissions: []string{PermBookings, PermCheckin, PermCheckout}}
	visit := func(kind, status string, officer int32, vip bool) VisitFacts {
		return VisitFacts{Kind: kind, OfficerID: officer, VIP: vip, Status: status, Start: start, Duration: 60}
	}
	before, during := start.Add(-time.Hour), start.Add(10*time.Minute)
	windowEnd := start.Add(time.Hour + RescheduleWindow)
	for _, tc := range []struct {
		name    string
		session dto.Session
		visit   VisitFacts
		now     time.Time
		want    []string
	}{
		{"assigned PA decides a pending visit", pa, visit("standard", "pending", 2, false), before,
			[]string{PermApprove, PermReject, PermCancel}},
		{"assigned PA manages an approved visit", pa, visit("standard", "approved", 2, false), before,
			[]string{PermReschedule, PermCancel}},
		{"unassigned PA sees nothing", pa, visit("standard", "pending", 3, false), before, []string{}},
		{"PA never decides VIP visits", pa, visit("standard", "pending", 2, true), before, []string{}},
		{"missed visit reschedules until the window closes", pa, visit("standard", "approved", 2, false), windowEnd,
			[]string{PermReschedule, PermCancel}},
		{"missed visit cannot reschedule after the window", pa, visit("standard", "approved", 2, false), windowEnd.Add(time.Millisecond),
			[]string{PermCancel}},
		{"Protocol decides and edits restricted visits", protocol, visit("restricted", "pending", 9, false), before,
			[]string{PermApprove, PermReject, PermCancel, ActionUpdate}},
		{"Protocol cannot touch general non-VIP visits", protocol, visit("standard", "pending", 9, false), before, []string{}},
		{"closed visits allow nothing", protocol, visit("restricted", "rejected", 9, false), before, []string{}},
		{"reception checks in inside the arrival window", reception, visit("standard", "approved", 2, false), during,
			[]string{PermCheckin}},
		{"reception cannot check in early", reception, visit("standard", "approved", 2, false), before, []string{}},
		{"reception checks out a visitor inside", reception, visit("standard", "checked_in", 2, false), during,
			[]string{PermCheckout}},
		{"reception cannot read pending visits", reception, visit("standard", "pending", 2, false), during, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := AllowedActions(tc.session, tc.visit, tc.now); !slices.Equal(got, tc.want) {
				t.Fatalf("AllowedActions = %v, want %v", got, tc.want)
			}
		})
	}
}
