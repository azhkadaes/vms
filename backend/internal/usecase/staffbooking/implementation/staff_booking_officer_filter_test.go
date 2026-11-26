package staffbookingusecaseimplementation

import (
	"context"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	staffserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	staffbookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type listReader []entity.StaffVisit

func (r listReader) List(context.Context, dto.Session) ([]entity.StaffVisit, time.Time, error) {
	return r, time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC), nil
}

func (r listReader) Find(context.Context, dto.Session, int32) (entity.StaffVisit, time.Time, error) {
	return entity.StaffVisit{}, time.Time{}, nil
}

func TestListFiltersByOfficerWithoutRevealingHiddenOfficials(t *testing.T) {
	visit := func(id, officer int32, kind string) entity.StaffVisit {
		return entity.StaffVisit{Visit: entity.Visit{ID: id, OfficerID: officer, OfficerName: "Officer", Kind: kind,
			Status: "approved", Version: 1, Date: "2026-11-20", Time: "10:00", Duration: 60}}
	}
	u := NewStaffBookingUsecase(listReader{visit(1, 5, "umum"), visit(2, 6, "umum"), visit(3, 5, "terbatas"), visit(4, 7, "vip")},
		staffserializer.NewStaffBookingSerializer(), noop.NewTracerProvider().Tracer("test"))
	reception := dto.Session{Active: true, Role: policy.RoleReceptionist, Permissions: []string{policy.PermBookings}}
	page, err := u.List(context.Background(), reception, staffbookingusecase.Query{Officer: "5,7"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, b := range page.Items {
		ids = append(ids, b.ID)
	}
	slices.Sort(ids)
	if !slices.Equal(ids, []string{"1", "4"}) || page.Total != 2 || page.TotalScoped != 4 {
		t.Fatalf("officer filter: %v total=%d scoped=%d", ids, page.Total, page.TotalScoped)
	}
	for _, bad := range []string{"0", "abc", "5,5", "5,"} {
		if _, err := u.List(context.Background(), reception, staffbookingusecase.Query{Officer: bad}); err == nil {
			t.Errorf("accepted officer=%q", bad)
		}
	}
}
