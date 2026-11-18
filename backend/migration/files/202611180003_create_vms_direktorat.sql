-- +migrate Up
-- Seed units only: officials, employees, roles and authority are never created here.
CREATE TABLE vms.direktorat (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    id_kedeputian INTEGER NOT NULL REFERENCES vms.kedeputian(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    nama VARCHAR(255) NOT NULL UNIQUE CHECK (char_length(btrim(nama)) BETWEEN 1 AND 255)
);
CREATE INDEX direktorat_kedeputian_idx ON vms.direktorat(id_kedeputian);
INSERT INTO vms.direktorat(id_kedeputian,nama)
SELECT k.id, u.nama FROM (VALUES
(1, 'PP', 'Direktorat Perencanaan Makro'),
(2, 'PP', 'Direktorat Perencanaan Mikro'),
(3, 'PP', 'Direktorat Pertanahan'),
(4, 'PDP', 'Direktorat Pengendalian Penyelenggaraan Pemerintahan dan Perizinan Pembangunan'),
(5, 'PDP', 'Direktorat Pengawasan, Pemantauan, dan Evaluasi'),
(6, 'PDP', 'Direktorat Ketentraman dan Ketertiban Umum'),
(7, 'SBPM', 'Direktorat Pelayanan Dasar'),
(8, 'SBPM', 'Direktorat Pemberdayaan Masyarakat'),
(9, 'SBPM', 'Direktorat Kebudayaan, Pariwisata, dan Ekonomi Kreatif'),
(10, 'THD', 'Direktorat Pengembangan Ekosistem Digital'),
(11, 'THD', 'Direktorat Transformasi Hijau'),
(12, 'THD', 'Direktorat Data dan Kecerdasan Buatan'),
(13, 'LHSDA', 'Direktorat Lingkungan Hidup dan Penanggulangan Bencana'),
(14, 'LHSDA', 'Direktorat Pengembangan Pemanfaatan Kehutanan dan Sumber Daya Air'),
(15, 'LHSDA', 'Direktorat Ketahanan Pangan'),
(16, 'PI', 'Direktorat Investasi dan Kemudahan Berusaha'),
(17, 'PI', 'Direktorat Pendanaan'),
(18, 'PI', 'Direktorat Pembiayaan'),
(19, 'SP', 'Direktorat Sarana Prasarana Dasar'),
(20, 'SP', 'Direktorat Sarana Prasarana Sosial'),
(21, 'SP', 'Direktorat Pengelolaan Gedung, Kawasan, dan Perkotaan')
) AS u(ord,kode,nama) JOIN vms.kedeputian k ON k.kode = u.kode ORDER BY u.ord;
GRANT SELECT ON vms.direktorat TO vms_api;

-- +migrate Down
REVOKE SELECT ON vms.direktorat FROM vms_api;
DROP TABLE vms.direktorat;
