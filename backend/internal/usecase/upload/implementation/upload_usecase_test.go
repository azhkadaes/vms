package uploadusecaseimplementation_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/infrastructure/storage"
	"github.com/azhkadaes/vms/backend/internal/policy"
	uploadrepository "github.com/azhkadaes/vms/backend/internal/repository/upload"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	uploadserializer "github.com/azhkadaes/vms/backend/internal/serializer/upload/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	authusecaseimplementation "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	uploadusecase "github.com/azhkadaes/vms/backend/internal/usecase/upload"
	implementation "github.com/azhkadaes/vms/backend/internal/usecase/upload/implementation"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

const (
	reference = "550e8400-e29b-41d4-a716-446655440000"
	visitorIP = "203.0.113.7"
)

// ---------------------------------------------------------------- synthetic files

// pdf builds a minimal well-formed PDF whose startxref points at its cross-reference table.
func pdf(body string) []byte {
	head := "%PDF-1.7\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [] /Count 0 /Note (" + body + ") >>\nendobj\n"
	xref := len(head)
	return []byte(head + "xref\n0 3\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \ntrailer\n<< /Size 3 /Root 1 0 R >>\nstartxref\n" + fmt.Sprint(xref) + "\n%%EOF\n")
}

func pictures(t *testing.T) (jpegBytes, pngBytes []byte) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for x := range 4 {
		for y := range 3 {
			img.Set(x, y, color.RGBA{uint8(60 * x), uint8(80 * y), 120, 255})
		}
	}
	var j, p bytes.Buffer
	if err := jpeg.Encode(&j, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&p, img); err != nil {
		t.Fatal(err)
	}
	return j.Bytes(), p.Bytes()
}

// ---------------------------------------------------------------- fakes

// store is an in-memory upload repository. A transaction reads the committed rows and its
// writes commit only when the operation returns nil and no commit failure is configured.
type store struct {
	now        time.Time
	snapshot   entity.SessionSnapshot
	committed  []entity.UploadRecord
	documents  []entity.FormDocument
	admissions map[string]int
	retryAfter int
	createErr  error
	commitErr  error
	calls      []string
}

func newStore() *store {
	now := time.Date(2026, 11, 19, 2, 30, 0, 0, time.UTC)
	return &store{now: now, admissions: map[string]int{}, retryAfter: 1800, snapshot: entity.SessionSnapshot{
		EmployeeID: 7, Name: "Staff Example", AccountActive: true, ActiveRole: policy.RolePA, AvailableRoles: []string{policy.RolePA},
		Permissions: []string{policy.PermBookings, policy.PermCreate}, CredentialGeneration: 1, RoleRevision: 1,
		CreatedAt: now.Add(-time.Hour), LastActivityAt: now.Add(-time.Minute), IdleExpiresAt: now.Add(7 * time.Hour), AbsoluteExpiresAt: now.Add(20 * time.Hour), ServerTime: now,
	}}
}

func proof(employee int32) entity.SessionProof {
	return entity.SessionProof{SessionID: "82d2bb6d-fdcb-4ff5-a914-91e5e9ce9559", EmployeeID: employee, CredentialGeneration: 1, RoleRevision: 1}
}

type tx struct {
	s         *store
	documents []entity.FormDocument
	records   []entity.UploadRecord
	admitted  []string
}

func (s *store) Transaction(_ context.Context, operation func(uploadrepository.UploadTransaction) error) error {
	s.calls = append(s.calls, "Transaction")
	t := &tx{s: s}
	if err := operation(t); err != nil {
		return err
	}
	if s.commitErr != nil {
		return s.commitErr
	}
	s.documents = append(s.documents, t.documents...)
	s.committed = append(s.committed, t.records...)
	for _, bucket := range t.admitted {
		s.admissions[bucket]++
	}
	return nil
}

