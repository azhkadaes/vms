package bookingusecaseimplementation

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking"
	auth "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	booking "github.com/azhkadaes/vms/backend/internal/usecase/booking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type publicBooking struct {
	repo          bookingrepo.BookingRepository
	serializer    bookingserializer.BookingSerializer
	noticeVersion string
	rateKey       []byte
	tracer        trace.Tracer
}

func NewPublicBookingUsecase(repo bookingrepo.BookingRepository, serializer bookingserializer.BookingSerializer,
	noticeVersion string, rateKey []byte, tracer trace.Tracer) booking.PublicBookingUsecase {
	return &publicBooking{repo: repo, serializer: serializer, noticeVersion: noticeVersion, rateKey: rateKey, tracer: tracer}
}

var publicOwner = security.Hash(entity.AudiencePublic)

func (u *publicBooking) Create(ctx context.Context, ip string, input dto.PublicBookingCreate) (dto.PublicBooking, bool, error) {
	ctx, span := u.tracer.Start(ctx, "PublicBooking.Create")
	defer span.End()
	officerID, start, err := validateCreate(&input)
	if err != nil {
		return dto.PublicBooking{}, false, err
	}
	address, err := requestAddress(ip)
	if err != nil {
		return dto.PublicBooking{}, false, err
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return dto.PublicBooking{}, false, err
	}
	digest := security.Digest(u.rateKey, "booking_payload", string(payload))
	email := security.Digest(u.rateKey, "booking_email", input.Email)
	var output dto.PublicBooking
	var created bool
	err = u.repo.Transaction(ctx, func(tx bookingrepo.BookingTransaction) error {
		if err := tx.LockReference(entity.AudiencePublic, publicOwner, input.IdempotencyKey); err != nil {
			return err
		}
		previous, err := tx.Request(entity.AudiencePublic, publicOwner, input.IdempotencyKey)
		var id int32
		switch {
		case err == nil:
			if previous.State != "committed" || !security.Equal(previous.PayloadDigest, digest) {
				return failure.New(409, "idempotency_conflict")
			}
			id = previous.VisitID
		case errors.Is(err, bookingrepo.ErrNotFound):
			id, err = u.create(tx, input, officerID, start, address, digest, email)
			if err != nil {
				return err
			}
			created = true
		default:
			return err
		}
		output, err = u.project(tx, id)
		return err
	})
	if err != nil {
		return dto.PublicBooking{}, false, err
	}
	return output, created, nil
}

func (u *publicBooking) create(tx bookingrepo.BookingTransaction, in dto.PublicBookingCreate, officerID int32, start time.Time, ip, digest, email string) (int32, error) {
	if u.noticeVersion == "" {
		return 0, failure.New(503, "service_unavailable")
	}
	if in.PrivacyVersion != u.noticeVersion {
		return 0, failure.New(409, "notice_outdated")
	}
	allowed, err := tx.AdmitBooking(security.Digest(u.rateKey, "public_booking", ip))
	if err != nil {
		return 0, failure.New(503, "service_unavailable").WithCause(err)
	}
	if !allowed.Allowed {
		return 0, &failure.Error{Status: 429, Code: "rate_limited", RetryAfter: max(1, allowed.RetryAfter)}
	}
	now, err := tx.Now()
	if err != nil {
		return 0, err
	}
	if !start.After(now) {
		return 0, failure.Validation().Add("date", "Pilih jadwal kunjungan di masa mendatang.").Err()
	}
	_, vip, err := tx.ActiveOfficer(officerID)
	if errors.Is(err, bookingrepo.ErrNotFound) {
		return 0, failure.Validation().Add("officerId", "Pilih pejabat yang aktif.").Err()
	}
	if err != nil {
		return 0, err
	}
	photoID, err := linkUpload(tx, in.VisitorPhotoToken, policy.PurposeVisitorPhoto, "visitorPhotoToken", now)
	if err != nil {
		return 0, err
	}
	var documentID *int32
	if in.AttachmentToken != "" {
		id, err := linkUpload(tx, in.AttachmentToken, policy.PurposeDocument, "attachmentToken", now)
		if err != nil {
			return 0, err
		}
		documentID = &id
	}
	kind := "umum"
	if vip {
		kind = "vip"
	}
	id, err := tx.CreatePublicVisit(entity.Visit{
		OfficerID: officerID, Kind: kind, Date: in.Date, Time: in.Time, Duration: 60, Purpose: in.Purpose,
		Name: in.Name, Organization: in.Organization, Email: in.Email, Phone: in.Phone, IdentityType: in.IdentityType, Visitors: in.Visitors,
	}, photoID, documentID, now, in.PrivacyVersion, ip)
	if err != nil {
		return 0, err
	}
	return id, tx.StoreRequest(entity.AudiencePublic, publicOwner, in.IdempotencyKey,
		entity.BookingRequest{State: "committed", PayloadDigest: digest, EmailDigest: email, VisitID: id})
}

