package receptionrepository

import (
	"context"
	"fmt"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

var ErrNotFound = fmt.Errorf("reception: %w", failure.ErrNotFound)

type ReceptionRepository interface {
	Resolve(context.Context, entity.ReceptionCredential) (entity.StaffVisit, entity.ReceptionMatch, time.Time, error)
	Transaction(context.Context, func(ReceptionTransaction) error) error
}

type ReceptionTransaction interface {
	authrepository.AuthorityLock
	LockVisit(int32) (entity.StaffVisit, error)
	MatchCredential(entity.ReceptionCredential, int32) (entity.ReceptionMatch, error)
	Now() (time.Time, error)
	Transition(id int32, version int, actor int32, action string, at time.Time) (entity.StaffVisit, error)
}
