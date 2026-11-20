package bookingusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type PublicBookingUsecase interface {
	Create(ctx context.Context, ip string, input dto.PublicBookingCreate) (dto.PublicBooking, bool, error)
	Resolve(ctx context.Context, input dto.PublicSubmissionProof) (dto.PublicSubmissionResolution, error)
}
