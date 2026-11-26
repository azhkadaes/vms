package staffbookingusecaseimplementation

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	staffrepo "github.com/azhkadaes/vms/backend/internal/repository/staffbooking"
	staffserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	staffbooking "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

// now is 08:00 WITA on the day of the fixtures' 10:00 visits.
var now = time.Date(2099, 8, 19, 8, 0, 0, 0, validate.WITA)

// decisionTx is the in-memory adapter behind DecisionRepository.
type decisionTx struct {
	visit         entity.StaffVisit
	inactiveRooms map[int32]bool
	inactive      map[int32]bool
	conflict      bool
	applied       []entity.VisitChange
}

func (tx *decisionTx) LockAuthority(entity.SessionProof) (entity.SessionSnapshot, error) {
	return entity.SessionSnapshot{}, nil
}
func (tx *decisionTx) LockVisit(id int32) (entity.StaffVisit, error) {
	if id != tx.visit.ID {
		return entity.StaffVisit{}, staffrepo.ErrNotFound
	}
	return tx.visit, nil
}
func (tx *decisionTx) Now() (time.Time, error)              { return now, nil }
func (tx *decisionTx) ActiveOfficer(id int32) (bool, error) { return !tx.inactive[id], nil }
func (tx *decisionTx) ActiveRoom(id int32) (bool, error)    { return !tx.inactiveRooms[id], nil }
func (tx *decisionTx) LockResources(_, _ []int32) error     { return nil }
func (tx *decisionTx) HasConflict(int32, entity.VisitSchedule, time.Time) (bool, error) {
	return tx.conflict, nil
}
func (tx *decisionTx) ApplyChange(id int32, version int, _ int32, _ string, _ time.Time, change entity.VisitChange) (entity.StaffVisit, error) {
	tx.applied = append(tx.applied, change)
	next := tx.visit
	next.Status, next.Version = change.Status, version+1
	if s := change.Schedule; s != nil {
		next.OfficerID, next.RoomID, next.Date, next.Time, next.Duration = s.OfficerID, s.RoomID, s.Date, s.Time, s.Duration
	}
	tx.visit = next
	return next, nil
}

type decisionRepo struct{ tx *decisionTx }

func (r decisionRepo) DecisionTransaction(_ context.Context, run func(staffrepo.DecisionTransaction) error) error {
	return run(r.tx)
}

type sessionValidator struct{ session dto.Session }

func (v sessionValidator) Validate(entity.SessionSnapshot, entity.SessionProof) (dto.Session, error) {
	return v.session, nil
}

func decisionsFor(session dto.Session, tx *decisionTx) staffbooking.DecisionUsecase {
	return NewDecisionUsecase(decisionRepo{tx}, sessionValidator{session}, staffserializer.NewStaffBookingSerializer(), noop.NewTracerProvider().Tracer("test"))
}

var staffPermissions = []string{policy.PermBookings, policy.PermCreate, policy.PermApprove, policy.PermReject, policy.PermReschedule, policy.PermCancel}

func staffSession(role string, assigned ...string) dto.Session {
	return dto.Session{Active: true, ID: "1", Name: role, Role: role, Permissions: staffPermissions, AssignedOfficerIDs: assigned}
}

var (
	assignedPA   = staffSession(policy.RolePA, "5")
	unassignedPA = staffSession(policy.RolePA, "6")
	protocol     = staffSession(policy.RoleProtocol)
	receptionist = dto.Session{Active: true, ID: "2", Name: "Desk", Role: policy.RoleReceptionist,
		Permissions: []string{policy.PermBookings, policy.PermCheckin, policy.PermCheckout}}
)

func visit(kind, status, date string) entity.StaffVisit {
	room := int32(3)
	return entity.StaffVisit{Visit: entity.Visit{ID: 42, Code: "00123456", Kind: kind, Status: status, Version: 4,
		OfficerID: 5, OfficerName: "Officer", RoomID: &room, Room: "Room", Date: date, Time: "10:00", Duration: 60,
		Name: "Visitor", Organization: "Agency"}}
}

func failureOf(t *testing.T, err error) *failure.Error {
	t.Helper()
	var target *failure.Error
	if !errors.As(err, &target) {
		t.Fatalf("expected a failure.Error, got %v", err)
	}
	return target
}

