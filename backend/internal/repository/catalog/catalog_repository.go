package catalogrepository

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
)

// CatalogRepository reads destination catalogs. The public read never exposes employee assignments.
type CatalogRepository interface {
	GetPublicCatalogs(ctx context.Context) (entity.PublicCatalog, error)
	// GetStaffCatalogs reads active officers and rooms in one snapshot; assignedInactiveOfficers
	// adds inactive officers that an employee still holds an active assignment to.
	GetStaffCatalogs(ctx context.Context, assignedInactiveOfficers bool) (entity.StaffCatalog, error)
}
