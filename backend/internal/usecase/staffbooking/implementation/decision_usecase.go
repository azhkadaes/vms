package staffbookingusecaseimplementation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	staffrepo "github.com/azhkadaes/vms/backend/internal/repository/staffbooking"
	staffserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking"
	auth "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	staffbooking "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type decisions struct {
	repo       staffrepo.DecisionRepository
	validator  auth.SessionValidator
	serializer staffserializer.StaffBookingSerializer
	tracer     trace.Tracer
}

func NewDecisionUsecase(repo staffrepo.DecisionRepository, validator auth.SessionValidator, serializer staffserializer.StaffBookingSerializer, tracer trace.Tracer) staffbooking.DecisionUsecase {
	return decisions{repo: repo, validator: validator, serializer: serializer, tracer: tracer}
}

type decision func(staffrepo.DecisionTransaction, entity.StaffVisit, time.Time) (entity.VisitChange, error)

func (u decisions) apply(ctx context.Context, proof entity.SessionProof, id int32, version int, permission string, decide decision) (dto.StaffBooking, error) {
	ctx, span := u.tracer.Start(ctx, "StaffBooking."+permission)
	defer span.End()
	if version <= 0 || int64(version) > 1<<31-1 {
		return dto.StaffBooking{}, field("version", "Versi kunjungan tidak valid.")
	}
	var output dto.StaffBooking
	err := u.repo.DecisionTransaction(ctx, func(tx staffrepo.DecisionTransaction) error {
		snapshot, err := tx.LockAuthority(proof)
		if err != nil {
			return err
		}
		session, err := u.validator.Validate(snapshot, proof)
		if err != nil {
			return err
		}
		if err := auth.RequireWorkspace(session); err != nil {
			return err
		}
		visit, err := tx.LockVisit(id)
		if err != nil {
			return err
		}
		if err := mayDecide(session, visit, permission); err != nil {
			return err
		}
		if visit.Version != version {
			return failure.New(409, "version_conflict")
		}
		now, err := tx.Now()
		if err != nil {
			return err
		}
		change, err := decide(tx, visit, now)
		if err != nil {
			return err
		}
		result, err := tx.ApplyChange(id, version, proof.EmployeeID, session.Role, now, change)
		if err != nil {
			return err
		}
		output = u.serializer.Project(result, session, now, false)
		return nil
	})
	if err != nil {
		return dto.StaffBooking{}, err
	}
	return output, nil
}

func mayDecide(s dto.Session, v entity.StaffVisit, perm string) error {
	kind, vip := visitKind(v), v.Kind == "vip"
	if !policy.CanRead(s, kind, v.OfficerID, vip, v.Status) {
		return staffrepo.ErrNotFound
	}
	allowed := policy.CanDecide(s, kind, v.OfficerID, vip, perm)
	if perm == policy.PermCreate {
		allowed = kind == "restricted" && s.Role == policy.RoleProtocol && policy.Has(s, perm)
	}
	if !allowed {
		return failure.New(403, "forbidden")
	}
	return nil
}

func visitKind(v entity.StaffVisit) string {
	if v.Kind == "terbatas" {
		return "restricted"
	}
	return "standard"
}

func (u decisions) Approve(ctx context.Context, proof entity.SessionProof, id int32, input dto.ApproveBooking) (dto.StaffBooking, error) {
	return u.apply(ctx, proof, id, input.Version, policy.PermApprove, func(tx staffrepo.DecisionTransaction, v entity.StaffVisit, _ time.Time) (entity.VisitChange, error) {
		if v.Status != "pending" {
			return entity.VisitChange{}, failure.New(409, "invalid_state")
		}
		s, err := editSchedule(v, input.RoomID, input.Duration, 60, input.VisitorInstructions, input.CoordinationNote)
		if err != nil {
			return entity.VisitChange{}, err
		}
		start, err := visitStart(v)
		if err != nil {
			return entity.VisitChange{}, err
		}
		if err := validateSchedule(tx, v, s, start, input.RoomID.Set, true); err != nil {
			return entity.VisitChange{}, err
		}
		return entity.VisitChange{Status: "approved", AuditType: "approved", Message: "Kunjungan disetujui", Schedule: &s}, nil
	})
}

func (u decisions) Reject(ctx context.Context, proof entity.SessionProof, id int32, input dto.RejectBooking) (dto.StaffBooking, error) {
	return u.apply(ctx, proof, id, input.Version, policy.PermReject, func(_ staffrepo.DecisionTransaction, v entity.StaffVisit, _ time.Time) (entity.VisitChange, error) {
		if v.Status != "pending" {
			return entity.VisitChange{}, failure.New(409, "invalid_state")
		}
		reason, note, err := decisionReason(input.RejectionReason, input.InternalRejectionNote, "rejectionReason", "internalRejectionNote")
		return entity.VisitChange{Status: "rejected", AuditType: "rejected", Message: "Kunjungan ditolak", Reason: reason, Internal: note}, err
	})
}

