package admissionrepository

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
)

// AdmissionRepository atomically admits requests under the durable overall public-IP budget.
type AdmissionRepository interface {
	PublicRequest(ctx context.Context, key string) (entity.RequestAdmission, error)
}
