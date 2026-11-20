package uploadrepositoryimplementation

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	admissionrepositoryimplementation "github.com/azhkadaes/vms/backend/internal/repository/admission/implementation"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authrepositoryimplementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	uploadrepository "github.com/azhkadaes/vms/backend/internal/repository/upload"
)

type repo struct{ db *gorm.DB }

var _ uploadrepository.UploadRepository = (*repo)(nil)

// NewUploadRepository runs upload transactions on db; nested in an open transaction they become savepoints.
func NewUploadRepository(db *gorm.DB) uploadrepository.UploadRepository { return &repo{db: db} }

// Transaction runs operation in one Read Committed transaction and rolls back on any error.
func (r *repo) Transaction(ctx context.Context, operation func(uploadrepository.UploadTransaction) error) error {
	err := r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		return operation(&uploadTx{AuthorityLock: authrepositoryimplementation.NewAuthorityLock(db), db: db})
	})
	if err != nil {
		return fmt.Errorf("upload transaction: %w", err)
	}
	return nil
}

type uploadTx struct {
	authrepository.AuthorityLock
	db  *gorm.DB
	now *time.Time
}

// LockReference takes a transaction-scoped advisory lock on the reference's key, which also
// covers a reference with no row yet, so concurrent first uploads run one after another.
func (t *uploadTx) LockReference(audience, ownerDigest, reference string) error {
	key := "vms.form_document_uploads:" + audience + ":" + ownerDigest + ":" + reference
	if err := t.db.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error; err != nil {
		return fmt.Errorf("lock upload reference: %w", err)
	}
	return nil
}

// Now reads clock_timestamp() once, on first use after the locks are held.
func (t *uploadTx) Now() (time.Time, error) {
	if t.now == nil {
		var now time.Time
		if err := t.db.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return time.Time{}, fmt.Errorf("read upload time: %w", err)
		}
		t.now = &now
	}
	return *t.now, nil
}

// uploadRow names every column explicitly; GORM's naming would not map content_sha256.
type uploadRow struct {
	DocumentID    int32      `gorm:"column:document_id"`
	Audience      string     `gorm:"column:audience"`
	OwnerDigest   string     `gorm:"column:owner_digest"`
	Reference     string     `gorm:"column:reference"`
	Purpose       string     `gorm:"column:purpose"`
	ContentSHA256 string     `gorm:"column:content_sha256"`
	TokenHash     string     `gorm:"column:token_hash"`
	KeyVersion    int16      `gorm:"column:key_version"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	ExpiresAt     time.Time  `gorm:"column:expires_at"`
	LinkedAt      *time.Time `gorm:"column:linked_at"`
	UploadedBy    *int32     `gorm:"column:uploaded_by"`
}

// FindUpload reads the committed upload of the reference with its document's purpose.
func (t *uploadTx) FindUpload(audience, ownerDigest, reference string) (entity.UploadRecord, error) {
	var row uploadRow
	result := t.db.Raw(`SELECT u.id_form_documents AS document_id, u.audience, u.owner_digest, u.reference::text AS reference, d.purpose,
		u.content_sha256, u.token_hash, u.key_version, u.created_at, u.expires_at, u.linked_at, u.uploaded_by
		FROM support.form_document_uploads u JOIN vms.form_documents d ON d.id = u.id_form_documents
		WHERE u.audience = ? AND u.owner_digest = ? AND u.reference = ?::uuid`, audience, ownerDigest, reference).Scan(&row)
	if result.Error != nil {
		return entity.UploadRecord{}, fmt.Errorf("find upload: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return entity.UploadRecord{}, fmt.Errorf("find upload: %w", uploadrepository.ErrNotFound)
	}
	return entity.UploadRecord{DocumentID: row.DocumentID, Audience: row.Audience, OwnerDigest: row.OwnerDigest, Reference: row.Reference, Purpose: row.Purpose,
		ContentSHA256: row.ContentSHA256, TokenHash: row.TokenHash, KeyVersion: row.KeyVersion, CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt,
		LinkedAt: row.LinkedAt, UploadedBy: row.UploadedBy}, nil
}

// AdmitUpload takes one slot of the bucket on this transaction, so a rolled-back upload uses none.
func (t *uploadTx) AdmitUpload(bucket string, limit int, window time.Duration) (entity.RequestAdmission, error) {
	admission, err := admissionrepositoryimplementation.Admit(t.db, bucket, limit, window)
	if err != nil {
		return entity.RequestAdmission{}, fmt.Errorf("admit upload: %w", err)
	}
	return admission, nil
}

// CreateDocument inserts the metadata row; the storage key goes in storage_path.
func (t *uploadTx) CreateDocument(document entity.FormDocument) (int32, error) {
	var id int32
	if err := t.db.Raw(`INSERT INTO vms.form_documents (nama, storage_path, mime_type, ukuran_file_bytes, purpose) VALUES (?, ?, ?, ?, ?) RETURNING id`,
		document.Name, document.StorageKey, document.MimeType, document.Size, document.Purpose).Scan(&id).Error; err != nil {
		return 0, fmt.Errorf("insert form document: %w", err)
	}
	if id <= 0 {
		return 0, fmt.Errorf("insert form document returned no id")
	}
	return id, nil
}

// CreateUpload inserts the lifecycle row, unlinked.
func (t *uploadTx) CreateUpload(record entity.UploadRecord) error {
	if err := t.db.Exec(`INSERT INTO support.form_document_uploads
		(id_form_documents, audience, owner_digest, reference, content_sha256, token_hash, key_version, created_at, expires_at, uploaded_by)
		VALUES (?, ?, ?, ?::uuid, ?, ?, ?, ?, ?, ?)`,
		record.DocumentID, record.Audience, record.OwnerDigest, record.Reference, record.ContentSHA256, record.TokenHash, record.KeyVersion,
		record.CreatedAt, record.ExpiresAt, record.UploadedBy).Error; err != nil {
		return fmt.Errorf("insert upload: %w", err)
	}
	return nil
}
