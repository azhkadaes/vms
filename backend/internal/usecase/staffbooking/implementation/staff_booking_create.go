package staffbookingusecaseimplementation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
	staffserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking"
	authcontract "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	auth "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	staff "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type creator struct {
	repo      bookingrepo.StaffBookingWriter
	validator authcontract.SessionValidator
	projector staffserializer.StaffBookingSerializer
	digestKey []byte
	tracer    trace.Tracer
}

func NewStaffBookingCreator(repo bookingrepo.StaffBookingWriter, validator authcontract.SessionValidator,
	projector staffserializer.StaffBookingSerializer, digestKey []byte, tracer trace.Tracer) staff.StaffBookingCreateUsecase {
	return &creator{repo: repo, validator: validator, projector: projector, digestKey: digestKey, tracer: tracer}
}

func canStartCreate(session dto.Session, kind string) bool {
	return policy.Has(session, policy.PermCreate) && (session.Role == policy.RoleProtocol ||
		(session.Role == policy.RolePA && kind != "restricted"))
}

func wireKind(kind string) string {
	if kind == "terbatas" {
		return "restricted"
	}
	return "standard"
}

func (u *creator) CreateStaffBooking(ctx context.Context, session dto.Session, proof entity.SessionProof, ip string, input dto.StaffBookingCreate) (dto.StaffBooking, bool, error) {
	ctx, span := u.tracer.Start(ctx, "staff_booking.create")
	defer span.End()
	if !canStartCreate(session, input.VisitKind) {
		return dto.StaffBooking{}, false, failure.New(403, "forbidden")
	}
	draft, err := buildCreate(&input)
	if err != nil {
		return dto.StaffBooking{}, false, err
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return dto.StaffBooking{}, false, err
	}
	payloadDigest := security.Digest(u.digestKey, "booking_payload", string(raw))
	owner := security.Hash(entity.AudienceStaff + ":" + strconv.FormatInt(int64(proof.EmployeeID), 10))
	var result dto.StaffBooking
	var created bool
	err = u.repo.StaffTransaction(ctx, func(tx bookingrepo.StaffBookingTransaction) error {
		snapshot, err := tx.LockAuthority(proof)
		if err != nil {
			return err
		}
		live, err := u.validator.Validate(snapshot, proof)
		if err != nil {
			return err
		}
		if !canStartCreate(live, input.VisitKind) {
			return failure.New(403, "forbidden")
		}
		if err := tx.LockReference(entity.AudienceStaff, owner, input.IdempotencyKey); err != nil {
			return err
		}
		request, err := tx.Request(entity.AudienceStaff, owner, input.IdempotencyKey)
		switch {
		case err == nil:
			if request.State != "committed" || !security.Equal(request.PayloadDigest, payloadDigest) {
				return failure.New(409, "idempotency_conflict")
			}
			now, err := tx.Now()
			if err != nil {
				return err
			}
			visit, err := tx.ReadStaffVisit(request.VisitID)
			if err != nil {
				return err
			}
			if !policy.CanCreate(live, wireKind(visit.Kind), visit.OfficerID, visit.Kind == "vip") {
				return failure.New(403, "forbidden")
			}
			result = u.projector.Project(visit, live, now, false)
			return nil
		case !errors.Is(err, bookingrepo.ErrNotFound):
			return err
		}
		draft.CreatorName, draft.CreatorRole = live.Name, live.Role
		visit, now, err := u.create(tx, live, proof.EmployeeID, ip, draft, input, owner)
		if err != nil {
			return err
		}
		err = tx.StoreRequest(entity.AudienceStaff, owner, input.IdempotencyKey,
			entity.BookingRequest{State: "committed", PayloadDigest: payloadDigest, VisitID: visit.ID})
		if err != nil {
			return err
		}
		created = true
		result = u.projector.Project(visit, live, now, false)
		return nil
	})
	if err != nil {
		return dto.StaffBooking{}, false, fmt.Errorf("create staff booking: %w", err)
	}
	return result, created, nil
}

