import DataRefresh from "../components/DataRefresh";
import { managementFreshness } from "../hooks/managementFreshness";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, preview, sessionKey, useSession } from "@vms/data";
import { management } from "@vms/data/management";
import type { RoleDefinition } from "@vms/data/management-contract";
import { Button, Empty, ErrorNotice, Loading, TableRegion } from "@vms/ui";
import { Check, LockKeyhole, Minus, Pencil, Users } from "lucide-react";
import { useRef, useState } from "react";
import { Link } from "react-router";
import ManagementDialog from "./ManagementDialog";

// Display groups only. The API's editable/assignablePermissions fields govern changes.
const permissionFeatures = [
  {
    name: "Kunjungan",
    permissions: ["bookings", "create", "approve", "reject", "reschedule", "cancel"],
  },
  { name: "Kedatangan & kepulangan", permissions: ["checkin", "checkout"] },
  { name: "Laporan", permissions: ["reports", "reportExport"] },
  { name: "Pegawai", permissions: ["people"] },
  { name: "Peran & akses", permissions: ["roles"] },
  { name: "Integrasi", permissions: ["integrations"] },
  { name: "Audit", permissions: ["audit"] },
];
const permissionLabels: Record<string, string> = {
  bookings: "Melihat kunjungan",
  create: "Membuat janji temu",
  approve: "Menyetujui permohonan",
  reject: "Menolak permohonan",
  reschedule: "Mengubah jadwal",
  cancel: "Membatalkan kunjungan",
  checkin: "Mencatat kedatangan",
  checkout: "Mencatat kepulangan",
  reports: "Melihat ringkasan laporan",
  reportExport: "Mengekspor ringkasan",
  people: "Mengelola pegawai",
  roles: "Mengelola akses peran",
  integrations: "Mengelola integrasi",
  audit: "Melihat audit",
};
function groupPermissions(permissions: string[]) {
  const keys = new Set(permissions);
  const known = new Set(permissionFeatures.flatMap((feature) => feature.permissions));
  return [
    ...permissionFeatures.map((feature) => ({
      ...feature,
      permissions: feature.permissions.filter((permission) => keys.has(permission)),
    })),
    { name: "Izin lainnya", permissions: [...keys].filter((permission) => !known.has(permission)) },
  ].filter((feature) => feature.permissions.length);
}
const labelFor = (permission: string) => permissionLabels[permission] || permission;

function PermissionMatrix({ records }: { records: RoleDefinition[] }) {
  const features = groupPermissions(
    records.flatMap((record) => [...record.permissions, ...record.assignablePermissions]),
  );
  return (
    <section className="panel role-matrix-panel" aria-labelledby="role-matrix-title">
      <div className="panel-heading role-matrix-heading">
        <div>
          <h2 id="role-matrix-title">Matriks izin</h2>
          <p>Bandingkan tindakan yang diizinkan pada setiap peran.</p>
          <p className="role-matrix-scroll-hint">Geser tabel untuk melihat peran lainnya.</p>
        </div>
        <div className="role-matrix-legend" aria-label="Keterangan matriks">
          <span>
            <Check size={16} aria-hidden="true" /> Diizinkan
          </span>
          <span>
            <Minus size={16} aria-hidden="true" /> Tidak diberikan
          </span>
          <span>
            <LockKeyhole size={14} aria-hidden="true" /> Izin tetap
          </span>
        </div>
      </div>
      {features.length ? (
        <TableRegion className="role-matrix-scroll" label="Perbandingan izin peran">
          <table className="role-matrix">
            <caption className="sr-only">Izin per fitur dan tindakan untuk setiap peran</caption>
            <thead>
              <tr>
                <th scope="col">Fitur / tindakan</th>
                {records.map((record) => (
                  <th scope="col" key={record.id}>
                    {record.name}
                  </th>
                ))}
              </tr>
            </thead>
            {features.map((feature) => (
              <tbody key={feature.name}>
                <tr className="role-matrix-feature">
                  <th scope="rowgroup" colSpan={records.length + 1}>
                    {feature.name}
                  </th>
                </tr>
                {feature.permissions.map((permission) => (
                  <tr key={permission}>
                    <th scope="row">{labelFor(permission)}</th>
                    {records.map((record) => {
                      const granted = record.permissions.includes(permission);
                      const fixed =
                        granted &&
                        (!record.editable || !record.assignablePermissions.includes(permission));
                      return (
                        <td key={record.id}>
                          <span className={`role-permission-state${granted ? " is-granted" : ""}`}>
                            {granted ? (
                              <Check size={17} aria-hidden="true" />
                            ) : (
                              <Minus size={17} aria-hidden="true" />
                            )}
                            <span className="sr-only">
                              {granted ? "Diizinkan" : "Tidak diberikan"}
                              {fixed ? "; izin tetap" : ""}
                            </span>
                            {fixed && <LockKeyhole size={12} aria-hidden="true" />}
                          </span>
                        </td>
                      );
                    })}
                  </tr>
                ))}
              </tbody>
            ))}
          </table>
        </TableRegion>
      ) : (
        <p className="role-matrix-note">Belum ada izin yang tersedia.</p>
      )}
      <p className="role-matrix-note">
        Cakupan kunjungan tetap mengikuti penugasan pejabat dan jenis kunjungan.
      </p>
    </section>
  );
}

