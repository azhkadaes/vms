package authusecaseimplementation_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	implementation "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	"gorm.io/gorm"
)

type repository struct {
	snapshot entity.SessionSnapshot
	err      error
	calls    int
	ctx      context.Context
}

func (r *repository) GetSessionSnapshot(ctx context.Context, id string) (entity.SessionSnapshot, error) {
	r.calls++
	r.ctx = ctx
	return r.snapshot, r.err
}

func fixture() (entity.SessionSnapshot, entity.SessionProof) {
	now := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	return entity.SessionSnapshot{EmployeeID: 7, Name: "Test employee", AccountActive: true,
			ActiveRole: "PA", AvailableRoles: []string{"PA", "Protocol Officer"}, Permissions: []string{"bookings"}, AssignedOfficerIDs: []int32{19},
			CredentialGeneration: 2, RoleRevision: 3, CreatedAt: now.Add(-time.Hour), LastActivityAt: now.Add(-time.Hour), ServerTime: now,
			IdleExpiresAt: now.Add(7 * time.Hour), AbsoluteExpiresAt: now.Add(23 * time.Hour)},
		entity.SessionProof{SessionID: "82d2bb6d-fdcb-4ff5-a914-91e5e9ce9559", EmployeeID: 7, CredentialGeneration: 2, RoleRevision: 3}
}

func TestCurrentSessionRejectsExpiredRevokedAndStaleAuthority(t *testing.T) {
	tests := []struct {
		name   string
		change func(*entity.SessionSnapshot)
		want   error
	}{
		{"inactive account", func(s *entity.SessionSnapshot) { s.AccountActive = false }, authusecase.ErrUnauthenticated},
		{"employee mismatch", func(s *entity.SessionSnapshot) { s.EmployeeID = 8 }, authusecase.ErrUnauthenticated},
		{"revoked", func(s *entity.SessionSnapshot) { s.RevokedAt = &s.ServerTime }, authusecase.ErrUnauthenticated},
		{"idle boundary", func(s *entity.SessionSnapshot) { s.IdleExpiresAt = s.ServerTime }, authusecase.ErrUnauthenticated},
		{"absolute boundary", func(s *entity.SessionSnapshot) { s.AbsoluteExpiresAt = s.ServerTime }, authusecase.ErrUnauthenticated},
		{"prior password generation", func(s *entity.SessionSnapshot) { s.CredentialGeneration++ }, authusecase.ErrUnauthenticated},
		{"prior role revision", func(s *entity.SessionSnapshot) { s.RoleRevision++ }, authusecase.ErrRoleChanged},
		{"removed selected role", func(s *entity.SessionSnapshot) { s.AvailableRoles = []string{"Protocol Officer"} }, authusecase.ErrRoleChanged},
		{"no assigned roles", func(s *entity.SessionSnapshot) { s.AvailableRoles = nil }, authusecase.ErrUnauthenticated},
		{"absolute lifetime too long", func(s *entity.SessionSnapshot) { s.AbsoluteExpiresAt = s.CreatedAt.Add(25 * time.Hour) }, authusecase.ErrUnauthenticated},
		{"idle lifetime too long", func(s *entity.SessionSnapshot) { s.IdleExpiresAt = s.LastActivityAt.Add(9 * time.Hour) }, authusecase.ErrUnauthenticated},
		{"future activity", func(s *entity.SessionSnapshot) { s.LastActivityAt = s.ServerTime.Add(time.Second) }, authusecase.ErrUnauthenticated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, proof := fixture()
			tt.change(&snapshot)
			r := &repository{snapshot: snapshot}
			u := implementation.NewAuthUsecase(r, authserializer.NewAuthSerializer())
			out, err := u.GetSession(context.Background(), proof)
			if !errors.Is(err, tt.want) || out.Active {
				t.Fatalf("got active=%v error=%v", out.Active, err)
			}
		})
	}
}

func TestSessionProjectionAndWorkspaceGates(t *testing.T) {
	for _, mode := range []string{"selected", "pending", "password change", "protocol"} {
		t.Run(mode, func(t *testing.T) {
			snapshot, proof := fixture()
			switch mode {
			case "pending":
				snapshot.ActiveRole = ""
			case "password change":
				snapshot.MustChangePassword = true
			case "protocol":
				snapshot.ActiveRole = "Protocol Officer"
			}
			r := &repository{snapshot: snapshot}
			u := implementation.NewAuthUsecase(r, authserializer.NewAuthSerializer())
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			out, err := u.GetSession(ctx, proof)
			if err != nil {
				t.Fatal(err)
			}
			if r.ctx != ctx || out.ID != "7" || out.Timing.ServerTime != snapshot.ServerTime.UnixMilli() || out.PasswordMinLength != 15 || out.PasswordMaxBytes != 72 {
				t.Fatal("wrong session projection or context")
			}
			blocked := mode == "pending" || mode == "password change"
			if blocked {
				if len(out.Permissions) != 0 || len(out.AssignedOfficerIDs) != 0 || !errors.Is(authusecase.RequireWorkspace(out), authusecase.ErrWorkspaceBlocked) {
					t.Fatal("gate grants workspace authority")
				}
			} else if authusecase.RequireWorkspace(out) != nil {
				t.Fatal("valid workspace blocked")
			}
			if mode == "selected" && (len(out.Permissions) != 1 || len(out.AssignedOfficerIDs) != 1 || out.AssignedOfficerIDs[0] != "19") {
				t.Fatal("selected authority missing")
			}
			if mode == "protocol" && len(out.AssignedOfficerIDs) != 0 {
				t.Fatal("PA assignments leaked into another role")
			}
			encoded, _ := json.Marshal(out)
			for _, private := range []string{proof.SessionID, "credentialGeneration", "refresh", "encodedHash", "null"} {
				if strings.Contains(string(encoded), private) {
					t.Fatalf("unexpected private/null field %s", private)
				}
			}
			// Projections cannot mutate a cached authority snapshot.
			out.AvailableRoles[0] = "changed"
			if r.snapshot.AvailableRoles[0] != "PA" {
				t.Fatal("aliased role slice")
			}
			if !r.snapshot.IdleExpiresAt.Equal(snapshot.IdleExpiresAt) {
				t.Fatal("read renewed inactivity")
			}
		})
	}
}

func TestMissingSessionAndDatabaseFailureStayDistinct(t *testing.T) {
	_, proof := fixture()
	for _, failure := range []error{gorm.ErrRecordNotFound, context.DeadlineExceeded} {
		r := &repository{err: failure}
		u := implementation.NewAuthUsecase(r, authserializer.NewAuthSerializer())
		_, err := u.GetSession(context.Background(), proof)
		if failure == gorm.ErrRecordNotFound {
			if !errors.Is(err, authusecase.ErrUnauthenticated) {
				t.Fatal(err)
			}
		} else if !errors.Is(err, failure) || errors.Is(err, authusecase.ErrUnauthenticated) {
			t.Fatal("database failure disguised as sign-out")
		}
	}
	r := &repository{}
	u := implementation.NewAuthUsecase(r, authserializer.NewAuthSerializer())
	proof.EmployeeID = 0
	if _, err := u.GetSession(context.Background(), proof); !errors.Is(err, authusecase.ErrUnauthenticated) || r.calls != 0 {
		t.Fatal("invalid proof reached database")
	}
}
