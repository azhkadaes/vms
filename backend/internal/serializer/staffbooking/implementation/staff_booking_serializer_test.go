package staffbookingserializerimplementation

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func TestRestrictedBookingKeys(t *testing.T) {
	at := time.Date(2026, 11, 19, 4, 0, 0, 0, time.UTC)
	private := "private-example"
	roomID := int32(6)
	visit := entity.StaffVisit{
		Visit: entity.Visit{ID: 123, Kind: "terbatas", Code: "00123456", Date: "2026-11-19", Time: "12:30", Duration: 60,
			Status: "approved", Version: 7, Name: "Restricted Name", Organization: "Organization", OfficerID: 5, OfficerName: "Officer", RoomID: &roomID,
			Email: "private@example.test", Phone: "08123456789", IdentityType: "KTP", Visitors: []string{"Other visitor"},
			NDName: &private, Instructions: &private, Rejection: &private, Cancellation: &private},
		ServiceID: &private, CoordinationNote: &private, InternalRejection: &private, InternalCancellation: &private,
		Rescheduled: true, AttachmentAvailable: true, VisitorPhotoName: &private, VisitorCount: 1,
		CreatorName: "Staff", CreatorRole: policy.RoleProtocol, History: []entity.StaffVisitEvent{{At: at, Message: "Created"}},
	}
	base := []string{"id", "date", "time", "duration", "room", "status", "version", "timing", "visitKind", "rescheduled", "reference", "restrictedView"}
	identity := append(slices.Clone(base), "name", "officerId", "pejabat", "serviceIdLast4")
	assigned := append(slices.Clone(identity), "organization")
	full := append(slices.Clone(assigned), "code", "roomId", "serviceId", "history", "createdBy", "visitorInstructions", "coordinationNote",
		"rejectionReason", "internalRejectionNote", "cancellationReason", "internalCancellationNote")
	for _, tc := range []struct {
		role, view string
		arrival    bool
		keys       []string
	}{
		{policy.RoleReceptionist, "reception", false, base},
		{policy.RoleReceptionist, "arrival", true, identity},
		{policy.RolePA, "assigned-pa", false, assigned},
		{policy.RoleProtocol, "full", false, full},
	} {
		t.Run(tc.view, func(t *testing.T) {
			session := dto.Session{Active: true, Role: tc.role, Permissions: []string{policy.PermBookings, policy.PermReject}}
			booking := Project(visit, session, at, tc.arrival)
			if booking.RestrictedView != tc.view || booking.Reference != "TRB-123" || booking.Version != 7 {
				t.Fatalf("wrong restricted booking: %+v", booking)
			}
			payload, err := json.Marshal(booking)
			var fields map[string]json.RawMessage
			if err != nil || json.Unmarshal(payload, &fields) != nil {
				t.Fatalf("cannot compare wire projection: %v", err)
			}
			want := slices.Clone(tc.keys)
			slices.Sort(want)
			if got := slices.Sorted(maps.Keys(fields)); !slices.Equal(got, want) {
				t.Fatalf("keys: %v, expected %v", got, want)
			}
		})
	}
}
