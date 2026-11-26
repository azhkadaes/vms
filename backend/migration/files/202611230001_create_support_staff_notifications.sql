-- +migrate Up
CREATE TABLE support.staff_notifications (
    event_id INTEGER NOT NULL REFERENCES vms.audit_log(id) ON DELETE CASCADE,
    employee_id INTEGER NOT NULL REFERENCES vms.pegawai(id) ON DELETE CASCADE,
    recipient_role VARCHAR(32) NOT NULL CHECK (recipient_role IN ('PA','Protocol Officer')),
    read_at TIMESTAMPTZ,
    PRIMARY KEY (employee_id, recipient_role, event_id)
);
CREATE INDEX staff_notifications_unread_idx ON support.staff_notifications(employee_id, recipient_role, event_id)
    WHERE read_at IS NULL;
GRANT SELECT, INSERT ON support.staff_notifications TO vms_api;
GRANT UPDATE (read_at) ON support.staff_notifications TO vms_api;

-- +migrate StatementBegin
CREATE FUNCTION support.capture_staff_notification() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO support.staff_notifications (event_id, employee_id, recipient_role)
    SELECT NEW.id, e.id, r.nama
    FROM vms.kunjungan k
    JOIN vms.detail_kunjungan d ON d.id = k.id_detail_kunjungan
    JOIN vms.role r ON r.nama = CASE WHEN k.tipe_kunjungan = 'umum' THEN 'PA' ELSE 'Protocol Officer' END
    JOIN vms.role_pegawai m ON m.id_role = r.id
    JOIN vms.pegawai e ON e.id = m.id_karyawan AND e.status_aktif
    WHERE k.id = NEW.entity_id
      AND (r.nama = 'Protocol Officer' OR EXISTS (
        SELECT 1 FROM vms.pegawai_pejabat p
        WHERE p.id_pegawai = e.id AND p.id_pejabat = d.id_pejabat AND p.status_aktif
      ));
    RETURN NEW;
END;
$$;
-- +migrate StatementEnd
CREATE TRIGGER staff_notification_after_visit_event
AFTER INSERT ON vms.audit_log
FOR EACH ROW WHEN (
    NEW.entity_type = 'kunjungan' AND
    NEW.jenis_kegiatan IN ('created','approved','updated','rescheduled','cancelled','checked_out')
) EXECUTE FUNCTION support.capture_staff_notification();

-- +migrate Down
DROP TRIGGER staff_notification_after_visit_event ON vms.audit_log;
DROP FUNCTION support.capture_staff_notification();
REVOKE UPDATE (read_at) ON support.staff_notifications FROM vms_api;
REVOKE SELECT, INSERT ON support.staff_notifications FROM vms_api;
DROP TABLE support.staff_notifications;
