package staffbookingserializerimplementation

import (
	"strconv"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	staffbookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type serializer struct{}

func NewStaffBookingSerializer() staffbookingserializer.StaffBookingSerializer { return serializer{} }

func (serializer) Project(v entity.StaffVisit, s dto.Session, at time.Time, arrival bool) dto.StaffBooking {
	return Project(v, s, at, arrival)
}

const (
	ViewFull       = "full"
	ViewAssignedPA = "assigned-pa"
	ViewReception  = "reception"
	ViewArrival    = "arrival"
)

func Project(v entity.StaffVisit, s dto.Session, at time.Time, arrival bool) dto.StaffBooking {
	start, _ := time.ParseInLocation("2006-01-02 15:04", v.Date+" "+v.Time, validate.WITA)
	b := dto.StaffBooking{
		ID: id(v.ID), Date: v.Date, Time: v.Time, Duration: v.Duration, Room: v.Room, Status: v.Status,
		Version: v.Version, Timing: policy.VisitTiming(v.Status, start, v.Duration, at), Rescheduled: v.Rescheduled,
	}
	if v.Kind == "terbatas" {
		return restricted(b, v, s, arrival)
	}
	b.VisitKind = "standard"
	b.Name, b.Organization = v.Name, v.Organization
	b.Officer = &dto.BookingOfficer{Name: v.OfficerName}
	vip := v.Kind == "vip"
	b.VIP, b.VisitorCount = &vip, &v.VisitorCount
	if s.Role == policy.RoleReceptionist && !arrival {
		return b
	}
	b.OfficerID = id(v.OfficerID)
	if v.RoomID != nil {
		room := id(*v.RoomID)
		b.RoomID = &room
	}
	b.Purpose, b.Visitors = v.Purpose, v.Visitors
	b.AttachmentAvailable, b.NDName = &v.AttachmentAvailable, v.NDName
	photo := v.VisitorPhotoName != nil
	b.VisitorPhotoAvailable, b.VisitorPhotoName = &photo, v.VisitorPhotoName
	b.IdentityType = v.IdentityType
	if s.Role == policy.RoleReceptionist {
		return b
	}
	b.Email, b.Phone = v.Email, v.Phone
	addFull(&b, v, s)
	return b
}

func restricted(b dto.StaffBooking, v entity.StaffVisit, s dto.Session, arrival bool) dto.StaffBooking {
	b.VisitKind, b.Reference = "restricted", "TRB-"+b.ID
	if s.Role == policy.RoleReceptionist && !arrival {
		b.RestrictedView = ViewReception
		return b
	}
	b.Name, b.OfficerID, b.Officer = v.Name, id(v.OfficerID), &dto.BookingOfficer{Name: v.OfficerName}
	if v.ServiceID != nil {
		tail := []rune(*v.ServiceID)
		b.ServiceIDLast4 = string(tail[max(0, len(tail)-4):])
	}
	if s.Role == policy.RoleReceptionist {
		b.RestrictedView = ViewArrival
		return b
	}
	b.Organization = v.Organization
	if s.Role == policy.RolePA {
		b.RestrictedView = ViewAssignedPA
		return b
	}
	b.RestrictedView, b.ServiceID = ViewFull, v.ServiceID
	addFull(&b, v, s)
	return b
}

func addFull(b *dto.StaffBooking, v entity.StaffVisit, s dto.Session) {
	b.Code = v.Code
	b.VisitorInstructions, b.CoordinationNote = v.Instructions, v.CoordinationNote
	b.RejectionReason, b.CancellationReason = v.Rejection, v.Cancellation
	b.InternalCancellationNote = v.InternalCancellation
	if policy.CanDecide(s, b.VisitKind, v.OfficerID, v.Kind == "vip", policy.PermReject) {
		b.InternalRejectionNote = v.InternalRejection
	}
	switch v.CreatorRole {
	case policy.RolePA, policy.RoleProtocol, policy.RoleReceptionist, policy.RoleSuperAdmin:
		if v.CreatorName != "" {
			b.CreatedBy = &dto.StaffBookingActor{Name: v.CreatorName, Role: v.CreatorRole}
		}
	}
	if v.History != nil {
		history := make([]dto.StaffBookingEvent, 0, len(v.History))
		for _, event := range v.History {
			history = append(history, dto.StaffBookingEvent{At: event.At.UTC(), Message: event.Message})
		}
		b.History = &history
	}
}

func id(value int32) string { return strconv.FormatInt(int64(value), 10) }
