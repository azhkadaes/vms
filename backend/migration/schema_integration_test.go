//go:build integration

package vmsmigration_test

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"

	vmsmigration "github.com/azhkadaes/vms/backend/migration"
)

// tableShape is the applied catalog expected for one table of the physical schema
// (docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md). Check, foreign-key and index
// definitions are PostgreSQL 18's normalized renderings.
type tableShape struct {
	name        string
	columns     []string
	checks      []string
	foreignKeys []string
	indexes     []string
	grants      string // vms_api table privileges
	columnGrant string // vms_api column privileges
	sequence    string // vms_api privileges on the identity sequence
}

const textLength1to255 = "CHECK (((char_length(btrim((nama)::text)) >= 1) AND (char_length(btrim((nama)::text)) <= 255)))"

var physicalSchema = []tableShape{
	{
		name:    "vms.kedeputian",
		columns: []string{"id integer NOT NULL IDENTITY ALWAYS", "kode varchar(16) NOT NULL", "nama varchar(255) NOT NULL"},
		checks:  []string{"CHECK ((id > 0))", textLength1to255},
		indexes: []string{
			"CREATE UNIQUE INDEX kedeputian_kode_key ON vms.kedeputian USING btree (kode)",
			"CREATE UNIQUE INDEX kedeputian_nama_key ON vms.kedeputian USING btree (nama)",
			"CREATE UNIQUE INDEX kedeputian_pkey ON vms.kedeputian USING btree (id)",
		},
		grants: "SELECT",
	},
	{
		name:        "vms.direktorat",
		columns:     []string{"id integer NOT NULL IDENTITY ALWAYS", "id_kedeputian integer NOT NULL", "nama varchar(255) NOT NULL"},
		checks:      []string{"CHECK ((id > 0))", textLength1to255},
		foreignKeys: []string{"FOREIGN KEY (id_kedeputian) REFERENCES vms.kedeputian(id) ON UPDATE RESTRICT ON DELETE RESTRICT"},
		indexes: []string{
			"CREATE INDEX direktorat_kedeputian_idx ON vms.direktorat USING btree (id_kedeputian)",
			"CREATE UNIQUE INDEX direktorat_nama_key ON vms.direktorat USING btree (nama)",
			"CREATE UNIQUE INDEX direktorat_pkey ON vms.direktorat USING btree (id)",
		},
		grants: "SELECT",
	},
	{
		name:    "vms.jabatan",
		columns: []string{"id integer NOT NULL IDENTITY ALWAYS", "nama varchar(255) NOT NULL", "pejabat_vip boolean NOT NULL", "id_direktorat integer", "id_kedeputian integer"},
		checks:  []string{"CHECK ((id > 0))", "CHECK (((id_direktorat IS NULL) OR (id_kedeputian IS NULL)))", textLength1to255},
		foreignKeys: []string{
			"FOREIGN KEY (id_direktorat) REFERENCES vms.direktorat(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (id_kedeputian) REFERENCES vms.kedeputian(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		indexes: []string{
			"CREATE INDEX jabatan_direktorat_idx ON vms.jabatan USING btree (id_direktorat) WHERE (id_direktorat IS NOT NULL)",
			"CREATE INDEX jabatan_kedeputian_idx ON vms.jabatan USING btree (id_kedeputian) WHERE (id_kedeputian IS NOT NULL)",
			"CREATE UNIQUE INDEX jabatan_pkey ON vms.jabatan USING btree (id)",
		},
		grants: "SELECT",
	},
	{
		name:        "vms.pejabat",
		columns:     []string{"id integer NOT NULL IDENTITY ALWAYS", "id_jabatan integer NOT NULL", "nama varchar(255) NOT NULL", "status_aktif boolean NOT NULL"},
		checks:      []string{"CHECK ((id > 0))", textLength1to255},
		foreignKeys: []string{"FOREIGN KEY (id_jabatan) REFERENCES vms.jabatan(id) ON UPDATE RESTRICT ON DELETE RESTRICT"},
		indexes: []string{
			"CREATE INDEX pejabat_jabatan_idx ON vms.pejabat USING btree (id_jabatan)",
			"CREATE UNIQUE INDEX pejabat_pkey ON vms.pejabat USING btree (id)",
		},
		grants: "SELECT",
	},
	{
		name:    "vms.ruangan",
		columns: []string{"id integer NOT NULL IDENTITY ALWAYS", "nama varchar(255) NOT NULL", "lokasi varchar(255)", "status_aktif boolean NOT NULL"},
		checks:  []string{"CHECK ((id > 0))", textLength1to255},
		indexes: []string{"CREATE UNIQUE INDEX ruangan_pkey ON vms.ruangan USING btree (id)"},
		grants:  "SELECT",
	},
	{
		name: "vms.pegawai",
		columns: []string{
			"id integer NOT NULL IDENTITY ALWAYS", "nama varchar(255) NOT NULL", "jabatan varchar(255)", "kedeputian varchar(255)",
			"direktorat varchar(255)", "email varchar(255) NOT NULL", "nomor_hp ARRAY NOT NULL DEFAULT ARRAY[]::text[]", "nip varchar(255)",
			"status_aktif boolean NOT NULL", "must_change_password boolean NOT NULL", "version integer NOT NULL DEFAULT 1",
		},
		checks: []string{
			"CHECK ((((email)::text = lower(btrim((email)::text))) AND (octet_length((email)::text) <= 254)))",
			"CHECK ((id > 0))",
			"CHECK ((version > 0))",
			"CHECK (((char_length(btrim((nama)::text)) >= 2) AND (char_length(btrim((nama)::text)) <= 255)))",
		},
		indexes: []string{
			"CREATE UNIQUE INDEX pegawai_email_key ON vms.pegawai USING btree (email)",
			"CREATE UNIQUE INDEX pegawai_pkey ON vms.pegawai USING btree (id)",
		},
		grants:   "INSERT,SELECT,UPDATE",
		sequence: "USAGE",
	},
	{
		name: "vms.role",
		columns: []string{
			"id integer NOT NULL IDENTITY ALWAYS", "nama varchar(255) NOT NULL", "version integer NOT NULL DEFAULT 1",
			"created_at timestamptz NOT NULL DEFAULT transaction_timestamp()", "created_by varchar(255) NOT NULL",
			"updated_at timestamptz NOT NULL DEFAULT transaction_timestamp()", "updated_by varchar(255) NOT NULL",
		},
		checks: []string{
			"CHECK ((((created_by)::text = 'system'::text) OR ((created_by)::text ~ '^[1-9][0-9]{0,9}$'::text)))",
			"CHECK (((nama)::text = ANY ((ARRAY['PA'::character varying, 'Protocol Officer'::character varying, 'Receptionist'::character varying, 'Super Admin'::character varying])::text[])))",
			"CHECK ((((updated_by)::text = 'system'::text) OR ((updated_by)::text ~ '^[1-9][0-9]{0,9}$'::text)))",
			"CHECK ((id > 0))",
			"CHECK ((version > 0))",
		},
		indexes: []string{
			"CREATE UNIQUE INDEX role_nama_key ON vms.role USING btree (nama)",
			"CREATE UNIQUE INDEX role_pkey ON vms.role USING btree (id)",
		},
		grants:      "SELECT",
		columnGrant: "UPDATE(updated_at,updated_by,version)",
	},
	{
		name:    "vms.pegawai_pejabat",
		columns: []string{"id integer NOT NULL IDENTITY ALWAYS", "id_pegawai integer NOT NULL", "id_pejabat integer NOT NULL", "status_aktif boolean NOT NULL"},
		checks:  []string{"CHECK ((id > 0))"},
		foreignKeys: []string{
			"FOREIGN KEY (id_pegawai) REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (id_pejabat) REFERENCES vms.pejabat(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		indexes: []string{
			"CREATE INDEX pegawai_pejabat_pejabat_idx ON vms.pegawai_pejabat USING btree (id_pejabat)",
			"CREATE UNIQUE INDEX pegawai_pejabat_id_pegawai_id_pejabat_key ON vms.pegawai_pejabat USING btree (id_pegawai, id_pejabat)",
			"CREATE UNIQUE INDEX pegawai_pejabat_one_active_idx ON vms.pegawai_pejabat USING btree (id_pegawai) WHERE status_aktif",
			"CREATE UNIQUE INDEX pegawai_pejabat_pkey ON vms.pegawai_pejabat USING btree (id)",
		},
		grants:   "INSERT,SELECT,UPDATE",
		sequence: "USAGE",
	},
	{
		name:    "vms.role_pegawai",
		columns: []string{"id integer NOT NULL IDENTITY ALWAYS", "id_karyawan integer NOT NULL", "id_role integer NOT NULL", "direktorat varchar(255)"},
		checks:  []string{"CHECK ((id > 0))"},
		foreignKeys: []string{
			"FOREIGN KEY (id_karyawan) REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (id_role) REFERENCES vms.role(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		indexes: []string{
			"CREATE INDEX role_pegawai_role_idx ON vms.role_pegawai USING btree (id_role)",
			"CREATE UNIQUE INDEX role_pegawai_id_karyawan_id_role_key ON vms.role_pegawai USING btree (id_karyawan, id_role)",
			"CREATE UNIQUE INDEX role_pegawai_pkey ON vms.role_pegawai USING btree (id)",
		},
		grants:   "DELETE,INSERT,SELECT",
		sequence: "USAGE",
	},
	{
		name:    "vms.feature",
		columns: []string{"id integer NOT NULL IDENTITY ALWAYS", "nama varchar(255) NOT NULL"},
		checks:  []string{"CHECK ((id > 0))", textLength1to255},
		indexes: []string{"CREATE UNIQUE INDEX feature_pkey ON vms.feature USING btree (id)"},
		grants:  "SELECT",
	},
	{
		name:    "vms.action",
		columns: []string{"id integer NOT NULL IDENTITY ALWAYS", "nama varchar(255) NOT NULL"},
		checks:  []string{"CHECK ((id > 0))", textLength1to255},
		indexes: []string{"CREATE UNIQUE INDEX action_pkey ON vms.action USING btree (id)"},
		grants:  "SELECT",
	},
	{
		name:    "vms.permission",
		columns: []string{"id integer NOT NULL IDENTITY ALWAYS", "id_feature integer NOT NULL", "id_action integer NOT NULL", "permission_key varchar(255) NOT NULL", "nama varchar(255) NOT NULL"},
		checks: []string{
			"CHECK (((permission_key)::text = ANY ((ARRAY['bookings'::character varying, 'create'::character varying, 'approve'::character varying, 'reject'::character varying, 'reschedule'::character varying, 'cancel'::character varying, 'checkin'::character varying, 'checkout'::character varying, 'people'::character varying, 'roles'::character varying, 'integrations'::character varying, 'reports'::character varying, 'reportExport'::character varying, 'audit'::character varying])::text[])))",
			"CHECK ((id > 0))",
			textLength1to255,
		},
		foreignKeys: []string{
			"FOREIGN KEY (id_action) REFERENCES vms.action(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (id_feature) REFERENCES vms.feature(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		indexes: []string{
			"CREATE INDEX permission_action_idx ON vms.permission USING btree (id_action)",
			"CREATE UNIQUE INDEX permission_id_feature_id_action_key ON vms.permission USING btree (id_feature, id_action)",
			"CREATE UNIQUE INDEX permission_permission_key_key ON vms.permission USING btree (permission_key)",
			"CREATE UNIQUE INDEX permission_pkey ON vms.permission USING btree (id)",
		},
		grants: "SELECT",
	},
	{
		name: "vms.policy",
		columns: []string{
			"id integer NOT NULL IDENTITY ALWAYS", "nama varchar(255) NOT NULL",
			"created_at timestamptz NOT NULL DEFAULT transaction_timestamp()", "updated_at timestamptz NOT NULL DEFAULT transaction_timestamp()",
			"created_by varchar(255) NOT NULL", "updated_by varchar(255) NOT NULL", "status_aktif boolean NOT NULL",
		},
		checks: []string{
			"CHECK ((((created_by)::text = 'system'::text) OR ((created_by)::text ~ '^[1-9][0-9]{0,9}$'::text)))",
			"CHECK ((((updated_by)::text = 'system'::text) OR ((updated_by)::text ~ '^[1-9][0-9]{0,9}$'::text)))",
			"CHECK ((id > 0))",
			textLength1to255,
		},
		indexes: []string{"CREATE UNIQUE INDEX policy_pkey ON vms.policy USING btree (id)"},
		grants:  "SELECT,UPDATE",
	},
	{
		name: "vms.role_policy",
		columns: []string{
			"id integer NOT NULL IDENTITY ALWAYS", "id_policy integer NOT NULL", "id_role integer NOT NULL",
			"created_at timestamptz NOT NULL DEFAULT transaction_timestamp()", "updated_at timestamptz NOT NULL DEFAULT transaction_timestamp()",
			"created_by varchar(255) NOT NULL", "updated_by varchar(255) NOT NULL", "status_aktif boolean NOT NULL",
		},
		checks: []string{
			"CHECK ((((created_by)::text = 'system'::text) OR ((created_by)::text ~ '^[1-9][0-9]{0,9}$'::text)))",
			"CHECK ((((updated_by)::text = 'system'::text) OR ((updated_by)::text ~ '^[1-9][0-9]{0,9}$'::text)))",
			"CHECK ((id > 0))",
		},
		foreignKeys: []string{
			"FOREIGN KEY (id_policy) REFERENCES vms.policy(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (id_role) REFERENCES vms.role(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		indexes: []string{
			"CREATE INDEX role_policy_policy_idx ON vms.role_policy USING btree (id_policy)",
			"CREATE UNIQUE INDEX role_policy_active_policy_idx ON vms.role_policy USING btree (id_policy) WHERE status_aktif",
			"CREATE UNIQUE INDEX role_policy_active_role_idx ON vms.role_policy USING btree (id_role) WHERE status_aktif",
			"CREATE UNIQUE INDEX role_policy_id_role_id_policy_key ON vms.role_policy USING btree (id_role, id_policy)",
			"CREATE UNIQUE INDEX role_policy_pkey ON vms.role_policy USING btree (id)",
		},
		grants: "SELECT",
	},
	{
		name:    "vms.policy_permission",
		columns: []string{"id integer NOT NULL IDENTITY ALWAYS", "id_permission integer NOT NULL", "id_policy integer NOT NULL"},
		checks:  []string{"CHECK ((id > 0))"},
		foreignKeys: []string{
			"FOREIGN KEY (id_permission) REFERENCES vms.permission(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (id_policy) REFERENCES vms.policy(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		indexes: []string{
			"CREATE INDEX policy_permission_permission_idx ON vms.policy_permission USING btree (id_permission)",
			"CREATE UNIQUE INDEX policy_permission_id_policy_id_permission_key ON vms.policy_permission USING btree (id_policy, id_permission)",
			"CREATE UNIQUE INDEX policy_permission_pkey ON vms.policy_permission USING btree (id)",
		},
		grants:   "DELETE,INSERT,SELECT",
		sequence: "USAGE",
	},
	{
		name: "vms.audit_log",
		columns: []string{
			"id integer NOT NULL IDENTITY ALWAYS", "actor_type varchar(16) NOT NULL", "id_pegawai integer",
			"entity_type varchar(32) NOT NULL", "entity_id integer NOT NULL", "jenis_kegiatan varchar(255) NOT NULL",
			"tanggal timestamptz NOT NULL", "deskripsi varchar(255) NOT NULL", "old_values json", "new_values json", "ip_address varchar(255)",
		},
		checks: []string{
			"CHECK (((((actor_type)::text = 'staff'::text) AND (id_pegawai IS NOT NULL)) OR (((actor_type)::text <> 'staff'::text) AND (id_pegawai IS NULL))))",
			"CHECK (((actor_type)::text = ANY ((ARRAY['staff'::character varying, 'visitor'::character varying, 'system'::character varying])::text[])))",
			"CHECK (((entity_type)::text = ANY ((ARRAY['kunjungan'::character varying, 'pegawai'::character varying, 'role'::character varying])::text[])))",
			"CHECK ((id > 0))",
		},
		foreignKeys: []string{"FOREIGN KEY (id_pegawai) REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT"},
		indexes: []string{
			"CREATE INDEX audit_log_entity_idx ON vms.audit_log USING btree (entity_type, entity_id, tanggal)",
			"CREATE INDEX audit_log_pegawai_idx ON vms.audit_log USING btree (id_pegawai) WHERE (id_pegawai IS NOT NULL)",
			"CREATE UNIQUE INDEX audit_log_pkey ON vms.audit_log USING btree (id)",
		},
		grants:   "INSERT,SELECT",
		sequence: "USAGE",
	},
	{
		name:    "support.rate_limit_buckets",
		columns: []string{"bucket_key char(64) NOT NULL", "last_seen_at timestamptz NOT NULL DEFAULT clock_timestamp()"},
		checks:  []string{"CHECK ((bucket_key ~ '^[0-9a-f]{64}$'::text))"},
		indexes: []string{"CREATE UNIQUE INDEX rate_limit_buckets_pkey ON support.rate_limit_buckets USING btree (bucket_key)"},
		grants:  "DELETE,INSERT,SELECT,UPDATE",
	},
	{
		name:        "support.rate_limit_events",
		columns:     []string{"id bigint NOT NULL IDENTITY ALWAYS", "bucket_key char(64) NOT NULL", "occurred_at timestamptz NOT NULL"},
		foreignKeys: []string{"FOREIGN KEY (bucket_key) REFERENCES support.rate_limit_buckets(bucket_key) ON DELETE CASCADE"},
		indexes: []string{
			"CREATE INDEX rate_limit_events_bucket_time_idx ON support.rate_limit_events USING btree (bucket_key, occurred_at)",
			"CREATE UNIQUE INDEX rate_limit_events_pkey ON support.rate_limit_events USING btree (id)",
		},
		grants:   "DELETE,INSERT,SELECT,UPDATE",
		sequence: "USAGE",
	},
	{
		name:        "support.employee_credentials",
		columns:     []string{"employee_id integer NOT NULL", "encoded_hash text NOT NULL", "updated_at timestamptz NOT NULL DEFAULT transaction_timestamp()"},
		checks:      []string{"CHECK ((encoded_hash ~~ 'bcrypt:%'::text))"},
		foreignKeys: []string{"FOREIGN KEY (employee_id) REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT"},
		indexes:     []string{"CREATE UNIQUE INDEX employee_credentials_pkey ON support.employee_credentials USING btree (employee_id)"},
		grants:      "INSERT,SELECT,UPDATE",
	},
	{
		name: "support.staff_sessions",
		columns: []string{
			"id uuid NOT NULL", "employee_id integer NOT NULL", "active_role_id integer", "credential_generation bigint NOT NULL",
			"role_revision bigint NOT NULL", "refresh_hash char(64) NOT NULL", "created_at timestamptz NOT NULL",
			"last_activity_at timestamptz NOT NULL", "idle_expires_at timestamptz NOT NULL", "absolute_expires_at timestamptz NOT NULL",
			"revoked_at timestamptz",
		},
		checks: []string{
			"CHECK (((absolute_expires_at > created_at) AND (absolute_expires_at <= (created_at + '24:00:00'::interval))))",
			"CHECK (((credential_generation >= 1) AND (credential_generation <= '9007199254740991'::bigint)))",
			"CHECK (((idle_expires_at > created_at) AND (idle_expires_at <= absolute_expires_at)))",
			"CHECK (((last_activity_at >= created_at) AND (last_activity_at < idle_expires_at)))",
			"CHECK (((revoked_at IS NULL) OR (revoked_at >= created_at)))",
			"CHECK (((role_revision >= 1) AND (role_revision <= '9007199254740991'::bigint)))",
			"CHECK ((idle_expires_at <= (last_activity_at + '08:00:00'::interval)))",
			"CHECK ((refresh_hash ~ '^[0-9a-f]{64}$'::text))",
		},
		foreignKeys: []string{
			"FOREIGN KEY (active_role_id) REFERENCES vms.role(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (employee_id) REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		indexes: []string{
			"CREATE INDEX staff_sessions_active_role_idx ON support.staff_sessions USING btree (active_role_id) WHERE (active_role_id IS NOT NULL)",
			"CREATE INDEX staff_sessions_employee_idx ON support.staff_sessions USING btree (employee_id)",
			"CREATE UNIQUE INDEX staff_sessions_pkey ON support.staff_sessions USING btree (id)",
			"CREATE UNIQUE INDEX staff_sessions_refresh_hash_key ON support.staff_sessions USING btree (refresh_hash)",
		},
		grants: "INSERT,SELECT,UPDATE",
	},
	{
		name:        "support.refresh_lineage",
		columns:     []string{"token_hash char(64) NOT NULL", "session_id uuid NOT NULL", "consumed_at timestamptz NOT NULL"},
		checks:      []string{"CHECK ((token_hash ~ '^[0-9a-f]{64}$'::text))"},
		foreignKeys: []string{"FOREIGN KEY (session_id) REFERENCES support.staff_sessions(id) ON DELETE RESTRICT"},
		indexes: []string{
			"CREATE INDEX refresh_lineage_session_idx ON support.refresh_lineage USING btree (session_id)",
			"CREATE UNIQUE INDEX refresh_lineage_pkey ON support.refresh_lineage USING btree (token_hash)",
		},
		grants: "INSERT,SELECT",
	},
	{
		name: "support.refresh_recovery",
		columns: []string{
			"session_id uuid NOT NULL", "request_id uuid NOT NULL", "predecessor_hash char(64) NOT NULL",
			"successor_hash char(64) NOT NULL", "successor_cipher bytea NOT NULL", "expires_at timestamptz NOT NULL",
		},
		checks: []string{
			"CHECK ((predecessor_hash ~ '^[0-9a-f]{64}$'::text))",
			"CHECK ((successor_hash ~ '^[0-9a-f]{64}$'::text))",
		},
		foreignKeys: []string{"FOREIGN KEY (session_id) REFERENCES support.staff_sessions(id) ON DELETE RESTRICT"},
		indexes:     []string{"CREATE UNIQUE INDEX refresh_recovery_pkey ON support.refresh_recovery USING btree (session_id, request_id)"},
		grants:      "DELETE,INSERT,SELECT",
	},
	{
		name: "vms.form_documents",
		columns: []string{
			"id integer NOT NULL IDENTITY ALWAYS", "nama varchar(255) NOT NULL", "storage_path varchar(255) NOT NULL",
			"mime_type varchar(255) NOT NULL", "ukuran_file_bytes integer NOT NULL", "purpose varchar(16) NOT NULL",
		},
		checks: []string{
			"CHECK (((((purpose)::text = 'document'::text) AND ((mime_type)::text = 'application/pdf'::text)) OR (((purpose)::text = 'visitorPhoto'::text) AND ((mime_type)::text = ANY ((ARRAY['image/jpeg'::character varying, 'image/png'::character varying])::text[])))))",
			textLength1to255,
			"CHECK (((mime_type)::text = ANY ((ARRAY['application/pdf'::character varying, 'image/jpeg'::character varying, 'image/png'::character varying])::text[])))",
			"CHECK (((purpose)::text = ANY ((ARRAY['document'::character varying, 'visitorPhoto'::character varying])::text[])))",
			"CHECK (((storage_path)::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'::text))",
			"CHECK (((ukuran_file_bytes >= 1) AND (ukuran_file_bytes <= 5242880)))",
			"CHECK ((id > 0))",
		},
		indexes: []string{
			"CREATE UNIQUE INDEX form_documents_pkey ON vms.form_documents USING btree (id)",
			"CREATE UNIQUE INDEX form_documents_storage_path_key ON vms.form_documents USING btree (storage_path)",
		},
		grants:   "INSERT,SELECT",
		sequence: "USAGE",
	},
	{
		name: "support.form_document_uploads",
		columns: []string{
			"id_form_documents integer NOT NULL", "audience varchar(8) NOT NULL", "owner_digest char(64) NOT NULL", "reference uuid NOT NULL",
			"content_sha256 char(64) NOT NULL", "token_hash char(64) NOT NULL", "key_version smallint NOT NULL",
			"created_at timestamptz NOT NULL", "expires_at timestamptz NOT NULL", "linked_at timestamptz", "uploaded_by integer",
		},
		checks: []string{
			"CHECK ((((audience)::text = 'staff'::text) = (uploaded_by IS NOT NULL)))",
			"CHECK (((audience)::text = ANY ((ARRAY['public'::character varying, 'staff'::character varying])::text[])))",
			"CHECK (((linked_at IS NULL) OR ((linked_at >= created_at) AND (linked_at < expires_at))))",
			"CHECK ((content_sha256 ~ '^[0-9a-f]{64}$'::text))",
			"CHECK ((expires_at = (created_at + '01:00:00'::interval)))",
			"CHECK ((key_version > 0))",
			"CHECK ((owner_digest ~ '^[0-9a-f]{64}$'::text))",
			"CHECK ((token_hash ~ '^[0-9a-f]{64}$'::text))",
		},
		foreignKeys: []string{
			"FOREIGN KEY (id_form_documents) REFERENCES vms.form_documents(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (uploaded_by) REFERENCES vms.pegawai(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		indexes: []string{
			"CREATE INDEX form_document_uploads_uploaded_by_idx ON support.form_document_uploads USING btree (uploaded_by) WHERE (uploaded_by IS NOT NULL)",
			"CREATE UNIQUE INDEX form_document_uploads_audience_owner_digest_reference_key ON support.form_document_uploads USING btree (audience, owner_digest, reference)",
			"CREATE UNIQUE INDEX form_document_uploads_pkey ON support.form_document_uploads USING btree (id_form_documents)",
			"CREATE UNIQUE INDEX form_document_uploads_token_hash_key ON support.form_document_uploads USING btree (token_hash)",
		},
		grants:      "INSERT,SELECT",
		columnGrant: "UPDATE(linked_at)",
	},
	{
		name:     "vms.detail_kunjungan",
		grants:   "INSERT,SELECT,UPDATE",
		sequence: "USAGE",
		checks: []string{
			"CHECK (((visit_duration >= 15) AND (visit_duration <= 240)))",
			"CHECK ((id > 0))",
		},
		foreignKeys: []string{
			"FOREIGN KEY (id_formdocuments) REFERENCES vms.form_documents(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (id_pejabat) REFERENCES vms.pejabat(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (id_ruangan) REFERENCES vms.ruangan(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		columns: []string{
			"id integer NOT NULL IDENTITY ALWAYS",
			"id_pejabat integer NOT NULL",
			"id_ruangan integer",
			"id_formdocuments integer",
			"tanggal date NOT NULL",
			"waktu_mulai time without time zone NOT NULL",
			"visit_duration integer NOT NULL",
			"keperluan varchar(1000)",
			"catatan varchar(2000)",
			"catatan_internal varchar(2000)",
			"no_nd varchar(100)",
		},
		indexes: []string{
			"CREATE INDEX detail_kunjungan_officer_idx ON vms.detail_kunjungan USING btree (id_pejabat)",
			"CREATE INDEX detail_kunjungan_room_idx ON vms.detail_kunjungan USING btree (id_ruangan)",
			"CREATE UNIQUE INDEX detail_kunjungan_id_formdocuments_key ON vms.detail_kunjungan USING btree (id_formdocuments)",
			"CREATE UNIQUE INDEX detail_kunjungan_pkey ON vms.detail_kunjungan USING btree (id)",
		},
	},
	{
		name:     "vms.detail_pengunjung",
		grants:   "INSERT,SELECT,UPDATE",
		sequence: "USAGE",
		checks: []string{
			"CHECK ((((asal_instansi)::text = btrim((asal_instansi)::text)) AND (char_length((asal_instansi)::text) >= 2)))",
			"CHECK ((((nama)::text = btrim((nama)::text)) AND ((char_length((nama)::text) >= 2) AND (char_length((nama)::text) <= 120))))",
			"CHECK (((email)::text = lower((email)::text)))",
			"CHECK (((nama_rombongan IS NULL) OR (NOT (jumlah_pengunjung IS DISTINCT FROM cardinality(nama_rombongan)))))",
			"CHECK (((nomor_telpon)::text ~ '^[0-9]{8,20}$'::text))",
			"CHECK ((id > 0))",
			"CHECK (vms.valid_visit_roster(nama_rombongan))",
		},
		foreignKeys: []string{
			"FOREIGN KEY (id_formdocuments) REFERENCES vms.form_documents(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		columns: []string{
			"id integer NOT NULL IDENTITY ALWAYS",
			"nama varchar(255)",
			"asal_instansi varchar(255)",
			"nomor_telpon varchar(255)",
			"email varchar(255)",
			"identity_type USER-DEFINED",
			"nama_rombongan ARRAY",
			"jumlah_pengunjung integer",
			"id_formdocuments integer",
		},
		indexes: []string{
			"CREATE UNIQUE INDEX detail_pengunjung_id_formdocuments_key ON vms.detail_pengunjung USING btree (id_formdocuments)",
			"CREATE UNIQUE INDEX detail_pengunjung_pkey ON vms.detail_pengunjung USING btree (id)",
		},
	},
	{
		name:     "vms.kunjungan",
		grants:   "INSERT,SELECT,UPDATE",
		sequence: "USAGE",
		checks: []string{
			"CHECK ((((status_penerimaan = 'checked_in'::vms.status_penerimaan) AND (waktu_cek_in IS NOT NULL) AND (waktu_cek_out IS NULL)) OR ((status_penerimaan = 'checked_out'::vms.status_penerimaan) AND (waktu_cek_in IS NOT NULL) AND (waktu_cek_out IS NOT NULL) AND (waktu_cek_out >= waktu_cek_in)) OR ((status_penerimaan <> ALL (ARRAY['checked_in'::vms.status_penerimaan, 'checked_out'::vms.status_penerimaan])) AND (waktu_cek_in IS NULL) AND (waktu_cek_out IS NULL))))",
			"CHECK ((id > 0))",
			"CHECK ((kode_kunjungan ~ '^[0-9]{8}$'::text))",
			"CHECK ((version > 0))",
		},
		foreignKeys: []string{
			"FOREIGN KEY (id_detail_kunjungan) REFERENCES vms.detail_kunjungan(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
			"FOREIGN KEY (id_detail_pengunjung) REFERENCES vms.detail_pengunjung(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		columns: []string{
			"id integer NOT NULL IDENTITY ALWAYS",
			"id_detail_pengunjung integer NOT NULL",
			"id_detail_kunjungan integer NOT NULL",
			"kode_kunjungan char(8) NOT NULL",
			"tipe_kunjungan USER-DEFINED NOT NULL",
			"status_penerimaan USER-DEFINED NOT NULL",
			"version integer NOT NULL DEFAULT 1",
			"waktu_cek_in timestamptz",
			"waktu_cek_out timestamptz",
			"tanggal_dibuat timestamptz NOT NULL",
			"alasan_penolakan varchar(1000)",
			"catatan_internal_penolakan varchar(2000)",
			"alasan_pembatalan varchar(1000)",
			"catatan_internal_pembatalan varchar(2000)",
		},
		indexes: []string{
			"CREATE UNIQUE INDEX kunjungan_id_detail_kunjungan_key ON vms.kunjungan USING btree (id_detail_kunjungan)",
			"CREATE UNIQUE INDEX kunjungan_id_detail_pengunjung_key ON vms.kunjungan USING btree (id_detail_pengunjung)",
			"CREATE UNIQUE INDEX kunjungan_kode_kunjungan_key ON vms.kunjungan USING btree (kode_kunjungan)",
			"CREATE UNIQUE INDEX kunjungan_pkey ON vms.kunjungan USING btree (id)",
		},
	},
	{
		name:   "support.booking_requests",
		grants: "INSERT,SELECT",
		checks: []string{
			"CHECK (((((state)::text = 'committed'::text) AND (id_kunjungan IS NOT NULL) AND (payload_digest IS NOT NULL)) OR (((state)::text = 'closed'::text) AND (id_kunjungan IS NULL) AND (payload_digest IS NULL))))",
			"CHECK ((((audience)::text = 'public'::text) = (email_digest IS NOT NULL)))",
			"CHECK (((audience)::text = ANY ((ARRAY['public'::character varying, 'staff'::character varying])::text[])))",
			"CHECK (((state)::text = ANY ((ARRAY['committed'::character varying, 'closed'::character varying])::text[])))",
			"CHECK ((email_digest ~ '^[0-9a-f]{64}$'::text))",
			"CHECK ((owner_digest ~ '^[0-9a-f]{64}$'::text))",
			"CHECK ((payload_digest ~ '^[0-9a-f]{64}$'::text))",
		},
		foreignKeys: []string{
			"FOREIGN KEY (id_kunjungan) REFERENCES vms.kunjungan(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		columns: []string{
			"audience varchar(8) NOT NULL",
			"owner_digest char(64) NOT NULL",
			"reference uuid NOT NULL",
			"payload_digest char(64)",
			"email_digest char(64)",
			"state varchar(9) NOT NULL",
			"id_kunjungan integer",
		},
		indexes: []string{
			"CREATE INDEX booking_requests_visit_idx ON support.booking_requests USING btree (id_kunjungan)",
			"CREATE UNIQUE INDEX booking_requests_pkey ON support.booking_requests USING btree (audience, owner_digest, reference)",
		},
	},
	{
		name:        "vms.qr_kunjungan",
		grants:      "INSERT,SELECT",
		columnGrant: "UPDATE(dicabut_pada)",
		sequence:    "USAGE",
		checks: []string{
			"CHECK ((id > 0))",
			"CHECK (((token_hash)::text ~ '^[0-9a-f]{64}$'::text))",
			"CHECK ((tanggal_expire > tanggal_dibuat))",
		},
		foreignKeys: []string{
			"FOREIGN KEY (id_kunjungan) REFERENCES vms.kunjungan(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		columns: []string{
			"id integer NOT NULL IDENTITY ALWAYS",
			"id_kunjungan integer NOT NULL",
			"token_hash varchar(255) NOT NULL",
			"tanggal_dibuat timestamptz NOT NULL",
			"tanggal_expire timestamptz NOT NULL",
			"dicabut_pada timestamptz",
		},
		indexes: []string{
			"CREATE INDEX qr_kunjungan_visit_idx ON vms.qr_kunjungan USING btree (id_kunjungan)",
			"CREATE UNIQUE INDEX qr_kunjungan_active_visit_key ON vms.qr_kunjungan USING btree (id_kunjungan) WHERE (dicabut_pada IS NULL)",
			"CREATE UNIQUE INDEX qr_kunjungan_pkey ON vms.qr_kunjungan USING btree (id)",
			"CREATE UNIQUE INDEX qr_kunjungan_token_hash_key ON vms.qr_kunjungan USING btree (token_hash)",
		},
	},
}

