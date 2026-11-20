package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/internal/entity"
	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	uploadusecase "github.com/azhkadaes/vms/backend/internal/usecase/upload"
)

type uploadHandler struct{ usecase uploadusecase.UploadUsecase }

// NewUploadHandler serves the public and staff upload operations, which share one request shape.
func NewUploadHandler(usecase uploadusecase.UploadUsecase) *uploadHandler {
	return &uploadHandler{usecase: usecase}
}

// PublicUpload accepts a visitor's document or photo (op 02): 201 when created, 200 when replayed.
func (h *uploadHandler) PublicUpload(c fiber.Ctx) error {
	upload, err := uploadRequest(c)
	if err != nil {
		return err
	}
	issued, err := h.usecase.UploadPublic(c.Context(), c.IP(), upload)
	if err != nil {
		return err
	}
	return uploaded(c, issued)
}

// StaffUpload accepts an upload by the signed-in staff member (op 19): 201 when created, 200 when replayed.
func (h *uploadHandler) StaffUpload(c fiber.Ctx) error {
	upload, err := uploadRequest(c)
	if err != nil {
		return err
	}
	issued, err := h.usecase.UploadStaff(c.Context(), restmiddleware.Proof(c), c.IP(), upload)
	if err != nil {
		return err
	}
	return uploaded(c, issued)
}

// uploadRequest takes no query and exactly the multipart parts file, reference and purpose.
func uploadRequest(c fiber.Ctx) (entity.UploadRequest, error) {
	if err := NoQuery(c); err != nil {
		return entity.UploadRequest{}, err
	}
	form, err := Multipart(c, uploadusecase.MaxFileBytes)
	if err != nil {
		return entity.UploadRequest{}, err
	}
	return entity.UploadRequest{Reference: form.Reference, Purpose: form.Purpose, Filename: form.Filename, Content: form.File}, nil
}

func uploaded(c fiber.Ctx, issued uploadusecase.Issued) error {
	status := fiber.StatusOK
	if issued.Created {
		status = fiber.StatusCreated
	}
	return c.Status(status).JSON(issued.Result)
}
