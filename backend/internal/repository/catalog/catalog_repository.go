package catalogrepository

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
)

// CatalogRepository reads active public destinations without exposing employee assignments.
type CatalogRepository interface {
	GetPublicCatalogs(ctx context.Context) (entity.PublicCatalog, error)
}