func (t *tx) LockAuthority(p entity.SessionProof) (entity.SessionSnapshot, error) {
	t.s.calls = append(t.s.calls, "LockAuthority")
	if p.EmployeeID != t.s.snapshot.EmployeeID {
		return entity.SessionSnapshot{}, nil
	}
	return t.s.snapshot, nil
}
func (t *tx) LockReference(audience, owner, ref string) error {
	t.s.calls = append(t.s.calls, "LockReference")
	return nil
}
func (t *tx) Now() (time.Time, error) { return t.s.now, nil }
func (t *tx) FindUpload(audience, owner, ref string) (entity.UploadRecord, error) {
	for _, r := range t.s.committed {
		if r.Audience == audience && r.OwnerDigest == owner && r.Reference == ref {
			return r, nil
		}
	}
	return entity.UploadRecord{}, uploadrepository.ErrNotFound
}
func (t *tx) AdmitUpload(bucket string, limit int, window time.Duration) (entity.RequestAdmission, error) {
	t.s.calls = append(t.s.calls, fmt.Sprintf("AdmitUpload %d/%v", limit, window))
	if t.s.admissions[bucket]+len(t.admitted) >= limit {
		return entity.RequestAdmission{RetryAfter: t.s.retryAfter}, nil
	}
	t.admitted = append(t.admitted, bucket)
	return entity.RequestAdmission{Allowed: true}, nil
}
func (t *tx) CreateDocument(d entity.FormDocument) (int32, error) {
	t.s.calls = append(t.s.calls, "CreateDocument")
	d.ID = int32(len(t.s.documents) + len(t.documents) + 1)
	t.documents = append(t.documents, d)
	return d.ID, nil
}
func (t *tx) CreateUpload(r entity.UploadRecord) error {
	t.s.calls = append(t.s.calls, "CreateUpload")
	if t.s.createErr != nil {
		return t.s.createErr
	}
	for _, d := range t.documents {
		if d.ID == r.DocumentID {
			r.Purpose = d.Purpose
		}
	}
	t.records = append(t.records, r)
	return nil
}

// files is an in-memory storage adapter.
type files struct {
	objects map[string][]byte
	puts    int
	deleted []string
	putErr  error
}

func (f *files) Put(_ context.Context, content []byte) (string, error) {
	if f.putErr != nil {
		return "", f.putErr
	}
	f.puts++
	key := fmt.Sprintf("00000000-0000-4000-8000-%012d", f.puts)
	f.objects[key] = append([]byte(nil), content...)
	return key, nil
}
func (f *files) Open(_ context.Context, key string) (io.ReadCloser, error) {
	content, ok := f.objects[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}
func (f *files) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	delete(f.objects, key)
	return nil
}

var (
	tokenKey = []byte(strings.Repeat("t", 32))
	rateKey  = []byte(strings.Repeat("r", 32))
)

func newUsecase(s *store, f *files, version int16) uploadusecase.UploadUsecase {
	validator := authusecaseimplementation.NewSessionValidator(authserializer.NewAuthSerializer())
	return implementation.NewUploadUsecase(s, f, uploadserializer.NewUploadSerializer(), validator,
		implementation.Keys{Token: tokenKey, Version: version, Rate: rateKey}, noop.NewTracerProvider().Tracer("upload-test"))
}

func setup() (*store, *files, uploadusecase.UploadUsecase) {
	s, f := newStore(), &files{objects: map[string][]byte{}}
	return s, f, newUsecase(s, f, 1)
}

