package employeeusecaseimplementation_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	employeeusecase "github.com/azhkadaes/vms/backend/internal/usecase/employee"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// failed returns the failure err carries, or fails the test.
func failed(t *testing.T, err error, status int, code string) *failure.Error {
	t.Helper()
	var f *failure.Error
	if !errors.As(err, &f) || f.Status != status || f.Code != code {
		t.Fatalf("want %d %s, got %v", status, code, err)
	}
	return f
}

// fieldKeys returns the sorted keys of a validation failure.
func fieldKeys(t *testing.T, err error) []string {
	t.Helper()
	f := failed(t, err, 422, "validation_failed")
	keys := []string{}
	for key := range f.Fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

var withoutPeople = []struct {
	name    string
	session dto.Session
}{
	{"super admin without people", session(policy.RoleSuperAdmin, policy.PermRoles, policy.PermReports)},
	{"PA", session(policy.RolePA, policy.PermBookings, policy.PermCreate)},
	{"role selection pending", dto.Session{Active: true, ID: "7", RoleSelectionRequired: true, AvailableRoles: []string{policy.RoleSuperAdmin}, Permissions: []string{policy.PermPeople}}},
	{"password change pending", dto.Session{Active: true, ID: "7", Role: policy.RoleSuperAdmin, MustChangePassword: true, AvailableRoles: []string{policy.RoleSuperAdmin}, Permissions: []string{policy.PermPeople}}},
}

func TestEmployeeReadsRefuseSessionsWithoutPeopleBeforeAnyRead(t *testing.T) {
	for _, tc := range withoutPeople {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore(entity.Employee{ID: 8, Name: "Staff Example", Email: "staff@example.test", Active: true, Roles: []string{policy.RolePA}, Version: 1})
			u := newUsecase(s, &hasher{})
			_, err := u.GetEmployees(context.Background(), tc.session, employeeusecase.EmployeeQuery{})
			failed(t, err, 403, "forbidden")
			_, err = u.GetEmployee(context.Background(), tc.session, 8)
			failed(t, err, 403, "forbidden")
			if len(s.calls) != 0 {
				t.Fatalf("a refused session reached the repository: %v", s.calls)
			}
		})
	}
}

func TestEmployeeListRefusesInvalidPagingAndSearchBeforeAnyRead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query employeeusecase.EmployeeQuery
		keys  []string
	}{
		{"page zero", employeeusecase.EmployeeQuery{Page: "0"}, []string{"page"}},
		{"negative page", employeeusecase.EmployeeQuery{Page: "-1"}, []string{"page"}},
		{"page not a number", employeeusecase.EmployeeQuery{Page: "two"}, []string{"page"}},
		{"page with a leading zero", employeeusecase.EmployeeQuery{Page: "01"}, []string{"page"}},
		{"page beyond any list", employeeusecase.EmployeeQuery{Page: "10000001"}, []string{"page"}},
		{"page size zero", employeeusecase.EmployeeQuery{PageSize: "0"}, []string{"pageSize"}},
		{"page size 251", employeeusecase.EmployeeQuery{PageSize: "251"}, []string{"pageSize"}},
		{"page size not a number", employeeusecase.EmployeeQuery{PageSize: "25.0"}, []string{"pageSize"}},
		{"both", employeeusecase.EmployeeQuery{Page: "0", PageSize: "0"}, []string{"page", "pageSize"}},
		{"search over 255 characters", employeeusecase.EmployeeQuery{Q: strings.Repeat("a", 256)}, []string{"q"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore()
			_, err := newUsecase(s, &hasher{}).GetEmployees(context.Background(), superAdmin, tc.query)
			if keys := fieldKeys(t, err); !slices.Equal(keys, tc.keys) {
				t.Fatalf("field errors on %v, want %v", keys, tc.keys)
			}
			if len(s.calls) != 0 {
				t.Fatalf("invalid paging reached the repository: %v", s.calls)
			}
		})
	}
}

