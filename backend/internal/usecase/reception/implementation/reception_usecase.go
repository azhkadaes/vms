package receptionusecaseimplementation

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	receptionrepository "github.com/azhkadaes/vms/backend/internal/repository/reception"
	staffbookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	receptionusecase "github.com/azhkadaes/vms/backend/internal/usecase/reception"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type usecase struct {
	repo       receptionrepository.ReceptionRepository
	serializer staffbookingserializer.StaffBookingSerializer
	validator  authusecase.SessionValidator
	tracer     trace.Tracer
}

func NewReceptionUsecase(repo receptionrepository.ReceptionRepository, serializer staffbookingserializer.StaffBookingSerializer, validator authusecase.SessionValidator, tracer trace.Tracer) receptionusecase.ReceptionUsecase {
	return &usecase{repo, serializer, validator, tracer}
}

func (u *usecase) Resolve(ctx context.Context, session dto.Session, req dto.ReceptionResolve) (dto.StaffBooking, error) {
	ctx, span := u.tracer.Start(ctx, "reception.resolve")
	defer span.End()
	if !policy.CanAttend(session, policy.PermCheckin) && !policy.CanAttend(session, policy.PermCheckout) {
		return dto.StaffBooking{}, failure.New(403, "forbidden")
	}
	credential, err := policy.ReceptionCredential(req.Credential)
	if err != nil {
		return dto.StaffBooking{}, err
	}
	visit, match, at, err := u.repo.Resolve(ctx, credential)
	if err != nil {
		return dto.StaffBooking{}, err
	}
	if !policy.CanResolveReception(session, visit.Status) || visit.Status == "approved" && expired(match, at) {
		return dto.StaffBooking{}, receptionrepository.ErrNotFound
	}
	return u.serializer.Project(visit, session, at, true), nil
}

func (u *usecase) Attendance(ctx context.Context, proof entity.SessionProof, id int32, req dto.ReceptionAttendance) (dto.StaffBooking, error) {
	ctx, span := u.tracer.Start(ctx, "reception.attendance")
	defer span.End()
	credential, permission, err := attendanceInput(req)
	if err != nil {
		return dto.StaffBooking{}, err
	}
	var out dto.StaffBooking
	err = u.repo.Transaction(ctx, func(tx receptionrepository.ReceptionTransaction) error {
		snapshot, err := tx.LockAuthority(proof)
		if err != nil {
			return err
		}
		session, err := u.validator.Validate(snapshot, proof)
		if err != nil {
			return err
		}
		if !policy.CanAttend(session, permission) {
			return failure.New(403, "forbidden")
		}
		visit, err := tx.LockVisit(id)
		if err != nil {
			return err
		}
		switch visit.Status {
		case "approved", "checked_in", "checked_out":
		default:
			return receptionrepository.ErrNotFound
		}
		match, err := tx.MatchCredential(credential, id)
		if err != nil {
			return err
		}
		if visit.Version != req.Version {
			return failure.New(409, "version_conflict")
		}
		at, err := tx.Now()
		if err != nil {
			return err
		}
		start, err := time.ParseInLocation("2006-01-02 15:04", visit.Date+" "+visit.Time, validate.WITA)
		if err != nil {
			return errors.New("invalid stored visit schedule")
		}
		timing := policy.VisitTiming(visit.Status, start, visit.Duration, at)
		if err := validateTransition(req.Action, visit.Status, timing, match, at); err != nil {
			return err
		}
		updated, err := tx.Transition(id, req.Version, snapshot.EmployeeID, req.Action, at)
		if err != nil {
			return err
		}
		out = u.serializer.Project(updated, session, at, true)
		return nil
	})
	if err != nil {
		return dto.StaffBooking{}, err
	}
	return out, nil
}

func attendanceInput(req dto.ReceptionAttendance) (entity.ReceptionCredential, string, error) {
	fields := failure.Validation()
	var permission string
	switch req.Action {
	case policy.AttendanceCheckIn:
		permission = policy.PermCheckin
	case policy.AttendanceCheckOut:
		permission = policy.PermCheckout
	default:
		fields.Add("action", "Pilih kedatangan atau kepulangan.")
	}
	if req.Version < 1 || req.Version > 2147483647 {
		fields.Add("version", "Versi kunjungan tidak valid. Muat ulang sebelum melanjutkan.")
	}
	credential, err := policy.ReceptionCredential(req.Credential)
	if validation, ok := errors.AsType[*failure.Error](err); ok {
		for _, msg := range validation.Fields["credential"] {
			fields.Add("credential", msg)
		}
	}
	return credential, permission, fields.Err()
}

func expired(match entity.ReceptionMatch, now time.Time) bool {
	return match.QRExpiresAt != nil && now.After(*match.QRExpiresAt)
}

func validateTransition(action, status string, timing dto.BookingTiming, match entity.ReceptionMatch, now time.Time) error {
	if action == policy.AttendanceCheckIn {
		if status == "approved" && expired(match, now) {
			return failure.New(409, "credential_expired")
		}
		if !timing.CanCheckIn {
			return failure.New(409, "invalid_state")
		}
	} else if !timing.CanCheckOut {
		return failure.New(409, "invalid_state")
	}
	return nil
}
