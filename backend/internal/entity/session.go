package entity

import "time"

// SessionProof is populated only after cryptographic access-token verification.
// It is not an HTTP request payload and does not grant authority by itself.
type SessionProof struct {
	SessionID            string
	EmployeeID           int32
	CredentialGeneration int64
	RoleRevision         int64
}

// SessionSnapshot contains current database authority, never browser credentials.
type SessionSnapshot struct {
	EmployeeID           int32
	Name                 string
	AccountActive        bool
	MustChangePassword   bool
	ActiveRole           string
	AvailableRoles       []string
	Permissions          []string
	AssignedOfficerIDs   []int32
	CredentialGeneration int64
	RoleRevision         int64
	CreatedAt            time.Time
	LastActivityAt       time.Time
	IdleExpiresAt        time.Time
	AbsoluteExpiresAt    time.Time
	RevokedAt            *time.Time
	ServerTime           time.Time
}