func document(content []byte) entity.UploadRequest {
	return entity.UploadRequest{Reference: reference, Purpose: policy.PurposeDocument, Filename: "nota dinas.pdf", Content: content}
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func expectFailure(t *testing.T, err error, status int, code string) *failure.Error {
	t.Helper()
	var f *failure.Error
	if !errors.As(err, &f) || f.Status != status || f.Code != code {
		t.Fatalf("got %v, want %d %s", err, status, code)
	}
	return f
}

func expectFields(t *testing.T, err error, keys ...string) {
	t.Helper()
	f := expectFailure(t, err, 422, "validation_failed")
	got := []string{}
	for key, messages := range f.Fields {
		if len(messages) == 0 || messages[0] == "" {
			t.Fatalf("field %s has no message", key)
		}
		got = append(got, key)
	}
	slices.Sort(got)
	if !slices.Equal(got, keys) {
		t.Fatalf("field errors %v, want %v", f.Fields, keys)
	}
}

func nothingWritten(t *testing.T, s *store, f *files) {
	t.Helper()
	if len(s.documents) != 0 || len(s.committed) != 0 || len(f.objects) != 0 || len(s.admissions) != 0 {
		t.Fatalf("rejected upload left documents %v, uploads %v, objects %d, admissions %v", s.documents, s.committed, len(f.objects), s.admissions)
	}
}

// ---------------------------------------------------------------- behaviour

func TestAFirstUploadStoresTheFilePrivatelyAndIssuesAOneHourToken(t *testing.T) {
	s, f, u := setup()
	content := pdf("first")
	issued, err := u.UploadPublic(context.Background(), visitorIP, document(content))
	if err != nil {
		t.Fatal(err)
	}
	if !issued.Created || !strings.HasPrefix(issued.Result.Token, "vmsu1.") || len(issued.Result.Token) < 16 || len(issued.Result.Token) > 4096 {
		t.Fatalf("issued %+v", issued)
	}
	if !issued.Result.ExpiresAt.Equal(s.now.Add(time.Hour)) || issued.Result.ExpiresAt.Location() != time.UTC || issued.Result.ServerTime != s.now.UnixMilli() {
		t.Fatalf("expiry %v / server time %d", issued.Result.ExpiresAt, issued.Result.ServerTime)
	}
	if len(s.documents) != 1 || len(s.committed) != 1 || len(f.objects) != 1 {
		t.Fatalf("documents %v uploads %v objects %d", s.documents, s.committed, len(f.objects))
	}
	d, r := s.documents[0], s.committed[0]
	if !bytes.Equal(f.objects[d.StorageKey], content) {
		t.Fatal("metadata does not name the stored object")
	}
	if d.Name != "nota dinas.pdf" || d.MimeType != "application/pdf" || d.Size != int32(len(content)) || d.Purpose != policy.PurposeDocument {
		t.Fatalf("document metadata %+v", d)
	}
	if r.DocumentID != d.ID || r.Audience != entity.AudiencePublic || r.Reference != reference || r.UploadedBy != nil || r.KeyVersion != 1 ||
		r.ContentSHA256 != digest(content) || !r.CreatedAt.Equal(s.now) || !r.ExpiresAt.Equal(s.now.Add(time.Hour)) || r.LinkedAt != nil {
		t.Fatalf("upload record %+v", r)
	}
	if r.TokenHash != digest([]byte(issued.Result.Token)) || strings.Contains(fmt.Sprintf("%+v", r), issued.Result.Token) {
		t.Fatal("the record must keep only the token's SHA-256")
	}
	if strings.Contains(fmt.Sprintf("%+v", issued.Result), d.StorageKey) {
		t.Fatal("the response exposes the storage key")
	}
}

func TestPhotosAreAcceptedAsJPEGOrPNGByTheirContent(t *testing.T) {
	jpegBytes, pngBytes := pictures(t)
	for _, tc := range []struct {
		name, mime string
		content    []byte
	}{{"jpeg", "image/jpeg", jpegBytes}, {"png", "image/png", pngBytes}} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, u := setup()
			// The client name says PDF; only the content decides.
			upload := entity.UploadRequest{Reference: reference, Purpose: policy.PurposeVisitorPhoto, Filename: "foto.pdf", Content: tc.content}
			if _, err := u.UploadPublic(context.Background(), visitorIP, upload); err != nil {
				t.Fatal(err)
			}
			if s.documents[0].MimeType != tc.mime || s.documents[0].Purpose != policy.PurposeVisitorPhoto {
				t.Fatalf("document %+v", s.documents[0])
			}
		})
	}
}

func TestReplayingTheSameReferenceAndContentReturnsTheSameTokenWithoutWriting(t *testing.T) {
	s, f, u := setup()
	content := pdf("replay")
	first, err := u.UploadPublic(context.Background(), visitorIP, document(content))
	if err != nil {
		t.Fatal(err)
	}
	s.now = s.now.Add(59*time.Minute + 59*time.Second)
	// A replay from another network and under another name is still the same upload.
	replay := document(content)
	replay.Filename = "renamed.pdf"
	second, err := u.UploadPublic(context.Background(), "198.51.100.20", replay)
	if err != nil {
		t.Fatal(err)
	}
	if second.Created || second.Result.Token != first.Result.Token || !second.Result.ExpiresAt.Equal(first.Result.ExpiresAt) || second.Result.ServerTime != s.now.UnixMilli() {
		t.Fatalf("replay %+v, first %+v", second, first)
	}
	if f.puts != 1 || len(s.documents) != 1 || len(s.committed) != 1 || len(s.admissions) != 1 {
		t.Fatalf("replay wrote: puts %d documents %d uploads %d admissions %v", f.puts, len(s.documents), len(s.committed), s.admissions)
	}
}

