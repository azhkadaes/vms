-- +migrate Up
-- actor_type and the staff-only id_pegawai follow D19; entity_type/entity_id follow D20.
-- Visit history and derived fields (createdBy, rescheduled, privacy version) read these rows.
CREATE TABLE vms.audit_log (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0),
    actor_type VARCHAR(16) NOT NULL CHECK (actor_type IN ('staff','visitor','system')),
    id_pegawai INTEGER REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    entity_type VARCHAR(32) NOT NULL CHECK (entity_type IN ('kunjungan','pegawai','role')),
    entity_id INTEGER NOT NULL,
    jenis_kegiatan VARCHAR(255) NOT NULL,
    tanggal TIMESTAMPTZ NOT NULL,
    deskripsi VARCHAR(255) NOT NULL,
    old_values JSON,
    new_values JSON,
    ip_address VARCHAR(255),
    CHECK ((actor_type = 'staff' AND id_pegawai IS NOT NULL) OR (actor_type <> 'staff' AND id_pegawai IS NULL))
);
CREATE INDEX audit_log_entity_idx ON vms.audit_log(entity_type, entity_id, tanggal);
CREATE INDEX audit_log_pegawai_idx ON vms.audit_log(id_pegawai) WHERE id_pegawai IS NOT NULL;
-- Append-only for the runtime: no UPDATE or DELETE.
GRANT SELECT, INSERT ON vms.audit_log TO vms_api;
GRANT USAGE ON SEQUENCE vms.audit_log_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE vms.audit_log_id_seq FROM vms_api;
REVOKE SELECT, INSERT ON vms.audit_log FROM vms_api;
DROP TABLE vms.audit_log;
