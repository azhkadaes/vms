package staffbookingusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type Query struct {
	// Status, Kind and Officer accept a comma-separated list; an empty Kind or Officer means every one.
	View, Search, Status, Kind, Officer, Date, VIP, Sort, Direction, Mode string
	Page, PageSize                                         string
}

type StaffBookingUsecase interface {
	List(context.Context, dto.Session, Query) (dto.StaffBookingPage, error)
	Get(context.Context, dto.Session, int32) (dto.StaffBooking, error)
}
