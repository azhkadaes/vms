-- +migrate Up
-- Consumed refresh hashes, kept to detect refresh-secret reuse.
CREATE TABLE support.refresh_lineage (
    token_hash CHAR(64) PRIMARY KEY CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    session_id UUID NOT NULL REFERENCES support.staff_sessions(id) ON DELETE RESTRICT,
    consumed_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX refresh_lineage_session_idx ON support.refresh_lineage(session_id);
GRANT SELECT, INSERT ON support.refresh_lineage TO vms_api;

-- +migrate Down
REVOKE SELECT, INSERT ON support.refresh_lineage FROM vms_api;
DROP TABLE support.refresh_lineage;
