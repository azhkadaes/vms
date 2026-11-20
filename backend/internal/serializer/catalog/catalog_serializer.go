package catalogserializer

import (
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// CatalogSerializer limits catalog output to the agreed wire projection of each audience.
type CatalogSerializer interface {
	ToPublicCatalog(in entity.PublicCatalog) dto.PublicCatalog
	ToStaffCatalog(in entity.StaffCatalog) dto.StaffCatalog
}
