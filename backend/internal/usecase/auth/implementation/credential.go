package authusecaseimplementation

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"golang.org/x/crypto/bcrypt"
)

type credentialVerifier struct {
	slots chan struct{}
	cost  int
	dummy string
}

// NewCredentialVerifier bounds bcrypt work across login, password changes and account creation.
func NewCredentialVerifier(cost, concurrency int) (authusecase.CredentialVerifier, error) {
	if cost < bcrypt.MinCost || cost > 16 || concurrency < 1 || concurrency > 64 {
		return nil, errors.New("invalid bcrypt work configuration")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("unmatched-dummy-credential-for-timing"), cost)
	if err != nil {
		return nil, err
	}
	return &credentialVerifier{slots: make(chan struct{}, concurrency), cost: cost, dummy: string(hash)}, nil
}
func (v *credentialVerifier) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case v.slots <- struct{}{}:
		return nil
	default:
		return failure.New(503, "service_unavailable")
	}
}
func (v *credentialVerifier) Verify(ctx context.Context, encoded, password string) (bool, error) {
	if err := v.acquire(ctx); err != nil {
		return false, err
	}
	defer func() { <-v.slots }()
	hash := v.dummy
	valid := strings.HasPrefix(encoded, "bcrypt:") && len(password) > 0 && len(password) <= 72 && utf8.ValidString(password)
	if valid {
		candidate := strings.TrimPrefix(encoded, "bcrypt:")
		cost, err := bcrypt.Cost([]byte(candidate))
		if err == nil && cost >= bcrypt.MinCost && cost <= 16 {
			hash = candidate
		} else {
			valid = false
		}
	}
	input := password
	if len(input) > 72 || !utf8.ValidString(input) {
		input = "invalid-password-length"
		valid = false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(input))
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	return valid && err == nil, nil
}
func (v *credentialVerifier) Hash(ctx context.Context, password string) (string, error) {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 15 || len(password) > 72 {
		return "", failure.Validation().Add("newPassword", "Kata sandi baru minimal 15 karakter dan maksimal 72 byte.").Err()
	}
	if err := v.acquire(ctx); err != nil {
		return "", err
	}
	defer func() { <-v.slots }()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), v.cost)
	if err != nil {
		return "", err
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return "bcrypt:" + string(hash), nil
}
