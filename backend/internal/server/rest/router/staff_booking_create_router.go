package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	staffserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	auth "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	staff "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	staffusecase "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking/implementation"
)

func staffBookingCreateRouter(deps *Dependencies) {
	validator := auth.NewSessionValidator(authserializer.NewAuthSerializer())
	creator := staffusecase.NewStaffBookingCreator(bookingrepo.NewStaffBookingWriter(deps.DB), validator,
		staffserializer.NewStaffBookingSerializer(), deps.Cfg.RateKey, deps.Tracer)
	RegisterStaffBookingCreateRoute(deps, creator)
}

func RegisterStaffBookingCreateRoute(deps *Dependencies, creator staff.StaffBookingCreateUsecase) {
	h := handler.NewStaffBookingCreateHandler(creator)
	deps.App.Post("/v1/staff/bookings", deps.StaffAuth, deps.Workspace, h.Create)
}
