package bookingfileusecaseimplementation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/infrastructure/storage"
	"github.com/azhkadaes/vms/backend/internal/policy"
	bookingfiles "github.com/azhkadaes/vms/backend/internal/repository/bookingfile"
	fileusecase "github.com/azhkadaes/vms/backend/internal/usecase/bookingfile"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type usecase struct {
	repo   bookingfiles.BookingFileRepository
	files  storage.Storage
	tracer trace.Tracer
}

func NewBookingFileUsecase(repo bookingfiles.BookingFileRepository, files storage.Storage, tracer trace.Tracer) fileusecase.BookingFileUsecase {
	return &usecase{repo: repo, files: files, tracer: tracer}
}

func unavailable() (fileusecase.Opened, error) {
	return fileusecase.Opened{}, failure.New(404, "not_found")
}

func (u *usecase) OpenStaffFile(ctx context.Context, session dto.Session, id int32, purpose string) (fileusecase.Opened, error) {
	ctx, span := u.tracer.Start(ctx, "booking_file.staff_read")
	defer span.End()
	if session.Role == policy.RoleReceptionist {
		return fileusecase.Opened{}, failure.New(403, "forbidden")
	}
	if !policy.Has(session, policy.PermBookings) {
		return unavailable()
	}
	file, err := u.repo.Lookup(ctx, id, purpose, nil)
	if err != nil {
		return lookupError(err)
	}
	if !ordinary(file.Kind) || !policy.CanRead(session, "standard", file.OfficerID, file.Kind == "vip", file.Status) {
		return unavailable()
	}
	return u.open(ctx, purpose, file)
}

func (u *usecase) OpenReceptionFile(ctx context.Context, session dto.Session, id int32, purpose string, request dto.ReceptionResolve) (fileusecase.Opened, error) {
	ctx, span := u.tracer.Start(ctx, "booking_file.reception_read")
	defer span.End()
	if !policy.CanAttend(session, policy.PermCheckin) && !policy.CanAttend(session, policy.PermCheckout) {
		return unavailable()
	}
	credential, err := policy.ReceptionCredential(request.Credential)
	if err != nil {
		return unavailable()
	}
	file, err := u.repo.Lookup(ctx, id, purpose, &credential)
	if err != nil {
		return lookupError(err)
	}
	if !ordinary(file.Kind) || file.Match.VisitID != id || !policy.CanResolveReception(session, file.Status) {
		return unavailable()
	}
	if file.Status == "approved" {
		start, err := time.ParseInLocation("2006-01-02 15:04", file.Date+" "+file.Time, validate.WITA)
		if err != nil || !policy.VisitTiming(file.Status, start, file.Duration, file.ServerTime).CanCheckIn ||
			(file.Match.QRExpiresAt != nil && file.ServerTime.After(*file.Match.QRExpiresAt)) {
			return unavailable()
		}
	}
	return u.open(ctx, purpose, file)
}

func ordinary(kind string) bool { return kind == "umum" || kind == "vip" }

func lookupError(err error) (fileusecase.Opened, error) {
	if errors.Is(err, failure.ErrNotFound) {
		return unavailable()
	}
	return fileusecase.Opened{}, fmt.Errorf("lookup private booking file: %w", err)
}

func (u *usecase) open(ctx context.Context, purpose string, file entity.BookingFile) (fileusecase.Opened, error) {
	const maxFileSize = 5 * 1024 * 1024
	validMIME := purpose == policy.PurposeDocument && file.MIME == "application/pdf" ||
		purpose == policy.PurposeVisitorPhoto && (file.MIME == "image/jpeg" || file.MIME == "image/png")
	if !validMIME || file.Size < 1 || file.Size > maxFileSize {
		return unavailable()
	}
	if u.files == nil {
		return fileusecase.Opened{}, failure.New(503, "service_unavailable")
	}
	content, err := u.files.Open(ctx, file.Key)
	if err != nil {
		return lookupError(err)
	}
	return fileusecase.Opened{MIME: file.MIME, Size: file.Size, Content: content}, nil
}
