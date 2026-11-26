package receptionusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type ReceptionUsecase interface {
	Resolve(context.Context, dto.Session, dto.ReceptionLookup) (dto.StaffBooking, error)
	Attendance(context.Context, entity.SessionProof, int32, dto.ReceptionAttendance) (dto.StaffBooking, error)
}
