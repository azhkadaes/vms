import DataRefresh from "../components/DataRefresh";
import { managementFreshness } from "../hooks/managementFreshness";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, preview, sessionKey, useSession } from "@vms/data";
import { management } from "@vms/data/management";
import type { RoleDefinition } from "@vms/data/management-contract";
import { Button, Empty, ErrorNotice, Loading } from "@vms/ui";
import { useRef, useState } from "react";
import ManagementDialog from "./ManagementDialog";
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

function RoleEditor({
  record,
  onClose,
  onSaved,
  onReload,
  trigger,
}: {
  record: RoleDefinition;
  onClose: () => void;
  onSaved: () => void;
  onReload: () => Promise<void>;
  trigger: HTMLElement | null;
}) {
  const [permissions, setPermissions] = useState(record.permissions);
  const [dirty, setDirty] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const lock = useRef(false);
  return (
    <ManagementDialog
      title={`Akses ${record.name}`}
      description="Perubahan berlaku untuk akun pada peran ini. Penugasan dan pembatasan jenis kunjungan tetap diperiksa layanan."
      dirty={dirty}
      busy={busy}
      error={error}
      onClose={onClose}
      onReload={onReload}
      returnFocusTo={trigger}
    >
      {(close) => (
        <form
          onSubmit={async (event) => {
            event.preventDefault();
            if (lock.current) return;
            lock.current = true;
            setBusy(true);
            setError(undefined);
            try {
              await management.saveRole(record, permissions);
              onSaved();
            } catch (error) {
              setError(error);
            } finally {
              lock.current = false;
              setBusy(false);
            }
          }}
        >
          <fieldset className="management-fieldset">
            <legend>Izin peran</legend>
            {[...new Set([...record.assignablePermissions, ...record.permissions])].map(
              (permission) => (
                <label key={permission} className="check-label">
                  <input
                    type="checkbox"
                    checked={permissions.includes(permission)}
                    disabled={!record.assignablePermissions.includes(permission)}
                    onChange={(event) => {
                      setDirty(true);
                      setPermissions((values) =>
                        event.target.checked
                          ? [...values, permission]
                          : values.filter((value) => value !== permission),
                      );
                    }}
                  />
                  <span>{permissionLabels[permission] || permission}</span>
                </label>
              ),
            )}
          </fieldset>
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
  const query = useQuery({
    ...managementFreshness,
    queryKey: ["roles", sessionKey(session)],
    queryFn: () => management.roles(),
  });
  const [editing, setEditing] = useState<RoleDefinition | null>(null);
  const [editorGeneration, setEditorGeneration] = useState(0);
  const [notice, setNotice] = useState("");
  const trigger = useRef<HTMLElement | null>(null);
  return (
    <>
      <div className="page-intro">
        <div>
          <h1>Peran & akses</h1>
          <p>Satu peran aktif per akun; izin mengikuti tanggung jawab dan penugasan.</p>
        </div>
      </div>
      {notice && <p role="status">{notice}</p>}
      <DataRefresh q={query} label="peran dan akses" />
      <section className="panel management-panel">
        <div className="panel-heading">
          <h2>Peran operasional</h2>
        </div>
        {query.isPending && <Loading />}
        {query.error && (
          <>
            <ErrorNotice error={query.error} />
            <Button onClick={() => void query.refetch()}>Coba lagi</Button>
          </>
        )}
        {!(query.error instanceof ApiError && [401, 403].includes(query.error.status)) &&
          query.data &&
          (query.data.length ? (
            <div className="management-records">
              {query.data.map((record) => (
                <article className="management-record" key={record.id}>
                  <h3>{record.name}</h3>
                  <p>{record.description}</p>
                  <p>
                    {record.permissions.length
                      ? record.permissions
                          .map((permission) => permissionLabels[permission] || permission)
                          .join(", ")
                      : "Tidak ada izin yang diberikan."}
                  </p>
                  <Button
                    kind="secondary"
                    disabled={!record.editable}
                    onClick={(event) => {
                      trigger.current = event.currentTarget;
                      setEditing(record);
                      setEditorGeneration((value) => value + 1);
                    }}
                    aria-label={`Edit akses ${record.name}`}
                  >
                    Edit akses
                  </Button>
                  {!record.editable && <p className="hint">Peran ini dilindungi layanan.</p>}
                </article>
              ))}
            </div>
          ) : (
            <Empty
              title="Belum ada definisi peran"
              text="Hubungi pengelola layanan untuk menyiapkan peran operasional."
            />
          ))}
        {preview && (
          <p className="hint">
            Contoh konfigurasi lokal. Perubahan ini tidak mengubah otorisasi peran pratinjau.
          </p>
        )}
      </section>
      {editing && !(query.error instanceof ApiError && [401, 403].includes(query.error.status)) && (
        <RoleEditor
          key={`${editing.id}:${editing.version}:${editorGeneration}`}
          record={editing}
          trigger={trigger.current}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            setNotice("Akses peran disimpan.");
            void client.invalidateQueries({ queryKey: ["roles"] });
            void refreshSession();
          }}
          onReload={async () => {
            const records = await management.roles();
            const record = records.find((record) => record.id === editing.id);
            if (!record) throw new Error("Peran tidak lagi tersedia.");
            setEditing(record);
            setEditorGeneration((value) => value + 1);
          }}
        />
      )}
    </>
  );
}
