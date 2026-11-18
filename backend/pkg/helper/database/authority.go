package databasehelper

import (
	"context"
	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"gorm.io/gorm"
)

// CurrentAuthority locks the employee before session, record and resource locks.
// Role-policy writers must lock affected employees in ascending ID order first.
func CurrentAuthority(ctx context.Context, tx *gorm.DB, proof entity.SessionProof) (dto.Session, error) {
	var id int32
	if err := tx.Raw("SELECT id FROM vms.pegawai WHERE id=? FOR UPDATE", proof.EmployeeID).Scan(&id).Error; err != nil {
		return dto.Session{}, err
	}
	reader := authusecase.NewAuthUsecase(authrepository.NewAuthRepository(tx), authserializer.NewAuthSerializer())
	return reader.GetSession(ctx, proof)
}
