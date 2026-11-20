package bookingusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type BookingAccessUsecase interface {
	Lookup(context.Context, dto.PublicLookup) (dto.PublicBooking, error)
	Cancel(context.Context, int32, dto.PublicCancel, string) (dto.PublicBooking, error)
}
