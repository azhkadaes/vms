-- +migrate Up
CREATE TABLE vms.policy_permission (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    id_permission INTEGER NOT NULL REFERENCES vms.permission(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    id_policy INTEGER NOT NULL REFERENCES vms.policy(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    UNIQUE (id_policy, id_permission)
);
CREATE INDEX policy_permission_permission_idx ON vms.policy_permission(id_permission);
-- Default grants per role, from the frontend's role defaults (packages/data/src/domain.ts sessionFor).
INSERT INTO vms.policy_permission(id_permission,id_policy)
SELECT p.id, rp.id_policy FROM (VALUES
('PA', 'bookings'), ('PA', 'create'), ('PA', 'approve'), ('PA', 'reject'),
('PA', 'reschedule'), ('PA', 'cancel'), ('PA', 'reports'), ('PA', 'reportExport'),
('Protocol Officer', 'bookings'), ('Protocol Officer', 'create'), ('Protocol Officer', 'approve'), ('Protocol Officer', 'reject'),
('Protocol Officer', 'reschedule'), ('Protocol Officer', 'cancel'), ('Protocol Officer', 'reports'), ('Protocol Officer', 'reportExport'),
('Receptionist', 'bookings'), ('Receptionist', 'checkin'), ('Receptionist', 'checkout'),
('Super Admin', 'bookings'), ('Super Admin', 'people'), ('Super Admin', 'roles'), ('Super Admin', 'integrations'),
('Super Admin', 'reports'), ('Super Admin', 'reportExport'), ('Super Admin', 'audit')
) AS u(role_nama,permission_key)
JOIN vms.role r ON r.nama = u.role_nama
JOIN vms.role_policy rp ON rp.id_role = r.id AND rp.status_aktif
JOIN vms.permission p ON p.permission_key = u.permission_key
ORDER BY r.id, p.id;
-- Role permission edits replace a policy's permission rows.
GRANT SELECT, INSERT, DELETE ON vms.policy_permission TO vms_api;
GRANT USAGE ON SEQUENCE vms.policy_permission_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE vms.policy_permission_id_seq FROM vms_api;
REVOKE SELECT, INSERT, DELETE ON vms.policy_permission FROM vms_api;
DROP TABLE vms.policy_permission;
