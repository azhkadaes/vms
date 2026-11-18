package catalogusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// CatalogUsecase returns a public catalog or an error, never an invented empty success on failure.
type CatalogUsecase interface {
	GetPublicCatalogs(ctx context.Context) (dto.PublicCatalog, error)
}
