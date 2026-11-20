package policy

import (
	"slices"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

// Upload purposes (D22): a Nota Dinas document or a representative visitor photo.
const (
	PurposeDocument     = "document"
	PurposeVisitorPhoto = "visitorPhoto"
)

// uploadTypes lists the verified content types each purpose accepts (D22).
var uploadTypes = map[string][]string{
	PurposeDocument:     {"application/pdf"},
	PurposeVisitorPhoto: {"image/jpeg", "image/png"},
}

// UploadPurpose reports whether purpose is one of the two upload purposes.
func UploadPurpose(purpose string) bool {
	_, ok := uploadTypes[purpose]
	return ok
}

// AcceptsUpload reports whether content verified as mime may be stored for purpose.
func AcceptsUpload(purpose, mime string) bool {
	return slices.Contains(uploadTypes[purpose], mime)
}

// CanUpload admits a staff upload: uploads feed staff visit creation, so the session must be
// in the workspace and its active role must hold the create permission.
func CanUpload(s dto.Session) bool {
	return WorkspaceReady(s) && slices.Contains(s.Permissions, PermCreate)
}
