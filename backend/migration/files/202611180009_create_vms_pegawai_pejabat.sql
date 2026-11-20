-- +migrate Up
-- One current official per employee (Org). Conflicting active assignments fail the
-- transaction; an assignment is never selected or removed arbitrarily.
CREATE TABLE vms.pegawai_pejabat (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    id_pegawai INTEGER NOT NULL REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    id_pejabat INTEGER NOT NULL REFERENCES vms.pejabat(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    status_aktif BOOLEAN NOT NULL,
    UNIQUE (id_pegawai, id_pejabat)
);
CREATE UNIQUE INDEX pegawai_pejabat_one_active_idx ON vms.pegawai_pejabat(id_pegawai) WHERE status_aktif;
CREATE INDEX pegawai_pejabat_pejabat_idx ON vms.pegawai_pejabat(id_pejabat);
GRANT SELECT, INSERT, UPDATE ON vms.pegawai_pejabat TO vms_api;
GRANT USAGE ON SEQUENCE vms.pegawai_pejabat_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE vms.pegawai_pejabat_id_seq FROM vms_api;
REVOKE SELECT, INSERT, UPDATE ON vms.pegawai_pejabat FROM vms_api;
DROP TABLE vms.pegawai_pejabat;
