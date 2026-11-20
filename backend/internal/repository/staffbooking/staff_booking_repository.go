package staffbookingrepository

import (
	"context"
	"fmt"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

var ErrNotFound = fmt.Errorf("staff booking: %w", failure.ErrNotFound)

type StaffBookingReader interface {
	List(context.Context, dto.Session) ([]entity.StaffVisit, time.Time, error)
	Find(context.Context, dto.Session, int32) (entity.StaffVisit, time.Time, error)
}
