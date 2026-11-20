-- +migrate Up
-- Actions behind the 14 permission keys; each (feature, action) pair is one permission.
CREATE TABLE vms.action (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    nama VARCHAR(255) NOT NULL CHECK (char_length(btrim(nama)) BETWEEN 1 AND 255)
);
INSERT INTO vms.action(nama)
SELECT u.nama FROM (VALUES
(1, 'Melihat'),
(2, 'Membuat'),
(3, 'Menyetujui'),
(4, 'Menolak'),
(5, 'Mengubah jadwal'),
(6, 'Membatalkan'),
(7, 'Mencatat kedatangan'),
(8, 'Mencatat kepulangan'),
(9, 'Mengekspor'),
(10, 'Mengelola')
) AS u(ord,nama) ORDER BY u.ord;
GRANT SELECT ON vms.action TO vms_api;

-- +migrate Down
REVOKE SELECT ON vms.action FROM vms_api;
DROP TABLE vms.action;
