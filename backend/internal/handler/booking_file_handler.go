package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/policy"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	bookingfile "github.com/azhkadaes/vms/backend/internal/usecase/bookingfile"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type bookingFileHandler struct {
	usecase bookingfile.BookingFileUsecase
}

func NewBookingFileHandler(usecase bookingfile.BookingFileUsecase) *bookingFileHandler {
	return &bookingFileHandler{usecase: usecase}
}

func (h *bookingFileHandler) StaffAttachment(c fiber.Ctx) error {
	return h.staff(c, policy.PurposeDocument)
}

func (h *bookingFileHandler) StaffVisitorPhoto(c fiber.Ctx) error {
	return h.staff(c, policy.PurposeVisitorPhoto)
}

func (h *bookingFileHandler) ReceptionAttachment(c fiber.Ctx) error {
	return h.reception(c, policy.PurposeDocument)
}

func (h *bookingFileHandler) ReceptionVisitorPhoto(c fiber.Ctx) error {
	return h.reception(c, policy.PurposeVisitorPhoto)
}

func bookingFileID(c fiber.Ctx) (int32, error) {
	if err := NoQuery(c); err != nil {
		return 0, err
	}
	return validate.ID(c.Params("id"))
}

func (h *bookingFileHandler) staff(c fiber.Ctx, purpose string) error {
	id, err := bookingFileID(c)
	if err != nil {
		return err
	}
	file, err := h.usecase.OpenStaffFile(c.Context(), restmiddleware.Session(c), id, purpose)
	if err != nil {
		return err
	}
	return sendBookingFile(c, file)
}

func (h *bookingFileHandler) reception(c fiber.Ctx, purpose string) error {
	id, err := bookingFileID(c)
	if err != nil {
		return err
	}
	var request dto.ReceptionResolve
	if err := DecodeJSON(c, &request); err != nil {
		return err
	}
	file, err := h.usecase.OpenReceptionFile(c.Context(), restmiddleware.Session(c), id, purpose, request)
	if err != nil {
		return err
	}
	return sendBookingFile(c, file)
}

func sendBookingFile(c fiber.Ctx, file bookingfile.Opened) error {
	c.Set("Content-Type", file.MIME)
	c.Set("Cache-Control", "no-store")
	// Fiber takes ownership of the reader and closes it after the response.
	if err := c.SendStream(file.Content, file.Size); err != nil {
		_ = file.Content.Close()
		return err
	}
	return nil
}
