package entity

import "time"

type ReportStatus struct {
	ID       int32
	Version  int
	Status   string
	StartsAt time.Time
	Duration int
}

type ReportVisit struct {
	ReportStatus `gorm:"embedded"`
	Reference    string
	Name         string
	Organization string
	OfficerID    int32
	Host         string
	Date         string
	Time         string
	RoomID       *int32
	Room         string
	Purpose      string
	Kind         string
	Visitors     *int
	CheckedInAt  *time.Time
	CheckedOutAt *time.Time
}

type ReportOfficer struct {
	ID   int32
	Name string
}

type ReportSnapshot struct {
	Rows     []ReportVisit
	Officers []ReportOfficer
	Rooms    []string
	At       time.Time
}
