package catalogusecaseimplementation_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	catalogserializer "github.com/azhkadaes/vms/backend/internal/serializer/catalog/implementation"
	implementation "github.com/azhkadaes/vms/backend/internal/usecase/catalog/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// repository records every staff catalog read and the scope it was asked for.
type repository struct {
	staff entity.StaffCatalog
	err   error
	reads []bool
}

func (r *repository) GetPublicCatalogs(context.Context) (entity.PublicCatalog, error) {
	return entity.PublicCatalog{}, errors.New("public catalog is not part of op 08")
}

func (r *repository) GetStaffCatalogs(_ context.Context, assignedInactiveOfficers bool) (entity.StaffCatalog, error) {
	r.reads = append(r.reads, assignedInactiveOfficers)
	return r.staff, r.err
}

func ready(role string, permissions ...string) dto.Session {
	return dto.Session{Active: true, ID: "7", Name: "Staff Example", Role: role, AvailableRoles: []string{role}, RoleRevision: 1, Permissions: permissions, AssignedOfficerIDs: []string{}}
}

func newUsecase(r *repository, notice *dto.PublishedNotice) interface {
	GetStaffCatalogs(context.Context, dto.Session) (dto.StaffCatalog, error)
} {
	return implementation.NewCatalogUsecase(r, catalogserializer.NewCatalogSerializer(notice, false), noop.NewTracerProvider().Tracer("catalog-test"))
}

func TestStaffCatalogsRefuseSessionsWithoutBookingsOrPeopleBeforeAnyRead(t *testing.T) {
	for _, tc := range []struct {
		name    string
		session dto.Session
	}{
		{"receptionist without bookings", ready(policy.RoleReceptionist, policy.PermCheckin, policy.PermCheckout)},
		{"super admin with reports and roles only", ready(policy.RoleSuperAdmin, policy.PermReports, policy.PermRoles)},
		{"pending role selection", dto.Session{Active: true, RoleSelectionRequired: true, AvailableRoles: []string{policy.RolePA}, Permissions: []string{policy.PermBookings}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &repository{}
			out, err := newUsecase(r, nil).GetStaffCatalogs(context.Background(), tc.session)
			var denied *failure.Error
			if !errors.As(err, &denied) || denied.Status != 403 || denied.Code != "forbidden" {
				t.Fatalf("want 403 forbidden, got %v", err)
			}
			if len(r.reads) != 0 || out.Officers != nil || out.Rooms != nil {
				t.Fatalf("a refused session read the catalog: reads %v, result %+v", r.reads, out)
			}
		})
	}
}

func TestStaffCatalogsAddAssignedInactiveOfficersOnlyForThePeoplePermission(t *testing.T) {
	stored := entity.StaffCatalog{
		Officers: []entity.CatalogOfficer{
			{ID: 4, Name: "Deputy Official", VIP: true, Active: true, JobTitle: "Deputi Bidang Contoh", Deputy: "Kedeputian Bidang Contoh"},
			{ID: 2147483647, Name: "Director Official", Active: true, JobTitle: "Direktur Contoh", Directorate: "Direktorat Contoh", Deputy: "Kedeputian Bidang Contoh"},
		},
		Rooms: []entity.CatalogRoom{{ID: 3, Name: "Ruang Contoh", Active: true}},
	}
	want := dto.StaffCatalog{
		Officers: []dto.CatalogOfficer{
			{ID: "4", Name: "Deputy Official", VIP: true, Active: true, JobTitle: "Deputi Bidang Contoh", Deputy: "Kedeputian Bidang Contoh"},
			{ID: "2147483647", Name: "Director Official", Active: true, JobTitle: "Direktur Contoh", Directorate: "Direktorat Contoh", Deputy: "Kedeputian Bidang Contoh"},
		},
		Rooms:         []dto.CatalogRoom{{ID: "3", Name: "Ruang Contoh", Active: true}},
		PrivacyNotice: &dto.PublishedNotice{Version: "2026-11-12", ID: "Pemberitahuan contoh", EN: "Example notice"},
	}
	for _, tc := range []struct {
		name         string
		session      dto.Session
		withAssigned bool
	}{
		{"PA with bookings", ready(policy.RolePA, policy.PermBookings), false},
		{"receptionist with bookings", ready(policy.RoleReceptionist, policy.PermBookings, policy.PermCheckin), false},
		{"super admin with people", ready(policy.RoleSuperAdmin, policy.PermPeople), true},
		{"protocol officer with bookings and people", ready(policy.RoleProtocol, policy.PermBookings, policy.PermPeople), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &repository{staff: stored}
			out, err := newUsecase(r, &dto.PublishedNotice{Version: "2026-11-12", ID: "Pemberitahuan contoh", EN: "Example notice"}).GetStaffCatalogs(context.Background(), tc.session)
			if err != nil {
				t.Fatal(err)
			}
			if len(r.reads) != 1 || r.reads[0] != tc.withAssigned {
				t.Fatalf("reads %v, want one read with assigned inactive officers = %v", r.reads, tc.withAssigned)
			}
			if !reflect.DeepEqual(out, want) {
				t.Fatalf("got %+v\nwant %+v", out, want)
			}
		})
	}
}

func TestStaffCatalogsReportARepositoryFailureInsteadOfAnEmptyCatalog(t *testing.T) {
	cause := errors.New("snapshot unavailable")
	out, err := newUsecase(&repository{err: cause}, nil).GetStaffCatalogs(context.Background(), ready(policy.RolePA, policy.PermBookings))
	if !errors.Is(err, cause) || out.Officers != nil || out.Rooms != nil {
		t.Fatalf("got %+v, %v", out, err)
	}
}