func TestApproveRecordsTheScheduleAndAnswersWithTheNextCommands(t *testing.T) {
	tx := &decisionTx{visit: visit("umum", "pending", "2099-08-19")}
	duration := 90
	got, err := decisionsFor(assignedPA, tx).Approve(context.Background(), entity.SessionProof{}, 42, dto.ApproveBooking{
		Version: 4, RoomID: dto.Optional[string]{Set: true, Value: "7"}, Duration: &duration,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.applied) != 1 || tx.applied[0].Status != "approved" || *tx.applied[0].Schedule.RoomID != 7 || tx.applied[0].Schedule.Duration != 90 {
		t.Fatalf("unexpected change: %+v", tx.applied)
	}
	if got.Status != "approved" || got.Version != 5 || !slices.Equal(got.AllowedActions, []string{policy.PermReschedule, policy.PermCancel}) {
		t.Fatalf("unexpected response: status=%s version=%d actions=%v", got.Status, got.Version, got.AllowedActions)
	}
}

func TestDecisionsRefuseWithoutTouchingTheVisit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		session dto.Session
		tx      *decisionTx
		version int
		status  int
		code    string
		field   string
	}{
		{"stale version", assignedPA, &decisionTx{visit: visit("umum", "pending", "2099-08-19")}, 3, 409, "version_conflict", ""},
		{"already decided", assignedPA, &decisionTx{visit: visit("umum", "rejected", "2099-08-19")}, 4, 409, "invalid_state", ""},
		{"schedule conflict", assignedPA, &decisionTx{visit: visit("umum", "pending", "2099-08-19"), conflict: true}, 4, 409, "schedule_conflict", ""},
		{"inactive room", assignedPA, &decisionTx{visit: visit("umum", "pending", "2099-08-19"), inactiveRooms: map[int32]bool{7: true}}, 4, 422, "", "roomId"},
		{"inactive officer", assignedPA, &decisionTx{visit: visit("umum", "pending", "2099-08-19"), inactive: map[int32]bool{5: true}}, 4, 422, "", "officerId"},
		{"Protocol on a general non-VIP visit", protocol, &decisionTx{visit: visit("umum", "pending", "2099-08-19")}, 4, 403, "forbidden", ""},
		{"version out of range", assignedPA, &decisionTx{visit: visit("umum", "pending", "2099-08-19")}, 0, 422, "", "version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decisionsFor(tc.session, tc.tx).Approve(context.Background(), entity.SessionProof{}, 42, dto.ApproveBooking{
				Version: tc.version, RoomID: dto.Optional[string]{Set: true, Value: "7"},
			})
			f := failureOf(t, err)
			if f.Status != tc.status || (tc.code != "" && f.Code != tc.code) || (tc.field != "" && len(f.Fields[tc.field]) == 0) {
				t.Fatalf("got %d %s %v", f.Status, f.Code, f.Fields)
			}
			if len(tc.tx.applied) != 0 {
				t.Fatalf("a refused decision changed the visit: %+v", tc.tx.applied)
			}
		})
	}
	t.Run("visit outside the PA's assignment reads as missing", func(t *testing.T) {
		tx := &decisionTx{visit: visit("umum", "pending", "2099-08-19")}
		_, err := decisionsFor(unassignedPA, tx).Approve(context.Background(), entity.SessionProof{}, 42, dto.ApproveBooking{Version: 4})
		if !errors.Is(err, failure.ErrNotFound) || len(tx.applied) != 0 {
			t.Fatalf("got %v, applied %v", err, tx.applied)
		}
	})
}

func TestRejectAndCancelNeedAReasonAndCancelRevokesTheQR(t *testing.T) {
	tx := &decisionTx{visit: visit("umum", "pending", "2099-08-19")}
	_, err := decisionsFor(assignedPA, tx).Reject(context.Background(), entity.SessionProof{}, 42, dto.RejectBooking{Version: 4, RejectionReason: "no"})
	if f := failureOf(t, err); f.Status != 422 || len(f.Fields["rejectionReason"]) == 0 || len(tx.applied) != 0 {
		t.Fatalf("short rejection reason: %v %v", f.Fields, tx.applied)
	}
	tx = &decisionTx{visit: visit("umum", "approved", "2099-08-19")}
	if _, err := decisionsFor(assignedPA, tx).Cancel(context.Background(), entity.SessionProof{}, 42, dto.CancelBooking{Version: 4, Reason: "Rapat dibatalkan"}); err != nil {
		t.Fatal(err)
	}
	if change := tx.applied[0]; change.Status != "cancelled" || !change.RevokeQR || change.Reason != "Rapat dibatalkan" {
		t.Fatalf("unexpected cancellation: %+v", change)
	}
}

func TestRescheduleWindowAndQRRevocation(t *testing.T) {
	reschedule := func(date string, input dto.RescheduleBooking) (*decisionTx, error) {
		tx := &decisionTx{visit: visit("umum", "approved", date)}
		input.Version = 4
		_, err := decisionsFor(assignedPA, tx).Reschedule(context.Background(), entity.SessionProof{}, 42, input)
		return tx, err
	}
	moved := dto.RescheduleBooking{Date: "2099-08-20", Time: "10:00", Reason: "Penyesuaian agenda"}
	// A visit that ended 2099-08-11 11:00 WITA left its seven-day window before `now`.
	if _, err := reschedule("2099-08-11", moved); failureOf(t, err).Code != "reschedule_window_closed" {
		t.Fatalf("a visit missed over a week ago was rescheduled: %v", err)
	}
	tx, err := reschedule("2099-08-15", moved)
	if err != nil {
		t.Fatal(err)
	}
	if change := tx.applied[0]; change.AuditType != "rescheduled" || !change.RevokeQR || change.Schedule.Date != "2099-08-20" {
		t.Fatalf("moving a visit must revoke its QR: %+v", change)
	}
	if _, err := reschedule("2099-08-19", dto.RescheduleBooking{Date: "2099-08-20", Time: "10:00"}); len(failureOf(t, err).Fields["reason"]) == 0 {
		t.Fatalf("a general visit moved without a reason: %v", err)
	}
	// General visits need a reason for every edit, even one that only changes notes.
	notes := dto.RescheduleBooking{Date: "2099-08-19", Time: "10:00", Reason: "Catatan untuk lobi",
		VisitorInstructions: dto.Optional[string]{Set: true, Value: "Lobi utama"}}
	tx, err = reschedule("2099-08-19", notes)
	if err != nil {
		t.Fatal(err)
	}
	if change := tx.applied[0]; change.AuditType != "updated" || change.RevokeQR {
		t.Fatalf("a notes-only edit must keep the QR: %+v", change)
	}
}

