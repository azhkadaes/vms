package entity

import "time"

// BookingFile carries private storage metadata and the visit state from one read snapshot.
type BookingFile struct {
	Key          string
	MIME         string `gorm:"column:mime"`
	Size         int
	OfficerID    int32
	Kind, Status string
	Date, Time   string
	Duration     int
	Match        ReceptionMatch `gorm:"-"`
	ServerTime   time.Time
}
