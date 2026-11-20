package dto

import "time"

type PublicBookingCreate struct {
	OfficerID         string   `json:"officerId"`
	Date              string   `json:"date"`
	Time              string   `json:"time"`
	Purpose           string   `json:"purpose"`
	Name              string   `json:"name"`
	Organization      string   `json:"organization"`
	Email             string   `json:"email"`
	Phone             string   `json:"phone"`
	IdentityType      string   `json:"identityType"`
	Visitors          []string `json:"visitors"`
	VisitKind         string   `json:"visitKind,omitempty"`
	NDName            string   `json:"ndName,omitempty"`
	AttachmentToken   string   `json:"attachmentToken,omitempty"`
	VisitorPhotoName  string   `json:"visitorPhotoName"`
	VisitorPhotoToken string   `json:"visitorPhotoToken"`
	Ack               bool     `json:"ack"`
	PrivacyVersion    string   `json:"privacyVersion"`
	IdempotencyKey    string   `json:"idempotencyKey"`
}

type PublicSubmissionProof struct {
	IdempotencyKey string `json:"idempotencyKey"`
	Email          string `json:"email"`
}

type PublicSubmissionResolution struct {
	Outcome string         `json:"outcome"`
	Booking *PublicBooking `json:"booking,omitempty"`
}

type BookingOfficer struct {
	Name string `json:"nama"`
}

type BookingTiming struct {
	ServerTime    int64  `json:"serverTime"`
	OpensAt       int64  `json:"opensAt"`
	EndsAt        int64  `json:"endsAt"`
	DisplayStatus string `json:"displayStatus"`
	CanCheckIn    bool   `json:"canCheckIn"`
	CanCheckOut   bool   `json:"canCheckOut"`
	CanCancel     bool   `json:"canCancel"`
}

type BookingHistory struct {
	At            time.Time `json:"at"`
	Action        string    `json:"action"`
	PublicMessage *string   `json:"publicMessage,omitempty"`
}

type PublicBooking struct {
	ID                  string           `json:"id"`
	Code                string           `json:"code"`
	Name                string           `json:"name"`
	Organization        string           `json:"organization"`
	Email               string           `json:"email"`
	Phone               string           `json:"phone"`
	OfficerID           string           `json:"officerId"`
	Officer             BookingOfficer   `json:"pejabat"`
	VIP                 bool             `json:"vip"`
	Date                string           `json:"date"`
	Time                string           `json:"time"`
	Duration            int              `json:"duration"`
	Room                string           `json:"room"`
	RoomID              *string          `json:"roomId"`
	Purpose             string           `json:"purpose"`
	Visitors            []string         `json:"visitors"`
	IdentityType        string           `json:"identityType"`
	Status              string           `json:"status"`
	Version             int              `json:"version"`
	Timing              BookingTiming    `json:"timing"`
	History             []BookingHistory `json:"history"`
	VisitKind           string           `json:"visitKind"`
	NDName              *string          `json:"ndName,omitempty"`
	VisitorInstructions *string          `json:"visitorInstructions,omitempty"`
	RejectionReason     *string          `json:"rejectionReason,omitempty"`
	CancellationReason  *string          `json:"cancellationReason,omitempty"`
	Rescheduled         bool             `json:"rescheduled,omitempty"`
}
