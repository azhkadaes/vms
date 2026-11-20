package dto

type ApproveBooking struct {
	Version             int              `json:"version"`
	RoomID              Optional[string] `json:"roomId,omitempty"`
	Duration            *int             `json:"duration,omitempty"`
	VisitorInstructions Optional[string] `json:"visitorInstructions,omitempty"`
	CoordinationNote    Optional[string] `json:"coordinationNote,omitempty"`
}

type RejectBooking struct {
	Version               int    `json:"version"`
	RejectionReason       string `json:"rejectionReason"`
	InternalRejectionNote string `json:"internalRejectionNote,omitempty"`
}

type CancelBooking struct {
	Version                  int    `json:"version"`
	Reason                   string `json:"reason"`
	InternalCancellationNote string `json:"internalCancellationNote,omitempty"`
}

type RescheduleBooking struct {
	Version             int              `json:"version"`
	Date                string           `json:"date"`
	Time                string           `json:"time"`
	Duration            *int             `json:"duration,omitempty"`
	RoomID              Optional[string] `json:"roomId,omitempty"`
	Reason              string           `json:"reason,omitempty"`
	VisitorInstructions Optional[string] `json:"visitorInstructions,omitempty"`
	CoordinationNote    Optional[string] `json:"coordinationNote,omitempty"`
}

type UpdateRestrictedBooking struct {
	Version             int              `json:"version"`
	VisitKind           string           `json:"visitKind,omitempty"`
	Date                Optional[string] `json:"date,omitempty"`
	Time                Optional[string] `json:"time,omitempty"`
	OfficerID           string           `json:"officerId"`
	RoomID              Optional[string] `json:"roomId,omitempty"`
	Name                Optional[string] `json:"name,omitempty"`
	Organization        Optional[string] `json:"organization,omitempty"`
	ServiceID           Optional[string] `json:"serviceId,omitempty"`
	VisitorInstructions Optional[string] `json:"visitorInstructions,omitempty"`
	CoordinationNote    Optional[string] `json:"coordinationNote,omitempty"`
}
