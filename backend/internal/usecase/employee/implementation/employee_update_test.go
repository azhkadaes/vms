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
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

func set[T any](value T) dto.Optional[T] { return dto.Optional[T]{Set: true, Value: value} }
func null[T any]() dto.Optional[T]       { return dto.Optional[T]{Set: true, Null: true} }

// paEmployee is employee 8: an active PA and Protocol Officer assigned to official 3, at version 4.
func paEmployee() entity.Employee {
	return entity.Employee{ID: 8, Name: "Petugas Contoh", Email: "staff@example.test", JobTitle: "Asisten", Deputy: "Kedeputian Contoh", Directorate: "Direktorat Contoh",
		PhoneNumbers: []string{"+62 811 0000"}, NIP: "198001012000011001", Active: true, MustChangePassword: true, Version: 4,
		Roles: []string{policy.RolePA, policy.RoleProtocol}, OfficerIDs: []int32{3}}
}

// adminEmployee is the caller, employee 7: an active Super Admin.
func adminEmployee() entity.Employee {
	return entity.Employee{ID: admin, Name: "Admin Example", Email: "admin@example.test", Active: true, Version: 2, Roles: []string{policy.RoleSuperAdmin}}
}

func update(s *store, id int32, req dto.EmployeeUpdate) (dto.Employee, error) {
	return newUsecase(s, &hasher{}).UpdateEmployee(context.Background(), superAdmin, adminProof, id, req)
}

func TestUpdatingAnEmployeeRefusesSessionsWithoutPeopleBeforeAnyRead(t *testing.T) {
	for _, tc := range withoutPeople {
		t.Run(tc.name, func(t *testing.T) {
			s := withOfficials(newStore(paEmployee()))
			_, err := newUsecase(s, &hasher{}).UpdateEmployee(context.Background(), tc.session, adminProof, 8, dto.EmployeeUpdate{Active: set(false), Version: 4})
			failed(t, err, 403, "forbidden")
			if len(s.calls) != 0 {
				t.Fatalf("a refused session reached the repository: %v", s.calls)
			}
		})
	}
}

func TestUpdatingAnEmployeeRejectsInvalidFieldsBeforeAnyRead(t *testing.T) {
	long := strings.Repeat("a", 256)
	for _, tc := range []struct {
		name string
		req  dto.EmployeeUpdate
		keys []string
	}{
		{"null name", dto.EmployeeUpdate{Name: null[string](), Version: 4}, []string{"name"}},
		{"one-character name", dto.EmployeeUpdate{Name: set("A"), Version: 4}, []string{"name"}},
		{"null email", dto.EmployeeUpdate{Email: null[string](), Version: 4}, []string{"email"}},
		{"invalid email", dto.EmployeeUpdate{Email: set("staff@"), Version: 4}, []string{"email"}},
		{"no roles", dto.EmployeeUpdate{Roles: []string{}, Version: 4}, []string{"roles"}},
		{"unknown role", dto.EmployeeUpdate{Roles: []string{"Admin"}, Version: 4}, []string{"roles"}},
		{"null active", dto.EmployeeUpdate{Active: null[bool](), Version: 4}, []string{"active"}},
		{"malformed official ID", dto.EmployeeUpdate{AssignedOfficerIDs: []string{"three"}, Version: 4}, []string{"assignedOfficerIds"}},
		{"NIP over 255", dto.EmployeeUpdate{NIP: set(long), Version: 4}, []string{"nip"}},
		{"phone over 255", dto.EmployeeUpdate{Phone: set(long), Version: 4}, []string{"phone"}},
		{"missing version", dto.EmployeeUpdate{Active: set(false)}, []string{"version"}},
		{"negative version", dto.EmployeeUpdate{Active: set(false), Version: -1}, []string{"version"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := withOfficials(newStore(paEmployee()))
			_, err := update(s, 8, tc.req)
			if keys := fieldKeys(t, err); !slices.Equal(keys, tc.keys) {
				t.Fatalf("field errors on %v, want %v", keys, tc.keys)
			}
			if len(s.calls) != 0 {
				t.Fatalf("an invalid request reached the repository: %v", s.calls)
			}
		})
	}
}

