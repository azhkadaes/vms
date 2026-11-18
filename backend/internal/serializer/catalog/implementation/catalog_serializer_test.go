package catalogserializerimplementation_test

import (
	"testing"

	"github.com/azhkadaes/vms/backend/internal/entity"
	serializer "github.com/azhkadaes/vms/backend/internal/serializer/catalog/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func TestCatalogKeepsIDsExactAndNoticeSnapshotPrivate(t *testing.T) {
	notice := &dto.PublishedNotice{Version: "v1", ID: "Disetujui", EN: "Approved"}
	s := serializer.NewCatalogSerializer(notice)
	notice.Version = "changed"
	input := entity.PublicCatalog{Officers: []entity.CatalogOfficer{{ID: 2147483647, Name: "Official", VIP: true, Active: true, JobTitle: "Direktur Pelayanan Dasar", Directorate: "Direktorat Pelayanan Dasar", Deputy: "Kedeputian Bidang Sosial, Budaya, dan Pemberdayaan Masyarakat"}}}
	first := s.ToPublicCatalog(input)
	if first.Officers[0].ID != "2147483647" || !first.Officers[0].VIP || first.Rooms == nil || first.AttachmentsEnabled {
		t.Fatal("catalog wire contract drifted")
	}
	if first.Officers[0].Directorate != input.Officers[0].Directorate || first.Officers[0].Deputy != input.Officers[0].Deputy || first.Officers[0].JobTitle != input.Officers[0].JobTitle {
		t.Fatal("officer organization was lost during catalog serialization")
	}
	first.PrivacyNotice.Version = "mutated response"
	if s.ToPublicCatalog(input).PrivacyNotice.Version != "v1" {
		t.Fatal("notice configuration escaped into a mutable response")
	}
}
