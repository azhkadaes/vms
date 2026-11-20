package bookingusecaseimplementation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/policy"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking"
	auth "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	bookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/booking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type accessUsecase struct {
	repo       bookingrepo.BookingAccessRepository
	serializer bookingserializer.BookingSerializer
	tracer     trace.Tracer
}

func NewBookingAccessUsecase(repo bookingrepo.BookingAccessRepository, serializer bookingserializer.BookingSerializer, tracer trace.Tracer) bookingusecase.BookingAccessUsecase {
	return &accessUsecase{repo: repo, serializer: serializer, tracer: tracer}
}

func (u *accessUsecase) Lookup(ctx context.Context, input dto.PublicLookup) (dto.PublicBooking, error) {
	ctx, span := u.tracer.Start(ctx, "booking.lookup")
	defer span.End()
	code, email, err := publicCredentials(input.Code, input.Email, nil)
	if err != nil {
		return dto.PublicBooking{}, err
	}
	return u.access(ctx, code, email, nil, 0, "")
}

func (u *accessUsecase) Cancel(ctx context.Context, id int32, input dto.PublicCancel, ip string) (dto.PublicBooking, error) {
	ctx, span := u.tracer.Start(ctx, "booking.cancel_public")
	defer span.End()
	code, email, err := publicCredentials(input.Code, input.Email, &input.Version)
	if err != nil {
		return dto.PublicBooking{}, err
	}
	return u.access(ctx, code, email, &id, input.Version, ip)
}

func publicCredentials(code, email string, version *int) (string, string, error) {
	code = strings.TrimSpace(code)
	invalid := failure.Validation()
	if !validate.Digits(code, 8, 8) {
		invalid.Add("code", "Kode kunjungan harus delapan digit.")
	}
	email, err := auth.NormalizeEmail(email)
	if err != nil {
		invalid.Add("email", "Masukkan alamat email yang valid.")
	}
	if version != nil && (*version <= 0 || int64(*version) > 1<<31-1) {
		invalid.Add("version", "Versi kunjungan tidak valid.")
	}
	return code, email, invalid.Err()
}

func (u *accessUsecase) access(ctx context.Context, code, email string, id *int32, version int, ip string) (dto.PublicBooking, error) {
	var result dto.PublicBooking
	err := u.repo.AccessTransaction(ctx, func(tx bookingrepo.BookingAccessTransaction) error {
		visit, err := tx.PublicVisit(code, email, id)
		if err != nil {
			return err
		}
		now, err := tx.Now()
		if err != nil {
			return fmt.Errorf("read visit time: %w", err)
		}
		if id != nil {
			if visit.Version != version {
				return failure.New(409, "version_conflict")
			}
			start, err := time.ParseInLocation("2006-01-02 15:04", visit.Date+" "+visit.Time, validate.WITA)
			if err != nil {
				return fmt.Errorf("parse stored visit schedule: %w", err)
			}
			if !policy.VisitTiming(visit.Status, start, visit.Duration, now).CanCancel {
				return failure.New(409, "invalid_state")
			}
			visit.Version, err = tx.CancelPublic(visit.ID, now, ip)
			if err != nil {
				return err
			}
			visit.Status = "cancelled"
		}
		history, err := tx.PublicHistory(visit.ID)
		if err != nil {
			return err
		}
		result = u.serializer.ToPublicBooking(visit, now, history)
		return nil
	})
	if err != nil {
		return dto.PublicBooking{}, err
	}
	return result, nil
}
