package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	staffbookingrepository "github.com/azhkadaes/vms/backend/internal/repository/staffbooking/implementation"
	staffbookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	staffbookingcontract "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	staffbookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking/implementation"
)

func StaffBookingRouter(deps *Dependencies) {
	reader := staffbookingrepository.NewStaffBookingReader(deps.DB)
	serializer := staffbookingserializer.NewStaffBookingSerializer()
	RegisterStaffBookingRoutes(deps, staffbookingusecase.NewStaffBookingUsecase(reader, serializer, deps.Tracer))
}

func RegisterStaffBookingRoutes(deps *Dependencies, usecase staffbookingcontract.StaffBookingUsecase) {
	h := handler.NewStaffBookingHandler(usecase)
	deps.App.Get("/v1/staff/bookings", deps.StaffAuth, deps.Workspace, h.List)
	deps.App.Get("/v1/staff/bookings/:id", deps.StaffAuth, deps.Workspace, h.Get)
}
