-- +migrate Up
CREATE TABLE vms.qr_kunjungan (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    id_kunjungan INTEGER NOT NULL REFERENCES vms.kunjungan(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    token_hash VARCHAR(255) NOT NULL UNIQUE CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    tanggal_dibuat TIMESTAMPTZ NOT NULL,
    tanggal_expire TIMESTAMPTZ NOT NULL CHECK (tanggal_expire > tanggal_dibuat),
    dicabut_pada TIMESTAMPTZ
);
CREATE INDEX qr_kunjungan_visit_idx ON vms.qr_kunjungan(id_kunjungan);
CREATE UNIQUE INDEX qr_kunjungan_active_visit_key ON vms.qr_kunjungan(id_kunjungan) WHERE dicabut_pada IS NULL;
GRANT SELECT, INSERT, UPDATE (dicabut_pada) ON vms.qr_kunjungan TO vms_api;
GRANT USAGE ON SEQUENCE vms.qr_kunjungan_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE vms.qr_kunjungan_id_seq FROM vms_api;
REVOKE SELECT, INSERT, UPDATE (dicabut_pada) ON vms.qr_kunjungan FROM vms_api;
DROP TABLE vms.qr_kunjungan;
