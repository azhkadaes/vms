package catalogserializerimplementation_test

import (
	"encoding/json"
	"testing"

	"github.com/azhkadaes/vms/backend/internal/entity"
	serializer "github.com/azhkadaes/vms/backend/internal/serializer/catalog/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func TestCatalogKeepsIDsExactAndNoticeSnapshotPrivate(t *testing.T) {
	notice := &dto.PublishedNotice{Version: "v1", ID: "Disetujui", EN: "Approved"}
	s := serializer.NewCatalogSerializer(notice, false)
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

func TestStaffCatalogWireKeepsInactiveOfficersOmitsUnknownOrganizationAndNeverSendsNull(t *testing.T) {
	notice := &dto.PublishedNotice{Version: "v1", ID: "Pemberitahuan", EN: "Notice"}
	for _, tc := range []struct {
		name   string
		notice *dto.PublishedNotice
		in     entity.StaffCatalog
		want   string
	}{
		{"empty catalog", nil, entity.StaffCatalog{}, `{"officers":[],"rooms":[]}`},
		{"assigned inactive officer and inactive room", notice, entity.StaffCatalog{
			Officers: []entity.CatalogOfficer{
				{ID: 2147483647, Name: "Retired Official", VIP: true, Active: false, JobTitle: "Direktur Contoh", Directorate: "Direktorat Contoh", Deputy: "Kedeputian Contoh"},
				{ID: 5, Name: "Unplaced Official", Active: true, JobTitle: "Staf Ahli"},
			},
			Rooms: []entity.CatalogRoom{{ID: 3, Name: "Ruang Contoh", Active: false}},
		}, `{"officers":[{"id":"2147483647","name":"Retired Official","vip":true,"active":false,"jobTitle":"Direktur Contoh","directorate":"Direktorat Contoh","deputy":"Kedeputian Contoh"},` +
			`{"id":"5","name":"Unplaced Official","vip":false,"active":true,"jobTitle":"Staf Ahli"}],` +
			`"rooms":[{"id":"3","name":"Ruang Contoh","active":false}],"privacyNotice":{"version":"v1","id":"Pemberitahuan","en":"Notice"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(serializer.NewCatalogSerializer(tc.notice, true).ToStaffCatalog(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			if string(raw) != tc.want {
				t.Fatalf("got  %s\nwant %s", raw, tc.want)
			}
		})
	}
}

// New uploads need attachmentsEnabled=true, so the public catalog reports the configured upload capability.
func TestCatalogReportsAttachmentsOnlyWhenUploadsAreEnabled(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		if got := serializer.NewCatalogSerializer(nil, enabled).ToPublicCatalog(entity.PublicCatalog{}).AttachmentsEnabled; got != enabled {
			t.Fatalf("uploads enabled %v reported attachmentsEnabled %v", enabled, got)
		}
	}
}
