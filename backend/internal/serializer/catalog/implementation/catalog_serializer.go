package catalogserializerimplementation

import (
	"strconv"

	"github.com/azhkadaes/vms/backend/internal/entity"
	catalogserializer "github.com/azhkadaes/vms/backend/internal/serializer/catalog"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type serializer struct {
	notice      *dto.PublishedNotice
	attachments bool
}

var _ catalogserializer.CatalogSerializer = (*serializer)(nil)

// NewCatalogSerializer snapshots notice configuration so responses cannot mutate the published version.
// attachments reports whether new uploads are enabled (verified storage capability).
func NewCatalogSerializer(notice *dto.PublishedNotice, attachments bool) catalogserializer.CatalogSerializer {
	var snapshot *dto.PublishedNotice
	if notice != nil {
		copied := *notice
		snapshot = &copied
	}
	return &serializer{notice: snapshot, attachments: attachments}
}

// ToPublicCatalog emits string IDs, empty arrays, no unconfigured notice, and attachmentsEnabled
// only while uploads are enabled.
func (s *serializer) ToPublicCatalog(in entity.PublicCatalog) dto.PublicCatalog {
	return dto.PublicCatalog{Officers: officers(in.Officers), Rooms: rooms(in.Rooms), AttachmentsEnabled: s.attachments, PrivacyNotice: s.publishedNotice()}
}

// ToStaffCatalog emits the same officer and room shapes and notice as the public catalog, without
// the public upload capability flag.
func (s *serializer) ToStaffCatalog(in entity.StaffCatalog) dto.StaffCatalog {
	return dto.StaffCatalog{Officers: officers(in.Officers), Rooms: rooms(in.Rooms), PrivacyNotice: s.publishedNotice()}
}

// publishedNotice copies the configured notice so a response cannot mutate it; nil when unconfigured.
func (s *serializer) publishedNotice() *dto.PublishedNotice {
	if s.notice == nil {
		return nil
	}
	notice := *s.notice
	return &notice
}

// officers serializes decimal string IDs and never returns nil, so an empty list is [].
func officers(rows []entity.CatalogOfficer) []dto.CatalogOfficer {
	out := make([]dto.CatalogOfficer, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.CatalogOfficer{ID: strconv.FormatInt(int64(row.ID), 10), Name: row.Name, VIP: row.VIP, Active: row.Active, JobTitle: row.JobTitle, Directorate: row.Directorate, Deputy: row.Deputy})
	}
	return out
}

// rooms serializes decimal string IDs and never returns nil, so an empty list is [].
func rooms(rows []entity.CatalogRoom) []dto.CatalogRoom {
	out := make([]dto.CatalogRoom, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.CatalogRoom{ID: strconv.FormatInt(int64(row.ID), 10), Name: row.Name, Active: row.Active})
	}
	return out
}