func (u *creator) create(tx bookingrepo.StaffBookingTransaction, live dto.Session, actor int32, ip string, draft entity.StaffVisit,
	input dto.StaffBookingCreate, owner string) (entity.StaffVisit, time.Time, error) {
	var rooms []int32
	if draft.RoomID != nil {
		rooms = append(rooms, *draft.RoomID)
	}
	if err := tx.LockScheduleResources([]int32{draft.OfficerID}, rooms); err != nil {
		return entity.StaffVisit{}, time.Time{}, err
	}
	now, err := tx.Now()
	if err != nil {
		return entity.StaffVisit{}, time.Time{}, err
	}
	start, ok := policy.StaffScheduleStart(draft.Date, draft.Time, now)
	if !ok {
		return entity.StaffVisit{}, time.Time{}, failure.Validation().Add("date", "Pilih jadwal kunjungan di masa mendatang.").Err()
	}
	_, vip, err := tx.ActiveOfficer(draft.OfficerID)
	if errors.Is(err, bookingrepo.ErrNotFound) || (err == nil && !policy.CanCreate(live, input.VisitKind, draft.OfficerID, vip)) {
		err = failure.Validation().Add("officerId", "Pilih pejabat aktif yang dapat Anda layani.").Err()
	}
	if err != nil {
		return entity.StaffVisit{}, time.Time{}, err
	}
	if draft.RoomID != nil {
		err = tx.ActiveRoom(*draft.RoomID)
		if errors.Is(err, bookingrepo.ErrNotFound) {
			err = failure.Validation().Add("roomId", "Pilih ruangan yang aktif.").Err()
		}
		if err != nil {
			return entity.StaffVisit{}, time.Time{}, err
		}
	}
	conflict, err := tx.ScheduleConflict(0, draft.OfficerID, draft.RoomID, start, draft.Duration)
	if err != nil {
		return entity.StaffVisit{}, time.Time{}, err
	}
	if conflict {
		return entity.StaffVisit{}, time.Time{}, failure.New(409, "schedule_conflict")
	}
	var photoID, documentID *int32
	if input.VisitKind == "standard" {
		draft.Kind = "umum"
		if vip {
			draft.Kind = "vip"
		}
		photoID, err = linkStaffUpload(tx, input.VisitorPhotoToken, policy.PurposeVisitorPhoto, "visitorPhotoToken", owner, now)
		if err != nil {
			return entity.StaffVisit{}, time.Time{}, err
		}
		if input.AttachmentToken != "" {
			documentID, err = linkStaffUpload(tx, input.AttachmentToken, policy.PurposeDocument, "attachmentToken", owner, now)
			if err != nil {
				return entity.StaffVisit{}, time.Time{}, err
			}
		}
	} else {
		draft.Kind = "terbatas"
	}
	id, err := tx.CreateStaffVisit(draft, photoID, documentID, actor, now, ip)
	if err != nil {
		return entity.StaffVisit{}, time.Time{}, err
	}
	result, err := tx.ReadStaffVisit(id)
	return result, now, err
}