func TestUpdatingAnEmployeeRechecksTheCallersAuthorityBeforeReadingTheTarget(t *testing.T) {
	s := withOfficials(newStore(paEmployee()))
	s.authority = liveAuthority(policy.RoleSuperAdmin, policy.PermRoles)
	_, err := update(s, 8, dto.EmployeeUpdate{Active: set(false), Version: 4})
	failed(t, err, 403, "forbidden")
	if !slices.Equal(s.calls, []string{"Transaction", "LockAuthority"}) {
		t.Fatalf("calls after a failed recheck: %v", s.calls)
	}
}

func TestUpdatingAnUnknownEmployeeIsNotFound(t *testing.T) {
	s := newStore(paEmployee())
	_, err := update(s, 404, dto.EmployeeUpdate{Active: set(false), Version: 1})
	if !errors.Is(err, failure.ErrNotFound) || len(s.writes()) != 0 {
		t.Fatalf("want not found without writes, got %v after %v", err, s.calls)
	}
}

func TestUpdatingFromAStaleVersionConflictsWithoutWriting(t *testing.T) {
	for _, version := range []int32{3, 5} {
		s := withOfficials(newStore(paEmployee()))
		_, err := update(s, 8, dto.EmployeeUpdate{Name: set("Nama Baru"), Version: version})
		failed(t, err, 409, "version_conflict")
		if len(s.writes()) != 0 || s.employees[8].Version != 4 {
			t.Fatalf("a stale save wrote: %v", s.calls)
		}
	}
}

func TestAnAdministratorCannotChangeTheirOwnRolesOrDeactivateThemselves(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  dto.EmployeeUpdate
	}{
		{"own role added", dto.EmployeeUpdate{Roles: []string{policy.RoleReceptionist, policy.RoleSuperAdmin}, Version: 2}},
		{"own role replaced", dto.EmployeeUpdate{Roles: []string{policy.RoleProtocol}, Version: 2}},
		{"own deactivation", dto.EmployeeUpdate{Active: set(false), Version: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore(adminEmployee())
			_, err := update(s, admin, tc.req)
			failed(t, err, 403, "forbidden")
			if len(s.writes()) != 0 {
				t.Fatalf("a self-lockout wrote: %v", s.calls)
			}
		})
	}
	// The editor resends the administrator's own unchanged roles and flag with profile edits.
	s := newStore(adminEmployee())
	out, err := update(s, admin, dto.EmployeeUpdate{Name: set("Admin Baru"), Roles: []string{policy.RoleSuperAdmin}, Active: set(true), Version: 2})
	if err != nil || out.Name != "Admin Baru" || out.Version != 3 {
		t.Fatalf("own profile edit: %+v %v", out, err)
	}
}

func TestTheLastActiveSuperAdminCannotBeDeactivatedOrLoseTheRole(t *testing.T) {
	target := entity.Employee{ID: 8, Name: "Last Admin", Email: "last@example.test", Active: true, Version: 1, Roles: []string{policy.RoleSuperAdmin}}
	for _, tc := range []struct {
		name   string
		req    dto.EmployeeUpdate
		others int
		key    string
	}{
		{"deactivating the last one", dto.EmployeeUpdate{Active: set(false), Version: 1}, 0, "active"},
		{"removing the role from the last one", dto.EmployeeUpdate{Roles: []string{policy.RoleReceptionist}, Version: 1}, 0, "roles"},
		{"deactivating one of two", dto.EmployeeUpdate{Active: set(false), Version: 1}, 1, ""},
		{"removing the role from one of two", dto.EmployeeUpdate{Roles: []string{policy.RoleReceptionist}, Version: 1}, 1, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore(target)
			// Only another role granted people can reach the last Super Admin: a Super Admin
			// caller is an active Super Admin too, and their own record is refused as self-lockout.
			s.authority = liveAuthority(policy.RolePA, policy.PermBookings, policy.PermPeople)
			s.otherAdmins = tc.others
			_, err := newUsecase(s, &hasher{}).UpdateEmployee(context.Background(), session(policy.RolePA, policy.PermBookings, policy.PermPeople), adminProof, 8, tc.req)
			if tc.key == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if keys := fieldKeys(t, err); !slices.Equal(keys, []string{tc.key}) {
				t.Fatalf("field errors on %v, want %s", keys, tc.key)
			}
			if len(s.writes()) != 0 {
				t.Fatalf("the last Super Admin was changed: %v", s.calls)
			}
		})
	}
}

