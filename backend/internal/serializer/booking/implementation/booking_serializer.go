package bookingserializerimplementation

import (
	"strconv"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type serializer struct{}

func NewBookingSerializer() bookingserializer.BookingSerializer { return serializer{} }

func (serializer) ToPublicBooking(v entity.Visit, now time.Time, events []entity.VisitHistory) dto.PublicBooking {
	start, _ := time.ParseInLocation("2006-01-02 15:04", v.Date+" "+v.Time, validate.WITA)
	history := make([]dto.BookingHistory, 0, len(events))
	var rescheduled bool
	for _, e := range events {
		rescheduled = rescheduled || e.Action == "rescheduled"
		history = append(history, dto.BookingHistory{At: e.At.UTC(), Action: e.Action, PublicMessage: e.PublicMessage})
	}
	var roomID *string
	if v.RoomID != nil {
		id := strconv.FormatInt(int64(*v.RoomID), 10)
		roomID = &id
	}
	return dto.PublicBooking{
		ID: strconv.FormatInt(int64(v.ID), 10), Code: v.Code, Name: v.Name, Organization: v.Organization,
		Email: v.Email, Phone: v.Phone, OfficerID: strconv.FormatInt(int64(v.OfficerID), 10),
		Officer: dto.BookingOfficer{Name: v.OfficerName}, VIP: v.Kind == "vip", Date: v.Date, Time: v.Time,
		Duration: v.Duration, Room: v.Room, RoomID: roomID, Purpose: v.Purpose, Visitors: v.Visitors, IdentityType: v.IdentityType,
		Status: v.Status, Version: v.Version, Timing: policy.VisitTiming(v.Status, start, v.Duration, now), History: history,
		VisitKind: "standard", NDName: v.NDName, VisitorInstructions: v.Instructions,
		RejectionReason: v.Rejection, CancellationReason: v.Cancellation, Rescheduled: rescheduled,
	}
}
