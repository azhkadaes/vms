package authusecase

import "context"

type CredentialVerifier interface {
	Verify(ctx context.Context, encodedHash, password string) (bool, error)
	Hash(ctx context.Context, password string) (string, error)
}