func TestUpdatingAnEmployeeRefusesAUsedEmailAndInvalidAssignmentsWithoutWriting(t *testing.T) {
	other := entity.Employee{ID: 9, Name: "Other Staff", Email: "other@example.test", Active: true, Version: 1, Roles: []string{policy.RoleReceptionist}}
	for _, tc := range []struct {
		name       string
		req        dto.EmployeeUpdate
		emailTaken bool
		keys       []string
	}{
		{"another employee's email", dto.EmployeeUpdate{Email: set(" Other@Example.test"), Version: 4}, false, []string{"email"}},
		{"an email taken concurrently", dto.EmployeeUpdate{Email: set("new@example.test"), Version: 4}, true, []string{"email"}},
		{"a newly chosen inactive official", dto.EmployeeUpdate{AssignedOfficerIDs: []string{"2"}, Version: 4}, false, []string{"assignedOfficerIds"}},
		{"a VIP official", dto.EmployeeUpdate{AssignedOfficerIDs: []string{"1"}, Version: 4}, false, []string{"assignedOfficerIds"}},
		{"two officials", dto.EmployeeUpdate{AssignedOfficerIDs: []string{"3", "4"}, Version: 4}, false, []string{"assignedOfficerIds"}},
		{"the PA role removed while the assignment stays", dto.EmployeeUpdate{Roles: []string{policy.RoleProtocol}, Version: 4}, false, []string{"assignedOfficerIds"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := withOfficials(newStore(paEmployee(), other))
			s.emailTaken = tc.emailTaken
			_, err := update(s, 8, tc.req)
			if keys := fieldKeys(t, err); !slices.Equal(keys, tc.keys) {
				t.Fatalf("field errors on %v, want %v", keys, tc.keys)
			}
			if !reflect.DeepEqual(s.employees[8], paEmployee()) || len(s.audits) != 0 {
				t.Fatalf("a refused update committed: %v", s.calls)
			}
		})
	}
}

func TestUpdatingAnEmployeeKeepsOmittedFieldsClearsBlankOnesAndAuditsTheChange(t *testing.T) {
	s := withOfficials(newStore(paEmployee()))
	out, err := update(s, 8, dto.EmployeeUpdate{Name: set(" Nama Baru "), JobTitle: set(""), Phone: null[string](), NIP: set(" 1990 "), Version: 4})
	if err != nil {
		t.Fatal(err)
	}
	after := paEmployee()
	after.Name, after.JobTitle, after.PhoneNumbers, after.NIP, after.Version = "Nama Baru", "", []string{}, "1990", 5
	if !reflect.DeepEqual(s.employees[8], after) {
		t.Fatalf("stored %+v, want %+v", s.employees[8], after)
	}
	want := dto.Employee{ID: "8", Name: "Nama Baru", Email: "staff@example.test", Roles: []string{"PA", "Protocol Officer"}, Deputy: "Kedeputian Contoh",
		Directorate: "Direktorat Contoh", PhoneNumbers: []string{}, NIP: "1990", Active: true, AssignedOfficerIDs: []string{"3"}, Version: 5}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("response %+v, want %+v", out, want)
	}
	before := paEmployee()
	if len(s.audits) != 1 || s.audits[0].Actor != admin || s.audits[0].Action != "employee.updated" || !reflect.DeepEqual(s.audits[0].Before, &before) ||
		!reflect.DeepEqual(s.audits[0].After, after) || !s.audits[0].At.Equal(serverTime) {
		t.Fatalf("audit %+v", s.audits)
	}
	if len(s.ended) != 0 || len(s.reloaded) != 0 {
		t.Fatalf("a profile edit touched sessions: ended %v reloaded %v", s.ended, s.reloaded)
	}
}

