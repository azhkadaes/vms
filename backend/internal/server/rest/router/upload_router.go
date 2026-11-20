package router

import (
	"github.com/azhkadaes/vms/backend/internal/handler"
	uploadcontract "github.com/azhkadaes/vms/backend/internal/usecase/upload"
)

// UploadRouter registers the staff upload operation; the public one lives in PublicRouter's group.
func UploadRouter(deps *Dependencies) {
	RegisterStaffUploadRoutes(deps, deps.Uploads)
}

// RegisterStaffUploadRoutes mounts the staff upload behind StaffAuth and the workspace gate;
// the usecase then rechecks authority and the create permission inside its transaction.
func RegisterStaffUploadRoutes(deps *Dependencies, uploads uploadcontract.UploadUsecase) {
	group := deps.App.Group("/v1/staff/attachments", deps.StaffAuth, deps.Workspace)
	h := handler.NewUploadHandler(uploads)
	group.Post("", h.StaffUpload)
}
