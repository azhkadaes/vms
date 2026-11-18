package catalogusecaseimplementation

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	catalogrepository "github.com/azhkadaes/vms/backend/internal/repository/catalog"
	catalogserializer "github.com/azhkadaes/vms/backend/internal/serializer/catalog"
	catalogusecase "github.com/azhkadaes/vms/backend/internal/usecase/catalog"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type usecase struct {
	repo       catalogrepository.CatalogRepository
	serializer catalogserializer.CatalogSerializer
	tracer     trace.Tracer
}

var _ catalogusecase.CatalogUsecase = (*usecase)(nil)

// NewCatalogUsecase composes the consistent read and its explicit public projection.
func NewCatalogUsecase(repo catalogrepository.CatalogRepository, serializer catalogserializer.CatalogSerializer, tracer trace.Tracer) catalogusecase.CatalogUsecase {
	return &usecase{repo: repo, serializer: serializer, tracer: tracer}
}

// GetPublicCatalogs preserves the caller's deadline through the traced repository operation.
func (u *usecase) GetPublicCatalogs(ctx context.Context) (dto.PublicCatalog, error) {
	ctx, span := u.tracer.Start(ctx, "catalog.public")
	defer span.End()
	rows, err := u.repo.GetPublicCatalogs(ctx)
	if err != nil {
		return dto.PublicCatalog{}, fmt.Errorf("public catalogs: %w", err)
	}
	return u.serializer.ToPublicCatalog(rows), nil
}
