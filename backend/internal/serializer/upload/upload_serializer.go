package uploadserializer

import (
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// UploadSerializer emits only the token, its expiry and the server time; never a storage key or path.
type UploadSerializer interface {
	ToUploadResult(issued entity.IssuedUpload) dto.UploadResult
}