func (u decisions) Cancel(ctx context.Context, proof entity.SessionProof, id int32, input dto.CancelBooking) (dto.StaffBooking, error) {
	return u.apply(ctx, proof, id, input.Version, policy.PermCancel, func(_ staffrepo.DecisionTransaction, v entity.StaffVisit, _ time.Time) (entity.VisitChange, error) {
		if v.Status != "pending" && v.Status != "approved" {
			return entity.VisitChange{}, failure.New(409, "invalid_state")
		}
		reason, note, err := decisionReason(input.Reason, input.InternalCancellationNote, "reason", "internalCancellationNote")
		return entity.VisitChange{Status: "cancelled", AuditType: "cancelled", Message: "Kunjungan dibatalkan", Reason: reason, Internal: note, RevokeQR: true}, err
	})
}

func (u decisions) Reschedule(ctx context.Context, proof entity.SessionProof, id int32, input dto.RescheduleBooking) (dto.StaffBooking, error) {
	return u.apply(ctx, proof, id, input.Version, policy.PermReschedule, func(tx staffrepo.DecisionTransaction, v entity.StaffVisit, now time.Time) (entity.VisitChange, error) {
		if v.Status != "approved" {
			return entity.VisitChange{}, failure.New(409, "invalid_state")
		}
		oldStart, err := visitStart(v)
		if err != nil {
			return entity.VisitChange{}, err
		}
		if now.Truncate(time.Millisecond).After(oldStart.Add(time.Duration(v.Duration)*time.Minute + 7*24*time.Hour)) {
			return entity.VisitChange{}, failure.New(409, "reschedule_window_closed")
		}
		s, err := editSchedule(v, input.RoomID, input.Duration, v.Duration, input.VisitorInstructions, input.CoordinationNote)
		if err != nil {
			return entity.VisitChange{}, err
		}
		start, err := validate.Start(input.Date, input.Time)
		if err != nil {
			return entity.VisitChange{}, err
		}
		s.Date, s.Time = input.Date, input.Time
		changed := !start.Equal(oldStart) || s.Duration != v.Duration
		if changed && !start.After(now) {
			return entity.VisitChange{}, field("date", "Pilih jadwal kunjungan di masa mendatang.")
		}
		reason := strings.TrimSpace(input.Reason)
		if (reason != "" || visitKind(v) == "standard") && !checkedText(reason, 5, 1000) {
			return entity.VisitChange{}, field("reason", "Masukkan alasan (5–1000 karakter).")
		}
		if err := validateSchedule(tx, v, s, start, input.RoomID.Set, true); err != nil {
			return entity.VisitChange{}, err
		}
		result := entity.VisitChange{Status: "approved", AuditType: "updated", Message: "Kunjungan diperbarui", Schedule: &s}
		if changed {
			result.AuditType, result.Message, result.Reason, result.RevokeQR = "rescheduled", "Jadwal kunjungan diubah", reason, true
		}
		return result, nil
	})
}

func (u decisions) UpdateRestricted(ctx context.Context, proof entity.SessionProof, id int32, input dto.UpdateRestrictedBooking) (dto.StaffBooking, error) {
	return u.apply(ctx, proof, id, input.Version, policy.PermCreate, func(tx staffrepo.DecisionTransaction, v entity.StaffVisit, _ time.Time) (entity.VisitChange, error) {
		if v.Status != "pending" && v.Status != "approved" {
			return entity.VisitChange{}, failure.New(409, "invalid_state")
		}
		s, err := editSchedule(v, input.RoomID, nil, v.Duration, input.VisitorInstructions, input.CoordinationNote)
		if err != nil {
			return entity.VisitChange{}, err
		}
		fields := failure.Validation()
		if input.VisitKind != "" && input.VisitKind != "restricted" {
			fields.Add("visitKind", "Jenis kunjungan tidak dapat diubah.")
		}
		if input.Date.Set && (input.Date.Null || input.Date.Value != v.Date) {
			fields.Add("date", "Tanggal tidak dapat diubah melalui formulir ini.")
		}
		if input.Time.Set && (input.Time.Null || input.Time.Value != v.Time) {
			fields.Add("time", "Waktu tidak dapat diubah melalui formulir ini.")
		}
		s.OfficerID, err = validate.ID(input.OfficerID)
		if err != nil {
			fields.Add("officerId", "Pilih pejabat yang aktif.")
		}
		s.ServiceID = optionalText(fields, "serviceId", input.ServiceID, v.ServiceID, 1, 100)
		identity := entity.RestrictedIdentity{
			Name:         optionalText(fields, "name", input.Name, textPointer(v.Name), 2, 120),
			Organization: optionalText(fields, "organization", input.Organization, textPointer(v.Organization), 2, 255),
		}
		if err := fields.Err(); err != nil {
			return entity.VisitChange{}, err
		}
		start, err := visitStart(v)
		if err != nil {
			return entity.VisitChange{}, err
		}
		if err := validateSchedule(tx, v, s, start, input.RoomID.Set, v.Status == "approved"); err != nil {
			return entity.VisitChange{}, err
		}
		return entity.VisitChange{Status: v.Status, AuditType: "updated", Message: "Kunjungan terbatas diperbarui", Schedule: &s, Identity: &identity}, nil
	})
}

