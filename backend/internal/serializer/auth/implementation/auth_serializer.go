package authserializerimplementation

import (
	"strconv"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type serializer struct{}

var _ authserializer.AuthSerializer = (*serializer)(nil)

func NewAuthSerializer() authserializer.AuthSerializer { return &serializer{} }

// ToSession projects an already validated snapshot without private session identifiers.
func (s *serializer) ToSession(in entity.SessionSnapshot) dto.Session {
	out := dto.Session{Active: true, ID: strconv.FormatInt(int64(in.EmployeeID), 10), Name: in.Name,
		Role: in.ActiveRole, AvailableRoles: append([]string{}, in.AvailableRoles...),
		RoleSelectionRequired: in.ActiveRole == "", RoleRevision: in.RoleRevision,
		Permissions: []string{}, AssignedOfficerIDs: []string{}, MustChangePassword: in.MustChangePassword,
		PasswordMinLength: policy.PasswordMinLength, PasswordMaxBytes: policy.PasswordMaxBytes,
		Timing: dto.SessionTiming{ServerTime: in.ServerTime.UnixMilli(), IdleExpiresAt: in.IdleExpiresAt.UTC(), AbsoluteExpiresAt: in.AbsoluteExpiresAt.UTC()}}
	if in.ActiveRole != "" && !in.MustChangePassword {
		out.Permissions = append(out.Permissions, in.Permissions...)
		if in.ActiveRole == policy.RolePA {
			for _, id := range in.AssignedOfficerIDs {
				out.AssignedOfficerIDs = append(out.AssignedOfficerIDs, strconv.FormatInt(int64(id), 10))
			}
		}
	}
	return out
}
