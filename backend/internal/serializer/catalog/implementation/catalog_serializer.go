package catalogserializerimplementation

import (
	"strconv"

	"github.com/azhkadaes/vms/backend/internal/entity"
	catalogserializer "github.com/azhkadaes/vms/backend/internal/serializer/catalog"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type serializer struct{ notice *dto.PublishedNotice }

var _ catalogserializer.CatalogSerializer = (*serializer)(nil)

// NewCatalogSerializer snapshots notice configuration so responses cannot mutate the published version.
func NewCatalogSerializer(notice *dto.PublishedNotice) catalogserializer.CatalogSerializer {
	var snapshot *dto.PublishedNotice
	if notice != nil {
		copied := *notice
		snapshot = &copied
	}
	return &serializer{notice: snapshot}
}

// ToPublicCatalog emits string IDs, empty arrays and no unconfigured notice or upload capability.
func (s *serializer) ToPublicCatalog(in entity.PublicCatalog) dto.PublicCatalog {
	out := dto.PublicCatalog{Officers: []dto.CatalogOfficer{}, Rooms: []dto.CatalogRoom{}, AttachmentsEnabled: false}
	for _, row := range in.Officers {
		out.Officers = append(out.Officers, dto.CatalogOfficer{ID: strconv.FormatInt(int64(row.ID), 10), Name: row.Name, VIP: row.VIP, Active: row.Active, JobTitle: row.JobTitle, Directorate: row.Directorate, Deputy: row.Deputy})
	}
	for _, row := range in.Rooms {
		out.Rooms = append(out.Rooms, dto.CatalogRoom{ID: strconv.FormatInt(int64(row.ID), 10), Name: row.Name, Active: row.Active})
	}
	if s.notice != nil {
		notice := *s.notice
		out.PrivacyNotice = &notice
	}
	return out
}
