package uploadusecase

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// MaxFileBytes is the per-file cap (5 MiB, D67) shared by the multipart reader and the usecase.
const MaxFileBytes = 5 << 20

// Issued is an upload result plus whether this request created it (201) or replayed it (200).
type Issued struct {
	Result  dto.UploadResult
	Created bool
}

// UploadUsecase verifies uploaded content against its purpose, stores it privately and issues
// a purpose-bound one-hour token. A replay of the same reference and content returns the same token.
type UploadUsecase interface {
	// UploadPublic accepts an anonymous visitor upload from the trusted client IP.
	UploadPublic(ctx context.Context, ip string, upload entity.UploadRequest) (Issued, error)
	// UploadStaff accepts an upload by the staff session the proof names.
	UploadStaff(ctx context.Context, proof entity.SessionProof, ip string, upload entity.UploadRequest) (Issued, error)
}
