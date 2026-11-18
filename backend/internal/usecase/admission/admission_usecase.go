package admissionusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
)

// AdmissionUsecase normalizes a trusted source IP and checks its shared public request budget.
type AdmissionUsecase interface {
	PublicRequest(ctx context.Context, ip string) (entity.RequestAdmission, error)
}
