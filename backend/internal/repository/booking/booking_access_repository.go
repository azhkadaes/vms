package bookingrepository

import (
	"context"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
)

type BookingAccessRepository interface {
	AccessTransaction(context.Context, func(BookingAccessTransaction) error) error
}

type BookingAccessTransaction interface {
	// PublicVisit checks the credentials and locks the visit before projecting it.
	// A mutation also matches expectedID and takes an exclusive lock.
	PublicVisit(code, email string, expectedID *int32) (entity.Visit, error)
	Now() (time.Time, error)
	PublicHistory(id int32) ([]entity.VisitHistory, error)
	CancelPublic(id int32, now time.Time, ip string) (int, error)
	CurrentQR(visitID int32) (entity.VisitQR, error)
	ReserveQRID() (int32, error)
	InsertQR(entity.VisitQR) error
}
