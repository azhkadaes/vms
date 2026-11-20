package authusecaseimplementation_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	implementation "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"gorm.io/gorm"
)

// sessionStore is one employee's stored session; the snapshot follows the stored row,
// and every write is recorded so tests can assert what was (not) changed.
type sessionStore struct {
	now           time.Time
	session       entity.StaffSession
	credential    entity.EmployeeCredential
	snapshotErr   error
	sessionErr    error
	transactions  int
	saved         []entity.StaffSession
	passwordsSet  int
	audits        []string
	rememberedTxs int
}

func newSessionStore() *sessionStore {
	now := time.Date(2026, 11, 18, 0, 30, 0, 0, time.UTC)
	role := int32(2)
	return &sessionStore{
		now:        now,
		credential: entity.EmployeeCredential{EmployeeID: 7, EncodedHash: "bcrypt:current", Active: true},
		session: entity.StaffSession{ID: "82d2bb6d-fdcb-4ff5-a914-91e5e9ce9559", EmployeeID: 7, ActiveRoleID: &role, CredentialGeneration: 2, RoleRevision: 3,
			RefreshHash: "old-refresh-hash", CreatedAt: now.Add(-time.Hour), LastActivityAt: now.Add(-time.Hour), IdleExpiresAt: now.Add(7 * time.Hour), AbsoluteExpiresAt: now.Add(23 * time.Hour)},
	}
}

func (s *sessionStore) proof() entity.SessionProof {
	return entity.SessionProof{SessionID: s.session.ID, EmployeeID: 7, CredentialGeneration: s.session.CredentialGeneration, RoleRevision: s.session.RoleRevision}
}

func (s *sessionStore) GetSessionSnapshot(context.Context, string) (entity.SessionSnapshot, error) {
	if s.snapshotErr != nil {
		return entity.SessionSnapshot{}, s.snapshotErr
	}
	// Like pgx, return timestamptz values in the host's local zone.
	r := s.session
	return entity.SessionSnapshot{EmployeeID: r.EmployeeID, Name: "Test employee", AccountActive: true, ActiveRole: "PA", AvailableRoles: []string{"PA"},
		Permissions: []string{"bookings"}, CredentialGeneration: r.CredentialGeneration, RoleRevision: r.RoleRevision, CreatedAt: r.CreatedAt.Local(),
		LastActivityAt: r.LastActivityAt.Local(), IdleExpiresAt: r.IdleExpiresAt.Local(), AbsoluteExpiresAt: r.AbsoluteExpiresAt.Local(), RevokedAt: r.RevokedAt, ServerTime: s.now.Local()}, nil
}
func (s *sessionStore) GetCredential(context.Context, string) (entity.EmployeeCredential, error) {
	return s.credential, nil
}
func (s *sessionStore) FindRefresh(context.Context, string) (string, error) { return s.session.ID, nil }
func (s *sessionStore) ReserveLogin(context.Context, string, string) (string, int, error) {
	return "1", 0, nil
}
func (s *sessionStore) FinishLogin(context.Context, string, bool) error { return nil }
func (s *sessionStore) Transaction(_ context.Context, _ int32, operation func(authrepository.SessionTransaction) error) error {
	s.transactions++
	return operation(s)
}
func (s *sessionStore) Now() time.Time { return s.now }
func (s *sessionStore) Credential(int32) (entity.EmployeeCredential, error) {
	return s.credential, nil
}
func (s *sessionStore) Roles(int32) (map[string]int32, error) {
	return map[string]int32{"PA": 2}, nil
}
func (s *sessionStore) Session(string) (entity.StaffSession, error) {
	if s.sessionErr != nil {
		return entity.StaffSession{}, s.sessionErr
	}
	return s.session, nil
}
func (s *sessionStore) Create(entity.StaffSession) error { return errors.New("unexpected create") }
func (s *sessionStore) Save(session entity.StaffSession) error {
	s.saved = append(s.saved, session)
	s.session = session
	return nil
}
func (s *sessionStore) SetPassword(_ int32, encoded, _ string) error {
	s.passwordsSet++
	s.credential.EncodedHash = encoded
	return nil
}
func (s *sessionStore) Recovery(string, string) (entity.SessionRecovery, error) {
	return entity.SessionRecovery{}, gorm.ErrRecordNotFound
}
func (s *sessionStore) SaveRecovery(entity.SessionRecovery) error { return nil }
func (s *sessionStore) ClearRecoveries(string) error              { return nil }
func (s *sessionStore) RememberRefresh(string, string) error {
	s.rememberedTxs++
	return nil
}
func (s *sessionStore) Audit(_ int32, action string) error {
	s.audits = append(s.audits, action)
	return nil
}

// verifier accepts exactly one current password and hashes anything long enough.
type verifier struct{ current string }

