-- +migrate Up
-- Carries the diagram's Pejabat.nama_jabatan and Pejabat.pejabat_vip (Org agreement).
-- A position belongs to at most one directorate or deputy group.
CREATE TABLE vms.jabatan (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    nama VARCHAR(255) NOT NULL CHECK (char_length(btrim(nama)) BETWEEN 1 AND 255),
    pejabat_vip BOOLEAN NOT NULL,
    id_direktorat INTEGER REFERENCES vms.direktorat(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    id_kedeputian INTEGER REFERENCES vms.kedeputian(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT jabatan_one_organization CHECK (id_direktorat IS NULL OR id_kedeputian IS NULL)
);
CREATE INDEX jabatan_direktorat_idx ON vms.jabatan(id_direktorat) WHERE id_direktorat IS NOT NULL;
CREATE INDEX jabatan_kedeputian_idx ON vms.jabatan(id_kedeputian) WHERE id_kedeputian IS NOT NULL;
GRANT SELECT ON vms.jabatan TO vms_api;

-- +migrate Down
REVOKE SELECT ON vms.jabatan FROM vms_api;
DROP TABLE vms.jabatan;