function RoleEditor({
  record,
  onClose,
  onSaved,
  onReload,
  trigger,
}: {
  record: RoleDefinition;
  onClose: () => void;
  onSaved: (record: RoleDefinition) => void;
  onReload: () => Promise<void>;
  trigger: HTMLElement | null;
}) {
  const [permissions, setPermissions] = useState([...new Set(record.permissions)]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const lock = useRef(false);
  const added = permissions.filter((permission) => !record.permissions.includes(permission));
  const removed = record.permissions.filter((permission) => !permissions.includes(permission));
  const dirty = added.length > 0 || removed.length > 0;
  const features = groupPermissions([...record.assignablePermissions, ...record.permissions]);
  return (
    <ManagementDialog
      title={`Akses ${record.name}`}
      description="Atur izin per fitur. Perubahan berlaku untuk seluruh pegawai yang menggunakan peran ini."
      dirty={dirty}
      busy={busy}
      error={error}
      onClose={onClose}
      onReload={onReload}
      returnFocusTo={trigger}
    >
      {(close) => (
        <form
          className="role-editor"
          onSubmit={async (event) => {
            event.preventDefault();
            if (lock.current || !dirty || !record.editable) return;
            lock.current = true;
            setBusy(true);
            setError(undefined);
            try {
              onSaved(await management.saveRole(record, permissions));
            } catch (error) {
              setError(error);
            } finally {
              lock.current = false;
              setBusy(false);
            }
          }}
        >
          <div className="role-editor-summary">
            <span>{permissions.length} izin aktif</span>
            <Button
              kind="ghost"
              type="button"
              disabled={busy || !dirty}
              onClick={() => setPermissions([...new Set(record.permissions)])}
            >
              Atur ulang
            </Button>
          </div>
          {features.map((feature) => (
            <fieldset className="management-fieldset role-permission-group" key={feature.name}>
              <legend>{feature.name}</legend>
              {feature.permissions.map((permission) => {
                const fixed =
                  !record.editable || !record.assignablePermissions.includes(permission);
                return (
                  <label key={permission} className="check-label role-permission-option">
                    <input
                      type="checkbox"
                      name="permissions"
                      value={permission}
                      checked={permissions.includes(permission)}
                      disabled={fixed}
                      onChange={(event) =>
                        setPermissions((values) =>
                          event.target.checked
                            ? [...values, permission]
                            : values.filter((value) => value !== permission),
                        )
                      }
                    />
                    <span>
                      {labelFor(permission)}
                      {fixed && (
                        <small>
                          <LockKeyhole size={12} aria-hidden="true" /> Izin tetap
                        </small>
                      )}
                    </span>
                  </label>
                );
              })}
            </fieldset>
          ))}
          {dirty && (
            <div className="role-access-changes" role="status" aria-atomic="true">
              <strong>Perubahan akses</strong>
              {!!added.length && <p>Ditambahkan: {added.map(labelFor).join(", ")}.</p>}
              {!!removed.length && <p>Dihapus: {removed.map(labelFor).join(", ")}.</p>}
            </div>
          )}
          <div className="form-actions">
            <Button kind="secondary" type="button" disabled={busy} onClick={close}>
              Batal
            </Button>
            <Button type="submit" disabled={busy || !dirty}>
              {busy ? "Menyimpan…" : "Simpan akses"}
            </Button>
          </div>
        </form>
      )}
    </ManagementDialog>
  );
}

export default function RoleManagement() {
  const { session, refreshSession } = useSession();
  const client = useQueryClient();
  const queryKey = ["roles", sessionKey(session)];
  const query = useQuery({
    ...managementFreshness,
    queryKey,
    queryFn: () => management.roles(),
    enabled: !!session,
  });
  const [editing, setEditing] = useState<RoleDefinition | null>(null);
  const [editorGeneration, setEditorGeneration] = useState(0);
  const [notice, setNotice] = useState("");
  const trigger = useRef<HTMLElement | null>(null);
  const accessError = query.error instanceof ApiError && [401, 403].includes(query.error.status);
  const records = accessError ? undefined : query.data;
  return (
    <>
      <div className="page-intro role-intro">
        <div>
          <h1>Peran & akses</h1>
          <p>Kelola izin setiap peran berdasarkan fitur dan tindakan.</p>
        </div>
        {session?.permissions.includes("people") && (
          <Link to="/employees" className="link-button secondary">
            <Users size={16} aria-hidden="true" /> Penugasan pegawai
          </Link>
        )}
      </div>
      {notice && (
        <p role="status" className="role-save-notice">
          <Check size={16} aria-hidden="true" />
          {notice}
        </p>
      )}
      <DataRefresh q={query} label="peran dan akses" />
      {query.isPending && <Loading />}
      {query.error && <ErrorNotice error={query.error} />}
      {records &&
        (records.length ? (
          <>
            <section aria-labelledby="role-list-title" className="role-overview">
              <div className="role-section-heading">
                <h2 id="role-list-title">Peran operasional</h2>
                <p>{records.length} peran tersedia</p>
              </div>
              <div className="role-cards">
                {records.map((record) => {
                  const canEdit = record.editable && record.assignablePermissions.length > 0;
                  return (
                    <article className="panel role-card" key={record.id}>
                      <div className="role-card-heading">
                        <h3>{record.name}</h3>
                        {!canEdit && <LockKeyhole size={16} aria-label="Peran dilindungi" />}
                      </div>
                      <p>{record.description || "Izin mengikuti peran dan penugasan pegawai."}</p>
                      <div className="role-card-footer">
                        <span>{new Set(record.permissions).size} izin aktif</span>
                        <Button
                          kind="secondary"
                          disabled={!canEdit}
                          onClick={(event) => {
                            trigger.current = event.currentTarget;
                            setNotice("");
                            setEditing(record);
                            setEditorGeneration((value) => value + 1);
                          }}
                          aria-label={`Edit akses ${record.name}`}
                        >
                          <Pencil size={14} aria-hidden="true" /> Edit akses
                        </Button>
                      </div>
                      {!canEdit && <p className="hint">Izin peran ini dilindungi.</p>}
                    </article>
                  );
                })}
              </div>
            </section>
            <PermissionMatrix records={records} />
          </>
        ) : (
          <Empty
            title="Belum ada definisi peran"
            text="Hubungi pengelola layanan untuk menyiapkan peran operasional."
          />
        ))}
      {preview && (
        <p className="role-preview-note hint">
          Contoh konfigurasi lokal. Perubahan izin pada pratinjau tidak mengubah akses akun demo.
        </p>
      )}
      {editing && !accessError && (
        <RoleEditor
          key={`${editing.id}:${editing.version}:${editorGeneration}`}
          record={editing}
          trigger={trigger.current}
          onClose={() => setEditing(null)}
          onSaved={(saved) => {
            setEditing(null);
            setNotice(`Akses ${saved.name} disimpan.`);
            client.setQueryData<RoleDefinition[]>(queryKey, (current) =>
              current?.map((record) => (record.id === saved.id ? saved : record)),
            );
            void client.invalidateQueries({ queryKey: ["roles"] });
            void refreshSession();
          }}
          onReload={async () => {
            const records = await management.roles();
            const record = records.find((record) => record.id === editing.id);
            if (!record) throw new Error("Peran tidak lagi tersedia.");
            if (!record.editable || !record.assignablePermissions.length)
              throw new Error(
                "Izin peran ini kini dilindungi. Tutup editor dan perbarui daftar peran.",
              );
            setEditing(record);
            setEditorGeneration((value) => value + 1);
          }}
        />
      )}
    </>
  );
}
