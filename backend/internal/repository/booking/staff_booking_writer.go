package bookingrepository

import (
	"context"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
)

type StaffBookingWriter interface {
	StaffTransaction(context.Context, func(StaffBookingTransaction) error) error
}

type StaffBookingTransaction interface {
	authrepository.AuthorityLock
	LockReference(audience, ownerDigest, reference string) error
	Request(audience, ownerDigest, reference string) (entity.BookingRequest, error)
	StoreRequest(audience, ownerDigest, reference string, request entity.BookingRequest) error
	Now() (time.Time, error)
	ReadStaffVisit(id int32) (entity.StaffVisit, error)
	ActiveOfficer(id int32) (name string, vip bool, err error)
	ActiveRoom(id int32) error
	LockScheduleResources(officerIDs, roomIDs []int32) error
	ScheduleConflict(excludeVisitID, officerID int32, roomID *int32, start time.Time, duration int) (bool, error)
	LinkUpload(tokenHash, purpose, audience, ownerDigest string, now time.Time) (int32, error)
	CreateStaffVisit(visit entity.StaffVisit, photoID, documentID *int32, actorID int32, now time.Time, ip string) (int32, error)
}
