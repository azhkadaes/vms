package uploadrepository

import (
	"context"
	"fmt"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// ErrNotFound marks a reference with no committed upload for its audience and owner.
var ErrNotFound = fmt.Errorf("upload: %w", failure.ErrNotFound)

// UploadTransaction is one upload's unit of work. Staff uploads lock their authority first;
// then the reference lock, the replay read, the new-upload allowance and both inserts
// commit together or not at all.
type UploadTransaction interface {
	authrepository.AuthorityLock
	// LockReference holds every other upload of the same audience, owner and reference until commit.
	LockReference(audience, ownerDigest, reference string) error
	// Now is the database clock, read once on first use.
	Now() (time.Time, error)
	// FindUpload returns the committed upload for the reference, or ErrNotFound.
	FindUpload(audience, ownerDigest, reference string) (entity.UploadRecord, error)
	// AdmitUpload records one new upload in bucket unless limit already fall inside window.
	AdmitUpload(bucket string, limit int, window time.Duration) (entity.RequestAdmission, error)
	// CreateDocument inserts the metadata row and returns its ID.
	CreateDocument(document entity.FormDocument) (int32, error)
	// CreateUpload inserts the lifecycle row of an inserted document.
	CreateUpload(record entity.UploadRecord) error
}

// UploadRepository runs upload transactions.
type UploadRepository interface {
	Transaction(ctx context.Context, operation func(tx UploadTransaction) error) error
}
