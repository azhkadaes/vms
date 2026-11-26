package receptionusecaseimplementation

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	receptionrepository "github.com/azhkadaes/vms/backend/internal/repository/reception"
	staffserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	receptionusecase "github.com/azhkadaes/vms/backend/internal/usecase/reception"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

// The fixture visit runs 10:00–11:00 WITA; check-in opens at 09:30.
var start = time.Date(2099, 8, 19, 10, 0, 0, 0, validate.WITA)

// receptionStore is the in-memory adapter behind ReceptionRepository.
type receptionStore struct {
	visit       entity.StaffVisit
	now         time.Time
	qrExpiresAt *time.Time
	transitions []string
}

func (s *receptionStore) match(credential entity.ReceptionCredential, id int32) (entity.ReceptionMatch, error) {
	if id != s.visit.ID || credential.Code != s.visit.Code {
		return entity.ReceptionMatch{}, receptionrepository.ErrNotFound
	}
	return entity.ReceptionMatch{VisitID: id, QRExpiresAt: s.qrExpiresAt}, nil
}
func (s *receptionStore) Resolve(_ context.Context, credential entity.ReceptionCredential) (entity.StaffVisit, entity.ReceptionMatch, time.Time, error) {
	match, err := s.match(credential, s.visit.ID)
	return s.visit, match, s.now, err
}
func (s *receptionStore) Transaction(_ context.Context, run func(receptionrepository.ReceptionTransaction) error) error {
	return run(s)
}
func (s *receptionStore) LockAuthority(entity.SessionProof) (entity.SessionSnapshot, error) {
	return entity.SessionSnapshot{}, nil
}
func (s *receptionStore) LockVisit(id int32) (entity.StaffVisit, error) {
	if id != s.visit.ID {
		return entity.StaffVisit{}, receptionrepository.ErrNotFound
	}
	return s.visit, nil
}
func (s *receptionStore) MatchCredential(credential entity.ReceptionCredential, id int32) (entity.ReceptionMatch, error) {
	return s.match(credential, id)
}
func (s *receptionStore) Now() (time.Time, error) { return s.now, nil }
func (s *receptionStore) Transition(id int32, version int, _ int32, action string, at time.Time) (entity.StaffVisit, error) {
	s.transitions = append(s.transitions, action)
	next := s.visit
	next.Version = version + 1
	if action == policy.AttendanceCheckIn {
		next.Status, next.CheckIn = "checked_in", &at
	} else {
		next.Status, next.CheckOut = "checked_out", &at
	}
	s.visit = next
	return next, nil
}

type sessionValidator struct{ session dto.Session }

func (v sessionValidator) Validate(entity.SessionSnapshot, entity.SessionProof) (dto.Session, error) {
	return v.session, nil
}

var desk = dto.Session{Active: true, ID: "2", Name: "Desk", Role: policy.RoleReceptionist,
	Permissions: []string{policy.PermBookings, policy.PermCheckin, policy.PermCheckout}}

func reception(session dto.Session, store *receptionStore) receptionusecase.ReceptionUsecase {
	return NewReceptionUsecase(store, staffserializer.NewStaffBookingSerializer(), sessionValidator{session}, noop.NewTracerProvider().Tracer("test"))
}

func store(status string, now time.Time) *receptionStore {
	return &receptionStore{now: now, visit: entity.StaffVisit{Visit: entity.Visit{ID: 42, Code: "00123456", Kind: "umum",
		Status: status, Version: 3, OfficerID: 5, OfficerName: "Officer", Date: "2099-08-19", Time: "10:00", Duration: 60,
		Name: "Visitor", Organization: "Agency"}}}
}

func attend(session dto.Session, s *receptionStore, action, credential string, version int) (dto.StaffBooking, error) {
	return reception(session, s).Attendance(context.Background(), entity.SessionProof{}, 42,
		dto.ReceptionAttendance{Action: action, Credential: credential, Version: version})
}

func code(err error) string {
	var f *failure.Error
	if errors.As(err, &f) {
		return f.Code
	}
	if errors.Is(err, failure.ErrNotFound) {
		return "not_found"
	}
	return ""
}

