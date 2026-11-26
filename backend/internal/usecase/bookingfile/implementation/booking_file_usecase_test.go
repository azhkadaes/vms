package bookingfileusecaseimplementation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/infrastructure/storage"
	"github.com/azhkadaes/vms/backend/internal/policy"
	bookingfiles "github.com/azhkadaes/vms/backend/internal/repository/bookingfile"
	fileusecase "github.com/azhkadaes/vms/backend/internal/usecase/bookingfile"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

var start = time.Date(2099, 8, 19, 10, 0, 0, 0, validate.WITA)

// files is the in-memory adapter for both the metadata lookup and private storage.
type files struct {
	file    entity.BookingFile
	objects map[string][]byte
}

func (f *files) Lookup(_ context.Context, id int32, _ string, credential *entity.ReceptionCredential) (entity.BookingFile, error) {
	if id != 42 {
		return entity.BookingFile{}, bookingfiles.ErrNotFound
	}
	file := f.file
	if credential != nil && credential.Code == "00123456" {
		file.Match.VisitID = 42
	}
	return file, nil
}
func (f *files) Put(context.Context, []byte) (string, error) { return "", errors.New("unused") }
func (f *files) Delete(context.Context, string) error        { return nil }
func (f *files) Open(_ context.Context, key string) (io.ReadCloser, error) {
	content, ok := f.objects[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}

func stored(kind, status, mime string, size int, at time.Time) *files {
	return &files{
		file: entity.BookingFile{Key: "object-1", MIME: mime, Size: size, OfficerID: 5, Kind: kind, Status: status,
			Date: "2099-08-19", Time: "10:00", Duration: 60, ServerTime: at},
		objects: map[string][]byte{"object-1": []byte("%PDF-1.7")},
	}
}

func usecaseFor(f *files, withStorage bool) fileusecase.BookingFileUsecase {
	var store storage.Storage
	if withStorage {
		store = f
	}
	return NewBookingFileUsecase(f, store, noop.NewTracerProvider().Tracer("test"))
}

var (
	assignedPA = dto.Session{Active: true, ID: "1", Name: "PA", Role: policy.RolePA,
		Permissions: []string{policy.PermBookings}, AssignedOfficerIDs: []string{"5"}}
	unassignedPA = dto.Session{Active: true, ID: "1", Name: "PA", Role: policy.RolePA,
		Permissions: []string{policy.PermBookings}, AssignedOfficerIDs: []string{"6"}}
	desk = dto.Session{Active: true, ID: "2", Name: "Desk", Role: policy.RoleReceptionist,
		Permissions: []string{policy.PermBookings, policy.PermCheckin, policy.PermCheckout}}
)

func status(err error) int {
	var f *failure.Error
	if errors.As(err, &f) {
		return f.Status
	}
	return 0
}

func TestStaffFileReadsFollowVisitScopeAndStoredType(t *testing.T) {
	pdf := func() *files { return stored("umum", "approved", "application/pdf", 8, start) }
	opened, err := usecaseFor(pdf(), true).OpenStaffFile(context.Background(), assignedPA, 42, policy.PurposeDocument)
	if err != nil {
		t.Fatal(err)
	}
	if content, _ := io.ReadAll(opened.Content); opened.MIME != "application/pdf" || string(content) != "%PDF-1.7" {
		t.Fatalf("unexpected file: %s %q", opened.MIME, content)
	}
	for _, tc := range []struct {
		name    string
		session dto.Session
		files   *files
		purpose string
		storage bool
		want    int
	}{
		{"reception must use the credential route", desk, pdf(), policy.PurposeDocument, true, 403},
		{"visit outside the PA's assignment", unassignedPA, pdf(), policy.PurposeDocument, true, 404},
		{"restricted visits have no staff files", assignedPA, stored("terbatas", "approved", "application/pdf", 8, start), policy.PurposeDocument, true, 404},
		{"a PDF is never served as a visitor photo", assignedPA, pdf(), policy.PurposeVisitorPhoto, true, 404},
		{"oversized object", assignedPA, stored("umum", "approved", "application/pdf", 5*1024*1024+1, start), policy.PurposeDocument, true, 404},
		{"object missing from storage", assignedPA, &files{file: pdf().file}, policy.PurposeDocument, true, 404},
		{"storage not configured", assignedPA, pdf(), policy.PurposeDocument, false, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := usecaseFor(tc.files, tc.storage).OpenStaffFile(context.Background(), tc.session, 42, tc.purpose)
			if got := status(err); got != tc.want {
				t.Fatalf("got %d (%v), want %d", got, err, tc.want)
			}
		})
	}
}

func TestReceptionFileReadsNeedTheCredentialInsideTheArrivalWindow(t *testing.T) {
	read := func(f *files, session dto.Session, credential string) error {
		_, err := usecaseFor(f, true).OpenReceptionFile(context.Background(), session, 42, policy.PurposeDocument,
			dto.ReceptionResolve{Credential: credential})
		return err
	}
	inWindow, early := start.Add(-10*time.Minute), start.Add(-31*time.Minute)
	if err := read(stored("umum", "approved", "application/pdf", 8, inWindow), desk, "00123456"); err != nil {
		t.Fatalf("arrival read refused: %v", err)
	}
	if err := read(stored("umum", "checked_in", "application/pdf", 8, start.Add(24*time.Hour)), desk, "00123456"); err != nil {
		t.Fatalf("checkout read refused: %v", err)
	}
	expired := stored("umum", "approved", "application/pdf", 8, inWindow)
	before := inWindow.Add(-time.Second)
	expired.file.Match.QRExpiresAt = &before
	for name, err := range map[string]error{
		"another visit's code": read(stored("umum", "approved", "application/pdf", 8, inWindow), desk, "00999999"),
		"before the window":    read(stored("umum", "approved", "application/pdf", 8, early), desk, "00123456"),
		"expired QR":           read(expired, desk, "00123456"),
		"pending visit":        read(stored("umum", "pending", "application/pdf", 8, inWindow), desk, "00123456"),
		"restricted visit":     read(stored("terbatas", "approved", "application/pdf", 8, inWindow), desk, "00123456"),
		"not a receptionist":   read(stored("umum", "approved", "application/pdf", 8, inWindow), assignedPA, "00123456"),
		"blank credential":     read(stored("umum", "approved", "application/pdf", 8, inWindow), desk, " "),
	} {
		if status(err) != 404 {
			t.Errorf("%s: got %v, want 404", name, err)
		}
	}
}
