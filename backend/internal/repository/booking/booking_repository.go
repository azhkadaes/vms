package bookingrepository

import (
	"context"
	"fmt"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

var ErrNotFound = fmt.Errorf("booking: %w", failure.ErrNotFound)

type BookingRepository interface {
	Transaction(context.Context, func(BookingTransaction) error) error
}

type BookingTransaction interface {
	LockReference(audience, ownerDigest, reference string) error
	Request(audience, ownerDigest, reference string) (entity.BookingRequest, error)
	StoreRequest(audience, ownerDigest, reference string, request entity.BookingRequest) error
	Now() (time.Time, error)
	ReadVisit(id int32) (entity.Visit, error)
	PublicHistory(id int32) ([]entity.VisitHistory, error)
	ActiveOfficer(id int32) (name string, vip bool, err error)
	LinkUpload(tokenHash, purpose, audience, ownerDigest string, now time.Time) (int32, error)
	AdmitBooking(bucket string) (entity.RequestAdmission, error)
	CreatePublicVisit(visit entity.Visit, photoID int32, documentID *int32, now time.Time, privacyVersion, ip string) (int32, error)
}
