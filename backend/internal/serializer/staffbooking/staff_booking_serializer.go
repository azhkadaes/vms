package staffbookingserializer

import (
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type StaffBookingSerializer interface {
	Project(entity.StaffVisit, dto.Session, time.Time, bool) dto.StaffBooking
}
