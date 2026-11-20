package staffbookingusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type StaffBookingCreateUsecase interface {
	CreateStaffBooking(context.Context, dto.Session, entity.SessionProof, string, dto.StaffBookingCreate) (dto.StaffBooking, bool, error)
}
