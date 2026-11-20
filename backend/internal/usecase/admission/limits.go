package admissionusecase

import (
	"context"
	"time"
)

type RequestLimiter interface {
	Admit(ctx context.Context, purpose, identity string, limit int, window time.Duration) error
}