func TestReusingAReferenceWithOtherContentOrPurposeIsAConflict(t *testing.T) {
	jpegBytes, _ := pictures(t)
	for _, tc := range []struct {
		name   string
		second entity.UploadRequest
	}{
		{"other content", document(pdf("other"))},
		{"other purpose", entity.UploadRequest{Reference: reference, Purpose: policy.PurposeVisitorPhoto, Filename: "foto.jpg", Content: jpegBytes}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f, u := setup()
			if _, err := u.UploadPublic(context.Background(), visitorIP, document(pdf("original"))); err != nil {
				t.Fatal(err)
			}
			_, err := u.UploadPublic(context.Background(), visitorIP, tc.second)
			expectFailure(t, err, 409, "idempotency_conflict")
			if f.puts != 1 || len(s.documents) != 1 || len(s.committed) != 1 {
				t.Fatal("a conflicting reuse wrote")
			}
		})
	}
}

func TestAReplayAfterTheTokenExpiredIsAConflictAndIssuesNothing(t *testing.T) {
	s, f, u := setup()
	content := pdf("expired")
	if _, err := u.UploadPublic(context.Background(), visitorIP, document(content)); err != nil {
		t.Fatal(err)
	}
	s.now = s.now.Add(time.Hour)
	_, err := u.UploadPublic(context.Background(), visitorIP, document(content))
	expectFailure(t, err, 409, "idempotency_conflict")
	if f.puts != 1 || len(s.committed) != 1 {
		t.Fatal("an expired reference was uploaded again")
	}
}

// D99: a replay never substitutes another key for the one its token was derived with.
func TestAReplayUnderARotatedKeyIsAConflict(t *testing.T) {
	s, f, u := setup()
	content := pdf("rotated")
	if _, err := u.UploadPublic(context.Background(), visitorIP, document(content)); err != nil {
		t.Fatal(err)
	}
	_, err := newUsecase(s, f, 2).UploadPublic(context.Background(), visitorIP, document(content))
	expectFailure(t, err, 409, "idempotency_conflict")
}

func TestReferencesAreScopedByAudienceAndUploader(t *testing.T) {
	s, f, u := setup()
	content := pdf("scoped")
	public, err := u.UploadPublic(context.Background(), visitorIP, document(content))
	if err != nil {
		t.Fatal(err)
	}
	staff, err := u.UploadStaff(context.Background(), proof(7), visitorIP, document(content))
	if err != nil {
		t.Fatal(err)
	}
	s.snapshot.EmployeeID = 8
	colleague, err := u.UploadStaff(context.Background(), proof(8), visitorIP, document(content))
	if err != nil {
		t.Fatal(err)
	}
	if !public.Created || !staff.Created || !colleague.Created || public.Result.Token == staff.Result.Token || staff.Result.Token == colleague.Result.Token {
		t.Fatal("one reference was shared across audiences or uploaders")
	}
	if f.puts != 3 || len(s.committed) != 3 {
		t.Fatalf("puts %d uploads %d", f.puts, len(s.committed))
	}
	if s.committed[1].Audience != entity.AudienceStaff || s.committed[1].UploadedBy == nil || *s.committed[1].UploadedBy != 7 || *s.committed[2].UploadedBy != 8 {
		t.Fatalf("staff uploads must record their uploader: %+v", s.committed[1:])
	}
}

