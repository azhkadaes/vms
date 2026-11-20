-- +migrate Up
-- The role set is fixed (ops 30-31); version is contract-required.
-- created_by/updated_by hold the acting employee ID as decimal text, or system.
CREATE TABLE vms.role (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    nama VARCHAR(255) NOT NULL UNIQUE CHECK (nama IN ('PA','Protocol Officer','Receptionist','Super Admin')),
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp(),
    created_by VARCHAR(255) NOT NULL CHECK (created_by = 'system' OR created_by ~ '^[1-9][0-9]{0,9}$'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp(),
    updated_by VARCHAR(255) NOT NULL CHECK (updated_by = 'system' OR updated_by ~ '^[1-9][0-9]{0,9}$')
);
INSERT INTO vms.role(nama,created_by,updated_by)
SELECT u.nama, 'system', 'system' FROM (VALUES
(1, 'PA'),
(2, 'Protocol Officer'),
(3, 'Receptionist'),
(4, 'Super Admin')
) AS u(ord,nama) ORDER BY u.ord;
GRANT SELECT ON vms.role TO vms_api;
-- Role permission edits bump only the version and the update audit columns.
GRANT UPDATE (version, updated_at, updated_by) ON vms.role TO vms_api;

-- +migrate Down
REVOKE UPDATE (version, updated_at, updated_by) ON vms.role FROM vms_api;
REVOKE SELECT ON vms.role FROM vms_api;
DROP TABLE vms.role;
