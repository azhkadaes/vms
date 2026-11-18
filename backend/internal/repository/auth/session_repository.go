package authrepository

import (
	"context"
	"github.com/azhkadaes/vms/backend/internal/entity"
	"time"
)

// SessionTransaction keeps lifecycle transitions and their audit in one transaction.
type SessionTransaction interface {
	Now() time.Time
	Credential(employeeID int32) (entity.EmployeeCredential, error)
	Roles(employeeID int32) (map[string]int32, error)
	Session(id string) (entity.StaffSession, error)
	Create(session entity.StaffSession) error
	Save(session entity.StaffSession) error
	SetPassword(employeeID int32, encodedHash string, currentSessionID string) error
	Recovery(sessionID, requestID string) (entity.SessionRecovery, error)
	SaveRecovery(recovery entity.SessionRecovery) error
	ClearRecoveries(sessionID string) error
	RememberRefresh(sessionID, hash string) error
	Audit(employeeID int32, action string) error
}
type SessionRepository interface {
	AuthRepository
	GetCredential(ctx context.Context, email string) (entity.EmployeeCredential, error)
	FindRefresh(ctx context.Context, hash string) (string, error)
	ReserveLogin(ctx context.Context, ipKey, accountKey string) (string, int, error)
	FinishLogin(ctx context.Context, reservation string, success bool) error
	Transaction(ctx context.Context, employeeID int32, operation func(tx SessionTransaction) error) error
}