func TestEmployeePhoneListsAndLegacyUpdates(t *testing.T) {
	s := newStore()
	s.employees[8] = paEmployee()
	out, err := update(s, 8, dto.EmployeeUpdate{Version: 4, PhoneNumbers: set([]string{" 0812345678 ", "", "+62 542 123456"})})
	if err != nil || out.Phone != "0812345678" || !slices.Equal(out.PhoneNumbers, []string{"0812345678", "+62 542 123456"}) {
		t.Fatalf("save phone list: %+v, %v", out, err)
	}
	out, err = update(s, 8, dto.EmployeeUpdate{Version: out.Version, Phone: set("0812340000")})
	if err != nil || out.Phone != "0812340000" || !slices.Equal(out.PhoneNumbers, []string{"0812340000", "+62 542 123456"}) {
		t.Fatalf("update legacy primary: %+v, %v", out, err)
	}
	out, err = update(s, 8, dto.EmployeeUpdate{Version: out.Version, PhoneNumbers: set([]string{})})
	if err != nil || out.Phone != "" || out.PhoneNumbers == nil || len(out.PhoneNumbers) != 0 {
		t.Fatalf("clear phone list: %+v, %v", out, err)
	}
}

func TestASaveThatChangesNothingWritesNothing(t *testing.T) {
	s := withOfficials(newStore(paEmployee()))
	// The editor resends every field, trimmed and with roles in any order.
	out, err := update(s, 8, dto.EmployeeUpdate{Name: set("Petugas Contoh"), Email: set("STAFF@example.test"), Roles: []string{policy.RoleProtocol, policy.RolePA},
		JobTitle: set("Asisten"), Deputy: set("Kedeputian Contoh"), Directorate: set("Direktorat Contoh"), Phone: set("+62 811 0000"), NIP: set("198001012000011001"),
		Active: set(true), AssignedOfficerIDs: []string{"3"}, Version: 4})
	if err != nil {
		t.Fatal(err)
	}
	if out.Version != 4 || len(s.writes()) != 0 {
		t.Fatalf("an unchanged save wrote %v and answered version %d", s.writes(), out.Version)
	}
}

func TestAccessChangesEndOrReloadTheEmployeesSessions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		req      dto.EmployeeUpdate
		ended    []int32
		reloaded map[int32][]string
		after    func(*entity.Employee)
	}{
		{"deactivation ends them", dto.EmployeeUpdate{Active: set(false), Version: 4}, []int32{8}, map[int32][]string{},
			func(e *entity.Employee) { e.Active = false }},
		{"a role removal reloads them", dto.EmployeeUpdate{Roles: []string{policy.RolePA}, Version: 4}, nil, map[int32][]string{8: {policy.RoleProtocol}},
			func(e *entity.Employee) { e.Roles = []string{policy.RolePA} }},
		{"removing the PA role with its assignment", dto.EmployeeUpdate{Roles: []string{policy.RoleProtocol}, AssignedOfficerIDs: []string{}, Version: 4}, nil,
			map[int32][]string{8: {policy.RolePA}}, func(e *entity.Employee) { e.Roles, e.OfficerIDs = []string{policy.RoleProtocol}, []int32{} }},
		{"a role addition reloads them", dto.EmployeeUpdate{Roles: []string{policy.RoleReceptionist, policy.RolePA, policy.RoleProtocol}, Version: 4}, nil,
			map[int32][]string{8: {}}, func(e *entity.Employee) {
				e.Roles = []string{policy.RolePA, policy.RoleProtocol, policy.RoleReceptionist}
			}},
		{"an assignment change reloads them", dto.EmployeeUpdate{AssignedOfficerIDs: []string{"4"}, Version: 4}, nil, map[int32][]string{8: {}},
			func(e *entity.Employee) { e.OfficerIDs = []int32{4} }},
		{"keeping an official that became inactive", dto.EmployeeUpdate{Name: set("Nama Baru"), AssignedOfficerIDs: []string{"3"}, Version: 4}, nil, map[int32][]string{},
			func(e *entity.Employee) { e.Name = "Nama Baru" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := withOfficials(newStore(paEmployee()))
			if tc.name == "keeping an official that became inactive" {
				s.officers[3] = entity.EmployeeOfficer{ID: 3}
			}
			if _, err := update(s, 8, tc.req); err != nil {
				t.Fatal(err)
			}
			want := paEmployee()
			tc.after(&want)
			want.Version = 5
			if !reflect.DeepEqual(s.employees[8], want) {
				t.Fatalf("stored %+v, want %+v", s.employees[8], want)
			}
			if !slices.Equal(s.ended, tc.ended) || !reflect.DeepEqual(s.reloaded, tc.reloaded) {
				t.Fatalf("ended %v reloaded %v, want %v and %v", s.ended, s.reloaded, tc.ended, tc.reloaded)
			}
		})
	}
}
