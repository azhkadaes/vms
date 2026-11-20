//go:build integration

package bookingrepositoryimplementation_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
	bookingimplementation "github.com/azhkadaes/vms/backend/internal/repository/booking/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking/implementation"
	bookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/booking/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

func TestOneStaticAdmissionQRIsIssuedAndRevokedUnderTheVisitLock(t *testing.T) {
	db := integrationtest.Open(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	start := time.Now().In(validate.WITA).AddDate(0, 0, 1).Truncate(time.Minute)
	random, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
	if err != nil {
		t.Fatal(err)
	}
	code := fmt.Sprintf("%08d", random.Int64())
	// A committed synthetic fixture is visible to the concurrent request transactions.
	var id int32
	if err := db.Transaction(func(tx *gorm.DB) error {
		id = accessFixture(t, tx, code, "umum", "approved", start)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	tracer := noop.NewTracerProvider().Tracer("qr-integration")
	repo := bookingimplementation.NewBookingAccessRepository(db)
	usecase := bookingusecase.NewAdmissionUsecase(repo, []byte(strings.Repeat("q", 32)), tracer)
	proof := dto.PublicCancel{Code: code, Email: "Visitor@Example.test", Version: 1}
	type issueResult struct {
		body dto.Admission
		err  error
	}
	const workers = 8
	issued := make(chan issueResult, workers)
	firstRequest := make(chan struct{})
	for range workers {
		go func() {
			<-firstRequest
			body, err := usecase.Issue(ctx, id, proof)
			issued <- issueResult{body, err}
		}()
	}
	close(firstRequest)
	var payload string
	for range workers {
		got := <-issued
		if got.err != nil {
			t.Fatalf("issue QR: %v", got.err)
		}
		if !got.body.ExpiresAt.Equal(start.Add(time.Hour)) || got.body.ExpiresAt.Location() != time.UTC || len(got.body.Payload) != 71 {
			t.Fatalf("invalid admission result shape: expires=%v, payload length=%d", got.body.ExpiresAt, len(got.body.Payload))
		}
		if payload == "" {
			payload = got.body.Payload
		}
		if got.body.Payload != payload {
			t.Fatal("concurrent issue returned a different credential")
		}
	}
	var row struct {
		Version, Count int
		Hash           string
	}
	if err := db.Raw(`SELECT k.version, count(q.id) AS count, max(q.token_hash) AS hash
		FROM vms.kunjungan k JOIN vms.qr_kunjungan q ON q.id_kunjungan=k.id WHERE k.id=? GROUP BY k.version`, id).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Version != 1 || row.Count != 1 || row.Hash != security.Hash(payload) {
		t.Fatal("QR issue changed the visit version, retained plaintext or wrote multiple rows")
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT id FROM vms.kunjungan WHERE id=? FOR UPDATE`, id).Error; err != nil {
			return err
		}
		return bookingimplementation.RevokeVisitQR(tx, id, time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	replacement, err := usecase.Issue(ctx, id, proof)
	if err != nil || replacement.Payload == payload {
		t.Fatalf("replacement did not get a new payload: %v", err)
	}
	access := bookingusecase.NewBookingAccessUsecase(repo, bookingserializer.NewBookingSerializer(), tracer)
	if _, err := access.Cancel(ctx, id, proof, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	var active int
	if err := db.Raw(`SELECT count(*) FROM vms.qr_kunjungan WHERE id_kunjungan=? AND dicabut_pada IS NULL`, id).Scan(&active).Error; err != nil || active != 0 {
		t.Fatalf("cancel left %d active QR, err=%v", active, err)
	}
	if _, err := usecase.Issue(ctx, id+1, proof); !errors.Is(err, bookingrepo.ErrNotFound) {
		t.Fatalf("mismatched ID did not return a generic not-found error: %v", err)
	}
	assertIssueError := func(input dto.PublicCancel, expected string) {
		t.Helper()
		_, err := usecase.Issue(ctx, id, input)
		if api, ok := errors.AsType[*failure.Error](err); !ok || api.Code != expected {
			t.Fatalf("expected %s, got %v", expected, err)
		}
	}
	assertIssueError(proof, "version_conflict")
	proof.Version++
	assertIssueError(proof, "invalid_state")
}
