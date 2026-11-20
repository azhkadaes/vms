//go:build integration

package bookingrepositoryimplementation_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	uploadrepo "github.com/azhkadaes/vms/backend/internal/repository/upload"
	uploadimpl "github.com/azhkadaes/vms/backend/internal/repository/upload/implementation"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking/implementation"
	booking "github.com/azhkadaes/vms/backend/internal/usecase/booking/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

func TestPublicBookingCommitAndRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db := integrationtest.Open(t)
	// Isolate limiter buckets from other runs and IPs, without changing rate policy.
	key := []byte(uuid.NewString())
	u := booking.NewPublicBookingUsecase(bookingrepo.NewBookingRepository(db), bookingserializer.NewBookingSerializer(),
		"test-notice", key, noop.NewTracerProvider().Tracer("test"))
	var id int32
	if err := db.Raw("SELECT p.id FROM vms.pejabat p JOIN vms.jabatan j ON j.id=p.id_jabatan WHERE p.status_aktif AND j.pejabat_vip ORDER BY p.id LIMIT 1").Scan(&id).Error; err != nil || id == 0 {
		t.Fatal("seed the synthetic active officer before running booking integration tests")
	}
	input := dto.PublicBookingCreate{
		OfficerID: strconv.FormatInt(int64(id), 10), Date: time.Now().In(validate.WITA).AddDate(0, 0, 1).Format("2006-01-02"), Time: "10:00",
		Purpose: "Pertemuan pengujian", Name: "Pengunjung Contoh", Organization: "Instansi Contoh", Email: "visitor@example.test",
		Phone: "081234567890", IdentityType: "KTP", Visitors: []string{"Pengunjung Contoh"},
		VisitorPhotoName: "photo.png", VisitorPhotoToken: fixturePhoto(t, ctx, db), Ack: true, PrivacyVersion: "test-notice", IdempotencyKey: uuid.NewString(),
	}
	type result struct {
		visit   dto.PublicBooking
		created bool
		err     error
	}
	results := make(chan result, 2)
	create := func(in dto.PublicBookingCreate) {
		v, created, err := u.Create(ctx, "127.0.0.10", in)
		results <- result{v, created, err}
	}
	go create(input)
	go create(input)
	a, b := <-results, <-results
	if a.err != nil || b.err != nil || a.created == b.created || a.visit.ID == "" || a.visit.ID != b.visit.ID {
		t.Fatalf("parallel create: %+v / %+v", a, b)
	}
	if a.visit.Status != "pending" || a.visit.Version != 1 || a.visit.Duration != 60 || a.visit.Officer.Name == "" || !a.visit.VIP ||
		!validate.Digits(a.visit.Code, 8, 8) || len(a.visit.Visitors) != 1 || !a.visit.Timing.CanCancel {
		t.Fatalf("incomplete receipt: %+v", a.visit)
	}
	proof := dto.PublicSubmissionProof{IdempotencyKey: input.IdempotencyKey, Email: "VISITOR@example.test"}
	recovered, err := u.Resolve(ctx, proof)
	if err != nil || recovered.Outcome != "committed" || recovered.Booking == nil || recovered.Booking.ID != a.visit.ID ||
		len(recovered.Booking.History) != 1 || recovered.Booking.History[0].Action != "created" {
		t.Fatalf("recover committed: %+v, %v", recovered, err)
	}
	proof.Email = "other@example.test"
	_, err = u.Resolve(ctx, proof)
	requireFailure(t, err, 404, "not_found")
	modified := input
	modified.Purpose = "Changed purpose"
	_, _, err = u.Create(ctx, "127.0.0.10", modified)
	requireFailure(t, err, 409, "idempotency_conflict")

	next := input
	next.IdempotencyKey, next.VisitorPhotoToken = uuid.NewString(), fixturePhoto(t, ctx, db)
	next.AttachmentToken = next.VisitorPhotoToken
	_, _, err = u.Create(ctx, "127.0.0.10", next)
	requireFailure(t, err, 422, "validation_failed")
	next.AttachmentToken = "" // A failed attachment claim must roll back the photo claim.
	go create(next)
	proof = dto.PublicSubmissionProof{IdempotencyKey: next.IdempotencyKey, Email: next.Email}
	resolved, resolveErr := u.Resolve(ctx, proof)
	c := <-results
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	switch resolved.Outcome {
	case "committed":
		if !c.created || c.err != nil || resolved.Booking == nil || resolved.Booking.ID != c.visit.ID {
			t.Fatalf("resolve and create disagree: %+v / %+v", resolved, c)
		}
	case "not_committed":
		requireFailure(t, c.err, 409, "idempotency_conflict")
		next.IdempotencyKey = uuid.NewString()
		if _, created, err := u.Create(ctx, "127.0.0.10", next); !created || err != nil {
			t.Fatalf("failed upload link was not rolled back: %v", err)
		}
	default:
		t.Fatalf("unexpected resolution outcome %q", resolved.Outcome)
	}
}

func fixturePhoto(t *testing.T, ctx context.Context, db *gorm.DB) string {
	t.Helper()
	token := uuid.NewString()
	err := uploadimpl.NewUploadRepository(db).Transaction(ctx, func(tx uploadrepo.UploadTransaction) error {
		now, err := tx.Now()
		if err != nil {
			return err
		}
		id, err := tx.CreateDocument(entity.FormDocument{Name: "photo.png", StorageKey: uuid.NewString(), MimeType: "image/png", Size: 10, Purpose: "visitorPhoto"})
		if err != nil {
			return err
		}
		return tx.CreateUpload(entity.UploadRecord{DocumentID: id, Audience: entity.AudiencePublic, OwnerDigest: security.Hash(entity.AudiencePublic),
			Reference: uuid.NewString(), ContentSHA256: security.Hash("photo"), TokenHash: security.Hash(token), KeyVersion: 1,
			CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func requireFailure(t *testing.T, err error, status int, code string) {
	t.Helper()
	var outcome *failure.Error
	if !errors.As(err, &outcome) || outcome.Status != status || outcome.Code != code {
		t.Fatalf("expected %d %s, got %v", status, code, err)
	}
}
