package staffbookingusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type DecisionUsecase interface {
	Approve(context.Context, entity.SessionProof, int32, dto.ApproveBooking) (dto.StaffBooking, error)
	Reject(context.Context, entity.SessionProof, int32, dto.RejectBooking) (dto.StaffBooking, error)
	Cancel(context.Context, entity.SessionProof, int32, dto.CancelBooking) (dto.StaffBooking, error)
	Reschedule(context.Context, entity.SessionProof, int32, dto.RescheduleBooking) (dto.StaffBooking, error)
	UpdateRestricted(context.Context, entity.SessionProof, int32, dto.UpdateRestrictedBooking) (dto.StaffBooking, error)
}
