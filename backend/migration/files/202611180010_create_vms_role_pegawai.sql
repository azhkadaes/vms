-- +migrate Up
-- id_karyawan follows the diagram (formerly id_pegawai). direktorat is metadata only and never grants scope.
CREATE TABLE vms.role_pegawai (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    id_karyawan INTEGER NOT NULL REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    id_role INTEGER NOT NULL REFERENCES vms.role(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    direktorat VARCHAR(255),
    UNIQUE (id_karyawan, id_role)
);
CREATE INDEX role_pegawai_role_idx ON vms.role_pegawai(id_role);
GRANT SELECT, INSERT, DELETE ON vms.role_pegawai TO vms_api;
GRANT USAGE ON SEQUENCE vms.role_pegawai_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE vms.role_pegawai_id_seq FROM vms_api;
REVOKE SELECT, INSERT, DELETE ON vms.role_pegawai FROM vms_api;
DROP TABLE vms.role_pegawai;
