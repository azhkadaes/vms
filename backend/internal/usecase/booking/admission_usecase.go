package bookingusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type AdmissionUsecase interface {
	Issue(context.Context, int32, dto.PublicCancel) (dto.Admission, error)
}
