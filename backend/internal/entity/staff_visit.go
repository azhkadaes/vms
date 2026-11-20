package entity

import "time"

type StaffVisit struct {
	Visit                `gorm:"embedded"`
	VisitorCount         int
	ServiceID            *string
	AttachmentAvailable  bool
	VisitorPhotoName     *string
	CoordinationNote     *string
	InternalRejection    *string
	InternalCancellation *string
	CreatorName          string
	CreatorRole          string
	Rescheduled          bool
	History              []StaffVisitEvent `gorm:"-"`
}

type StaffVisitEvent struct {
	At      time.Time `json:"at"`
	Message string    `json:"message"`
}
