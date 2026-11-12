import DataRefresh from "../components/DataRefresh";
import { managementFreshness } from "../hooks/managementFreshness";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, preview, sessionKey, useSession } from "@vms/data";
import { management } from "@vms/data/management";
import type { Integration } from "@vms/data/management-contract";
import { Button, Empty, ErrorNotice, Loading, RequiredFieldsHint, RequiredMark } from "@vms/ui";
import { useRef, useState } from "react";
import ManagementDialog, { focusServerField } from "./ManagementDialog";

const healthLabels = {
  ready: "Siap",
  degraded: "Perlu perhatian",
  not_configured: "Belum dikonfigurasi",
};
function IntegrationEditor({
  record,
  onClose,
  onSaved,
  onReload,
  trigger,
}: {
  record: Integration;
  onClose: () => void;
  onSaved: () => void;
  onReload: () => Promise<void>;
  trigger: HTMLElement | null;
}) {
  const [enabled, setEnabled] = useState(record.enabled);
  const [clearSecrets, setClearSecrets] = useState<string[]>([]);
  const [dirty, setDirty] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const [fields, setFields] = useState<Record<string, string[]>>({});
  const lock = useRef(false);
  return (
    <ManagementDialog
      title={`Konfigurasi ${record.name}`}
      description="Kredensial tersimpan tidak ditampilkan. Biarkan isian kredensial kosong untuk mempertahankannya."
      dirty={dirty}
      busy={busy}
      error={error}
      onClose={onClose}
      onReload={onReload}
      returnFocusTo={trigger}
    >
      {(close) => (
        <form
          onChange={() => setDirty(true)}
          onSubmit={async (event) => {
            event.preventDefault();
            if (lock.current) return;
            const form = event.currentTarget;
            const data = new FormData(form);
            const values = Object.fromEntries(
              record.fields
                .filter((field) => !clearSecrets.includes(field.key))
                .map((field) => [field.key, String(data.get(field.key) || "")])
                .filter(
                  ([key, value]) =>
                    value !== "" ||
                    record.fields.find((field) => field.key === key)?.kind !== "secret",
                ),
            );
            lock.current = true;
            setBusy(true);
            setError(undefined);
            setFields({});
            try {
              await management.saveIntegration(record, {
                enabled,
                values,
                clearSecrets,
                version: record.version,
              });
              onSaved();
            } catch (error) {
              setError(error);
              if (error instanceof ApiError) setFields(error.fieldErrors || {});
              focusServerField(form, error);
            } finally {
              lock.current = false;
              setBusy(false);
            }
          }}
        >
          <RequiredFieldsHint />
          <label className="check-label">
            <input
              type="checkbox"
              name="enabled"
              checked={enabled}
              onChange={(event) => setEnabled(event.target.checked)}
            />
            <span>Integrasi aktif</span>
          </label>
          <div className="form-grid">
            {record.fields.map((field) => {
              const secret = field.kind === "secret";
              const clearing = clearSecrets.includes(field.key);
              const required =
                enabled && field.required && !(secret && field.configured && !clearing);
              return (
                <div className="field span-2" key={field.key}>
                  <label htmlFor={`integration-${field.key}`}>
                    {field.label}
                    {required && <RequiredMark />}
                  </label>
                  <input
                    id={`integration-${field.key}`}
                    name={field.key}
                    type={
                      secret
                        ? "password"
                        : field.kind === "number"
                          ? "number"
                          : field.kind === "text"
                            ? "text"
                            : field.kind
                    }
                    defaultValue={secret ? "" : field.value || ""}
                    required={required}
                    disabled={clearing}
                    autoComplete={secret ? "new-password" : "off"}
                    aria-invalid={!!fields[field.key]}
                    aria-describedby={`integration-${field.key}-hint`}
                  />
                  <small id={`integration-${field.key}-hint`}>
                    {secret
                      ? field.configured
                        ? "Kredensial sudah tersimpan. Isi hanya untuk menggantinya."
                        : "Belum ada kredensial tersimpan."
                      : "Nilai konfigurasi layanan."}
                  </small>
                  {fields[field.key] && (
                    <span className="field-error" role="alert">
                      {fields[field.key].join(" ")}
                    </span>
                  )}
                  {secret && field.configured && (
                    <label className="check-label">
                      <input
                        type="checkbox"
                        checked={clearing}
                        onChange={(event) =>
                          setClearSecrets((keys) =>
                            event.target.checked
                              ? [...keys, field.key]
                              : keys.filter((key) => key !== field.key),
                          )
                        }
                      />
                      <span>Hapus kredensial tersimpan</span>
                    </label>
                  )}
                </div>
              );
            })}
          </div>
          {enabled &&
            clearSecrets.some((key) =>
              record.fields.some((field) => field.key === key && field.required),
            ) && <p role="alert">Nonaktifkan integrasi sebelum menghapus kredensial wajib.</p>}
          <div className="form-actions">
            <Button kind="secondary" type="button" disabled={busy} onClick={close}>
              Batal
            </Button>
            <Button
              type="submit"
              disabled={
                busy ||
                (enabled &&
                  clearSecrets.some((key) =>
                    record.fields.some((field) => field.key === key && field.required),
                  ))
              }
            >
              {busy ? "Menyimpan…" : "Simpan konfigurasi"}
            </Button>
          </div>
        </form>
      )}
    </ManagementDialog>
  );
}

