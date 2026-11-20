//go:build integration

package uploadrepositoryimplementation_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/infrastructure/storage"
	storageimplementation "github.com/azhkadaes/vms/backend/internal/infrastructure/storage/implementation"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authrepositoryimplementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	implementation "github.com/azhkadaes/vms/backend/internal/repository/upload/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	uploadserializer "github.com/azhkadaes/vms/backend/internal/serializer/upload/implementation"
	authusecaseimplementation "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	uploadusecase "github.com/azhkadaes/vms/backend/internal/usecase/upload"
	uploadusecaseimplementation "github.com/azhkadaes/vms/backend/internal/usecase/upload/implementation"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
)

// pdf builds a minimal well-formed PDF; body makes each file's content distinct.
func pdf(body string) []byte {
	head := "%PDF-1.7\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [] /Count 0 /Note (" + body + ") >>\nendobj\n"
	return []byte(head + "xref\n0 3\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \ntrailer\n<< /Size 3 /Root 1 0 R >>\nstartxref\n" + fmt.Sprint(len(head)) + "\n%%EOF\n")
}

func randomKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

// uploads wires the real usecase over db and a fresh private storage directory.
func uploads(t *testing.T, db *gorm.DB, rateKey []byte) (uploadusecase.UploadUsecase, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "uploads")
	files, err := storageimplementation.NewLocalStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	return withStorage(db, files, rateKey), dir
}

func withStorage(db *gorm.DB, files storage.Storage, rateKey []byte) uploadusecase.UploadUsecase {
	validator := authusecaseimplementation.NewSessionValidator(authserializer.NewAuthSerializer())
	keys := uploadusecaseimplementation.Keys{Token: []byte(strings.Repeat("t", 32)), Version: 1, Rate: rateKey}
	return uploadusecaseimplementation.NewUploadUsecase(implementation.NewUploadRepository(db), files, uploadserializer.NewUploadSerializer(), validator, keys, noop.NewTracerProvider().Tracer("upload-integration"))
}

