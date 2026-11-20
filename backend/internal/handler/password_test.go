package handler_test

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/config"
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/handler"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	authimplementation "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
)

// passwordStore serves one employee's credential and records every write attempt.
type passwordStore struct {
	encoded string
	writes  []string
}

func (s *passwordStore) GetSessionSnapshot(context.Context, string) (entity.SessionSnapshot, error) {
	return entity.SessionSnapshot{EmployeeID: 7}, nil
}
func (s *passwordStore) GetCredential(context.Context, string) (entity.EmployeeCredential, error) {
	return entity.EmployeeCredential{EmployeeID: 7, EncodedHash: s.encoded, Active: true}, nil
}
func (s *passwordStore) FindRefresh(context.Context, string) (string, error) { return "", nil }
func (s *passwordStore) ReserveLogin(context.Context, string, string) (string, int, error) {
	return "", 0, nil
}
func (s *passwordStore) FinishLogin(context.Context, string, bool) error { return nil }
func (s *passwordStore) Transaction(_ context.Context, _ int32, operation func(authrepository.SessionTransaction) error) error {
	return operation(s)
}
func (s *passwordStore) Now() time.Time { return time.Now().UTC() }
func (s *passwordStore) Credential(int32) (entity.EmployeeCredential, error) {
	return s.GetCredential(context.Background(), "")
}
func (s *passwordStore) Roles(int32) (map[string]int32, error) { return map[string]int32{"PA": 2}, nil }
func (s *passwordStore) Session(string) (entity.StaffSession, error) {
	s.writes = append(s.writes, "lock session")
	return entity.StaffSession{}, nil
}
func (s *passwordStore) Create(entity.StaffSession) error {
	s.writes = append(s.writes, "create")
	return nil
}
func (s *passwordStore) Save(entity.StaffSession) error {
	s.writes = append(s.writes, "save session")
	return nil
}
func (s *passwordStore) SetPassword(int32, string, string) error {
	s.writes = append(s.writes, "set password")
	return nil
}
func (s *passwordStore) Recovery(string, string) (entity.SessionRecovery, error) {
	return entity.SessionRecovery{}, nil
}
func (s *passwordStore) SaveRecovery(entity.SessionRecovery) error {
	s.writes = append(s.writes, "save recovery")
	return nil
}
func (s *passwordStore) ClearRecoveries(string) error {
	s.writes = append(s.writes, "clear recoveries")
	return nil
}
func (s *passwordStore) RememberRefresh(string, string) error {
	s.writes = append(s.writes, "remember refresh")
	return nil
}
func (s *passwordStore) Audit(int32, string) error {
	s.writes = append(s.writes, "audit")
	return nil
}

type unusedReader struct{ t *testing.T }

func (r unusedReader) GetSession(context.Context, entity.SessionProof) (dto.Session, error) {
	r.t.Error("session reissued after a rejected password change")
	return dto.Session{}, authusecase.ErrUnauthenticated
}

func TestPasswordChangeWithWrongCurrentPasswordIsA422FieldErrorAndKeepsTheSession(t *testing.T) {
	verifier, err := authimplementation.NewCredentialVerifier(4, 1)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := verifier.Hash(context.Background(), "current synthetic passphrase")
	if err != nil {
		t.Fatal(err)
	}
	store := &passwordStore{encoded: encoded}
	usecase := authimplementation.NewSessionUsecase(store, unusedReader{t}, verifier, security.Tokens{Key: []byte(strings.Repeat("k", 32))}, []byte(strings.Repeat("r", 32)), []byte(strings.Repeat("c", 32)))
	h := handler.NewAuthHandler(usecase, config.Auth{Origin: "https://staff.example.test", CSRFKey: []byte(strings.Repeat("k", 32))})
	app := transportApp(fiber.Config{}, func(app *fiber.App) {
		app.Use(func(c fiber.Ctx) error {
			c.Locals("vms.proof", entity.SessionProof{SessionID: "82d2bb6d-fdcb-4ff5-a914-91e5e9ce9559", EmployeeID: 7, CredentialGeneration: 2, RoleRevision: 3})
			return c.Next()
		})
		app.Post("/v1/staff/password", h.Password)
	})
	request := httptest.NewRequest("POST", "/v1/staff/password", strings.NewReader(`{"currentPassword":"wrong synthetic passphrase","newPassword":"next synthetic passphrase"}`))
	request.Header.Set("Content-Type", "application/json")
	got := do(t, app, request)
	expectError(t, got, 422, "validation_failed")
	if got.body["message"] != "Periksa isian yang ditandai." {
		t.Fatalf("message %v", got.body["message"])
	}
	if !reflect.DeepEqual(got.body["fieldErrors"], map[string]any{"currentPassword": []any{"Kata sandi saat ini tidak sesuai."}}) {
		t.Fatalf("fieldErrors %v", got.body["fieldErrors"])
	}
	if len(got.header.Values("Set-Cookie")) != 0 {
		t.Fatal("rejected change touched session cookies")
	}
	if len(store.writes) != 0 {
		t.Fatalf("session touched: %v", store.writes)
	}
}
