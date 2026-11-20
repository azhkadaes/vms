-- +migrate Up
CREATE TYPE vms.identity_type AS ENUM ('KTP', 'Paspor', 'SIM');
-- +migrate StatementBegin
CREATE FUNCTION vms.valid_visit_roster(roster TEXT[]) RETURNS BOOLEAN
LANGUAGE SQL IMMUTABLE STRICT AS $$
    SELECT array_ndims(roster) = 1 AND cardinality(roster) BETWEEN 1 AND 20
        AND NOT EXISTS (SELECT FROM unnest(roster) AS member(name)
            WHERE name IS NULL OR name <> btrim(name) OR char_length(name) NOT BETWEEN 2 AND 120);
$$;
-- +migrate StatementEnd
CREATE TABLE vms.detail_pengunjung (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    nama VARCHAR(255) CHECK (nama = btrim(nama) AND char_length(nama) BETWEEN 2 AND 120),
    asal_instansi VARCHAR(255) CHECK (asal_instansi = btrim(asal_instansi) AND char_length(asal_instansi) >= 2),
    nomor_telpon VARCHAR(255) CHECK (nomor_telpon ~ '^[0-9]{8,20}$'),
    email VARCHAR(255) CHECK (email = lower(email)),
    identity_type vms.identity_type,
    nama_rombongan TEXT[] CHECK (vms.valid_visit_roster(nama_rombongan)),
    jumlah_pengunjung INTEGER,
    id_formdocuments INTEGER UNIQUE REFERENCES vms.form_documents(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (nama_rombongan IS NULL OR jumlah_pengunjung IS NOT DISTINCT FROM cardinality(nama_rombongan))
);
GRANT SELECT, INSERT, UPDATE ON vms.detail_pengunjung TO vms_api;
GRANT USAGE ON SEQUENCE vms.detail_pengunjung_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE vms.detail_pengunjung_id_seq FROM vms_api;
REVOKE SELECT, INSERT, UPDATE ON vms.detail_pengunjung FROM vms_api;
DROP TABLE vms.detail_pengunjung;
DROP FUNCTION vms.valid_visit_roster(TEXT[]);
DROP TYPE vms.identity_type;
