import { ApiError, preview, useCatalogs, useSession } from "@vms/data";
import { management } from "@vms/data/management";
import {
  employeeInputSchema,
  type Employee,
  type EmployeeInput,
} from "@vms/data/management-contract";
import { Button, ErrorNotice, RequiredFieldsHint, RequiredMark } from "@vms/ui";
import { ReferenceDataNotice } from "@vms/ui/reference-data-notice";
import { useRef, useState } from "react";
import ManagementDialog, { focusServerField } from "./ManagementDialog";
import RoleAssignments from "./RoleAssignments";

export default function EmployeeEditor({
  employee,
  minimumPassword,
  onClose,
  onSaved,
  onReload,
  returnFocusTo,
}: {
  employee: Employee | null;
  minimumPassword: number;
  onClose: () => void;
  onSaved: () => void;
  onReload: () => Promise<void>;
  returnFocusTo?: HTMLElement | null;
}) {
  const { session } = useSession();
  const catalogs = useCatalogs();
  const [selectedRoles, setSelectedRoles] = useState(employee?.roles || [employee?.role || "PA"]);
  const [assignments, setAssignments] = useState(employee?.assignedOfficerIds || []);
  const unavailable = catalogs.data
    ? assignments.filter(
        (id) => !catalogs.data.officers.some((officer) => officer.id === id && !officer.vip),
      )
    : [];
  const [dirty, setDirty] = useState(false);
  const [busy, setBusy] = useState(false);
  const lock = useRef(false);
  const [error, setError] = useState<unknown>();
  const [fields, setFields] = useState<Record<string, string>>({});
  const own = !!employee && employee.id === session?.id;
  const title = employee ? "Edit pegawai" : preview ? "Tambah pegawai demo" : "Tambah pegawai";
  async function submit(form: HTMLFormElement) {
    if (lock.current) return;
    const values = new FormData(form);
    const input: EmployeeInput = {
      name: String(values.get("name") || ""),
      email: String(values.get("email") || ""),
      roles: selectedRoles,
      jobTitle: String(values.get("jobTitle") || ""),
      deputy: String(values.get("deputy") || ""),
      directorate: String(values.get("directorate") || ""),
      phone: String(values.get("phone") || ""),
      nip: String(values.get("nip") || ""),
      active: own || values.get("active") === "on",
      assignedOfficerIds: selectedRoles.includes("PA") ? assignments : [],
      initialPassword: employee ? undefined : String(values.get("initialPassword") || ""),
      requirePasswordChange: employee ? undefined : values.get("requirePasswordChange") === "on",
    };
    const parsed = employeeInputSchema(minimumPassword, !!employee).safeParse(input);
    if (!parsed.success) {
      if (parsed.error.issues[0].path[0] === "roles") {
        const picker = form.querySelector<HTMLDetailsElement>("#person-roles-picker");
        if (picker) picker.open = true;
        form.querySelector<HTMLElement>("#person-role-search")?.focus();
      }
      setFields(
        Object.fromEntries(
          parsed.error.issues.map((issue) => [String(issue.path[0]), issue.message]),
        ),
      );
      form
        .querySelector<HTMLElement>(
          `[name="${CSS.escape(String(parsed.error.issues[0].path[0]))}"]`,
        )
        ?.focus();
      return;
    }
    lock.current = true;
    setBusy(true);
    setError(undefined);
    setFields({});
    try {
      await management.saveEmployee(employee?.id || null, parsed.data, employee?.version);
      onSaved();
    } catch (error) {
      setError(error);
      if (error instanceof ApiError && error.fieldErrors)
        setFields(
          Object.fromEntries(
            Object.entries(error.fieldErrors).map(([key, value]) => [key, value.join(" ")]),
          ),
        );
      focusServerField(form, error);
    } finally {
      lock.current = false;
      setBusy(false);
    }
  }
  return (
    <ManagementDialog
      title={title}
      description={
        preview
          ? "Contoh lokal; tidak membuat akun atau menyimpan kata sandi nyata."
          : "Pegawai dapat memiliki beberapa peran dan menggunakan satu peran pada satu waktu."
      }
      dirty={dirty}
      busy={busy}
      error={error}
      onClose={onClose}
      onReload={onReload}
      returnFocusTo={returnFocusTo}
    >
      {(requestClose) => (
        <form
          onChange={() => setDirty(true)}
          onSubmit={(event) => {
            event.preventDefault();
            void submit(event.currentTarget);
          }}
        >
          <RequiredFieldsHint />
          <div className="form-grid">
            <div className="field">
              <label htmlFor="person-name">
                Nama lengkap <RequiredMark />
              </label>
              <input
                id="person-name"
                name="name"
                required
                minLength={2}
                maxLength={255}
                defaultValue={employee?.name}
                autoComplete="name"
                aria-invalid={!!fields.name}
                aria-describedby={fields.name ? "person-name-error" : undefined}
                onInput={(event) =>
                  event.currentTarget.setCustomValidity(
                    event.currentTarget.value.trim().length < 2
                      ? "Masukkan nama minimal dua karakter."
                      : "",
                  )
                }
              />
              {fields.name && (
                <span id="person-name-error" className="field-error" role="alert">
                  {fields.name}
                </span>
              )}
            </div>
            <div className="field">
              <label htmlFor="person-email">
                Email <RequiredMark />
              </label>
              <input
                id="person-email"
                name="email"
                type="email"
                required
                maxLength={254}
                defaultValue={employee?.email}
                autoComplete="email"
                aria-invalid={!!fields.email}
                aria-describedby={fields.email ? "person-email-error" : undefined}
              />
              {fields.email && (
                <span id="person-email-error" className="field-error" role="alert">
                  {fields.email}
                </span>
              )}
            </div>
            <div className="field span-2">
              <RoleAssignments
                selected={selectedRoles}
                disabled={own}
                error={fields.roles || fields.role}
                onChange={(value) => {
                  setSelectedRoles(value);
                  setDirty(true);
                }}
              />
            </div>
            {selectedRoles.includes("PA") && (
              <fieldset className="management-fieldset span-2">
                <legend>Penugasan pejabat</legend>
                <ReferenceDataNotice
                  loading={catalogs.isPending}
                  error={catalogs.error}
                  onRetry={() => void catalogs.refetch()}
                />
                {catalogs.data?.officers
                  .filter(
                    (officer) =>
                      !officer.vip && (officer.active || assignments.includes(officer.id)),
                  )
                  .map((officer) => (
                    <label className="check-label" key={officer.id}>
                      <input
                        type="checkbox"
                        name="assignedOfficerIds"
                        value={officer.id}
                        checked={assignments.includes(officer.id)}
                        disabled={!officer.active && !assignments.includes(officer.id)}
                        onChange={(event) =>
                          setAssignments((ids) =>
                            event.target.checked
                              ? [...ids, officer.id]
                              : ids.filter((id) => id !== officer.id),
                          )
                        }
                      />
                      <span>
                        {officer.name}
                        {!officer.active && " (tidak aktif)"}
                      </span>
                    </label>
                  ))}
                {!!unavailable.length && (
                  <div className="hint">
                    <p>
                      {unavailable.length} penugasan tidak tersedia dalam direktori saat ini.
                      Nilainya dipertahankan sampai Anda menghapusnya.
                    </p>
                    <Button
                      type="button"
                      kind="secondary"
                      onClick={() => {
                        setAssignments((ids) => ids.filter((id) => !unavailable.includes(id)));
                        setDirty(true);
                      }}
                    >
                      Hapus penugasan yang tidak tersedia
                    </Button>
                  </div>
                )}
                <ErrorNotice error={fields.assignedOfficerIds} />
              </fieldset>
            )}
            {!employee && (
              <div className="field span-2">
                <label htmlFor="person-password">
                  Kata sandi awal <RequiredMark />
                </label>
                <input
                  id="person-password"
                  name="initialPassword"
                  type="password"
                  required
                  minLength={minimumPassword}
                  maxLength={256}
                  autoComplete="new-password"
                  aria-invalid={!!fields.initialPassword}
                  aria-describedby="person-password-hint"
                />
                <small id="person-password-hint">
                  Minimal {minimumPassword} karakter, sesuai aturan layanan. Bagikan melalui saluran
                  yang disetujui organisasi.
                </small>
                <ErrorNotice error={fields.initialPassword} />
                <label className="check-label">
                  <input name="requirePasswordChange" type="checkbox" defaultChecked />
                  <span>Minta penggantian kata sandi saat masuk pertama</span>
                </label>
              </div>
            )}
            <label className="check-label span-2">
              <input
                type="checkbox"
                name="active"
                defaultChecked={employee?.active ?? true}
                disabled={own}
              />
              <span>Akun aktif</span>
            </label>
          </div>
          <details className="profile-details">
            <summary>Detail profil tambahan</summary>
            <div className="form-grid">
              {(
                [
                  ["jobTitle", "Jabatan"],
                  ["deputy", "Kedeputian"],
                  ["directorate", "Direktorat"],
                  ["phone", "Nomor telepon"],
                  ["nip", "NIP"],
                ] as const
              ).map(([key, label]) => (
                <div className="field" key={key}>
                  <label htmlFor={`person-${key}`}>
                    {label} <small>(opsional)</small>
                  </label>
                  <input
                    id={`person-${key}`}
                    name={key}
                    defaultValue={employee?.[key] || ""}
                    type={key === "phone" ? "tel" : "text"}
                    maxLength={255}
                    aria-invalid={!!fields[key]}
                    aria-describedby={fields[key] ? `person-${key}-error` : undefined}
                  />
                  {fields[key] && (
                    <span id={`person-${key}-error`} className="field-error" role="alert">
                      {fields[key]}
                    </span>
                  )}
                </div>
              ))}
            </div>
          </details>
          {own && (
            <p className="hint">
              Peran dan status akun sendiri tidak dapat diubah dari formulir ini.
            </p>
          )}
          <div className="form-actions">
            <Button type="button" kind="secondary" onClick={requestClose} disabled={busy}>
              Batal
            </Button>
            <Button
              type="submit"
              disabled={busy || (selectedRoles.includes("PA") && !catalogs.data)}
            >
              {busy ? "Menyimpan…" : preview ? "Simpan contoh" : "Simpan pegawai"}
            </Button>
          </div>
        </form>
      )}
    </ManagementDialog>
  );
}