func TestCheckInAndCheckOutFollowTheArrivalWindow(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status string
		at     time.Time
		action string
		want   string
	}{
		{"before the window opens", "approved", start.Add(-31 * time.Minute), policy.AttendanceCheckIn, "invalid_state"},
		{"when the window opens", "approved", start.Add(-30 * time.Minute), policy.AttendanceCheckIn, ""},
		{"at the scheduled end", "approved", start.Add(time.Hour), policy.AttendanceCheckIn, ""},
		{"after the visit ended", "approved", start.Add(time.Hour + time.Millisecond), policy.AttendanceCheckIn, "invalid_state"},
		{"checkout before check-in", "approved", start, policy.AttendanceCheckOut, "invalid_state"},
		{"checkout the next day", "checked_in", start.Add(24 * time.Hour), policy.AttendanceCheckOut, ""},
		{"pending visits stay hidden", "pending", start, policy.AttendanceCheckIn, "not_found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := store(tc.status, tc.at)
			got, err := attend(desk, s, tc.action, "00123456", 3)
			if code(err) != tc.want {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
			if tc.want != "" {
				if len(s.transitions) != 0 {
					t.Fatalf("a refused attendance changed the visit: %v", s.transitions)
				}
				return
			}
			if !slices.Equal(s.transitions, []string{tc.action}) || got.Version != 4 {
				t.Fatalf("unexpected transition %v, version %d", s.transitions, got.Version)
			}
		})
	}
}

func TestAttendanceNeedsTheVisitorsCredentialAndCurrentVersion(t *testing.T) {
	during := start.Add(-10 * time.Minute)
	if _, err := attend(desk, store("approved", during), policy.AttendanceCheckIn, "00999999", 3); code(err) != "not_found" {
		t.Fatalf("another visit's code was accepted: %v", err)
	}
	if _, err := attend(desk, store("approved", during), policy.AttendanceCheckIn, "00123456", 2); code(err) != "version_conflict" {
		t.Fatalf("a stale version was accepted: %v", err)
	}
	expired := store("approved", during)
	before := during.Add(-time.Minute)
	expired.qrExpiresAt = &before
	if _, err := attend(desk, expired, policy.AttendanceCheckIn, "00123456", 3); code(err) != "credential_expired" {
		t.Fatalf("an expired QR checked the visitor in: %v", err)
	}
	expiredInside := store("checked_in", during)
	expiredInside.qrExpiresAt = &before
	if _, err := attend(desk, expiredInside, policy.AttendanceCheckOut, "00123456", 3); err != nil {
		t.Fatalf("an expired QR must still check a visitor out: %v", err)
	}
	if _, err := attend(desk, store("approved", during), "arrive", "00123456", 3); code(err) != "validation_failed" {
		t.Fatalf("an unknown action was accepted: %v", err)
	}
	pa := dto.Session{Active: true, ID: "1", Name: "PA", Role: policy.RolePA, Permissions: []string{policy.PermBookings, policy.PermCheckin}}
	if _, err := attend(pa, store("approved", during), policy.AttendanceCheckIn, "00123456", 3); code(err) != "forbidden" {
		t.Fatalf("a PA recorded attendance: %v", err)
	}
}

func TestResolveFindsOnlyVisitsTheDeskCanActOn(t *testing.T) {
	during := start.Add(-10 * time.Minute)
	resolve := func(s *receptionStore, credential string) (dto.StaffBooking, error) {
		return reception(desk, s).Resolve(context.Background(), desk, dto.ReceptionResolve{Credential: credential})
	}
	got, err := resolve(store("approved", during), "00123456")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Visitor" || !slices.Equal(got.AllowedActions, []string{policy.PermCheckin}) {
		t.Fatalf("arrival projection: name=%q actions=%v", got.Name, got.AllowedActions)
	}
	for name, s := range map[string]*receptionStore{
		"checked out": store("checked_out", during),
		"pending":     store("pending", during),
		"wrong code":  store("approved", during),
		"expired QR": func() *receptionStore {
			s := store("approved", during)
			e := during.Add(-time.Second)
			s.qrExpiresAt = &e
			return s
		}(),
	} {
		credential := "00123456"
		if name == "wrong code" {
			credential = "00999999"
		}
		if _, err := resolve(s, credential); code(err) != "not_found" {
			t.Errorf("%s: got %v", name, err)
		}
	}
}
