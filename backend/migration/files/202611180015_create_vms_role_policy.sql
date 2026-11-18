-- +migrate Up
-- At most one active policy per role and one active role per policy.
CREATE TABLE vms.role_policy (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    id_policy INTEGER NOT NULL REFERENCES vms.policy(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    id_role INTEGER NOT NULL REFERENCES vms.role(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp(),
    created_by VARCHAR(255) NOT NULL CHECK (created_by = 'system' OR created_by ~ '^[1-9][0-9]{0,9}$'),
    updated_by VARCHAR(255) NOT NULL CHECK (updated_by = 'system' OR updated_by ~ '^[1-9][0-9]{0,9}$'),
    status_aktif BOOLEAN NOT NULL,
    UNIQUE (id_role, id_policy)
);
CREATE UNIQUE INDEX role_policy_active_role_idx ON vms.role_policy(id_role) WHERE status_aktif;
CREATE UNIQUE INDEX role_policy_active_policy_idx ON vms.role_policy(id_policy) WHERE status_aktif;
CREATE INDEX role_policy_policy_idx ON vms.role_policy(id_policy);
INSERT INTO vms.role_policy(id_policy,id_role,created_by,updated_by,status_aktif)
SELECT p.id, r.id, 'system', 'system', TRUE
FROM vms.role r JOIN vms.policy p ON p.nama = 'Kebijakan bawaan ' || r.nama ORDER BY r.id;
GRANT SELECT ON vms.role_policy TO vms_api;

-- +migrate Down
REVOKE SELECT ON vms.role_policy FROM vms_api;
DROP TABLE vms.role_policy;
