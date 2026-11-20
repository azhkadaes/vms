-- +migrate Up
-- status_aktif, must_change_password and version are contract-required (employee ops 26-29, op 36).
-- Email is stored lower-case and trimmed; the 254-octet bound is the address limit the API enforces.
CREATE TABLE vms.pegawai (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    nama VARCHAR(255) NOT NULL CHECK (char_length(btrim(nama)) BETWEEN 2 AND 255),
    jabatan VARCHAR(255),
    kedeputian VARCHAR(255),
    direktorat VARCHAR(255),
    email VARCHAR(255) NOT NULL UNIQUE CHECK (email = lower(btrim(email)) AND octet_length(email) <= 254),
    nomor_hp VARCHAR(255),
    nip VARCHAR(255),
    status_aktif BOOLEAN NOT NULL,
    must_change_password BOOLEAN NOT NULL,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0)
);
-- Employee administration creates and edits employees; there is no delete.
GRANT SELECT, INSERT, UPDATE ON vms.pegawai TO vms_api;
GRANT USAGE ON SEQUENCE vms.pegawai_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE vms.pegawai_id_seq FROM vms_api;
REVOKE SELECT, INSERT, UPDATE ON vms.pegawai FROM vms_api;
DROP TABLE vms.pegawai;
