-- +migrate Up
-- Diagram "Form Documents" plus the agreed purpose (D22); file_data is omitted because the bytes
-- live in private storage (D21). storage_path holds the server-generated UUID storage key, never
-- a client path or URL, and each purpose admits only its verified content types.
CREATE TABLE vms.form_documents (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    nama VARCHAR(255) NOT NULL CHECK (char_length(btrim(nama)) BETWEEN 1 AND 255),
    storage_path VARCHAR(255) NOT NULL UNIQUE
        CHECK (storage_path ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    mime_type VARCHAR(255) NOT NULL CHECK (mime_type IN ('application/pdf','image/jpeg','image/png')),
    ukuran_file_bytes INTEGER NOT NULL CHECK (ukuran_file_bytes BETWEEN 1 AND 5242880),
    purpose VARCHAR(16) NOT NULL CHECK (purpose IN ('document','visitorPhoto')),
    CHECK ((purpose = 'document' AND mime_type = 'application/pdf')
        OR (purpose = 'visitorPhoto' AND mime_type IN ('image/jpeg','image/png')))
);
-- Uploads insert metadata; bookings and file reads select it. Rows are never edited or deleted by the runtime.
GRANT SELECT, INSERT ON vms.form_documents TO vms_api;
GRANT USAGE ON SEQUENCE vms.form_documents_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE vms.form_documents_id_seq FROM vms_api;
REVOKE SELECT, INSERT ON vms.form_documents FROM vms_api;
DROP TABLE vms.form_documents;
