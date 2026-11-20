package authusecaseimplementation

import (
	"context"
	"fmt"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type usecase struct {
	repo      authrepository.AuthRepository
	validator authusecase.SessionValidator
}

var _ authusecase.AuthUsecase = (*usecase)(nil)

func NewAuthUsecase(repo authrepository.AuthRepository, serializer authserializer.AuthSerializer) authusecase.AuthUsecase {
	return &usecase{repo: repo, validator: NewSessionValidator(serializer)}
}

// GetSession requires verified token proof plus current database authority on every call.
// Mutations must recheck authority inside their own transaction before committing.
func (u *usecase) GetSession(ctx context.Context, proof entity.SessionProof) (dto.Session, error) {
	if !validProof(proof) {
		return dto.Session{}, authusecase.ErrUnauthenticated
	}
	in, err := u.repo.GetSessionSnapshot(ctx, proof.SessionID)
	if missing(err) {
		return dto.Session{}, authusecase.ErrUnauthenticated
	}
	if err != nil {
		return dto.Session{}, fmt.Errorf("current session: %w", err)
	}
	return u.validator.Validate(in, proof)
}
