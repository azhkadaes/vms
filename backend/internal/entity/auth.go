package entity

import "time"

type EmployeeCredential struct {
	EmployeeID  int32
	EncodedHash string
	Active      bool
}
type StaffSession struct {
	ID                   string
	EmployeeID           int32
	ActiveRoleID         *int32
	CredentialGeneration int64
	RoleRevision         int64
	RefreshHash          string
	CreatedAt            time.Time
	LastActivityAt       time.Time
	IdleExpiresAt        time.Time
	AbsoluteExpiresAt    time.Time
	RevokedAt            *time.Time
}
type SessionRecovery struct {
	RequestID       string
	SessionID       string
	PredecessorHash string
	SuccessorHash   string
	SuccessorCipher []byte
	ExpiresAt       time.Time
}
