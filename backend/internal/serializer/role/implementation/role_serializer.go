package roleserializerimplementation

import (
	"strconv"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	roleserializer "github.com/azhkadaes/vms/backend/internal/serializer/role"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type serializer struct{}

var _ roleserializer.RoleSerializer = (*serializer)(nil)

func NewRoleSerializer() roleserializer.RoleSerializer { return &serializer{} }

// ToRoleDefinition copies the stored permissions and adds the role's fixed description, assignable
// permissions and editability. Lists are never null.
func (s *serializer) ToRoleDefinition(in entity.Role) dto.RoleDefinition {
	return dto.RoleDefinition{
		ID:                    strconv.FormatInt(int64(in.ID), 10),
		Name:                  in.Name,
		Description:           policy.RoleDescription(in.Name),
		Permissions:           append([]string{}, in.Permissions...),
		AssignablePermissions: policy.AssignablePermissions(in.Name),
		Editable:              policy.RoleEditable(in.Name, in.PolicyID > 0),
		Version:               in.Version,
	}
}

// ToRoleDefinitions keeps the stored order; an empty list is [], never null.
func (s *serializer) ToRoleDefinitions(in []entity.Role) []dto.RoleDefinition {
	out := make([]dto.RoleDefinition, 0, len(in))
	for _, role := range in {
		out = append(out, s.ToRoleDefinition(role))
	}
	return out
}
