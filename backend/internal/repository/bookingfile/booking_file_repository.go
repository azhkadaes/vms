package bookingfilerepository

import (
	"context"
	"fmt"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

var ErrNotFound = fmt.Errorf("booking file: %w", failure.ErrNotFound)

type BookingFileRepository interface {
	Lookup(context.Context, int32, string, *entity.ReceptionCredential) (entity.BookingFile, error)
}