func TestEmployeeListPassesTheTrimmedSearchAndPagingAndEchoesThem(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query employeeusecase.EmployeeQuery
		want  entity.EmployeeFilter
	}{
		{"defaults", employeeusecase.EmployeeQuery{}, entity.EmployeeFilter{Page: 1, PageSize: 25}},
		{"search and paging", employeeusecase.EmployeeQuery{Q: "  ani@example ", Page: "3", PageSize: "250"}, entity.EmployeeFilter{Query: "ani@example", Page: 3, PageSize: 250}},
		{"blank search", employeeusecase.EmployeeQuery{Q: "   ", PageSize: "1"}, entity.EmployeeFilter{Page: 1, PageSize: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore()
			s.page = entity.EmployeePage{Items: nil, Total: 0, TotalAll: 12}
			out, err := newUsecase(s, &hasher{}).GetEmployees(context.Background(), superAdmin, tc.query)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(s.filters, []entity.EmployeeFilter{tc.want}) {
				t.Fatalf("repository filters %+v, want %+v", s.filters, tc.want)
			}
			// A page past the end is an empty page, never null.
			want := dto.EmployeePage{Items: []dto.Employee{}, Total: 0, TotalAll: 12, Page: tc.want.Page, PageSize: tc.want.PageSize, PasswordMinLength: 15}
			if !reflect.DeepEqual(out, want) {
				t.Fatalf("page %+v, want %+v", out, want)
			}
		})
	}
}

func TestEmployeeReadsProjectProfileRolesAndAssignmentWithoutCredentials(t *testing.T) {
	pa := entity.Employee{ID: 8, Name: "Petugas Contoh", Email: "staff@example.test", JobTitle: "Asisten", Deputy: "Kedeputian Contoh", Directorate: "Direktorat Contoh",
		PhoneNumbers: []string{"+62 811 0000", "0542 123456"}, NIP: "198001012000011001", Active: true, MustChangePassword: true, Version: 4, Roles: []string{policy.RolePA, policy.RoleProtocol}, OfficerIDs: []int32{2147483647}}
	bare := entity.Employee{ID: 9, Name: "Resepsionis Contoh", Email: "front@example.test", Active: false, Version: 1, Roles: []string{policy.RoleReceptionist}}
	s := newStore(pa, bare)
	s.page = entity.EmployeePage{Items: []entity.Employee{pa, bare}, Total: 2, TotalAll: 40}
	u := newUsecase(s, &hasher{})
	wantPA := dto.Employee{ID: "8", Name: "Petugas Contoh", Email: "staff@example.test", Roles: []string{"PA", "Protocol Officer"}, JobTitle: "Asisten", Deputy: "Kedeputian Contoh",
		Directorate: "Direktorat Contoh", Phone: "+62 811 0000", PhoneNumbers: []string{"+62 811 0000", "0542 123456"}, NIP: "198001012000011001", Active: true, AssignedOfficerIDs: []string{"2147483647"}, Version: 4}
	wantBare := dto.Employee{ID: "9", Name: "Resepsionis Contoh", Email: "front@example.test", Roles: []string{"Receptionist"}, PhoneNumbers: []string{}, Active: false, AssignedOfficerIDs: []string{}, Version: 1}
	page, err := u.GetEmployees(context.Background(), superAdmin, employeeusecase.EmployeeQuery{Q: "contoh", Page: "1", PageSize: "2"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(page.Items, []dto.Employee{wantPA, wantBare}) || page.Total != 2 || page.TotalAll != 40 {
		t.Fatalf("page %+v", page)
	}
	one, err := u.GetEmployee(context.Background(), superAdmin, 8)
	if err != nil || !reflect.DeepEqual(one, wantPA) {
		t.Fatalf("detail %+v %v, want %+v", one, err, wantPA)
	}
	// The projection is a copy: changing it cannot reach the stored employee.
	one.Roles[0], one.AssignedOfficerIDs[0], one.PhoneNumbers[0] = "Super Admin", "1", "changed"
	if s.employees[8].Roles[0] != policy.RolePA || s.employees[8].OfficerIDs[0] != 2147483647 || s.employees[8].PhoneNumbers[0] != "+62 811 0000" {
		t.Fatal("the projection shares slices with the entity")
	}
}

func TestAnUnknownEmployeeIsNotFound(t *testing.T) {
	_, err := newUsecase(newStore(), &hasher{}).GetEmployee(context.Background(), superAdmin, 404)
	if !errors.Is(err, failure.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}
