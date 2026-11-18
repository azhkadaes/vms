package catalogserializer

import (
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// CatalogSerializer limits catalog output to the agreed visitor-safe wire projection.
type CatalogSerializer interface {
	ToPublicCatalog(in entity.PublicCatalog) dto.PublicCatalog
}
