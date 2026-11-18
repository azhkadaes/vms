-- +migrate Up
-- Password verifiers live apart from the employee profile.
CREATE TABLE support.employee_credentials (
    employee_id INTEGER PRIMARY KEY REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    encoded_hash TEXT NOT NULL CHECK (encoded_hash LIKE 'bcrypt:%'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp()
);
-- INSERT stores the initial password when an employee is created; UPDATE changes it.
GRANT SELECT, INSERT, UPDATE ON support.employee_credentials TO vms_api;

-- +migrate Down
REVOKE SELECT, INSERT, UPDATE ON support.employee_credentials FROM vms_api;
DROP TABLE support.employee_credentials;
