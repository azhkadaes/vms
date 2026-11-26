package staffbookingusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type Query struct {
	// Status and Kind accept a comma-separated list; an empty Kind means every kind.
	View, Search, Status, Kind, Date, VIP, Sort, Direction, Mode string
	Page, PageSize                                         string
}

type StaffBookingUsecase interface {
	List(context.Context, dto.Session, Query) (dto.StaffBookingPage, error)
	Get(context.Context, dto.Session, int32) (dto.StaffBooking, error)
}
