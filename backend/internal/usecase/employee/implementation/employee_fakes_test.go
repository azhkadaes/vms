package employeeusecaseimplementation_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	employeerepository "github.com/azhkadaes/vms/backend/internal/repository/employee"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	employeeserializer "github.com/azhkadaes/vms/backend/internal/serializer/employee/implementation"
	authusecaseimplementation "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	employeeusecase "github.com/azhkadaes/vms/backend/internal/usecase/employee"
	implementation "github.com/azhkadaes/vms/backend/internal/usecase/employee/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// serverTime is the database time every fake transaction reports.
var serverTime = time.Date(2026, 11, 19, 2, 0, 0, 0, time.UTC)

// store is a hand-written fake of the employee repository and its transaction. Every call is
// recorded in order; writes made inside a failed transaction are discarded like a rollback.
type store struct {
	employees map[int32]entity.Employee
	officers  map[int32]entity.EmployeeOfficer
	// authority is what LockAuthority returns for the caller.
	authority   entity.SessionSnapshot
	otherAdmins int
	// emailTaken makes the next create or save fail on the unique email constraint.
	emailTaken bool
	page       entity.EmployeePage
	filters    []entity.EmployeeFilter
	calls      []string
	// Committed effects.
	hashes   map[int32]string
	ended    []int32
	reloaded map[int32][]string
	audits   []entity.EmployeeEvent
	nextID   int32
}

func newStore(employees ...entity.Employee) *store {
	s := &store{employees: map[int32]entity.Employee{}, officers: map[int32]entity.EmployeeOfficer{}, hashes: map[int32]string{}, reloaded: map[int32][]string{}, nextID: 100}
	for _, e := range employees {
		s.employees[e.ID] = e
	}
	return s
}

// writes lists the recorded calls that change data.
func (s *store) writes() []string {
	out := []string{}
	for _, call := range s.calls {
		for _, prefix := range []string{"CreateEmployee", "SaveEmployee", "EndSessions", "ReloadSessions", "Audit"} {
			if strings.HasPrefix(call, prefix) {
				out = append(out, call)
			}
		}
	}
	return out
}

func (s *store) GetEmployees(_ context.Context, filter entity.EmployeeFilter) (entity.EmployeePage, error) {
	s.calls = append(s.calls, "GetEmployees")
	s.filters = append(s.filters, filter)
	return s.page, nil
}

func (s *store) GetEmployee(_ context.Context, id int32) (entity.Employee, error) {
	s.calls = append(s.calls, "GetEmployee")
	e, ok := s.employees[id]
	if !ok {
		return entity.Employee{}, employeerepository.ErrNotFound
	}
	return e, nil
}

func (s *store) Transaction(_ context.Context, fn func(employeerepository.EmployeeTransaction) error) error {
	s.calls = append(s.calls, "Transaction")
	tx := &transaction{s: s, employees: map[int32]entity.Employee{}, hashes: map[int32]string{}, reloaded: map[int32][]string{}, nextID: s.nextID}
	for id, e := range s.employees {
		tx.employees[id] = e
	}
	if err := fn(tx); err != nil {
		return err
	}
	s.employees, s.nextID = tx.employees, tx.nextID
	for id, hash := range tx.hashes {
		s.hashes[id] = hash
	}
	s.ended = append(s.ended, tx.ended...)
	for id, roles := range tx.reloaded {
		s.reloaded[id] = roles
	}
	s.audits = append(s.audits, tx.audits...)
	return nil
}

type transaction struct {
	s         *store
	employees map[int32]entity.Employee
	hashes    map[int32]string
	ended     []int32
	reloaded  map[int32][]string
	audits    []entity.EmployeeEvent
	nextID    int32
}

func (t *transaction) LockAuthority(proof entity.SessionProof) (entity.SessionSnapshot, error) {
	t.s.calls = append(t.s.calls, "LockAuthority")
	if proof.EmployeeID != t.s.authority.EmployeeID {
		return entity.SessionSnapshot{}, nil
	}
	return t.s.authority, nil
}

func (t *transaction) LockEmployee(id int32) (entity.Employee, error) {
	t.s.calls = append(t.s.calls, "LockEmployee")
	e, ok := t.employees[id]
	if !ok {
		return entity.Employee{}, employeerepository.ErrNotFound
	}
	return e, nil
}