func objects(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func document(reference string, content []byte) entity.UploadRequest {
	return entity.UploadRequest{Reference: reference, Purpose: policy.PurposeDocument, Filename: "nota dinas.pdf", Content: content}
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// The invariant only row locks give: concurrent first uploads of one reference store one object.
// These requests commit, so each run leaves one synthetic, already expiring public upload behind
// (the runtime role cannot delete uploads); its rate bucket is removed.
func TestParallelUploadsWithOneReferenceStoreOneObjectAndShareOneToken(t *testing.T) {
	db := integrationtest.Open(t)
	rateKey := randomKey(t)
	dir := filepath.Join(t.TempDir(), "uploads")
	files, err := storageimplementation.NewLocalStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Slow storage keeps the first upload in flight while every other request arrives.
	u := withStorage(db, slow{files}, rateKey)
	reference := uuid.NewString()
	content := pdf("parallel " + reference)
	ip := "198.51.100.77"
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		bucket := security.Digest(rateKey, "upload", ip)
		if err := db.WithContext(ctx).Exec("DELETE FROM support.rate_limit_buckets WHERE bucket_key = ?", bucket).Error; err != nil {
			t.Error("could not remove this test's rate bucket")
		}
	})
	const workers = 8
	type outcome struct {
		issued uploadusecase.Issued
		err    error
	}
	results := make(chan outcome, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			issued, err := u.UploadPublic(context.Background(), ip, document(reference, content))
			results <- outcome{issued, err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	created, tokens := 0, map[string]bool{}
	for result := range results {
		if result.err != nil {
			t.Fatalf("parallel upload failed: %v", result.err)
		}
		if result.issued.Created {
			created++
		}
		tokens[result.issued.Result.Token] = true
	}
	if created != 1 || len(tokens) != 1 {
		t.Fatalf("%d creations and %d distinct tokens, want 1 and 1", created, len(tokens))
	}
	if stored := objects(t, dir); len(stored) != 1 {
		t.Fatalf("storage holds %v, want one object", stored)
	}
	var rows, events int64
	if err := db.Raw("SELECT count(*) FROM support.form_document_uploads WHERE reference = ?::uuid", reference).Scan(&rows).Error; err != nil || rows != 1 {
		t.Fatalf("%d upload rows for the reference (%v)", rows, err)
	}
	if err := db.Raw("SELECT count(*) FROM support.rate_limit_events WHERE bucket_key = ?", security.Digest(rateKey, "upload", ip)).Scan(&events).Error; err != nil || events != 1 {
		t.Fatalf("replays consumed upload slots: %d events (%v)", events, err)
	}
}

func TestAReplayRederivesTheTokenFromTheStoredRecordWhichKeepsOnlyItsHash(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	u, dir := uploads(t, tx, randomKey(t))
	reference := uuid.NewString()
	content := pdf("replay")
	first, err := u.UploadPublic(context.Background(), "203.0.113.10", document(reference, content))
	if err != nil {
		t.Fatal(err)
	}
	second, err := u.UploadPublic(context.Background(), "203.0.113.10", document(reference, content))
	if err != nil {
		t.Fatal(err)
	}
	if !first.Created || second.Created || second.Result.Token != first.Result.Token || !second.Result.ExpiresAt.Equal(first.Result.ExpiresAt) {
		t.Fatalf("first %+v, replay %+v", first, second)
	}
	var row struct {
		Nama, StoragePath, MimeType, Purpose, Audience, TokenHash, ContentSha256 string
		UkuranFileBytes                                                          int32
		KeyVersion                                                               int16
		CreatedAt, ExpiresAt                                                     time.Time
		UploadedBy                                                               *int32
	}
	if err := tx.Raw(`SELECT d.nama, d.storage_path, d.mime_type, d.purpose, d.ukuran_file_bytes, u.audience, u.token_hash, u.content_sha256 AS content_sha256,
		u.key_version, u.created_at, u.expires_at, u.uploaded_by
		FROM support.form_document_uploads u JOIN vms.form_documents d ON d.id = u.id_form_documents WHERE u.reference = ?::uuid`, reference).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	stored := objects(t, dir)
	sum := sha256.Sum256(content)
	if len(stored) != 1 || row.StoragePath != stored[0] || row.Nama != "nota dinas.pdf" || row.MimeType != "application/pdf" || row.Purpose != policy.PurposeDocument ||
		row.UkuranFileBytes != int32(len(content)) || row.Audience != entity.AudiencePublic || row.UploadedBy != nil || row.KeyVersion != 1 ||
		row.ContentSha256 != hex.EncodeToString(sum[:]) || !row.ExpiresAt.Equal(row.CreatedAt.Add(time.Hour)) || !row.ExpiresAt.Equal(first.Result.ExpiresAt) {
		t.Fatalf("stored metadata %+v, objects %v", row, stored)
	}
	if row.TokenHash != tokenHash(first.Result.Token) {
		t.Fatal("the stored hash is not the token's SHA-256")
	}
	var copies int64
	if err := tx.Raw(`SELECT count(*) FROM support.form_document_uploads u JOIN vms.form_documents d ON d.id = u.id_form_documents
		WHERE position(? IN row_to_json(u)::text || row_to_json(d)::text) > 0`, first.Result.Token[len("vmsu1."):]).Scan(&copies).Error; err != nil || copies != 0 {
		t.Fatalf("the token itself is stored (%d rows, %v)", copies, err)
	}
	if _, err := u.UploadPublic(context.Background(), "203.0.113.10", document(reference, pdf("changed"))); !isFailure(err, 409, "idempotency_conflict") {
		t.Fatalf("changed content on a used reference: %v", err)
	}
}

func TestTheSixtyFirstNewUploadFromOneIPIsRefusedAndReplaysDoNotCount(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	u, dir := uploads(t, tx, randomKey(t))
	const ip = "203.0.113.61"
	first := uuid.NewString()
	for i := range 60 {
		reference := first
		if i > 0 {
			reference = uuid.NewString()
		}
		if _, err := u.UploadPublic(context.Background(), ip, document(reference, pdf(reference))); err != nil {
			t.Fatalf("upload %d: %v", i+1, err)
		}
		if i == 0 {
			// A replay before the limit takes no slot either.
			if issued, err := u.UploadPublic(context.Background(), ip, document(first, pdf(first))); err != nil || issued.Created {
				t.Fatalf("early replay: %+v %v", issued, err)
			}
		}
	}
	if issued, err := u.UploadPublic(context.Background(), ip, document(first, pdf(first))); err != nil || issued.Created {
		t.Fatalf("replay at the limit: %+v %v", issued, err)
	}
	_, err := u.UploadPublic(context.Background(), ip, document(uuid.NewString(), pdf("sixty-first")))
	var refused *failure.Error
	if !errors.As(err, &refused) || refused.Status != 429 || refused.Code != "rate_limited" || refused.RetryAfter < 1 || refused.RetryAfter > 3600 {
		t.Fatalf("61st upload: %v", err)
	}
	if stored := objects(t, dir); len(stored) != 60 {
		t.Fatalf("%d stored objects, want 60", len(stored))
	}
}

// slow delays each stored object, widening the window in which concurrent uploads overlap.
type slow struct{ storage.Storage }

func (s slow) Put(ctx context.Context, content []byte) (string, error) {
	time.Sleep(300 * time.Millisecond)
	return s.Storage.Put(ctx, content)
}

// keyCase stores objects normally but reports their keys upper-cased, which the storage_path
// check rejects, so the metadata insert fails after the bytes are already stored.
type keyCase struct{ storage.Storage }

func (s keyCase) Put(ctx context.Context, content []byte) (string, error) {
	key, err := s.Storage.Put(ctx, content)
	return strings.ToUpper(key), err
}
func (s keyCase) Delete(ctx context.Context, key string) error {
	return s.Storage.Delete(ctx, strings.ToLower(key))
}

func TestADatabaseFailureAfterStoringLeavesNoRowsNoBytesAndNoToken(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	dir := filepath.Join(t.TempDir(), "uploads")
	files, err := storageimplementation.NewLocalStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	reference := uuid.NewString()
	issued, err := withStorage(tx, keyCase{files}, randomKey(t)).UploadPublic(context.Background(), "203.0.113.11", document(reference, pdf("fails")))
	if err == nil || issued.Result.Token != "" {
		t.Fatalf("issued %+v after a failed insert", issued)
	}
	if stored := objects(t, dir); len(stored) != 0 {
		t.Fatalf("bytes of a rolled-back upload remain: %v", stored)
	}
	var rows int64
	if err := tx.Raw("SELECT count(*) FROM support.form_document_uploads WHERE reference = ?::uuid", reference).Scan(&rows).Error; err != nil || rows != 0 {
		t.Fatalf("rolled-back upload rows: %d (%v)", rows, err)
	}
}

// signedIn creates an employee with role selected and returns the proof of its live session.
func signedIn(t *testing.T, tx *gorm.DB, role string) entity.SessionProof {
	t.Helper()
	var employee, roleID int32
	if err := tx.Raw(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password) VALUES ('Upload Staff',?,TRUE,FALSE) RETURNING id`,
		"upload-"+uuid.NewString()+"@example.test").Scan(&employee).Error; err != nil || employee <= 0 {
		t.Fatalf("cannot create employee: %v", err)
	}
	if err := tx.Raw(`INSERT INTO vms.role_pegawai(id_karyawan,id_role) SELECT ?, id FROM vms.role WHERE nama = ? RETURNING id_role`, employee, role).Scan(&roleID).Error; err != nil || roleID <= 0 {
		t.Fatalf("cannot assign role: %v", err)
	}
	sessionID := uuid.NewString()
	if err := authrepositoryimplementation.NewSessionRepository(tx).Transaction(context.Background(), employee, func(s authrepository.SessionTransaction) error {
		now := s.Now()
		return s.Create(entity.StaffSession{ID: sessionID, EmployeeID: employee, ActiveRoleID: &roleID, CredentialGeneration: 1, RoleRevision: 1,
			RefreshHash: tokenHash(sessionID), CreatedAt: now, LastActivityAt: now, IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(2 * time.Hour)})
	}); err != nil {
		t.Fatalf("cannot create session: %v", err)
	}
	return entity.SessionProof{SessionID: sessionID, EmployeeID: employee, CredentialGeneration: 1, RoleRevision: 1}
}

func TestStaffUploadsRecordTheirUploaderOnlyWhenTheirRoleMayCreateVisits(t *testing.T) {
	tx := integrationtest.RolledBack(t, integrationtest.Open(t))
	u, dir := uploads(t, tx, randomKey(t))
	pa := signedIn(t, tx, policy.RolePA)
	reference := uuid.NewString()
	if _, err := u.UploadStaff(context.Background(), pa, "203.0.113.12", document(reference, pdf("staff"))); err != nil {
		t.Fatal(err)
	}
	var uploader *int32
	if err := tx.Raw("SELECT uploaded_by FROM support.form_document_uploads WHERE audience = 'staff' AND reference = ?::uuid", reference).Scan(&uploader).Error; err != nil || uploader == nil || *uploader != pa.EmployeeID {
		t.Fatalf("uploader %v, want %d (%v)", uploader, pa.EmployeeID, err)
	}
	receptionist := signedIn(t, tx, policy.RoleReceptionist)
	if _, err := u.UploadStaff(context.Background(), receptionist, "203.0.113.12", document(uuid.NewString(), pdf("denied"))); !isFailure(err, 403, "forbidden") {
		t.Fatalf("receptionist upload: %v", err)
	}
	if stored := objects(t, dir); len(stored) != 1 {
		t.Fatalf("a refused upload stored bytes: %v", stored)
	}
}

func isFailure(err error, status int, code string) bool {
	var f *failure.Error
	return errors.As(err, &f) && f.Status == status && f.Code == code
}
