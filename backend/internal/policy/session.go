package policy

import (
	"slices"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// SessionOperation is a staff session operation, numbered as in the REST contract.
type SessionOperation int

const (
	OpGetSession     SessionOperation = 33
	OpSelectRole     SessionOperation = 34
	OpLogout         SessionOperation = 35
	OpChangePassword SessionOperation = 36
	OpRefreshSession SessionOperation = 39
	OpRenewSession   SessionOperation = 40
)

// SessionStage is how far a session has come towards the workspace.
type SessionStage int

const (
	StageSignedOut SessionStage = iota
	// StageRoleSelection comes first even when a password change is also pending,
	// because the UI asks for the role before the new password.
	StageRoleSelection
	StagePasswordChange
	StageWorkspace
)

// stageOperations lists the session operations each stage may call (A-18).
var stageOperations = map[SessionStage][]SessionOperation{
	StageRoleSelection:  {OpGetSession, OpSelectRole, OpLogout, OpRefreshSession, OpRenewSession},
	StagePasswordChange: {OpGetSession, OpLogout, OpChangePassword, OpRefreshSession, OpRenewSession},
	StageWorkspace:      {OpGetSession, OpSelectRole, OpLogout, OpChangePassword, OpRefreshSession, OpRenewSession},
}

// Stage places a session on the way to the workspace.
func Stage(s dto.Session) SessionStage {
	switch {
	case !s.Active:
		return StageSignedOut
	case s.RoleSelectionRequired || s.Role == "":
		return StageRoleSelection
	case s.MustChangePassword:
		return StagePasswordChange
	}
	return StageWorkspace
}

// WorkspaceReady reports whether s may call business operations; they still check permission and scope.
func WorkspaceReady(s dto.Session) bool {
	return Stage(s) == StageWorkspace
}

// AllowsSessionOperation reports whether a session in its current stage may call op (A-18).
func AllowsSessionOperation(s dto.Session, op SessionOperation) bool {
	return slices.Contains(stageOperations[Stage(s)], op)
}
