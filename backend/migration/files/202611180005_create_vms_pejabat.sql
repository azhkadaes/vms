-- +migrate Up
CREATE TABLE vms.pejabat (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    id_jabatan INTEGER NOT NULL REFERENCES vms.jabatan(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    nama VARCHAR(255) NOT NULL CHECK (char_length(btrim(nama)) BETWEEN 1 AND 255),
    status_aktif BOOLEAN NOT NULL
);
CREATE INDEX pejabat_jabatan_idx ON vms.pejabat(id_jabatan);
GRANT SELECT ON vms.pejabat TO vms_api;

-- +migrate Down
REVOKE SELECT ON vms.pejabat FROM vms_api;
DROP TABLE vms.pejabat;
