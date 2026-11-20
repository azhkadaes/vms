package bookingusecaseimplementation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/entity"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
	bookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/booking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type admissionUsecase struct {
	repo   bookingrepo.BookingAccessRepository
	key    []byte
	tracer trace.Tracer
}

func NewAdmissionUsecase(repo bookingrepo.BookingAccessRepository, key []byte, tracer trace.Tracer) bookingusecase.AdmissionUsecase {
	return &admissionUsecase{repo: repo, key: append([]byte(nil), key...), tracer: tracer}
}

func (u *admissionUsecase) Issue(ctx context.Context, id int32, input dto.PublicCancel) (dto.Admission, error) {
	ctx, span := u.tracer.Start(ctx, "booking.admission")
	defer span.End()
	if len(u.key) == 0 {
		return dto.Admission{}, failure.New(503, "service_unavailable")
	}
	code, email, err := publicCredentials(input.Code, input.Email, &input.Version)
	if err != nil {
		return dto.Admission{}, err
	}
	var result dto.Admission
	err = u.repo.AccessTransaction(ctx, func(tx bookingrepo.BookingAccessTransaction) error {
		visit, err := tx.PublicVisit(code, email, &id)
		if err != nil {
			return err
		}
		if visit.Version != input.Version {
			return failure.New(409, "version_conflict")
		}
		start, err := time.ParseInLocation("2006-01-02 15:04", visit.Date+" "+visit.Time, validate.WITA)
		if err != nil {
			return fmt.Errorf("parse stored visit schedule: %w", err)
		}
		ends := start.Add(time.Duration(visit.Duration) * time.Minute)
		now, err := tx.Now()
		if err != nil {
			return fmt.Errorf("read admission time: %w", err)
		}
		if visit.Status != "approved" || !now.Before(ends) {
			return failure.New(409, "invalid_state")
		}
		qr, err := tx.CurrentQR(visit.ID)
		if errors.Is(err, bookingrepo.ErrNotFound) {
			qr, err = u.createQR(tx, visit.ID, now, ends)
		}
		if err != nil {
			return err
		}
		payload := u.payload(qr)
		if !qr.ExpiresAt.Equal(ends) || !security.Equal(qr.TokenHash, security.Hash(payload)) {
			return failure.New(503, "service_unavailable").WithCause(errors.New("stored visit QR does not match current configuration or schedule"))
		}
		result = dto.Admission{Payload: payload, ExpiresAt: qr.ExpiresAt.UTC(), ServerTime: now.UnixMilli()}
		return nil
	})
	if err != nil {
		return dto.Admission{}, err
	}
	return result, nil
}

func (u *admissionUsecase) createQR(tx bookingrepo.BookingAccessTransaction, visitID int32, now, ends time.Time) (entity.VisitQR, error) {
	id, err := tx.ReserveQRID()
	if err != nil {
		return entity.VisitQR{}, err
	}
	qr := entity.VisitQR{ID: id, VisitID: visitID, CreatedAt: now, ExpiresAt: ends}
	qr.TokenHash = security.Hash(u.payload(qr))
	if err := tx.InsertQR(qr); err != nil {
		return entity.VisitQR{}, err
	}
	return qr, nil
}

func (u *admissionUsecase) payload(qr entity.VisitQR) string {
	return "vmsqr1." + security.Digest(u.key, "visit-admission-v1", fmt.Sprintf("%d:%d", qr.ID, qr.VisitID))
}