func (v verifier) Verify(_ context.Context, encoded, password string) (bool, error) {
	return encoded == "bcrypt:current" && password == v.current, nil
}
func (v verifier) Hash(_ context.Context, password string) (string, error) {
	if len(password) < 15 {
		return "", failure.Validation().Add("newPassword", "Kata sandi baru minimal 15 karakter dan maksimal 72 byte.").Err()
	}
	return "bcrypt:next", nil
}

func sessionUsecase(store *sessionStore) authusecase.SessionUsecase {
	reader := implementation.NewAuthUsecase(store, authserializer.NewAuthSerializer())
	tokens := security.Tokens{Key: []byte(strings.Repeat("k", 32)), Issuer: "vms", Audience: "staff"}
	return implementation.NewSessionUsecase(store, reader, verifier{current: "current synthetic passphrase"}, tokens, []byte(strings.Repeat("r", 32)), []byte(strings.Repeat("c", 32)))
}

func TestWrongCurrentPasswordIsAFieldErrorThatKeepsTheSession(t *testing.T) {
	store := newSessionStore()
	before := store.session
	_, err := sessionUsecase(store).ChangePassword(context.Background(), store.proof(), dto.PasswordChange{CurrentPassword: "wrong synthetic passphrase", NewPassword: "next synthetic passphrase"})
	var got *failure.Error
	if !errors.As(err, &got) || got.Status != 422 || got.Code != "validation_failed" || len(got.Fields["currentPassword"]) == 0 {
		t.Fatalf("got %v", err)
	}
	if errors.Is(err, authusecase.ErrUnauthenticated) {
		t.Fatal("wrong current password would sign the user out")
	}
	if len(store.saved) != 0 || store.passwordsSet != 0 || len(store.audits) != 0 || store.rememberedTxs != 0 {
		t.Fatalf("session or credential written: saves=%d passwords=%d audits=%v", len(store.saved), store.passwordsSet, store.audits)
	}
	if store.session != before || store.session.RevokedAt != nil {
		t.Fatal("session changed")
	}
}

func TestPasswordChangeOnAMissingSessionSignsOutInsteadOfFailing(t *testing.T) {
	for name, missing := range map[string]error{
		"gorm":       gorm.ErrRecordNotFound,
		"repository": fmt.Errorf("session snapshot: %w", fmt.Errorf("auth: %w", failure.ErrNotFound)),
	} {
		t.Run(name, func(t *testing.T) {
			store := newSessionStore()
			store.snapshotErr = missing
			_, err := sessionUsecase(store).ChangePassword(context.Background(), store.proof(), dto.PasswordChange{CurrentPassword: "current synthetic passphrase", NewPassword: "next synthetic passphrase"})
			if !errors.Is(err, authusecase.ErrUnauthenticated) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestSessionMutationOnAMissingSessionRowSignsOutInsteadOfFailing(t *testing.T) {
	for name, missing := range map[string]error{
		"gorm":       gorm.ErrRecordNotFound,
		"repository": fmt.Errorf("lock session: %w", failure.ErrNotFound),
	} {
		t.Run(name, func(t *testing.T) {
			store := newSessionStore()
			store.sessionErr = missing
			usecase := sessionUsecase(store)
			if _, err := usecase.Renew(context.Background(), store.proof()); !errors.Is(err, authusecase.ErrUnauthenticated) {
				t.Fatalf("renew: %v", err)
			}
			if _, err := usecase.SwitchRole(context.Background(), store.proof(), "PA"); !errors.Is(err, authusecase.ErrUnauthenticated) {
				t.Fatalf("switch role: %v", err)
			}
			if len(store.saved) != 0 {
				t.Fatal("missing session written")
			}
		})
	}
}

var utcTimestamp = regexp.MustCompile(`"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z"`)
var anyTimestamp = regexp.MustCompile(`"\d{4}-\d{2}-\d{2}T[^"]*"`)

func TestIssuedTokenTimestampsAreUTCEvenOnAWITAHost(t *testing.T) {
	local := time.Local
	time.Local = time.FixedZone("WITA", 8*60*60)
	t.Cleanup(func() { time.Local = local })
	store := newSessionStore()
	result, err := sessionUsecase(store).ChangePassword(context.Background(), store.proof(), dto.PasswordChange{CurrentPassword: "current synthetic passphrase", NewPassword: "next synthetic passphrase"})
	if err != nil {
		t.Fatal(err)
	}
	if store.passwordsSet != 1 || store.session.CredentialGeneration != 3 {
		t.Fatal("password change did not complete")
	}
	encoded, err := json.Marshal(result.Result)
	if err != nil {
		t.Fatal(err)
	}
	all := anyTimestamp.FindAllString(string(encoded), -1)
	if len(all) != 3 || len(utcTimestamp.FindAllString(string(encoded), -1)) != len(all) {
		t.Fatalf("non-UTC timestamp in %s", encoded)
	}
}
