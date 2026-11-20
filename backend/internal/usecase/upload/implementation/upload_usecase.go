package uploadusecaseimplementation

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/infrastructure/storage"
	"github.com/azhkadaes/vms/backend/internal/policy"
	uploadrepository "github.com/azhkadaes/vms/backend/internal/repository/upload"
	uploadserializer "github.com/azhkadaes/vms/backend/internal/serializer/upload"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	uploadusecase "github.com/azhkadaes/vms/backend/internal/usecase/upload"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

const (
	// tokenValidity is fixed from issue, never extended by a replay (D18).
	tokenValidity = time.Hour
	// uploadAllowance new uploads per trusted IP per allowanceWindow, shared by both audiences (D31).
	uploadAllowance = 60
	allowanceWindow = time.Hour
	// discardTimeout bounds removing the bytes of an upload whose transaction rolled back.
	discardTimeout = 5 * time.Second
)

// Keys are the upload secrets: the dedicated token key and its version (D99), and the
// limiter key that keeps client IPs out of stored rate buckets.
type Keys struct {
	Token   []byte
	Version int16
	Rate    []byte
}

type usecase struct {
	repo       uploadrepository.UploadRepository
	files      storage.Storage
	serializer uploadserializer.UploadSerializer
	validator  authusecase.SessionValidator
	keys       Keys
	tracer     trace.Tracer
}

var _ uploadusecase.UploadUsecase = (*usecase)(nil)

// NewUploadUsecase privately copies the keys so later changes to the caller's slices cannot alter tokens.
func NewUploadUsecase(repo uploadrepository.UploadRepository, files storage.Storage, serializer uploadserializer.UploadSerializer,
	validator authusecase.SessionValidator, keys Keys, tracer trace.Tracer) uploadusecase.UploadUsecase {
	keys.Token, keys.Rate = append([]byte(nil), keys.Token...), append([]byte(nil), keys.Rate...)
	return &usecase{repo: repo, files: files, serializer: serializer, validator: validator, keys: keys, tracer: tracer}
}

// UploadPublic issues anonymous visitor uploads; every visitor shares the public owner.
func (u *usecase) UploadPublic(ctx context.Context, ip string, upload entity.UploadRequest) (uploadusecase.Issued, error) {
	ctx, span := u.tracer.Start(ctx, "upload.public")
	defer span.End()
	return u.upload(ctx, entity.AudiencePublic, nil, ip, upload)
}

// UploadStaff issues uploads owned by the employee whose live authority still allows creating visits.
func (u *usecase) UploadStaff(ctx context.Context, proof entity.SessionProof, ip string, upload entity.UploadRequest) (uploadusecase.Issued, error) {
	ctx, span := u.tracer.Start(ctx, "upload.staff")
	defer span.End()
	return u.upload(ctx, entity.AudienceStaff, &proof, ip, upload)
}

// accepted is an upload that passed every field and content check.
type accepted struct {
	reference, purpose, name, mime, digest string
	content                                []byte
}

func (u *usecase) upload(ctx context.Context, audience string, proof *entity.SessionProof, ip string, in entity.UploadRequest) (uploadusecase.Issued, error) {
	file, err := check(in)
	if err != nil {
		return uploadusecase.Issued{}, err
	}
	address, err := netip.ParseAddr(ip)
	if err != nil {
		return uploadusecase.Issued{}, failure.New(503, "service_unavailable").WithCause(errors.New("trusted request address unavailable"))
	}
	bucket := security.Digest(u.keys.Rate, "upload", address.Unmap().String())
	owner, uploader := security.Hash(entity.AudiencePublic), (*int32)(nil)
	if proof != nil {
		id := proof.EmployeeID
		owner, uploader = security.Hash(entity.AudienceStaff+":"+strconv.FormatInt(int64(id), 10)), &id
	}
	var issued entity.IssuedUpload
	var created, finished bool
	var stored string
	err = u.repo.Transaction(ctx, func(tx uploadrepository.UploadTransaction) error {
		if proof != nil {
			if err := u.authorize(tx, *proof); err != nil {
				return err
			}
		}
		if err := tx.LockReference(audience, owner, file.reference); err != nil {
			return fmt.Errorf("lock upload reference: %w", err)
		}
		now, err := tx.Now()
		if err != nil {
			return fmt.Errorf("read upload time: %w", err)
		}
		existing, err := tx.FindUpload(audience, owner, file.reference)
		switch {
		case err == nil:
			if issued, err = u.replay(existing, file, now); err != nil {
				return err
			}
		case errors.Is(err, uploadrepository.ErrNotFound):
			admission, err := tx.AdmitUpload(bucket, uploadAllowance, allowanceWindow)
			if err != nil {
				return failure.New(503, "service_unavailable").WithCause(fmt.Errorf("admit upload: %w", err))
			}
			if !admission.Allowed {
				return &failure.Error{Status: 429, Code: "rate_limited", RetryAfter: max(1, admission.RetryAfter)}
			}
			if stored, err = u.files.Put(ctx, file.content); err != nil {
				return failure.New(503, "service_unavailable").WithCause(fmt.Errorf("store upload: %w", err))
			}
			record := entity.UploadRecord{Audience: audience, OwnerDigest: owner, Reference: file.reference, Purpose: file.purpose,
				ContentSHA256: file.digest, KeyVersion: u.keys.Version, CreatedAt: now, ExpiresAt: now.Add(tokenValidity), UploadedBy: uploader}
			if issued, err = u.create(tx, file, stored, record); err != nil {
				return err
			}
			created = true
		default:
			return fmt.Errorf("find upload: %w", err)
		}
		finished = true
		return nil
	})
	if err != nil {
		// A failed commit may still have committed; only a rollback makes the stored bytes an orphan.
		if stored != "" && !finished {
			u.discard(ctx, stored)
		}
		return uploadusecase.Issued{}, err
	}
	return uploadusecase.Issued{Result: u.serializer.ToUploadResult(issued), Created: created}, nil
}

