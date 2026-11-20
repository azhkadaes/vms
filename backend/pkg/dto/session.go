package dto

import "time"

type SessionTiming struct {
	ServerTime        int64     `json:"serverTime"`
	IdleExpiresAt     time.Time `json:"idleExpiresAt"`
	AbsoluteExpiresAt time.Time `json:"absoluteExpiresAt"`
}

type Session struct {
	Active                bool          `json:"active"`
	ID                    string        `json:"id"`
	Name                  string        `json:"name"`
	Role                  string        `json:"role"`
	AvailableRoles        []string      `json:"availableRoles"`
	RoleSelectionRequired bool          `json:"roleSelectionRequired"`
	RoleRevision          int64         `json:"roleRevision"`
	Permissions           []string      `json:"permissions"`
	AssignedOfficerIDs    []string      `json:"assignedOfficerIds"`
	MustChangePassword    bool          `json:"mustChangePassword"`
	PasswordMinLength     int           `json:"passwordMinLength"`
	PasswordMaxBytes      int           `json:"passwordMaxBytes"`
	Timing                SessionTiming `json:"timing"`
}
