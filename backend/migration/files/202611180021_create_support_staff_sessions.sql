-- +migrate Up
-- Sessions, active role and expiry. Revocation is an UPDATE; rows are never deleted by the runtime.
CREATE TABLE support.staff_sessions (
    id UUID PRIMARY KEY,
    employee_id INTEGER NOT NULL REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    active_role_id INTEGER REFERENCES vms.role(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    credential_generation BIGINT NOT NULL CHECK (credential_generation BETWEEN 1 AND 9007199254740991),
    role_revision BIGINT NOT NULL CHECK (role_revision BETWEEN 1 AND 9007199254740991),
    refresh_hash CHAR(64) NOT NULL UNIQUE CHECK (refresh_hash ~ '^[0-9a-f]{64}$'),
    created_at TIMESTAMPTZ NOT NULL,
    last_activity_at TIMESTAMPTZ NOT NULL,
    idle_expires_at TIMESTAMPTZ NOT NULL,
    absolute_expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    CHECK (idle_expires_at > created_at AND idle_expires_at <= absolute_expires_at),
    CHECK (last_activity_at >= created_at AND last_activity_at < idle_expires_at),
    CHECK (idle_expires_at <= last_activity_at + INTERVAL '8 hours'),
    CHECK (absolute_expires_at > created_at AND absolute_expires_at <= created_at + INTERVAL '24 hours'),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);
CREATE INDEX staff_sessions_employee_idx ON support.staff_sessions(employee_id);
CREATE INDEX staff_sessions_active_role_idx ON support.staff_sessions(active_role_id) WHERE active_role_id IS NOT NULL;
GRANT SELECT, INSERT, UPDATE ON support.staff_sessions TO vms_api;

-- +migrate Down
REVOKE SELECT, INSERT, UPDATE ON support.staff_sessions FROM vms_api;
DROP TABLE support.staff_sessions;
