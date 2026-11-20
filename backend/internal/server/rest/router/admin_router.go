package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	rolerepository "github.com/azhkadaes/vms/backend/internal/repository/role/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	roleserializer "github.com/azhkadaes/vms/backend/internal/serializer/role/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	rolecontract "github.com/azhkadaes/vms/backend/internal/usecase/role"
	roleusecase "github.com/azhkadaes/vms/backend/internal/usecase/role/implementation"
)

// RoleRouter wires role administration; its writes recheck authority with the shared validator.
func RoleRouter(deps *Dependencies) {
	validator := authusecase.NewSessionValidator(authserializer.NewAuthSerializer())
	roles := roleusecase.NewRoleUsecase(rolerepository.NewRoleRepository(deps.DB), roleserializer.NewRoleSerializer(), validator, deps.Tracer)
	RegisterRoleRoutes(deps, roles)
}

// RegisterRoleRoutes mounts role administration (ops 30, 31) on its own protected staff group.
func RegisterRoleRoutes(deps *Dependencies, roles rolecontract.RoleUsecase) {
	h := handler.NewRoleHandler(roles)
	group := deps.App.Group("/v1/staff/roles", deps.StaffAuth, deps.Workspace)
	group.Get("", h.Roles)
	group.Patch("/:id", h.UpdateRole)
}
