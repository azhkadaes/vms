package catalogusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// CatalogUsecase returns a catalog or an error, never an invented empty success on failure.
type CatalogUsecase interface {
	GetPublicCatalogs(ctx context.Context) (dto.PublicCatalog, error)
	// GetStaffCatalogs is op 08 for the live session StaffAuth read: 403 without bookings or people.
	GetStaffCatalogs(ctx context.Context, session dto.Session) (dto.StaffCatalog, error)
}
