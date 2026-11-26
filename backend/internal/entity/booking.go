package entity

import "time"

type Visit struct {
	ID           int32
	Code         string
	Kind         string
	Status       string
	Version      int
	OfficerID    int32
	OfficerName  string
	OfficerPAs   []OfficerPA `gorm:"-"`
	RoomID       *int32
	Room         string
	Date         string
	Time         string
	Duration     int
	Purpose      string
	Name         string
	Organization string
	Email        string
	Phone        string
	IdentityType string
	Visitors     []string `gorm:"-"`
	NDName       *string
	Instructions *string
	Rejection    *string
	Cancellation *string
	CheckIn      *time.Time
	CheckOut     *time.Time
}

// OfficerPA is an active employee on an active PA assignment to the visit's official.
type OfficerPA struct {
	Name   string   `json:"nama"`
	Phones []string `json:"nomorHp"`
}

type BookingRequest struct {
	State         string
	PayloadDigest string
	EmailDigest   string
	VisitID       int32
}

type VisitHistory struct {
	At            time.Time
	Action        string
	PublicMessage *string
}
