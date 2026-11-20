package entity

import "time"

// Upload audiences: the route an upload came through. Tokens and replays never cross audiences.
const (
	AudiencePublic = "public"
	AudienceStaff  = "staff"
)

// UploadRequest is one received multipart upload before any field or content check.
type UploadRequest struct {
	Reference string
	Purpose   string
	Filename  string
	Content   []byte
}

// FormDocument is the metadata of one stored file (vms.form_documents). StorageKey is private.
type FormDocument struct {
	ID         int32
	Name       string
	StorageKey string
	MimeType   string
	Size       int32
	Purpose    string
}

// UploadRecord is the lifecycle of one upload (support.form_document_uploads) with its
// document's purpose. The token is not stored: it is re-derived from the fixed columns.
type UploadRecord struct {
	DocumentID    int32
	Audience      string
	OwnerDigest   string
	Reference     string
	Purpose       string
	ContentSHA256 string
	TokenHash     string
	KeyVersion    int16
	CreatedAt     time.Time
	ExpiresAt     time.Time
	LinkedAt      *time.Time
	UploadedBy    *int32
}

// IssuedUpload is a token handed to the uploader, with its expiry and the database time it was read at.
type IssuedUpload struct {
	Token      string
	ExpiresAt  time.Time
	ServerTime time.Time
}
