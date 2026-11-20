-- +migrate Up
-- Features behind the 14 permission keys, named as the admin role screen groups them.
CREATE TABLE vms.feature (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    nama VARCHAR(255) NOT NULL CHECK (char_length(btrim(nama)) BETWEEN 1 AND 255)
);
INSERT INTO vms.feature(nama)
SELECT u.nama FROM (VALUES
(1, 'Kunjungan'),
(2, 'Kedatangan & kepulangan'),
(3, 'Laporan'),
(4, 'Pegawai'),
(5, 'Peran & akses'),
(6, 'Integrasi'),
(7, 'Audit')
) AS u(ord,nama) ORDER BY u.ord;
GRANT SELECT ON vms.feature TO vms_api;

-- +migrate Down
REVOKE SELECT ON vms.feature FROM vms_api;
DROP TABLE vms.feature;