// openScratch refuses every target except a loopback vms_scratch database reached as
// vms_migrator: the cycle drops every VMS table, so it never runs against vms_stage1.
func openScratch(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("VMS_INTEGRATION_MIGRATION_DATABASE_URL")
	u, err := url.Parse(dsn)
	if err != nil || u == nil || u.Path != "/vms_scratch" || u.User == nil || u.User.Username() != "vms_migrator" {
		t.Fatal("explicit vms_migrator URL for a disposable vms_scratch database is required")
	}
	if u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback() {
		t.Fatal("migration cycle target must be loopback")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot open scratch database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var database, role string
	var version int
	if err := db.QueryRow("SELECT current_database(), current_user, current_setting('server_version_num')::integer").Scan(&database, &role, &version); err != nil ||
		database != "vms_scratch" || role != "vms_migrator" || version < 180000 || version >= 190000 {
		t.Fatal("wrong scratch database, role or PostgreSQL major")
	}
	return db
}

func TestMigrationsRebuildThePhysicalSchemaAcrossAnUpDownUpCycle(t *testing.T) {
	db := openScratch(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	set := vmsmigration.Set()
	total := len(migrationOrder)

	if _, err := set.ExecContext(ctx, db, "postgres", vmsmigration.Source(), migrate.Down); err != nil {
		t.Fatalf("initial down: %v", err)
	}
	assertNoVMSTables(t, db)

	if n, err := set.ExecContext(ctx, db, "postgres", vmsmigration.Source(), migrate.Up); err != nil || n != total {
		t.Fatalf("first up applied %d of %d: %v", n, total, err)
	}
	assertPhysicalSchema(t, db)

	if n, err := set.ExecContext(ctx, db, "postgres", vmsmigration.Source(), migrate.Down); err != nil || n != total {
		t.Fatalf("down reverted %d of %d: %v", n, total, err)
	}
	assertNoVMSTables(t, db)

	if n, err := set.ExecContext(ctx, db, "postgres", vmsmigration.Source(), migrate.Up); err != nil || n != total {
		t.Fatalf("second up applied %d of %d: %v", n, total, err)
	}
	assertPhysicalSchema(t, db)
}

func assertNoVMSTables(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := queryStrings(t, db, `SELECT table_schema || '.' || table_name FROM information_schema.tables
		WHERE table_schema IN ('vms','support') ORDER BY 1`)
	if !slices.Equal(tables, []string{"support.schema_migrations"}) {
		t.Fatalf("down left tables behind: %v", tables)
	}
	var vmsSchema bool
	if err := db.QueryRow("SELECT to_regnamespace('vms') IS NOT NULL").Scan(&vmsSchema); err != nil || vmsSchema {
		t.Fatal("prerequisite down must drop the vms schema")
	}
}

func assertPhysicalSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	want := []string{"support.schema_migrations"}
	for _, table := range physicalSchema {
		want = append(want, table.name)
	}
	slices.Sort(want)
	if got := sortedQuery(t, db, `SELECT table_schema || '.' || table_name FROM information_schema.tables
		WHERE table_schema IN ('vms','support')`); !slices.Equal(got, want) {
		t.Fatalf("tables differ from the physical schema:\n got %v\nwant %v", got, want)
	}
	for _, table := range physicalSchema {
		schema, name, _ := strings.Cut(table.name, ".")
		t.Run(table.name, func(t *testing.T) {
			assertEqual(t, "columns", queryStrings(t, db, `SELECT column_name || ' ' ||
				CASE data_type WHEN 'character varying' THEN 'varchar(' || character_maximum_length || ')'
					WHEN 'character' THEN 'char(' || character_maximum_length || ')'
					WHEN 'timestamp with time zone' THEN 'timestamptz' ELSE data_type END ||
				CASE WHEN is_nullable = 'NO' THEN ' NOT NULL' ELSE '' END ||
				CASE WHEN is_identity = 'YES' THEN ' IDENTITY ' || identity_generation ELSE '' END ||
				COALESCE(' DEFAULT ' || column_default, '')
				FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`, schema, name), table.columns)
			assertEqual(t, "checks", sortedQuery(t, db, `SELECT pg_get_constraintdef(oid) FROM pg_constraint
				WHERE conrelid = $1::regclass AND contype = 'c' ORDER BY 1`, table.name), sorted(table.checks))
			assertEqual(t, "foreign keys", sortedQuery(t, db, `SELECT pg_get_constraintdef(oid) FROM pg_constraint
				WHERE conrelid = $1::regclass AND contype = 'f' ORDER BY 1`, table.name), sorted(table.foreignKeys))
			assertEqual(t, "indexes", sortedQuery(t, db, `SELECT indexdef FROM pg_indexes
				WHERE schemaname = $1 AND tablename = $2 ORDER BY 1`, schema, name), sorted(table.indexes))
			assertEqual(t, "table grants", queryStrings(t, db, `SELECT COALESCE(string_agg(a.privilege_type, ',' ORDER BY a.privilege_type), '')
				FROM pg_class c LEFT JOIN LATERAL aclexplode(c.relacl) a ON a.grantee = 'vms_api'::regrole
				WHERE c.oid = $1::regclass`, table.name), []string{table.grants})
			assertEqual(t, "column grants", queryStrings(t, db, `SELECT COALESCE(string_agg(privilege_type || '(' || columns || ')', ';'), '') FROM (
				SELECT a.privilege_type, string_agg(att.attname, ',' ORDER BY att.attname) AS columns
				FROM pg_attribute att CROSS JOIN LATERAL aclexplode(att.attacl) a
				WHERE att.attrelid = $1::regclass AND a.grantee = 'vms_api'::regrole GROUP BY a.privilege_type) p`, table.name), []string{table.columnGrant})
			assertEqual(t, "identity sequence grants", queryStrings(t, db, `SELECT COALESCE(string_agg(a.privilege_type, ',' ORDER BY a.privilege_type), '')
				FROM pg_attribute att
				JOIN pg_class s ON s.oid = pg_get_serial_sequence($1::text, att.attname)::regclass
				CROSS JOIN LATERAL aclexplode(s.relacl) a
				WHERE att.attrelid = $1::text::regclass AND att.attidentity <> '' AND a.grantee = 'vms_api'::regrole`, table.name), []string{table.sequence})
		})
	}
	t.Run("every foreign key column leads an index", func(t *testing.T) {
		unindexed := queryStrings(t, db, `SELECT c.conrelid::regclass::text || '.' || a.attname FROM pg_constraint c
			JOIN pg_namespace n ON n.oid = c.connamespace AND n.nspname IN ('vms','support')
			JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
			WHERE c.contype = 'f' AND NOT EXISTS (SELECT FROM pg_index i WHERE i.indrelid = c.conrelid AND i.indkey[0] = c.conkey[1])
			ORDER BY 1`)
		if len(unindexed) != 0 {
			t.Fatalf("foreign key columns without an index: %v", unindexed)
		}
	})
	t.Run("runtime role reaches schemas but cannot create objects or read the ledger", func(t *testing.T) {
		assertEqual(t, "schema privileges", queryStrings(t, db, `SELECT s || ':' || has_schema_privilege('vms_api', s, 'USAGE') || ':' || has_schema_privilege('vms_api', s, 'CREATE')
			FROM unnest(ARRAY['support','vms']) AS s ORDER BY 1`), []string{"support:true:false", "vms:true:false"})
		assertEqual(t, "ledger privileges", queryStrings(t, db, `SELECT has_table_privilege('vms_api', 'support.schema_migrations', 'SELECT,INSERT,UPDATE,DELETE')::text`), []string{"false"})
	})
	assertSeeds(t, db)
	assertAuditActorRules(t, db)
	assertUploadRules(t, db)
}

func assertSeeds(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Run("seeds", func(t *testing.T) {
		assertEqual(t, "deputy groups", queryStrings(t, db, `SELECT kode FROM vms.kedeputian ORDER BY id`),
			[]string{"PP", "PDP", "SBPM", "THD", "LHSDA", "PI", "SP"})
		assertEqual(t, "directorates", queryStrings(t, db, `SELECT k.kode || ' ' || d.nama FROM vms.direktorat d JOIN vms.kedeputian k ON k.id = d.id_kedeputian ORDER BY d.id`), []string{
			"PP Direktorat Perencanaan Makro", "PP Direktorat Perencanaan Mikro", "PP Direktorat Pertanahan",
			"PDP Direktorat Pengendalian Penyelenggaraan Pemerintahan dan Perizinan Pembangunan", "PDP Direktorat Pengawasan, Pemantauan, dan Evaluasi", "PDP Direktorat Ketentraman dan Ketertiban Umum",
			"SBPM Direktorat Pelayanan Dasar", "SBPM Direktorat Pemberdayaan Masyarakat", "SBPM Direktorat Kebudayaan, Pariwisata, dan Ekonomi Kreatif",
			"THD Direktorat Pengembangan Ekosistem Digital", "THD Direktorat Transformasi Hijau", "THD Direktorat Data dan Kecerdasan Buatan",
			"LHSDA Direktorat Lingkungan Hidup dan Penanggulangan Bencana", "LHSDA Direktorat Pengembangan Pemanfaatan Kehutanan dan Sumber Daya Air", "LHSDA Direktorat Ketahanan Pangan",
			"PI Direktorat Investasi dan Kemudahan Berusaha", "PI Direktorat Pendanaan", "PI Direktorat Pembiayaan",
			"SP Direktorat Sarana Prasarana Dasar", "SP Direktorat Sarana Prasarana Sosial", "SP Direktorat Pengelolaan Gedung, Kawasan, dan Perkotaan",
		})
		assertEqual(t, "roles", queryStrings(t, db, `SELECT id || ' ' || nama || ' v' || version || ' ' || created_by || '/' || updated_by FROM vms.role ORDER BY id`),
			[]string{"1 PA v1 system/system", "2 Protocol Officer v1 system/system", "3 Receptionist v1 system/system", "4 Super Admin v1 system/system"})
		assertEqual(t, "permissions", queryStrings(t, db, `SELECT p.permission_key || ' = ' || f.nama || ' / ' || a.nama || ' : ' || p.nama
			FROM vms.permission p JOIN vms.feature f ON f.id = p.id_feature JOIN vms.action a ON a.id = p.id_action ORDER BY p.id`), []string{
			"bookings = Kunjungan / Melihat : Melihat kunjungan",
			"create = Kunjungan / Membuat : Membuat janji temu",
			"approve = Kunjungan / Menyetujui : Menyetujui permohonan",
			"reject = Kunjungan / Menolak : Menolak permohonan",
			"reschedule = Kunjungan / Mengubah jadwal : Mengubah jadwal",
			"cancel = Kunjungan / Membatalkan : Membatalkan kunjungan",
			"checkin = Kedatangan & kepulangan / Mencatat kedatangan : Mencatat kedatangan",
			"checkout = Kedatangan & kepulangan / Mencatat kepulangan : Mencatat kepulangan",
			"reports = Laporan / Melihat : Melihat ringkasan laporan",
			"reportExport = Laporan / Mengekspor : Mengekspor ringkasan",
			"people = Pegawai / Mengelola : Mengelola pegawai",
			"roles = Peran & akses / Mengelola : Mengelola akses peran",
			"integrations = Integrasi / Mengelola : Mengelola integrasi",
			"audit = Audit / Melihat : Melihat audit",
		})
		assertEqual(t, "feature and action counts", queryStrings(t, db, `SELECT (SELECT count(*) FROM vms.feature) || '/' || (SELECT count(*) FROM vms.action)`), []string{"7/10"})
		assertEqual(t, "default policies", queryStrings(t, db, `SELECT r.nama || ' -> ' || p.nama || ' ' || p.status_aktif || ' ' || p.created_by || '/' || p.updated_by || ' link ' || rp.created_by || '/' || rp.updated_by
			FROM vms.role r JOIN vms.role_policy rp ON rp.id_role = r.id AND rp.status_aktif JOIN vms.policy p ON p.id = rp.id_policy ORDER BY r.id`), []string{
			"PA -> Kebijakan bawaan PA true system/system link system/system",
			"Protocol Officer -> Kebijakan bawaan Protocol Officer true system/system link system/system",
			"Receptionist -> Kebijakan bawaan Receptionist true system/system link system/system",
			"Super Admin -> Kebijakan bawaan Super Admin true system/system link system/system",
		})
		// The frontend's role defaults (frontendx/packages/data/src/domain.ts sessionFor), in permission ID order.
		assertEqual(t, "default permissions per role", queryStrings(t, db, `SELECT r.nama || ': ' || string_agg(p.permission_key, ',' ORDER BY p.id)
			FROM vms.role r JOIN vms.role_policy rp ON rp.id_role = r.id AND rp.status_aktif
			JOIN vms.policy_permission pp ON pp.id_policy = rp.id_policy JOIN vms.permission p ON p.id = pp.id_permission
			GROUP BY r.id, r.nama ORDER BY r.id`), []string{
			"PA: bookings,create,approve,reject,reschedule,cancel,reports,reportExport",
			"Protocol Officer: bookings,create,approve,reject,reschedule,cancel,reports,reportExport",
			"Receptionist: bookings,checkin,checkout",
			"Super Admin: bookings,reports,reportExport,people,roles,integrations,audit",
		})
		assertEqual(t, "tables without seeds", queryStrings(t, db, `SELECT (SELECT count(*) FROM vms.jabatan) + (SELECT count(*) FROM vms.pejabat) +
			(SELECT count(*) FROM vms.ruangan) + (SELECT count(*) FROM vms.pegawai) + (SELECT count(*) FROM vms.audit_log) +
			(SELECT count(*) FROM vms.form_documents) + (SELECT count(*) FROM support.form_document_uploads) || ''`), []string{"0"})
	})
}

// assertAuditActorRules inserts inside a rolled-back transaction, so the scratch database keeps no rows.
func assertAuditActorRules(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Run("audit_log actor rules", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		var employee int32
		if err := tx.QueryRow(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password)
			VALUES ('Audit Fixture','audit-fixture@example.test',TRUE,FALSE) RETURNING id`).Scan(&employee); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name, actor, entity string
			employee            any
			accepted            bool
		}{
			{"staff with employee", "staff", "pegawai", employee, true},
			{"staff without employee", "staff", "pegawai", nil, false},
			{"visitor without employee", "visitor", "kunjungan", nil, true},
			{"visitor with employee", "visitor", "kunjungan", employee, false},
			{"system without employee", "system", "role", nil, true},
			{"system with employee", "system", "role", employee, false},
			{"retired public actor", "public", "kunjungan", nil, false},
			{"unknown entity", "system", "visit", nil, false},
		} {
			if _, err := tx.Exec("SAVEPOINT audit_case"); err != nil {
				t.Fatal(err)
			}
			_, err := tx.Exec(`INSERT INTO vms.audit_log(actor_type,id_pegawai,entity_type,entity_id,jenis_kegiatan,tanggal,deskripsi,new_values)
				VALUES ($1,$2,$3,1,'test.case',clock_timestamp(),'Audit rule case','{"k":"v"}')`, tc.actor, tc.employee, tc.entity)
			var pgErr *pgconn.PgError
			switch {
			case tc.accepted && err != nil:
				t.Fatalf("%s rejected: %v", tc.name, err)
			case !tc.accepted && !(errors.As(err, &pgErr) && pgErr.Code == "23514"):
				t.Fatalf("%s must violate a check constraint, got %v", tc.name, err)
			}
			if _, err := tx.Exec("ROLLBACK TO SAVEPOINT audit_case"); err != nil {
				t.Fatal(err)
			}
		}
	})
}

// assertUploadRules checks the form_documents and upload lifecycle backstops inside a
// rolled-back transaction: purpose and content type agree, and tokens last exactly one hour.
func assertUploadRules(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Run("upload rules", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		var employee int32
		if err := tx.QueryRow(`INSERT INTO vms.pegawai(nama,email,status_aktif,must_change_password)
			VALUES ('Upload Fixture','upload-fixture@example.test',TRUE,FALSE) RETURNING id`).Scan(&employee); err != nil {
			t.Fatal(err)
		}
		const key = "0f8fad5b-d9cb-469f-a165-70867728950e"
		digest := strings.Repeat("a", 64)
		for _, tc := range []struct {
			name, statement string
			args            []any
			accepted        bool
		}{
			{"document as PDF", `INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose) VALUES ('nd.pdf',$1,'application/pdf',10,'document')`, []any{key}, true},
			{"photo as PNG", `INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose) VALUES ('foto.png',$1,'image/png',10,'visitorPhoto')`, []any{key}, true},
			{"document as PNG", `INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose) VALUES ('nd.png',$1,'image/png',10,'document')`, []any{key}, false},
			{"photo as PDF", `INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose) VALUES ('foto.pdf',$1,'application/pdf',10,'visitorPhoto')`, []any{key}, false},
			{"storage path that is a client path", `INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose) VALUES ('nd.pdf','../nd.pdf','application/pdf',10,'document')`, nil, false},
			{"empty file", `INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose) VALUES ('nd.pdf',$1,'application/pdf',0,'document')`, []any{key}, false},
			{"file over 5 MiB", `INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose) VALUES ('nd.pdf',$1,'application/pdf',5242881,'document')`, []any{key}, false},
			{"blank name", `INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose) VALUES ('  ',$1,'application/pdf',10,'document')`, []any{key}, false},
			{"public upload for one hour", uploadRow, []any{"public", nil, "1 hour", nil, digest}, true},
			{"staff upload by its employee", uploadRow, []any{"staff", employee, "1 hour", nil, digest}, true},
			{"staff upload without employee", uploadRow, []any{"staff", nil, "1 hour", nil, digest}, false},
			{"public upload with employee", uploadRow, []any{"public", employee, "1 hour", nil, digest}, false},
			{"token valid for two hours", uploadRow, []any{"public", nil, "2 hours", nil, digest}, false},
			{"linked while valid", uploadRow, []any{"public", nil, "1 hour", "30 minutes", digest}, true},
			{"linked after expiry", uploadRow, []any{"public", nil, "1 hour", "1 hour", digest}, false},
			{"token hash that is not a digest", uploadRow, []any{"public", nil, "1 hour", nil, "token"}, false},
		} {
			if _, err := tx.Exec("SAVEPOINT upload_case"); err != nil {
				t.Fatal(err)
			}
			_, err := tx.Exec(tc.statement, tc.args...)
			var pgErr *pgconn.PgError
			switch {
			case tc.accepted && err != nil:
				t.Fatalf("%s rejected: %v", tc.name, err)
			case !tc.accepted && !(errors.As(err, &pgErr) && pgErr.Code == "23514"):
				t.Fatalf("%s must violate a check constraint, got %v", tc.name, err)
			}
			if _, err := tx.Exec("ROLLBACK TO SAVEPOINT upload_case"); err != nil {
				t.Fatal(err)
			}
		}
	})
}

// uploadRow inserts a document and its lifecycle row: $1 audience, $2 uploader, $3 validity,
// $4 link delay after creation (or NULL) and $5 token hash.
const uploadRow = `WITH document AS (
	INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose)
	VALUES ('nd.pdf','0f8fad5b-d9cb-469f-a165-70867728950e','application/pdf',10,'document') RETURNING id)
INSERT INTO support.form_document_uploads(id_form_documents,audience,owner_digest,reference,content_sha256,token_hash,key_version,created_at,expires_at,linked_at,uploaded_by)
SELECT id, $1, repeat('b',64), gen_random_uuid(), repeat('c',64), $5, 1, now(), now() + $3::interval, now() + $4::interval, $2 FROM document`

func queryStrings(t *testing.T, db *sql.DB, query string, args ...any) []string {
	t.Helper()
	rows, err := db.Query(query, args...)
	if err != nil {
		t.Fatalf("catalog query failed: %v", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var value sql.NullString
		if err := rows.Scan(&value); err != nil {
			t.Fatal(err)
		}
		out = append(out, value.String)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// sortedQuery orders in Go so expectations do not depend on the database collation.
func sortedQuery(t *testing.T, db *sql.DB, query string, args ...any) []string {
	t.Helper()
	return sorted(queryStrings(t, db, query, args...))
}

func sorted(values []string) []string {
	out := append([]string{}, values...)
	slices.Sort(out)
	if out == nil {
		return []string{}
	}
	return out
}

func assertEqual(t *testing.T, what string, got, want []string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s differ:\n got %q\nwant %q", what, got, want)
	}
}
