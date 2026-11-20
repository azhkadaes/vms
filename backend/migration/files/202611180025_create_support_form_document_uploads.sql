-- +migrate Up
-- Upload lifecycle of one form_documents row: who uploaded it under which client reference, the
-- content digest that replays must match, and the one-hour token (D18). Only the token's SHA-256
-- is kept; the token is re-derived from these fixed columns and the key_version key (D99).
-- Public uploads are anonymous; staff uploads record the uploading employee.
CREATE TABLE support.form_document_uploads (
    id_form_documents INTEGER PRIMARY KEY REFERENCES vms.form_documents(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    audience VARCHAR(8) NOT NULL CHECK (audience IN ('public','staff')),
    owner_digest CHAR(64) NOT NULL CHECK (owner_digest ~ '^[0-9a-f]{64}$'),
    reference UUID NOT NULL,
    content_sha256 CHAR(64) NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    token_hash CHAR(64) NOT NULL UNIQUE CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    key_version SMALLINT NOT NULL CHECK (key_version > 0),
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    linked_at TIMESTAMPTZ,
    uploaded_by INTEGER REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    UNIQUE (audience, owner_digest, reference),
    CHECK (expires_at = created_at + INTERVAL '1 hour'),
    CHECK ((audience = 'staff') = (uploaded_by IS NOT NULL)),
    -- A token links only while it is valid (D18).
    CHECK (linked_at IS NULL OR (linked_at >= created_at AND linked_at < expires_at))
);
CREATE INDEX form_document_uploads_uploaded_by_idx ON support.form_document_uploads(uploaded_by) WHERE uploaded_by IS NOT NULL;
-- Uploads insert and replay; booking creation links a token by setting linked_at.
GRANT SELECT, INSERT ON support.form_document_uploads TO vms_api;
GRANT UPDATE (linked_at) ON support.form_document_uploads TO vms_api;

-- +migrate Down
REVOKE UPDATE (linked_at) ON support.form_document_uploads FROM vms_api;
REVOKE SELECT, INSERT ON support.form_document_uploads FROM vms_api;
DROP TABLE support.form_document_uploads;
