-- +migrate Up
CREATE TABLE vms.detail_kunjungan (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    id_pejabat INTEGER NOT NULL REFERENCES vms.pejabat(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    id_ruangan INTEGER REFERENCES vms.ruangan(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    id_formdocuments INTEGER UNIQUE REFERENCES vms.form_documents(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    tanggal DATE NOT NULL,
    waktu_mulai TIME NOT NULL,
    visit_duration INTEGER NOT NULL CHECK (visit_duration BETWEEN 15 AND 240),
    keperluan VARCHAR(1000),
    catatan VARCHAR(2000),
    catatan_internal VARCHAR(2000),
    no_nd VARCHAR(100)
);
CREATE INDEX detail_kunjungan_officer_idx ON vms.detail_kunjungan(id_pejabat);
CREATE INDEX detail_kunjungan_room_idx ON vms.detail_kunjungan(id_ruangan);
GRANT SELECT, INSERT, UPDATE ON vms.detail_kunjungan TO vms_api;
GRANT USAGE ON SEQUENCE vms.detail_kunjungan_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE vms.detail_kunjungan_id_seq FROM vms_api;
REVOKE SELECT, INSERT, UPDATE ON vms.detail_kunjungan FROM vms_api;
DROP TABLE vms.detail_kunjungan;
