package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	bookingrepo "github.com/azhkadaes/vms/backend/internal/repository/booking/implementation"
	authserializer "github.com/azhkadaes/vms/backend/internal/serializer/auth/implementation"
	staffserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking/implementation"
	auth "github.com/azhkadaes/vms/backend/internal/usecase/auth/implementation"
	staffbooking "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	staffimpl "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking/implementation"
)

func DecisionRouter(deps *Dependencies) {
	validator := auth.NewSessionValidator(authserializer.NewAuthSerializer())
	RegisterDecisionRoutes(deps, staffimpl.NewDecisionUsecase(bookingrepo.NewDecisionRepository(deps.DB), validator, staffserializer.NewStaffBookingSerializer(), deps.Tracer))
}

func RegisterDecisionRoutes(deps *Dependencies, usecase staffbooking.DecisionUsecase) {
	h := handler.NewDecisionHandler(usecase)
	deps.App.Post("/v1/staff/bookings/:id/approve", deps.StaffAuth, deps.Workspace, h.Approve)
	deps.App.Post("/v1/staff/bookings/:id/reject", deps.StaffAuth, deps.Workspace, h.Reject)
	deps.App.Post("/v1/staff/bookings/:id/cancel", deps.StaffAuth, deps.Workspace, h.Cancel)
	deps.App.Post("/v1/staff/bookings/:id/reschedule", deps.StaffAuth, deps.Workspace, h.Reschedule)
	deps.App.Post("/v1/staff/bookings/:id/update", deps.StaffAuth, deps.Workspace, h.UpdateRestricted)
}
