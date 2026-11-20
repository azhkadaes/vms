package admissionrepositoryimplementation

import (
	"context"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	admissionrepository "github.com/azhkadaes/vms/backend/internal/repository/admission"
)

type repo struct{ db *gorm.DB }

var _ admissionrepository.AdmissionRepository = (*repo)(nil)

// NewAdmissionRepository uses PostgreSQL state so API restarts and instances share the same allowance.
func NewAdmissionRepository(db *gorm.DB) admissionrepository.AdmissionRepository {
	return &repo{db: db}
}

// PublicRequest serializes admission per keyed IP digest, including an empty bucket.
func (r *repo) PublicRequest(ctx context.Context, key string) (entity.RequestAdmission, error) {
	return r.RequestLimit(ctx, key, 120, time.Minute)
}

func NewRequestLimitRepository(db *gorm.DB) admissionrepository.RequestLimitRepository {
	return &repo{db: db}
}

func (r *repo) RequestLimit(ctx context.Context, key string, limit int, window time.Duration) (entity.RequestAdmission, error) {
	var result entity.RequestAdmission
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = Admit(tx, key, limit, window)
		return err
	})
	if err != nil {
		return entity.RequestAdmission{}, fmt.Errorf("admit public request: %w", err)
	}
	return result, nil
}

// Admit takes one event of key's bucket on tx unless limit events already fall inside window.
// It locks the bucket row (creating it when missing) until tx ends, so callers that admit inside
// a larger transaction count an event only if that transaction commits.
func Admit(tx *gorm.DB, key string, limit int, window time.Duration) (entity.RequestAdmission, error) {
	var result entity.RequestAdmission
	if err := tx.Exec("INSERT INTO support.rate_limit_buckets (bucket_key) VALUES (?) ON CONFLICT DO NOTHING", key).Error; err != nil {
		return result, err
	}
	var locked string
	if err := tx.Raw("SELECT bucket_key FROM support.rate_limit_buckets WHERE bucket_key = ? FOR UPDATE", key).Scan(&locked).Error; err != nil {
		return result, err
	}
	if locked != key {
		return result, fmt.Errorf("rate bucket missing after insertion")
	}
	var clock struct{ Now time.Time }
	if err := tx.Raw("SELECT clock_timestamp() AS now").Scan(&clock).Error; err != nil {
		return result, err
	}
	cutoff := clock.Now.Add(-window)
	if err := tx.Exec("DELETE FROM support.rate_limit_events WHERE bucket_key = ? AND occurred_at <= ?", key, cutoff).Error; err != nil {
		return result, err
	}
	var count struct {
		Count  int64
		Oldest *time.Time
	}
	if err := tx.Raw("SELECT count(*) AS count, min(occurred_at) AS oldest FROM support.rate_limit_events WHERE bucket_key = ?", key).Scan(&count).Error; err != nil {
		return result, err
	}
	if count.Count >= int64(limit) {
		if count.Oldest == nil {
			return result, fmt.Errorf("rate bucket has invalid event history")
		}
		result.RetryAfter = max(1, int(math.Ceil(count.Oldest.Add(window).Sub(clock.Now).Seconds())))
		return result, nil
	}
	if err := tx.Exec("INSERT INTO support.rate_limit_events (bucket_key,occurred_at) VALUES (?,?)", key, clock.Now).Error; err != nil {
		return result, err
	}
	if err := tx.Exec("UPDATE support.rate_limit_buckets SET last_seen_at = ? WHERE bucket_key = ?", clock.Now, key).Error; err != nil {
		return result, err
	}
	result.Allowed = true
	return result, nil
}
