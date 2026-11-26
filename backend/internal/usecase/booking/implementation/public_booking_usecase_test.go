package bookingusecaseimplementation

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking"
	bookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/booking/implementation"
	booking "github.com/azhkadaes/vms/backend/internal/usecase/booking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

const (
	notice    = "notice-2099"
	reference = "6f1c3f5e-8f53-4a35-9a0e-2b8d0f0d5a11"
	photo     = "photo-token-0123456789"
)

// bookingStore is the in-memory adapter behind BookingRepository.
type bookingStore struct {
	now       time.Time
	requests  map[string]entity.BookingRequest
	visits    map[int32]entity.Visit
	officers  map[int32]bool // id -> VIP
	uploads   map[string]int32
	rateLimit int // seconds to wait; 0 admits
	created   int
}

func newStore() *bookingStore {
	return &bookingStore{
		now:      time.Date(2099, 8, 19, 8, 0, 0, 0, validate.WITA),
		requests: map[string]entity.BookingRequest{},
		visits:   map[int32]entity.Visit{},
		officers: map[int32]bool{5: false, 7: true},
		uploads:  map[string]int32{security.Hash(photo): 11},
	}
}

func (s *bookingStore) Transaction(_ context.Context, run func(bookingrepo.BookingTransaction) error) error {
	return run(s)
}
func (s *bookingStore) LockReference(string, string, string) error { return nil }
func (s *bookingStore) Request(_, _, ref string) (entity.BookingRequest, error) {
	request, ok := s.requests[ref]
	if !ok {
		return entity.BookingRequest{}, bookingrepo.ErrNotFound
	}
	return request, nil
}
func (s *bookingStore) StoreRequest(_, _, ref string, request entity.BookingRequest) error {
	s.requests[ref] = request
	return nil
}
func (s *bookingStore) Now() (time.Time, error) { return s.now, nil }
func (s *bookingStore) ReadVisit(id int32) (entity.Visit, error) {
	visit, ok := s.visits[id]
	if !ok {
		return entity.Visit{}, bookingrepo.ErrNotFound
	}
	return visit, nil
}
func (s *bookingStore) PublicHistory(int32) ([]entity.VisitHistory, error) { return nil, nil }
func (s *bookingStore) ActiveOfficer(id int32) (string, bool, error) {
	vip, ok := s.officers[id]
	if !ok {
		return "", false, bookingrepo.ErrNotFound
	}
	return "Officer", vip, nil
}
func (s *bookingStore) LinkUpload(tokenHash, _, _, _ string, _ time.Time) (int32, error) {
	id, ok := s.uploads[tokenHash]
	if !ok {
		return 0, bookingrepo.ErrNotFound
	}
	delete(s.uploads, tokenHash) // a token links to one visit only
	return id, nil
}
func (s *bookingStore) AdmitBooking(string) (entity.RequestAdmission, error) {
	return entity.RequestAdmission{Allowed: s.rateLimit == 0, RetryAfter: s.rateLimit}, nil
}
func (s *bookingStore) CreatePublicVisit(visit entity.Visit, _ int32, _ *int32, _ time.Time, _, _ string) (int32, error) {
	s.created++
	visit.ID, visit.Status, visit.Version, visit.Code = int32(100+s.created), "pending", 1, "00123456"
	s.visits[visit.ID] = visit
	return visit.ID, nil
}

func publicBookings(s *bookingStore) booking.PublicBookingUsecase {
	return NewPublicBookingUsecase(s, bookingserializer.NewBookingSerializer(), notice, []byte("rate-key"), noop.NewTracerProvider().Tracer("test"))
}

func request() dto.PublicBookingCreate {
	return dto.PublicBookingCreate{
		OfficerID: "5", Date: "2099-08-20", Time: "10:00", Purpose: "Koordinasi layanan", Name: "Visitor Name",
		Organization: "Agency", Email: "visitor@example.test", Phone: "081234567890", IdentityType: "KTP",
		Visitors: []string{"Visitor Name"}, VisitorPhotoName: "photo.jpg", VisitorPhotoToken: photo,
		Ack: true, PrivacyVersion: notice, IdempotencyKey: reference,
	}
}

func create(s *bookingStore, input dto.PublicBookingCreate) (dto.PublicBooking, bool, error) {
	return publicBookings(s).Create(context.Background(), "203.0.113.7", input)
}

func failed(err error) *failure.Error {
	var f *failure.Error
	errors.As(err, &f)
	return f
}