func TestContentThatIsNotWhatItsPurposeNeedsIsAFieldErrorOnFile(t *testing.T) {
	jpegBytes, pngBytes := pictures(t)
	gif := []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")
	valid := pdf("structure")
	for _, tc := range []struct {
		name, purpose string
		content       []byte
	}{
		{"PNG sent as a document", policy.PurposeDocument, pngBytes},
		{"JPEG sent as a document", policy.PurposeDocument, jpegBytes},
		{"PDF sent as a photo", policy.PurposeVisitorPhoto, valid},
		{"text renamed to PDF", policy.PurposeDocument, []byte("Nota dinas, not really a PDF\n")},
		{"PDF header without a body", policy.PurposeDocument, []byte("%PDF-1.7\n")},
		{"PDF without any cross-reference entries", policy.PurposeDocument, []byte("%PDF-1.7\nxref\nstartxref\n9\n%%EOF\n")},
		{"ordinary object in place of cross-reference stream", policy.PurposeDocument, []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\nstartxref\n9\n%%EOF\n")},
		{"PDF whose startxref points nowhere", policy.PurposeDocument, bytes.Replace(valid, []byte("startxref\n"), []byte("startxref\n9"), 1)},
		{"PDF cut off before its trailer", policy.PurposeDocument, valid[:len(valid)-40]},
		{"truncated JPEG", policy.PurposeVisitorPhoto, jpegBytes[:len(jpegBytes)/2]},
		{"truncated PNG", policy.PurposeVisitorPhoto, pngBytes[:len(pngBytes)-20]},
		{"GIF photo", policy.PurposeVisitorPhoto, gif},
		{"text renamed to JPEG", policy.PurposeVisitorPhoto, []byte("\xff\xd8\xff not a JPEG")},
		{"empty file", policy.PurposeDocument, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f, u := setup()
			_, err := u.UploadPublic(context.Background(), visitorIP, entity.UploadRequest{Reference: reference, Purpose: tc.purpose, Filename: "berkas.pdf", Content: tc.content})
			expectFields(t, err, "file")
			nothingWritten(t, s, f)
			if len(s.calls) != 0 {
				t.Fatalf("invalid content reached the repository: %v", s.calls)
			}
		})
	}
}

func TestEveryInvalidFieldIsReportedTogether(t *testing.T) {
	for _, tc := range []struct {
		name   string
		upload entity.UploadRequest
		keys   []string
	}{
		{"upper-case reference", entity.UploadRequest{Reference: strings.ToUpper(reference), Purpose: policy.PurposeDocument, Content: pdf("x")}, []string{"reference"}},
		{"braced reference", entity.UploadRequest{Reference: "{" + reference + "}", Purpose: policy.PurposeDocument, Content: pdf("x")}, []string{"reference"}},
		{"nil reference", entity.UploadRequest{Reference: "00000000-0000-0000-0000-000000000000", Purpose: policy.PurposeDocument, Content: pdf("x")}, []string{"reference"}},
		{"unknown purpose", entity.UploadRequest{Reference: reference, Purpose: "photo", Content: pdf("x")}, []string{"purpose"}},
		{"everything wrong", entity.UploadRequest{Reference: "x", Purpose: "", Content: nil}, []string{"file", "purpose", "reference"}},
		{"file name over 255 characters", entity.UploadRequest{Reference: reference, Purpose: policy.PurposeDocument, Filename: strings.Repeat("é", 252) + ".pdf", Content: pdf("x")}, []string{"file"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f, u := setup()
			_, err := u.UploadPublic(context.Background(), visitorIP, tc.upload)
			expectFields(t, err, tc.keys...)
			nothingWritten(t, s, f)
		})
	}
}

func TestAFileOverFiveMebibytesIsTooLarge(t *testing.T) {
	s, f, u := setup()
	content := pdf(strings.Repeat("x", uploadusecase.MaxFileBytes))
	_, err := u.UploadPublic(context.Background(), visitorIP, document(content))
	expectFailure(t, err, 413, "payload_too_large")
	nothingWritten(t, s, f)
}

