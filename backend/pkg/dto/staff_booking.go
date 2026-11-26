package dto

import "time"

type StaffBooking struct {
	ID                       string               `json:"id"`
	Date                     string               `json:"date"`
	Time                     string               `json:"time"`
	Duration                 int                  `json:"duration"`
	Room                     string               `json:"room"`
	Status                   string               `json:"status"`
	Version                  int                  `json:"version"`
	Timing                   BookingTiming        `json:"timing"`
	AllowedActions           []string             `json:"allowedActions"`
	VisitKind                string               `json:"visitKind"`
	Rescheduled              bool                 `json:"rescheduled,omitempty"`
	RoomID                   *string              `json:"roomId,omitempty"`
	Reference                string               `json:"reference,omitempty"`
	RestrictedView           string               `json:"restrictedView,omitempty"`
	Code                     string               `json:"code,omitempty"`
	Name                     string               `json:"name,omitempty"`
	Organization             string               `json:"organization,omitempty"`
	Email                    string               `json:"email,omitempty"`
	Phone                    string               `json:"phone,omitempty"`
	OfficerID                string               `json:"officerId,omitempty"`
	Officer                  *BookingOfficer      `json:"pejabat,omitempty"`
	VIP                      *bool                `json:"vip,omitempty"`
	Purpose                  string               `json:"purpose,omitempty"`
	Visitors                 []string             `json:"visitors,omitempty"`
	IdentityType             string               `json:"identityType,omitempty"`
	VisitorCount             *int                 `json:"visitorCount,omitempty"`
	AttachmentAvailable      *bool                `json:"attachmentAvailable,omitempty"`
	VisitorPhotoAvailable    *bool                `json:"visitorPhotoAvailable,omitempty"`
	VisitorPhotoName         *string              `json:"visitorPhotoName,omitempty"`
	NDName                   *string              `json:"ndName,omitempty"`
	CreatedBy                *StaffBookingActor   `json:"createdBy,omitempty"`
	ServiceID                *string              `json:"serviceId,omitempty"`
	ServiceIDLast4           string               `json:"serviceIdLast4,omitempty"`
	VisitorInstructions      *string              `json:"visitorInstructions,omitempty"`
	CoordinationNote         *string              `json:"coordinationNote,omitempty"`
	RejectionReason          *string              `json:"rejectionReason,omitempty"`
	InternalRejectionNote    *string              `json:"internalRejectionNote,omitempty"`
	CancellationReason       *string              `json:"cancellationReason,omitempty"`
	InternalCancellationNote *string              `json:"internalCancellationNote,omitempty"`
	History                  *[]StaffBookingEvent `json:"history,omitempty"`
}

type StaffBookingActor struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type StaffBookingEvent struct {
	At      time.Time `json:"at"`
	Message string    `json:"message"`
}

type StaffBookingPage struct {
	Items        []StaffBooking       `json:"items"`
	Total        int                  `json:"total"`
	TotalScoped  int                  `json:"totalScoped"`
	Page         int                  `json:"page"`
	PageSize     int                  `json:"pageSize"`
	StatusCounts *StaffBookingCounts  `json:"statusCounts,omitempty"`
	Metrics      *StaffBookingMetrics `json:"metrics,omitempty"`
}

type StaffBookingCounts struct {
	All        int `json:"all"`
	Pending    int `json:"pending"`
	Approved   int `json:"approved"`
	Missed     int `json:"missed"`
	Rejected   int `json:"rejected"`
	Cancelled  int `json:"cancelled"`
	CheckedIn  int `json:"checked_in"`
	CheckedOut int `json:"checked_out"`
}

type StaffBookingMetrics struct {
	Today     int `json:"today"`
	Pending   int `json:"pending"`
	Arrivals  int `json:"arrivals"`
	Inside    int `json:"inside"`
	Completed int `json:"completed"`
}
