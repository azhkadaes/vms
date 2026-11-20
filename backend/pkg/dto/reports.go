package dto

import (
	"encoding/json"
	"time"
)

// ReportFilterValues preserve repeated query values and the frontend's scalar representation.
type ReportFilterValues []string

func (v ReportFilterValues) MarshalJSON() ([]byte, error) {
	switch len(v) {
	case 0:
		return json.Marshal("")
	case 1:
		return json.Marshal(v[0])
	default:
		return json.Marshal([]string(v))
	}
}

type SummaryFilter struct {
	From   string             `json:"from"`
	To     string             `json:"to"`
	Status ReportFilterValues `json:"status"`
}

type VisitSummary struct {
	Filter      SummaryFilter  `json:"filter"`
	Scope       string         `json:"scope"`
	GeneratedAt time.Time      `json:"generatedAt"`
	CanExport   bool           `json:"canExport"`
	Revision    string         `json:"revision"`
	Counts      map[string]int `json:"counts"`
	Total       int            `json:"total"`
}

type SummaryExport struct {
	Filename string `json:"filename"`
	CSV      string `json:"csv"`
	Revision string `json:"revision"`
}

type VisitLogFilter struct {
	From    string             `json:"from"`
	To      string             `json:"to"`
	Q       string             `json:"q"`
	Status  ReportFilterValues `json:"status"`
	Kind    ReportFilterValues `json:"kind"`
	Officer ReportFilterValues `json:"officer"`
	Room    ReportFilterValues `json:"room"`
	Sort    string             `json:"sort"`
	Dir     string             `json:"dir"`
}

type VisitLogRow struct {
	ID           string     `json:"id"`
	Reference    string     `json:"reference"`
	Name         string     `json:"name"`
	Organization string     `json:"organization"`
	OfficerID    string     `json:"officerId"`
	Host         string     `json:"host"`
	Date         string     `json:"date"`
	Time         string     `json:"time"`
	Duration     int        `json:"duration"`
	Room         string     `json:"room"`
	Purpose      string     `json:"purpose"`
	Kind         string     `json:"kind"`
	Visitors     *int       `json:"visitors"`
	Status       string     `json:"status"`
	CheckedInAt  *time.Time `json:"checkedInAt"`
	CheckedOutAt *time.Time `json:"checkedOutAt"`
}

type VisitLogSummary struct {
	Total     int `json:"total"`
	Visitors  int `json:"visitors"`
	Inside    int `json:"inside"`
	Completed int `json:"completed"`
	Pending   int `json:"pending"`
}

type VisitLogOfficer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type VisitLogSnapshot struct {
	Filter      VisitLogFilter  `json:"filter"`
	Items       []VisitLogRow   `json:"items"`
	Summary     VisitLogSummary `json:"summary"`
	GeneratedAt time.Time       `json:"generatedAt"`
	Revision    string          `json:"revision"`
}

type VisitLogResponse struct {
	Filter      VisitLogFilter    `json:"filter"`
	Items       []VisitLogRow     `json:"items"`
	Summary     VisitLogSummary   `json:"summary"`
	GeneratedAt time.Time         `json:"generatedAt"`
	Revision    string            `json:"revision"`
	Page        int               `json:"page"`
	PageSize    int               `json:"pageSize"`
	Total       int               `json:"total"`
	CanExport   bool              `json:"canExport"`
	Officers    []VisitLogOfficer `json:"officers"`
	Rooms       []string          `json:"rooms"`
}
