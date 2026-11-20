package bookingserializer

import (
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type BookingSerializer interface {
	ToPublicBooking(entity.Visit, time.Time, []entity.VisitHistory) dto.PublicBooking
}
