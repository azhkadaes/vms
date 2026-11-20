package roleserializer

import (
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// RoleSerializer projects stored roles with their fixed description and assignable permissions.
type RoleSerializer interface {
	ToRoleDefinition(in entity.Role) dto.RoleDefinition
	ToRoleDefinitions(in []entity.Role) []dto.RoleDefinition
}
