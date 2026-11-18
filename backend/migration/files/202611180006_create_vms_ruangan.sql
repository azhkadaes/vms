-- +migrate Up
-- The former extra ruangan column is gone: it is not in the diagram and has no agreement.
CREATE TABLE vms.ruangan (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    nama VARCHAR(255) NOT NULL CHECK (char_length(btrim(nama)) BETWEEN 1 AND 255),
    lokasi VARCHAR(255),
    status_aktif BOOLEAN NOT NULL
);
GRANT SELECT ON vms.ruangan TO vms_api;

-- +migrate Down
REVOKE SELECT ON vms.ruangan FROM vms_api;
DROP TABLE vms.ruangan;
