package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	receptionrepository "github.com/azhkadaes/vms/backend/internal/repository/reception/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	staffbookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	receptioncontract "github.com/azhkadaes/vms/backend/internal/usecase/reception"
	receptionusecase "github.com/azhkadaes/vms/backend/internal/usecase/reception/implementation"
)

func ReceptionRouter(deps *Dependencies) {
	repo := receptionrepository.NewReceptionRepository(deps.DB)
	serializer := staffbookingserializer.NewStaffBookingSerializer()
	validator := authusecase.NewSessionValidator(authserializer.NewAuthSerializer())
	RegisterReceptionRoutes(deps, receptionusecase.NewReceptionUsecase(repo, serializer, validator, deps.Tracer))
}

func RegisterReceptionRoutes(deps *Dependencies, usecase receptioncontract.ReceptionUsecase) {
	h := handler.NewReceptionHandler(usecase)
	deps.App.Post("/v1/staff/resolve", deps.StaffAuth, deps.Workspace, h.Resolve)
	deps.App.Post("/v1/staff/bookings/:id/attendance", deps.StaffAuth, deps.Workspace, h.Attendance)
}
