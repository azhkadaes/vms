-- +migrate Up
-- Short-lived sealed successors for a refresh whose response may have been lost.
CREATE TABLE support.refresh_recovery (
    session_id UUID NOT NULL REFERENCES support.staff_sessions(id) ON DELETE RESTRICT,
    request_id UUID NOT NULL,
    predecessor_hash CHAR(64) NOT NULL CHECK (predecessor_hash ~ '^[0-9a-f]{64}$'),
    successor_hash CHAR(64) NOT NULL CHECK (successor_hash ~ '^[0-9a-f]{64}$'),
    successor_cipher BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (session_id, request_id)
);
GRANT SELECT, INSERT, DELETE ON support.refresh_recovery TO vms_api;

-- +migrate Down
REVOKE SELECT, INSERT, DELETE ON support.refresh_recovery FROM vms_api;
DROP TABLE support.refresh_recovery;
