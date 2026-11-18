-- +migrate Up
-- The 14 contract permission keys; nama is the admin screen's label.
CREATE TABLE vms.permission (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    id_feature INTEGER NOT NULL REFERENCES vms.feature(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    id_action INTEGER NOT NULL REFERENCES vms.action(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    permission_key VARCHAR(255) NOT NULL UNIQUE CHECK (permission_key IN
        ('bookings','create','approve','reject','reschedule','cancel','checkin','checkout','people','roles','integrations','reports','reportExport','audit')),
    nama VARCHAR(255) NOT NULL CHECK (char_length(btrim(nama)) BETWEEN 1 AND 255),
    UNIQUE (id_feature, id_action)
);
CREATE INDEX permission_action_idx ON vms.permission(id_action);
INSERT INTO vms.permission(id_feature,id_action,permission_key,nama)
SELECT f.id, a.id, u.permission_key, u.nama FROM (VALUES
(1, 'Kunjungan', 'Melihat', 'bookings', 'Melihat kunjungan'),
(2, 'Kunjungan', 'Membuat', 'create', 'Membuat janji temu'),
(3, 'Kunjungan', 'Menyetujui', 'approve', 'Menyetujui permohonan'),
(4, 'Kunjungan', 'Menolak', 'reject', 'Menolak permohonan'),
(5, 'Kunjungan', 'Mengubah jadwal', 'reschedule', 'Mengubah jadwal'),
(6, 'Kunjungan', 'Membatalkan', 'cancel', 'Membatalkan kunjungan'),
(7, 'Kedatangan & kepulangan', 'Mencatat kedatangan', 'checkin', 'Mencatat kedatangan'),
(8, 'Kedatangan & kepulangan', 'Mencatat kepulangan', 'checkout', 'Mencatat kepulangan'),
(9, 'Laporan', 'Melihat', 'reports', 'Melihat ringkasan laporan'),
(10, 'Laporan', 'Mengekspor', 'reportExport', 'Mengekspor ringkasan'),
(11, 'Pegawai', 'Mengelola', 'people', 'Mengelola pegawai'),
(12, 'Peran & akses', 'Mengelola', 'roles', 'Mengelola akses peran'),
(13, 'Integrasi', 'Mengelola', 'integrations', 'Mengelola integrasi'),
(14, 'Audit', 'Melihat', 'audit', 'Melihat audit')
) AS u(ord,feature,action,permission_key,nama)
JOIN vms.feature f ON f.nama = u.feature
JOIN vms.action a ON a.nama = u.action
ORDER BY u.ord;
GRANT SELECT ON vms.permission TO vms_api;

-- +migrate Down
REVOKE SELECT ON vms.permission FROM vms_api;
DROP TABLE vms.permission;