func TestRestrictedUpdateKeepsTheScheduleAndRequiresAnOfficer(t *testing.T) {
	update := func(input dto.UpdateRestrictedBooking) (*decisionTx, error) {
		tx := &decisionTx{visit: visit("terbatas", "approved", "2099-08-19")}
		input.Version = 4
		_, err := decisionsFor(protocol, tx).UpdateRestricted(context.Background(), entity.SessionProof{}, 42, input)
		return tx, err
	}
	if _, err := update(dto.UpdateRestrictedBooking{OfficerID: ""}); len(failureOf(t, err).Fields["officerId"]) == 0 {
		t.Fatalf("restricted visit saved without an officer: %v", err)
	}
	if _, err := update(dto.UpdateRestrictedBooking{OfficerID: "5", Date: dto.Optional[string]{Set: true, Value: "2099-08-20"}}); len(failureOf(t, err).Fields["date"]) == 0 {
		t.Fatalf("restricted update moved the date: %v", err)
	}
	name := "Tamu Terbatas"
	tx, err := update(dto.UpdateRestrictedBooking{OfficerID: "5", Name: dto.Optional[string]{Set: true, Value: name}})
	if err != nil {
		t.Fatal(err)
	}
	if change := tx.applied[0]; change.Status != "approved" || *change.Identity.Name != name {
		t.Fatalf("unexpected restricted update: %+v", change)
	}
}

// TestDecisionsMatchAllowedActions pins ADR-0003: a command succeeds exactly when the
// allowedActions the client was shown include it.
func TestDecisionsMatchAllowedActions(t *testing.T) {
	commands := map[string]func(staffbooking.DecisionUsecase) error{
		policy.PermApprove: func(u staffbooking.DecisionUsecase) error {
			_, err := u.Approve(context.Background(), entity.SessionProof{}, 42, dto.ApproveBooking{Version: 4})
			return err
		},
		policy.PermReject: func(u staffbooking.DecisionUsecase) error {
			_, err := u.Reject(context.Background(), entity.SessionProof{}, 42, dto.RejectBooking{Version: 4, RejectionReason: "Tidak sesuai agenda"})
			return err
		},
		policy.PermCancel: func(u staffbooking.DecisionUsecase) error {
			_, err := u.Cancel(context.Background(), entity.SessionProof{}, 42, dto.CancelBooking{Version: 4, Reason: "Rapat dibatalkan"})
			return err
		},
		policy.PermReschedule: func(u staffbooking.DecisionUsecase) error {
			_, err := u.Reschedule(context.Background(), entity.SessionProof{}, 42, dto.RescheduleBooking{Version: 4, Date: "2099-08-21", Time: "10:00", Reason: "Penyesuaian agenda"})
			return err
		},
		policy.ActionUpdate: func(u staffbooking.DecisionUsecase) error {
			_, err := u.UpdateRestricted(context.Background(), entity.SessionProof{}, 42, dto.UpdateRestrictedBooking{Version: 4, OfficerID: "5"})
			return err
		},
	}
	sessions := map[string]dto.Session{"assigned PA": assignedPA, "unassigned PA": unassignedPA, "Protocol": protocol, "Receptionist": receptionist}
	var visits []entity.StaffVisit
	for _, kind := range []string{"umum", "vip", "terbatas"} {
		for _, status := range []string{"pending", "approved", "rejected", "cancelled", "checked_in", "checked_out"} {
			visits = append(visits, visit(kind, status, "2099-08-19"))
		}
		visits = append(visits, visit(kind, "approved", "2099-08-15"), visit(kind, "approved", "2099-08-11"))
	}
	for sessionName, session := range sessions {
		for _, v := range visits {
			allowed := staffserializer.Project(v, session, now, false).AllowedActions
			for action, run := range commands {
				name := fmt.Sprintf("%s %s on %s %s@%s", sessionName, action, v.Kind, v.Status, v.Date)
				err := run(decisionsFor(session, &decisionTx{visit: v}))
				if want := slices.Contains(allowed, action); (err == nil) != want {
					t.Errorf("%s: allowedActions=%v but the usecase returned %v", name, allowed, err)
				}
			}
		}
	}
}
