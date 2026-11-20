package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	filerepo "github.com/azhkadaes/vms/backend/internal/repository/bookingfile/implementation"
	filecontract "github.com/azhkadaes/vms/backend/internal/usecase/bookingfile"
	fileusecase "github.com/azhkadaes/vms/backend/internal/usecase/bookingfile/implementation"
)

func BookingFileRouter(deps *Dependencies) {
	usecase := fileusecase.NewBookingFileUsecase(filerepo.NewBookingFileRepository(deps.DB), deps.BookingFiles, deps.Tracer)
	RegisterBookingFileRoutes(deps, usecase)
}

func RegisterBookingFileRoutes(deps *Dependencies, usecase filecontract.BookingFileUsecase) {
	h := handler.NewBookingFileHandler(usecase)
	deps.App.Get("/v1/staff/bookings/:id/attachment", deps.StaffAuth, deps.Workspace, h.StaffAttachment)
	deps.App.Get("/v1/staff/bookings/:id/visitor-photo", deps.StaffAuth, deps.Workspace, h.StaffVisitorPhoto)
	deps.App.Post("/v1/staff/bookings/:id/attachment", deps.StaffAuth, deps.Workspace, h.ReceptionAttachment)
	deps.App.Post("/v1/staff/bookings/:id/visitor-photo", deps.StaffAuth, deps.Workspace, h.ReceptionVisitorPhoto)
}
