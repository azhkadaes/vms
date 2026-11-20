package dto

type StaffBookingCreate struct {
	OfficerID           string   `json:"officerId"`
	Date                string   `json:"date"`
	Time                string   `json:"time"`
	VisitKind           string   `json:"visitKind,omitempty"`
	Duration            *int     `json:"duration,omitempty"`
	RoomID              *string  `json:"roomId,omitempty"`
	Name                string   `json:"name,omitempty"`
	Organization        string   `json:"organization,omitempty"`
	Purpose             string   `json:"purpose,omitempty"`
	Email               string   `json:"email,omitempty"`
	Phone               string   `json:"phone,omitempty"`
	IdentityType        string   `json:"identityType,omitempty"`
	Visitors            []string `json:"visitors,omitempty"`
	VisitorPhotoName    string   `json:"visitorPhotoName,omitempty"`
	VisitorPhotoToken   string   `json:"visitorPhotoToken,omitempty"`
	NDName              string   `json:"ndName,omitempty"`
	AttachmentToken     string   `json:"attachmentToken,omitempty"`
	ServiceID           string   `json:"serviceId,omitempty"`
	VisitorInstructions string   `json:"visitorInstructions,omitempty"`
	CoordinationNote    string   `json:"coordinationNote,omitempty"`
	IdempotencyKey      string   `json:"idempotencyKey"`
}
