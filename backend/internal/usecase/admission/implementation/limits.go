package admissionusecaseimplementation

import (
	"context"
	admissionrepository "github.com/azhkadaes/vms/backend/internal/repository/admission"
	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"time"
)

type limiter struct {
	repo admissionrepository.RequestLimitRepository
	key  []byte
}

func NewRequestLimiter(repo admissionrepository.RequestLimitRepository, key []byte) admissionusecase.RequestLimiter {
	return &limiter{repo: repo, key: key}
}
func (l *limiter) Admit(ctx context.Context, purpose, identity string, limit int, window time.Duration) error {
	decision, err := l.repo.RequestLimit(ctx, security.Digest(l.key, purpose, identity), limit, window)
	if err != nil {
		return failure.New(503, "service_unavailable").WithCause(err)
	}
	if !decision.Allowed {
		return &failure.Error{Status: 429, Code: "rate_limited", RetryAfter: decision.RetryAfter}
	}
	return nil
}
