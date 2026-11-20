-- +migrate Up
CREATE TYPE vms.tipe_kunjungan AS ENUM ('umum', 'vip', 'terbatas');
CREATE TYPE vms.status_penerimaan AS ENUM ('pending', 'approved', 'rejected', 'cancelled', 'checked_in', 'checked_out');
CREATE TABLE vms.kunjungan (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    id_detail_pengunjung INTEGER NOT NULL UNIQUE REFERENCES vms.detail_pengunjung(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    id_detail_kunjungan INTEGER NOT NULL UNIQUE REFERENCES vms.detail_kunjungan(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    kode_kunjungan CHAR(8) NOT NULL UNIQUE CHECK (kode_kunjungan ~ '^[0-9]{8}$'),
    tipe_kunjungan vms.tipe_kunjungan NOT NULL,
    status_penerimaan vms.status_penerimaan NOT NULL,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    waktu_cek_in TIMESTAMPTZ,
    waktu_cek_out TIMESTAMPTZ,
    tanggal_dibuat TIMESTAMPTZ NOT NULL,
    alasan_penolakan VARCHAR(1000),
    catatan_internal_penolakan VARCHAR(2000),
    alasan_pembatalan VARCHAR(1000),
    catatan_internal_pembatalan VARCHAR(2000),
    CHECK (
        (status_penerimaan = 'checked_in' AND waktu_cek_in IS NOT NULL AND waktu_cek_out IS NULL)
        OR (status_penerimaan = 'checked_out' AND waktu_cek_in IS NOT NULL AND waktu_cek_out IS NOT NULL AND waktu_cek_out >= waktu_cek_in)
        OR (status_penerimaan NOT IN ('checked_in', 'checked_out') AND waktu_cek_in IS NULL AND waktu_cek_out IS NULL)
    )
);
GRANT SELECT, INSERT, UPDATE ON vms.kunjungan TO vms_api;
GRANT USAGE ON SEQUENCE vms.kunjungan_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE vms.kunjungan_id_seq FROM vms_api;
REVOKE SELECT, INSERT, UPDATE ON vms.kunjungan FROM vms_api;
DROP TABLE vms.kunjungan;
DROP TYPE vms.status_penerimaan;
DROP TYPE vms.tipe_kunjungan;