func TestTheStoredNameIsTheClientsBaseNameWithoutControlCharacters(t *testing.T) {
	jpegBytes, pngBytes := pictures(t)
	for _, tc := range []struct {
		name, purpose, filename, want string
		content                       []byte
	}{
		{"windows path", policy.PurposeDocument, `C:\Users\visitor\Nota Dinas.pdf`, "Nota Dinas.pdf", pdf("x")},
		{"unix path", policy.PurposeDocument, "/home/visitor/nd.pdf", "nd.pdf", pdf("x")},
		{"control characters and padding", policy.PurposeDocument, "  nota\x00\r\n dinas.pdf ", "nota dinas.pdf", pdf("x")},
		{"invalid UTF-8", policy.PurposeDocument, "nota\xffdinas.pdf", "notadinas.pdf", pdf("x")},
		{"no document name", policy.PurposeDocument, "", "nota-dinas.pdf", pdf("x")},
		{"no JPEG name", policy.PurposeVisitorPhoto, " / ", "foto.jpg", jpegBytes},
		{"no PNG name", policy.PurposeVisitorPhoto, "", "foto.png", pngBytes},
		{"exactly 255 characters", policy.PurposeDocument, strings.Repeat("é", 251) + ".pdf", strings.Repeat("é", 251) + ".pdf", pdf("x")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, u := setup()
			if _, err := u.UploadPublic(context.Background(), visitorIP, entity.UploadRequest{Reference: reference, Purpose: tc.purpose, Filename: tc.filename, Content: tc.content}); err != nil {
				t.Fatal(err)
			}
			if s.documents[0].Name != tc.want {
				t.Fatalf("stored name %q, want %q", s.documents[0].Name, tc.want)
			}
		})
	}
}

func TestTheSixtyFirstNewUploadFromOneIPWithinAnHourIsRateLimited(t *testing.T) {
	s, f, u := setup()
	references := []string{}
	for i := range 61 {
		references = append(references, fmt.Sprintf("550e8400-e29b-41d4-a716-%012d", i))
	}
	content := pdf("allowance")
	for i, ref := range references[:60] {
		upload := document(content)
		upload.Reference = ref
		// Staff uploads from the same IP draw on the same allowance (D31).
		var err error
		if i%2 == 0 {
			_, err = u.UploadPublic(context.Background(), visitorIP, upload)
		} else {
			_, err = u.UploadStaff(context.Background(), proof(7), visitorIP, upload)
		}
		if err != nil {
			t.Fatalf("upload %d: %v", i+1, err)
		}
	}
	// Replays add no new object, so they neither count nor get refused.
	replay := document(content)
	replay.Reference = references[0]
	if issued, err := u.UploadPublic(context.Background(), visitorIP, replay); err != nil || issued.Created {
		t.Fatalf("replay at the limit: %+v %v", issued, err)
	}
	last := document(content)
	last.Reference = references[60]
	_, err := u.UploadPublic(context.Background(), visitorIP, last)
	if f := expectFailure(t, err, 429, "rate_limited"); f.RetryAfter != s.retryAfter {
		t.Fatalf("Retry-After %d, want %d", f.RetryAfter, s.retryAfter)
	}
	if f.puts != 60 || len(s.committed) != 60 {
		t.Fatalf("puts %d uploads %d", f.puts, len(s.committed))
	}
	if !slices.Contains(s.calls, fmt.Sprintf("AdmitUpload 60/%v", time.Hour)) {
		t.Fatalf("allowance is not 60 per hour: %v", s.calls)
	}
	// Another IP has its own allowance.
	if _, err := u.UploadPublic(context.Background(), "203.0.113.8", last); err != nil {
		t.Fatal(err)
	}
}

func TestIPv4AndItsMappedIPv6FormShareOneAllowance(t *testing.T) {
	s, _, u := setup()
	for i, ip := range []string{"203.0.113.7", "::ffff:203.0.113.7"} {
		upload := document(pdf("mapped"))
		upload.Reference = fmt.Sprintf("550e8400-e29b-41d4-a716-%012d", i)
		if _, err := u.UploadPublic(context.Background(), ip, upload); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.admissions) != 1 {
		t.Fatalf("buckets %v", s.admissions)
	}
}

func TestAnUnusableClientAddressFailsClosed(t *testing.T) {
	s, f, u := setup()
	_, err := u.UploadPublic(context.Background(), "", document(pdf("x")))
	expectFailure(t, err, 503, "service_unavailable")
	nothingWritten(t, s, f)
}

func TestStaffUploadsRecheckAuthorityBeforeAnythingElse(t *testing.T) {
	s, f, u := setup()
	if _, err := u.UploadStaff(context.Background(), proof(7), visitorIP, document(pdf("staff"))); err != nil {
		t.Fatal(err)
	}
	if len(s.calls) < 2 || s.calls[0] != "Transaction" || s.calls[1] != "LockAuthority" {
		t.Fatalf("the authority lock must be the transaction's first step: %v", s.calls)
	}
	if f.puts != 1 {
		t.Fatal("upload not stored")
	}
}

