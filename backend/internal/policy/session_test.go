package policy

import (
	"slices"
	"testing"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

var sessionOperations = []SessionOperation{OpGetSession, OpSelectRole, OpLogout, OpChangePassword, OpRefreshSession, OpRenewSession}

func TestPendingSessionsReachOnlyTheSessionOperationsListedInA18(t *testing.T) {
	for _, tc := range []struct {
		name    string
		session dto.Session
		allowed []int
	}{
		{"role selection required", dto.Session{Active: true, RoleSelectionRequired: true, AvailableRoles: []string{RolePA, RoleReceptionist}}, []int{33, 34, 35, 39, 40}},
		{"password change required", dto.Session{Active: true, Role: RolePA, AvailableRoles: []string{RolePA}, MustChangePassword: true}, []int{33, 35, 36, 39, 40}},
		// The UI asks for the role first, then the password; blocking 34 here would strand the session.
		{"role selection before a pending password change", dto.Session{Active: true, RoleSelectionRequired: true, MustChangePassword: true, AvailableRoles: []string{RolePA, RoleProtocol}}, []int{33, 34, 35, 39, 40}},
		{"ready workspace", dto.Session{Active: true, Role: RoleProtocol, AvailableRoles: []string{RoleProtocol}, Permissions: []string{PermBookings}}, []int{33, 34, 35, 36, 39, 40}},
		{"signed out", dto.Session{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, op := range sessionOperations {
				if got, want := AllowsSessionOperation(tc.session, op), slices.Contains(tc.allowed, int(op)); got != want {
					t.Errorf("operation %d allowed = %v, want %v", op, got, want)
				}
			}
		})
	}
}

func TestOnlyAReadySessionEntersTheWorkspace(t *testing.T) {
	for _, tc := range []struct {
		name    string
		session dto.Session
		want    SessionStage
	}{
		{"signed out", dto.Session{Role: RolePA}, StageSignedOut},
		{"role selection required", dto.Session{Active: true, RoleSelectionRequired: true}, StageRoleSelection},
		{"no active role", dto.Session{Active: true}, StageRoleSelection},
		{"role selection before password change", dto.Session{Active: true, RoleSelectionRequired: true, MustChangePassword: true}, StageRoleSelection},
		{"password change required", dto.Session{Active: true, Role: RolePA, MustChangePassword: true}, StagePasswordChange},
		{"ready", dto.Session{Active: true, Role: RoleSuperAdmin}, StageWorkspace},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Stage(tc.session); got != tc.want {
				t.Fatalf("stage = %v, want %v", got, tc.want)
			}
			if got := WorkspaceReady(tc.session); got != (tc.want == StageWorkspace) {
				t.Fatalf("workspace ready = %v", got)
			}
		})
	}
}

func TestPendingSessionsHoldNoPermission(t *testing.T) {
	for _, session := range []dto.Session{
		{Active: true, RoleSelectionRequired: true, Permissions: []string{PermBookings}},
		{Active: true, Permissions: []string{PermBookings}},
		{Active: true, Role: RolePA, MustChangePassword: true, Permissions: []string{PermBookings}},
		{Role: RolePA, Permissions: []string{PermBookings}},
	} {
		if Has(session, PermBookings) {
			t.Fatalf("pending or signed-out session holds a permission: %+v", session)
		}
	}
	if !Has(dto.Session{Active: true, Role: RolePA, Permissions: []string{PermBookings}}, PermBookings) {
		t.Fatal("ready session lost its permission")
	}
}