// authorize rechecks the staff session on the transaction: live, in the workspace, allowed to create visits.
func (u *usecase) authorize(tx uploadrepository.UploadTransaction, proof entity.SessionProof) error {
	snapshot, err := tx.LockAuthority(proof)
	if err != nil {
		return fmt.Errorf("lock upload authority: %w", err)
	}
	session, err := u.validator.Validate(snapshot, proof)
	if err != nil {
		return err
	}
	if err := authusecase.RequireWorkspace(session); err != nil {
		return err
	}
	if !policy.CanUpload(session) {
		return failure.New(403, "forbidden")
	}
	return nil
}

// replay re-derives the original token while it is valid; other content or purpose, an expired
// token or a retired key version means the reference cannot be replayed.
func (u *usecase) replay(existing entity.UploadRecord, file accepted, now time.Time) (entity.IssuedUpload, error) {
	if existing.ContentSHA256 != file.digest || existing.Purpose != file.purpose || !now.Before(existing.ExpiresAt) || existing.KeyVersion != u.keys.Version {
		return entity.IssuedUpload{}, failure.New(409, "idempotency_conflict")
	}
	token := u.token(existing)
	if !security.Equal(security.Hash(token), existing.TokenHash) {
		return entity.IssuedUpload{}, errors.New("upload token key changed without a new key version")
	}
	return entity.IssuedUpload{Token: token, ExpiresAt: existing.ExpiresAt, ServerTime: now}, nil
}

// create inserts the metadata and lifecycle rows of the stored object; only the token's hash is kept.
func (u *usecase) create(tx uploadrepository.UploadTransaction, file accepted, key string, record entity.UploadRecord) (entity.IssuedUpload, error) {
	id, err := tx.CreateDocument(entity.FormDocument{Name: file.name, StorageKey: key, MimeType: file.mime, Size: int32(len(file.content)), Purpose: file.purpose})
	if err != nil {
		return entity.IssuedUpload{}, fmt.Errorf("create form document: %w", err)
	}
	record.DocumentID = id
	token := u.token(record)
	record.TokenHash = security.Hash(token)
	if err := tx.CreateUpload(record); err != nil {
		return entity.IssuedUpload{}, fmt.Errorf("create upload: %w", err)
	}
	return entity.IssuedUpload{Token: token, ExpiresAt: record.ExpiresAt, ServerTime: record.CreatedAt}, nil
}

// token derives the upload token from the record's fixed columns and the versioned key (D99),
// so a replay regenerates it without any stored copy.
func (u *usecase) token(r entity.UploadRecord) string {
	mac := hmac.New(sha256.New, u.keys.Token)
	_, _ = fmt.Fprintf(mac, "vms.upload.token.v1\x00%d\x00%s\x00%s\x00%s\x00%d", r.DocumentID, r.Audience, r.OwnerDigest, r.Reference, r.CreatedAt.UnixMicro())
	return "vmsu" + strconv.Itoa(int(r.KeyVersion)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// discard removes the bytes of a rolled-back upload even when the request was canceled.
func (u *usecase) discard(ctx context.Context, key string) {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), discardTimeout)
	defer cancel()
	if err := u.files.Delete(cleanup, key); err != nil {
		trace.SpanFromContext(ctx).RecordError(fmt.Errorf("discard rolled-back upload: %w", err))
	}
}

// check applies the field rules and verifies the content against its purpose before any IO.
func check(in entity.UploadRequest) (accepted, error) {
	if len(in.Content) > uploadusecase.MaxFileBytes {
		return accepted{}, failure.New(413, "payload_too_large")
	}
	invalid := failure.Validation()
	if !validate.Reference(in.Reference) {
		invalid.Add("reference", "Referensi unggahan harus UUID yang valid.")
	}
	purposeKnown := policy.UploadPurpose(in.Purpose)
	if !purposeKnown {
		invalid.Add("purpose", "Jenis berkas harus document atau visitorPhoto.")
	}
	var mime string
	switch {
	case len(in.Content) == 0:
		invalid.Add("file", "Berkas tidak boleh kosong.")
	case purposeKnown:
		var problem string
		if mime, problem = inspect(in.Content, in.Purpose); problem != "" {
			invalid.Add("file", problem)
		}
	}
	name, ok := fileName(in.Filename, mime)
	if !ok {
		invalid.Add("file", "Nama berkas maksimal 255 karakter.")
	}
	if err := invalid.Err(); err != nil {
		return accepted{}, err
	}
	sum := sha256.Sum256(in.Content)
	return accepted{reference: in.Reference, purpose: in.Purpose, name: name, mime: mime, digest: hex.EncodeToString(sum[:]), content: in.Content}, nil
}

type disabled struct{}

// NewDisabledUploadUsecase answers 503 while upload storage is not configured, so no client
// ever receives a token for bytes that were not kept.
func NewDisabledUploadUsecase() uploadusecase.UploadUsecase { return disabled{} }

func (disabled) UploadPublic(context.Context, string, entity.UploadRequest) (uploadusecase.Issued, error) {
	return uploadusecase.Issued{}, failure.New(503, "service_unavailable")
}

func (disabled) UploadStaff(context.Context, entity.SessionProof, string, entity.UploadRequest) (uploadusecase.Issued, error) {
	return uploadusecase.Issued{}, failure.New(503, "service_unavailable")
}