func TestPublicCreateIsIdempotentPerReference(t *testing.T) {
	s := newStore()
	first, created, err := create(s, request())
	if err != nil || !created {
		t.Fatalf("first submission: created=%v err=%v", created, err)
	}
	if first.Status != "pending" || s.visits[101].Kind != "umum" || s.visits[101].Duration != 60 {
		t.Fatalf("unexpected visit: %+v %+v", first, s.visits[101])
	}
	again, created, err := create(s, request())
	if err != nil || created || again.ID != first.ID || s.created != 1 {
		t.Fatalf("a replayed submission must return the first visit: created=%v err=%v visits=%d", created, err, s.created)
	}
	changed := request()
	changed.Purpose = "Keperluan yang berbeda"
	if _, _, err := create(s, changed); failed(err) == nil || failed(err).Code != "idempotency_conflict" || s.created != 1 {
		t.Fatalf("a changed payload reused the reference: %v", err)
	}
}

func TestPublicCreateRefusesWithoutCreatingAVisit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		store  func(*bookingStore)
		input  func(*dto.PublicBookingCreate)
		ip     string
		status int
		key    string
	}{
		{"outdated privacy notice", nil, func(in *dto.PublicBookingCreate) { in.PrivacyVersion = "notice-old" }, "", 409, "notice_outdated"},
		{"rate limited", func(s *bookingStore) { s.rateLimit = 30 }, nil, "", 429, "rate_limited"},
		{"start already passed", nil, func(in *dto.PublicBookingCreate) { in.Date, in.Time = "2099-08-19", "08:00" }, "", 422, "date"},
		{"inactive officer", nil, func(in *dto.PublicBookingCreate) { in.OfficerID = "9" }, "", 422, "officerId"},
		{"photo token already used", func(s *bookingStore) { s.uploads = map[string]int32{} }, nil, "", 422, "visitorPhotoToken"},
		{"roster must start with the representative", nil, func(in *dto.PublicBookingCreate) { in.Visitors = []string{"Someone Else"} }, "", 422, "visitors"},
		{"restricted visits are staff-only", nil, func(in *dto.PublicBookingCreate) { in.VisitKind = "restricted" }, "", 422, "visitKind"},
		{"untrusted client address", nil, nil, "not-an-ip", 503, "service_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore()
			if tc.store != nil {
				tc.store(s)
			}
			input := request()
			if tc.input != nil {
				tc.input(&input)
			}
			ip := "203.0.113.7"
			if tc.ip != "" {
				ip = tc.ip
			}
			_, _, err := publicBookings(s).Create(context.Background(), ip, input)
			f := failed(err)
			if f == nil || f.Status != tc.status || (f.Code != tc.key && len(f.Fields[tc.key]) == 0) {
				t.Fatalf("got %v %+v", err, f)
			}
			if tc.status == 429 && f.RetryAfter != 30 {
				t.Fatalf("Retry-After %d", f.RetryAfter)
			}
			if s.created != 0 || len(s.requests) != 0 {
				t.Fatalf("a refused submission left state behind: visits=%d requests=%v", s.created, s.requests)
			}
		})
	}
}

func TestVIPOfficerMakesAVIPVisit(t *testing.T) {
	s := newStore()
	input := request()
	input.OfficerID = "7"
	if _, _, err := create(s, input); err != nil {
		t.Fatal(err)
	}
	if s.visits[101].Kind != "vip" {
		t.Fatalf("kind %q", s.visits[101].Kind)
	}
}

func TestResolveRecoversOrClosesAReference(t *testing.T) {
	s := newStore()
	resolve := func(email string) (dto.PublicSubmissionResolution, error) {
		return publicBookings(s).Resolve(context.Background(), dto.PublicSubmissionProof{IdempotencyKey: reference, Email: email})
	}
	missing, err := resolve("visitor@example.test")
	if err != nil || missing.Outcome != "not_committed" || s.requests[reference].State != "closed" {
		t.Fatalf("unknown reference: %+v %v %+v", missing, err, s.requests)
	}
	// Closing the reference means a late-arriving submission cannot commit under it.
	if _, _, err := create(s, request()); failed(err) == nil || failed(err).Code != "idempotency_conflict" || s.created != 0 {
		t.Fatalf("a closed reference accepted a submission: %v", err)
	}

	s = newStore()
	first, _, err := create(s, request())
	if err != nil {
		t.Fatal(err)
	}
	found, err := resolve("Visitor@Example.test")
	if err != nil || found.Outcome != "committed" || found.Booking.ID != first.ID {
		t.Fatalf("committed reference: %+v %v", found, err)
	}
	if _, err := resolve("someone@example.test"); failed(err) == nil || failed(err).Status != 404 {
		t.Fatalf("another email recovered the visit: %v", err)
	}
}
