package admissionrepository

import (
	"context"
	"github.com/azhkadaes/vms/backend/internal/entity"
	"time"
)

type RequestLimitRepository interface {
	RequestLimit(ctx context.Context, key string, limit int, window time.Duration) (entity.RequestAdmission, error)
}
