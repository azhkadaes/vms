-- +migrate Up
-- One default policy per fixed role; role_policy links them and policy_permission holds the grants.
CREATE TABLE vms.policy (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    nama VARCHAR(255) NOT NULL CHECK (char_length(btrim(nama)) BETWEEN 1 AND 255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp(),
    created_by VARCHAR(255) NOT NULL CHECK (created_by = 'system' OR created_by ~ '^[1-9][0-9]{0,9}$'),
    updated_by VARCHAR(255) NOT NULL CHECK (updated_by = 'system' OR updated_by ~ '^[1-9][0-9]{0,9}$'),
    status_aktif BOOLEAN NOT NULL
);
INSERT INTO vms.policy(nama,created_by,updated_by,status_aktif)
SELECT 'Kebijakan bawaan ' || r.nama, 'system', 'system', TRUE FROM vms.role r ORDER BY r.id;
GRANT SELECT, UPDATE ON vms.policy TO vms_api;

-- +migrate Down
REVOKE SELECT, UPDATE ON vms.policy FROM vms_api;
DROP TABLE vms.policy;