func (t *transaction) EmailInUse(email string, except int32) (bool, error) {
	t.s.calls = append(t.s.calls, "EmailInUse")
	for id, e := range t.employees {
		if id != except && e.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (t *transaction) Officers(ids []int32) ([]entity.EmployeeOfficer, error) {
	t.s.calls = append(t.s.calls, "Officers")
	out := []entity.EmployeeOfficer{}
	for _, id := range ids {
		if officer, ok := t.s.officers[id]; ok {
			out = append(out, officer)
		}
	}
	return out, nil
}

func (t *transaction) OtherActiveSuperAdmins(except int32) (int, error) {
	t.s.calls = append(t.s.calls, "OtherActiveSuperAdmins")
	return t.s.otherAdmins, nil
}

func (t *transaction) CreateEmployee(e entity.Employee, encodedHash string) (entity.Employee, error) {
	t.s.calls = append(t.s.calls, "CreateEmployee")
	if t.s.emailTaken {
		return entity.Employee{}, employeerepository.ErrEmailTaken
	}
	t.nextID++
	e.ID, e.Version = t.nextID, 1
	t.employees[e.ID] = e
	t.hashes[e.ID] = encodedHash
	return e, nil
}

func (t *transaction) SaveEmployee(before, after entity.Employee) (entity.Employee, error) {
	t.s.calls = append(t.s.calls, "SaveEmployee")
	if t.s.emailTaken {
		return entity.Employee{}, employeerepository.ErrEmailTaken
	}
	if t.employees[before.ID].Version != before.Version {
		return entity.Employee{}, errors.New("fake: save from a stale version")
	}
	after.ID, after.Version = before.ID, before.Version+1
	t.employees[after.ID] = after
	return after, nil
}

func (t *transaction) EndSessions(employeeID int32, at time.Time) error {
	t.s.calls = append(t.s.calls, "EndSessions")
	if !at.Equal(serverTime) {
		return errors.New("fake: sessions ended at a time other than the locked server time")
	}
	t.ended = append(t.ended, employeeID)
	return nil
}

func (t *transaction) ReloadSessions(employeeID int32, removedRoles []string, at time.Time) error {
	t.s.calls = append(t.s.calls, "ReloadSessions")
	if !at.Equal(serverTime) {
		return errors.New("fake: sessions reloaded at a time other than the locked server time")
	}
	t.reloaded[employeeID] = slices.Clone(removedRoles)
	return nil
}

func (t *transaction) Audit(event entity.EmployeeEvent) error {
	t.s.calls = append(t.s.calls, "Audit")
	t.audits = append(t.audits, event)
	return nil
}

// hasher stands in for bcrypt: it records the passwords it hashed.
type hasher struct{ hashed []string }

func (h *hasher) Verify(context.Context, string, string) (bool, error) {
	return false, errors.New("employee administration never verifies passwords")
}

func (h *hasher) Hash(_ context.Context, password string) (string, error) {
	h.hashed = append(h.hashed, password)
	return "bcrypt:hashed-" + password, nil
}

// admin is employee 7: an active Super Admin acting in that role.
const admin int32 = 7

var adminProof = entity.SessionProof{SessionID: "550e8400-e29b-41d4-a716-446655440000", EmployeeID: admin, CredentialGeneration: 1, RoleRevision: 1}

// liveAuthority is the locked snapshot of a ready session acting in role with permissions.
func liveAuthority(role string, permissions ...string) entity.SessionSnapshot {
	return entity.SessionSnapshot{EmployeeID: admin, Name: "Admin Example", AccountActive: true, ActiveRole: role, AvailableRoles: []string{role},
		Permissions: permissions, AssignedOfficerIDs: []int32{}, CredentialGeneration: 1, RoleRevision: 1,
		CreatedAt: serverTime.Add(-time.Hour), LastActivityAt: serverTime.Add(-time.Hour), IdleExpiresAt: serverTime.Add(7 * time.Hour),
		AbsoluteExpiresAt: serverTime.Add(23 * time.Hour), ServerTime: serverTime}
}

// session is the live session StaffAuth hands the handler.
func session(role string, permissions ...string) dto.Session {
	return dto.Session{Active: true, ID: "7", Name: "Admin Example", Role: role, AvailableRoles: []string{role}, RoleRevision: 1,
		Permissions: permissions, AssignedOfficerIDs: []string{}, PasswordMinLength: 15, PasswordMaxBytes: 72}
}

var superAdmin = session(policy.RoleSuperAdmin, policy.PermPeople, policy.PermRoles)

func newUsecase(s *store, h *hasher) employeeusecase.EmployeeUsecase {
	if s.authority.EmployeeID == 0 {
		s.authority = liveAuthority(policy.RoleSuperAdmin, policy.PermPeople, policy.PermRoles)
	}
	return implementation.NewEmployeeUsecase(s, employeeserializer.NewEmployeeSerializer(),
		authusecaseimplementation.NewSessionValidator(authserializer.NewAuthSerializer()), h, noop.NewTracerProvider().Tracer("employee-test"))
}