func linkStaffUpload(tx bookingrepo.StaffBookingTransaction, token, purpose, field, owner string, now time.Time) (*int32, error) {
	id, err := tx.LinkUpload(security.Hash(token), purpose, entity.AudienceStaff, owner, now)
	if errors.Is(err, bookingrepo.ErrNotFound) {
		err = failure.Validation().Add(field, "Unggahan tidak dapat digunakan. Unggah kembali berkas.").Err()
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func buildCreate(input *dto.StaffBookingCreate) (entity.StaffVisit, error) {
	fields := failure.Validation()
	officer, err := validate.ID(input.OfficerID)
	if err != nil {
		fields.Add("officerId", "Pilih pejabat dari daftar yang tersedia.")
	}
	var roomID *int32
	if input.RoomID != nil {
		room, err := validate.ID(*input.RoomID)
		if err != nil {
			fields.Add("roomId", "Pilih ruangan dari daftar yang tersedia.")
		} else {
			roomID = &room
		}
	}
	if _, err := validate.Start(input.Date, input.Time); err != nil {
		var f *failure.Error
		if errors.As(err, &f) {
			for key, messages := range f.Fields {
				for _, message := range messages {
					fields.Add(key, message)
				}
			}
		}
	}
	if !validate.Reference(input.IdempotencyKey) {
		fields.Add("idempotencyKey", "Referensi pengajuan tidak valid.")
	}
	duration := 60
	if input.Duration != nil {
		duration = *input.Duration
	}
	input.Duration = &duration
	if duration < 15 || duration > 240 {
		fields.Add("duration", "Durasi harus antara 15 hingga 240 menit.")
	}
	if input.VisitKind == "" {
		input.VisitKind = "standard"
	}
	input.Name = createText(fields, "name", input.Name, 2, 120, input.VisitKind == "standard")
	input.Organization = createText(fields, "organization", input.Organization, 2, 255, input.VisitKind == "standard")
	input.VisitorInstructions = createText(fields, "visitorInstructions", input.VisitorInstructions, 1, 2000, false)
	input.CoordinationNote = createText(fields, "coordinationNote", input.CoordinationNote, 1, 2000, false)
	switch input.VisitKind {
	case "standard":
		validateStandard(input, fields)
	case "restricted":
		if input.Purpose != "" || input.Email != "" || input.Phone != "" || input.IdentityType != "" ||
			input.Visitors != nil || input.VisitorPhotoName != "" || input.VisitorPhotoToken != "" ||
			input.NDName != "" || input.AttachmentToken != "" {
			return entity.StaffVisit{}, failure.New(400, "invalid_request")
		}
		input.ServiceID = createText(fields, "serviceId", input.ServiceID, 1, 100, false)
	default:
		fields.Add("visitKind", "Pilih jenis kunjungan yang berlaku.")
	}
	return entity.StaffVisit{
		Visit: entity.Visit{OfficerID: officer, RoomID: roomID, Date: input.Date, Time: input.Time, Duration: duration,
			Name: input.Name, Organization: input.Organization, Purpose: input.Purpose, Email: input.Email, Phone: input.Phone,
			IdentityType: input.IdentityType, Visitors: input.Visitors, Instructions: textPointer(input.VisitorInstructions)},
		ServiceID: textPointer(input.ServiceID), CoordinationNote: textPointer(input.CoordinationNote),
	}, fields.Err()
}

func validateStandard(input *dto.StaffBookingCreate, fields *failure.ValidationBuilder) {
	if input.ServiceID != "" {
		fields.Add("serviceId", "Nomor ID dinas hanya untuk kunjungan terbatas.")
	}
	input.Purpose = createText(fields, "purpose", input.Purpose, 5, 1000, true)
	if value, err := auth.NormalizeEmail(input.Email); err == nil {
		input.Email = value
	} else {
		fields.Add("email", "Masukkan alamat email yang valid.")
	}
	if !validate.Digits(input.Phone, 8, 20) {
		fields.Add("phone", "Masukkan 8–20 digit nomor telepon.")
	}
	switch input.IdentityType {
	case "KTP", "Paspor", "SIM":
	default:
		fields.Add("identityType", "Pilih jenis identitas.")
	}
	validRoster := len(input.Visitors) >= 1 && len(input.Visitors) <= 20
	if validRoster {
		input.Visitors = slices.Clone(input.Visitors)
		for i, member := range input.Visitors {
			member = strings.TrimSpace(member)
			input.Visitors[i] = member
			validRoster = validRoster && validate.Text(member, 2, 120) && !strings.ContainsRune(member, 0)
		}
	}
	if !validRoster || input.Visitors[0] != input.Name {
		fields.Add("visitors", "Masukkan 1–20 nama, dimulai dengan nama perwakilan.")
	}
	input.VisitorPhotoName = createText(fields, "visitorPhotoName", input.VisitorPhotoName, 1, 255, true)
	input.NDName = createText(fields, "ndName", input.NDName, 1, 255, false)
	for _, field := range []struct {
		name, value string
		required    bool
	}{
		{"visitorPhotoToken", input.VisitorPhotoToken, true},
		{"attachmentToken", input.AttachmentToken, input.NDName != ""},
	} {
		if (field.required || field.value != "") && (!validate.Text(field.value, 16, 4096) || strings.ContainsRune(field.value, 0)) {
			fields.Add(field.name, "Unggah berkas yang diperlukan.")
		}
	}
}

func createText(fields *failure.ValidationBuilder, key, value string, min, max int, required bool) string {
	value = strings.TrimSpace(value)
	if (required || value != "") && (!validate.Text(value, min, max) || strings.ContainsRune(value, 0)) {
		fields.Add(key, fmt.Sprintf("Masukkan %d–%d karakter.", min, max))
	}
	return value
}

func textPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
