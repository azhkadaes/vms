package uploadserializerimplementation

import (
	"github.com/azhkadaes/vms/backend/internal/entity"
	uploadserializer "github.com/azhkadaes/vms/backend/internal/serializer/upload"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type serializer struct{}

var _ uploadserializer.UploadSerializer = (*serializer)(nil)

// NewUploadSerializer projects issued uploads; it has no access to storage keys or metadata.
func NewUploadSerializer() uploadserializer.UploadSerializer { return &serializer{} }

// ToUploadResult emits the expiry in UTC and the server time in epoch milliseconds.
func (s *serializer) ToUploadResult(in entity.IssuedUpload) dto.UploadResult {
	return dto.UploadResult{Token: in.Token, ExpiresAt: in.ExpiresAt.UTC(), ServerTime: in.ServerTime.UnixMilli()}
}