func TestStaffWithoutAuthorityToCreateVisitsCannotUpload(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*entity.SessionSnapshot)
		want   func(t *testing.T, err error)
	}{
		{"receptionist", func(s *entity.SessionSnapshot) {
			s.ActiveRole, s.AvailableRoles, s.Permissions = policy.RoleReceptionist, []string{policy.RoleReceptionist}, []string{policy.PermBookings, policy.PermCheckin}
		}, func(t *testing.T, err error) { expectFailure(t, err, 403, "forbidden") }},
		{"role selection pending", func(s *entity.SessionSnapshot) { s.ActiveRole = "" }, func(t *testing.T, err error) {
			if !errors.Is(err, authusecase.ErrWorkspaceBlocked) {
				t.Fatalf("got %v", err)
			}
		}},
		{"password change pending", func(s *entity.SessionSnapshot) { s.MustChangePassword = true }, func(t *testing.T, err error) {
			if !errors.Is(err, authusecase.ErrWorkspaceBlocked) {
				t.Fatalf("got %v", err)
			}
		}},
		{"revoked session", func(s *entity.SessionSnapshot) { revoked := s.ServerTime; s.RevokedAt = &revoked }, func(t *testing.T, err error) {
			if !errors.Is(err, authusecase.ErrUnauthenticated) {
				t.Fatalf("got %v", err)
			}
		}},
		{"role revision changed", func(s *entity.SessionSnapshot) { s.RoleRevision = 2 }, func(t *testing.T, err error) {
			if !errors.Is(err, authusecase.ErrRoleChanged) {
				t.Fatalf("got %v", err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f, u := setup()
			tc.change(&s.snapshot)
			_, err := u.UploadStaff(context.Background(), proof(7), visitorIP, document(pdf("denied")))
			tc.want(t, err)
			nothingWritten(t, s, f)
			if !slices.Equal(s.calls, []string{"Transaction", "LockAuthority"}) {
				t.Fatalf("work after an authorization failure: %v", s.calls)
			}
		})
	}
}

func TestAStorageFailureLeavesNoUsableToken(t *testing.T) {
	s, f, u := setup()
	f.putErr = errors.New("disk full")
	issued, err := u.UploadPublic(context.Background(), visitorIP, document(pdf("x")))
	expectFailure(t, err, 503, "service_unavailable")
	if issued.Result.Token != "" {
		t.Fatal("token issued after a storage failure")
	}
	nothingWritten(t, s, f)
}

func TestADatabaseFailureAfterStoringDeletesTheStoredBytes(t *testing.T) {
	s, f, u := setup()
	s.createErr = errors.New("insert failed")
	issued, err := u.UploadPublic(context.Background(), visitorIP, document(pdf("x")))
	if err == nil || issued.Result.Token != "" {
		t.Fatalf("issued %+v after a failed insert", issued)
	}
	if f.puts != 1 || len(f.deleted) != 1 || len(f.objects) != 0 {
		t.Fatalf("stored bytes kept after rollback: puts %d deleted %v", f.puts, f.deleted)
	}
	nothingWritten(t, s, f)
}

// When the commit itself fails, the rows may still have committed; deleting the bytes could
// strand a committed token, so they stay (an unreferenced object is only an orphan).
func TestAnUncertainCommitKeepsTheStoredBytes(t *testing.T) {
	s, f, u := setup()
	s.commitErr = errors.New("connection lost during commit")
	issued, err := u.UploadPublic(context.Background(), visitorIP, document(pdf("x")))
	if err == nil || issued.Result.Token != "" {
		t.Fatalf("issued %+v after a failed commit", issued)
	}
	if f.puts != 1 || len(f.deleted) != 0 {
		t.Fatalf("bytes deleted after an uncertain commit: %v", f.deleted)
	}
}

func TestDisabledUploadsAreUnavailable(t *testing.T) {
	u := implementation.NewDisabledUploadUsecase()
	_, err := u.UploadPublic(context.Background(), visitorIP, document(pdf("x")))
	expectFailure(t, err, 503, "service_unavailable")
	_, err = u.UploadStaff(context.Background(), proof(7), visitorIP, document(pdf("x")))
	expectFailure(t, err, 503, "service_unavailable")
}