export default function IntegrationManagement() {
  const { session } = useSession();
  const client = useQueryClient();
  const query = useQuery({
    ...managementFreshness,
    queryKey: ["integrations", sessionKey(session)],
    queryFn: () => management.integrations(),
  });
  const [editing, setEditing] = useState<Integration | null>(null);
  const [editorGeneration, setEditorGeneration] = useState(0);
  const [notice, setNotice] = useState("");
  const trigger = useRef<HTMLElement | null>(null);
  return (
    <>
      <div className="page-intro">
        <div>
          <h1>Integrasi</h1>
          <p>Koneksi layanan untuk operasional kunjungan.</p>
        </div>
      </div>
      {notice && <p role="status">{notice}</p>}
      <DataRefresh q={query} label="integrasi" />
      <section className="panel management-panel">
        <div className="panel-heading">
          <h2>Layanan terdaftar</h2>
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
                    {record.enabled ? "Aktif" : "Nonaktif"} · {healthLabels[record.health]}
                  </p>
                  <Button
                    kind="secondary"
                    onClick={(event) => {
                      trigger.current = event.currentTarget;
                      setEditing(record);
                      setEditorGeneration((value) => value + 1);
                    }}
                    aria-label={`Konfigurasi ${record.name}`}
                  >
                    Konfigurasi
                  </Button>
                </article>
              ))}
            </div>
          ) : (
            <Empty
              title="Belum ada integrasi terdaftar"
              text="Layanan belum menyediakan konfigurasi integrasi."
            />
          ))}
        {preview && (
          <p className="hint">
            Konfigurasi contoh lokal; tidak menghubungkan email, berkas, atau perangkat nyata.
          </p>
        )}
      </section>
      {editing && !(query.error instanceof ApiError && [401, 403].includes(query.error.status)) && (
        <IntegrationEditor
          key={`${editing.id}:${editing.version}:${editorGeneration}`}
          record={editing}
          trigger={trigger.current}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            setNotice("Konfigurasi disimpan. Status koneksi mengikuti pemeriksaan layanan.");
            void client.invalidateQueries({ queryKey: ["integrations"] });
          }}
          onReload={async () => {
            const records = await management.integrations();
            const record = records.find((record) => record.id === editing.id);
            if (!record) throw new Error("Integrasi tidak lagi tersedia.");
            setEditing(record);
            setEditorGeneration((value) => value + 1);
          }}
        />
      )}
    </>
  );
}
