package bookingfileusecase

import (
	"context"
	"io"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// Opened owns a storage reader that the HTTP response must close.
type Opened struct {
	MIME    string
	Size    int
	Content io.ReadCloser
}

type BookingFileUsecase interface {
	OpenStaffFile(context.Context, dto.Session, int32, string) (Opened, error)
	OpenReceptionFile(context.Context, dto.Session, int32, string, dto.ReceptionResolve) (Opened, error)
}
