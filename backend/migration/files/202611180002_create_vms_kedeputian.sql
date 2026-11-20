-- +migrate Up
-- Hierarchy verified against https://ikn.go.id/id/struktur-organisasi and
-- Perka OIKN 1/2022 articles 33, 43, 53, 63, 73, 83, 93.
CREATE TABLE vms.kedeputian (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    kode VARCHAR(16) NOT NULL UNIQUE,
    nama VARCHAR(255) NOT NULL UNIQUE CHECK (char_length(btrim(nama)) BETWEEN 1 AND 255)
);
INSERT INTO vms.kedeputian(kode,nama)
SELECT u.kode, u.nama FROM (VALUES
(1, 'PP', 'Kedeputian Bidang Perencanaan dan Pertanahan'),
(2, 'PDP', 'Kedeputian Bidang Pengendalian Pembangunan'),
(3, 'SBPM', 'Kedeputian Bidang Sosial, Budaya, dan Pemberdayaan Masyarakat'),
(4, 'THD', 'Kedeputian Bidang Transformasi Hijau dan Digital'),
(5, 'LHSDA', 'Kedeputian Bidang Lingkungan Hidup dan Sumber Daya Alam'),
(6, 'PI', 'Kedeputian Bidang Pendanaan dan Investasi'),
(7, 'SP', 'Kedeputian Bidang Sarana dan Prasarana')
) AS u(ord,kode,nama) ORDER BY u.ord;
GRANT SELECT ON vms.kedeputian TO vms_api;

-- +migrate Down
REVOKE SELECT ON vms.kedeputian FROM vms_api;
DROP TABLE vms.kedeputian;