func (u *publicBooking) Resolve(ctx context.Context, input dto.PublicSubmissionProof) (dto.PublicSubmissionResolution, error) {
	ctx, span := u.tracer.Start(ctx, "PublicBooking.Resolve")
	defer span.End()
	email, err := auth.NormalizeEmail(input.Email)
	if err != nil {
		return dto.PublicSubmissionResolution{}, err
	}
	if !validate.Reference(input.IdempotencyKey) {
		return dto.PublicSubmissionResolution{}, failure.Validation().Add("idempotencyKey", "Referensi pengajuan tidak valid.").Err()
	}
	digest := security.Digest(u.rateKey, "booking_email", email)
	var output dto.PublicSubmissionResolution
	err = u.repo.Transaction(ctx, func(tx bookingrepo.BookingTransaction) error {
		if err := tx.LockReference(entity.AudiencePublic, publicOwner, input.IdempotencyKey); err != nil {
			return err
		}
		previous, err := tx.Request(entity.AudiencePublic, publicOwner, input.IdempotencyKey)
		switch {
		case errors.Is(err, bookingrepo.ErrNotFound):
			output.Outcome = "not_committed"
			return tx.StoreRequest(entity.AudiencePublic, publicOwner, input.IdempotencyKey,
				entity.BookingRequest{State: "closed", EmailDigest: digest})
		case err != nil:
			return err
		case !security.Equal(previous.EmailDigest, digest):
			return failure.New(404, "not_found")
		case previous.State == "closed":
			output.Outcome = "not_committed"
			return nil
		default:
			v, err := u.project(tx, previous.VisitID)
			if err != nil {
				return err
			}
			output = dto.PublicSubmissionResolution{Outcome: "committed", Booking: &v}
			return nil
		}
	})
	if err != nil {
		return dto.PublicSubmissionResolution{}, err
	}
	return output, nil
}

func (u *publicBooking) project(tx bookingrepo.BookingTransaction, id int32) (dto.PublicBooking, error) {
	visit, err := tx.ReadVisit(id)
	if err != nil {
		return dto.PublicBooking{}, err
	}
	now, err := tx.Now()
	if err != nil {
		return dto.PublicBooking{}, err
	}
	history, err := tx.PublicHistory(id)
	if err != nil {
		return dto.PublicBooking{}, err
	}
	return u.serializer.ToPublicBooking(visit, now, history), nil
}

func linkUpload(tx bookingrepo.BookingTransaction, token, purpose, field string, now time.Time) (int32, error) {
	id, err := tx.LinkUpload(security.Hash(token), purpose, entity.AudiencePublic, publicOwner, now)
	if errors.Is(err, bookingrepo.ErrNotFound) {
		err = failure.Validation().Add(field, "Unggahan tidak dapat digunakan. Unggah kembali berkas.").Err()
	}
	return id, err
}

func requestAddress(ip string) (string, error) {
	address, err := netip.ParseAddr(ip)
	if err != nil {
		return "", failure.New(503, "service_unavailable").WithCause(errors.New("trusted request address unavailable"))
	}
	return address.Unmap().String(), nil
}

func validateCreate(in *dto.PublicBookingCreate) (int32, time.Time, error) {
	fields := failure.Validation()
	id, err := validate.ID(in.OfficerID)
	if err != nil {
		fields.Add("officerId", "Pilih pejabat yang aktif.")
	}
	start, err := validate.Start(in.Date, in.Time)
	var startError *failure.Error
	if errors.As(err, &startError) {
		for key, values := range startError.Fields {
			for _, message := range values {
				fields.Add(key, message)
			}
		}
	}
	in.Name, in.Organization, in.Purpose = strings.TrimSpace(in.Name), strings.TrimSpace(in.Organization), strings.TrimSpace(in.Purpose)
	checkText := func(key, value string, min, max int, message string) {
		if !validate.Text(value, min, max) || strings.ContainsRune(value, 0) {
			fields.Add(key, message)
		}
	}
	checkText("name", in.Name, 2, 120, "Masukkan nama perwakilan (2–120 karakter).")
	checkText("organization", in.Organization, 2, 255, "Masukkan asal instansi (2–255 karakter).")
	checkText("purpose", in.Purpose, 5, 1000, "Masukkan keperluan (5–1000 karakter).")
	if email, err := auth.NormalizeEmail(in.Email); err == nil {
		in.Email = email
	} else {
		fields.Add("email", "Masukkan alamat email yang valid.")
	}
	if !validate.Digits(in.Phone, 8, 20) {
		fields.Add("phone", "Masukkan 8–20 digit nomor telepon.")
	}
	switch in.IdentityType {
	case "KTP", "Paspor", "SIM":
	default:
		fields.Add("identityType", "Pilih jenis identitas.")
	}
	rosterOK := len(in.Visitors) >= 1 && len(in.Visitors) <= 20
	in.Visitors = slices.Clone(in.Visitors)
	for i, member := range in.Visitors {
		in.Visitors[i] = strings.TrimSpace(member)
		rosterOK = rosterOK && validate.Text(in.Visitors[i], 2, 120) && !strings.ContainsRune(member, 0)
	}
	if !rosterOK || in.Visitors[0] != in.Name {
		fields.Add("visitors", "Masukkan 1–20 nama, dimulai dengan nama perwakilan.")
	}
	if in.VisitKind == "" {
		in.VisitKind = "standard"
	}
	if in.VisitKind != "standard" {
		fields.Add("visitKind", "Pilih kunjungan umum.")
	}
	checkText("visitorPhotoName", in.VisitorPhotoName, 1, 255, "Masukkan nama berkas foto.")
	checkText("visitorPhotoToken", in.VisitorPhotoToken, 16, 4096, "Unggah foto pengunjung.")
	if in.NDName != "" {
		checkText("ndName", in.NDName, 1, 255, "Nama berkas maksimal 255 karakter.")
	}
	if in.AttachmentToken != "" || in.NDName != "" {
		checkText("attachmentToken", in.AttachmentToken, 16, 4096, "Unggah nota dinas.")
	}
	if !in.Ack {
		fields.Add("ack", "Setujui pemberitahuan privasi.")
	}
	if in.PrivacyVersion == "" {
		fields.Add("privacyVersion", "Versi pemberitahuan privasi wajib diisi.")
	}
	if !validate.Reference(in.IdempotencyKey) {
		fields.Add("idempotencyKey", "Referensi pengajuan tidak valid.")
	}
	return id, start, fields.Err()
}
