package dto

type ReceptionResolve struct {
	Credential string `json:"credential"`
}

type ReceptionLookup struct {
	ReceptionResolve
	VisitID string `json:"visitId,omitempty"`
}

type ReceptionAttendance struct {
	Action     string `json:"action"`
	Credential string `json:"credential"`
	Version    int    `json:"version"`
}
