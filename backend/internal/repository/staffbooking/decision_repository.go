package staffbookingrepository

import (
	"context"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	auth "github.com/azhkadaes/vms/backend/internal/repository/auth"
)

type DecisionRepository interface {
	DecisionTransaction(context.Context, func(DecisionTransaction) error) error
}

type DecisionTransaction interface {
	auth.AuthorityLock
	LockVisit(id int32) (entity.StaffVisit, error)
	Now() (time.Time, error)
	ActiveOfficer(id int32) (bool, error)
	ActiveRoom(id int32) (bool, error)
	LockResources(officers, rooms []int32) error
	HasConflict(visitID int32, schedule entity.VisitSchedule, start time.Time) (bool, error)
	ApplyChange(id int32, version int, actor int32, role string, now time.Time, change entity.VisitChange) (entity.StaffVisit, error)
}