func editSchedule(v entity.StaffVisit, room dto.Optional[string], duration *int, defaultDuration int, instructions, coordination dto.Optional[string]) (entity.VisitSchedule, error) {
	fields := failure.Validation()
	s := entity.VisitSchedule{OfficerID: v.OfficerID, RoomID: v.RoomID, Date: v.Date, Time: v.Time, Duration: defaultDuration, ServiceID: v.ServiceID}
	if duration != nil {
		s.Duration = *duration
	}
	if s.Duration < 15 || s.Duration > 240 {
		fields.Add("duration", "Durasi kunjungan harus 15–240 menit.")
	}
	if room.Set {
		s.RoomID = nil
		if !room.Null {
			id, err := validate.ID(room.Value)
			if err != nil {
				fields.Add("roomId", "Pilih ruangan yang aktif atau kosongkan pilihan.")
			} else {
				s.RoomID = &id
			}
		}
	}
	s.Instructions = optionalText(fields, "visitorInstructions", instructions, v.Instructions, 1, 2000)
	s.Coordination = optionalText(fields, "coordinationNote", coordination, v.CoordinationNote, 1, 2000)
	return s, fields.Err()
}

func validateSchedule(tx staffrepo.DecisionTransaction, v entity.StaffVisit, s entity.VisitSchedule, start time.Time, roomChanged, checkConflict bool) error {
	rooms := make([]int32, 0, 2)
	for _, room := range []*int32{v.RoomID, s.RoomID} {
		if room != nil {
			rooms = append(rooms, *room)
		}
	}
	if err := tx.LockResources([]int32{v.OfficerID, s.OfficerID}, rooms); err != nil {
		return err
	}
	active, err := tx.ActiveOfficer(s.OfficerID)
	if err != nil {
		return err
	}
	if !active {
		return field("officerId", "Pilih pejabat yang aktif.")
	}
	if roomChanged && s.RoomID != nil {
		active, err := tx.ActiveRoom(*s.RoomID)
		if err != nil {
			return err
		}
		if !active {
			return field("roomId", "Pilih ruangan yang aktif.")
		}
	}
	if checkConflict {
		conflict, err := tx.HasConflict(v.ID, s, start)
		if err != nil {
			return err
		}
		if conflict {
			return failure.New(409, "schedule_conflict")
		}
	}
	return nil
}

func visitStart(v entity.StaffVisit) (time.Time, error) {
	start, err := time.ParseInLocation("2006-01-02 15:04", v.Date+" "+v.Time, validate.WITA)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid stored visit date or time: %w", err)
	}
	return start, nil
}

func decisionReason(reason, note, reasonKey, noteKey string) (string, *string, error) {
	fields := failure.Validation()
	reason = strings.TrimSpace(reason)
	if !checkedText(reason, 5, 1000) {
		fields.Add(reasonKey, "Masukkan alasan (5–1000 karakter).")
	}
	optional := optionalText(fields, noteKey, dto.Optional[string]{Set: true, Value: note}, nil, 1, 2000)
	return reason, optional, fields.Err()
}

func optionalText(fields *failure.ValidationBuilder, key string, change dto.Optional[string], previous *string, min, max int) *string {
	if !change.Set {
		return previous
	}
	value := strings.TrimSpace(change.Value)
	if change.Null || (value != "" && !checkedText(value, min, max)) {
		fields.Add(key, fmt.Sprintf("Masukkan nilai yang valid (maksimal %d karakter).", max))
	}
	return textPointer(value)
}

func checkedText(value string, min, max int) bool {
	return validate.Text(value, min, max) && !strings.ContainsRune(value, 0)
}

func field(key, message string) error { return failure.Validation().Add(key, message).Err() }
