import {
  ApiError,
  officerOrganization,
  preview,
  sessionKey,
  useCatalogs,
  useSession,
} from "@vms/data";
import { management } from "@vms/data/management";
import {
  employeeInputSchema,
  employeePhoneNumbers,
  type Employee,
  type EmployeeInput,
} from "@vms/data/management-contract";
import {
  Button,
  ErrorNotice,
  Loading,
  RequiredFieldsHint,
  RequiredMark,
  SearchableMultiSelect,
} from "@vms/ui";
import { ReferenceDataNotice } from "@vms/ui/reference-data-notice";
import { useQuery } from "@tanstack/react-query";
import { ChevronDown, Pencil } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import ManagementDialog, { focusServerField } from "./ManagementDialog";
import RoleAssignments from "./RoleAssignments";
import EmployeeDetails from "./EmployeeDetails";

export default function EmployeeEditor({
  employee: initialEmployee,
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
  const [employee, setEmployee] = useState(initialEmployee);
  const [editing, setEditing] = useState(!initialEmployee);
  const detail = useQuery({
    queryKey: ["employee-detail", sessionKey(session), initialEmployee?.id],
    queryFn: () => management.employee(initialEmployee!.id),
    enabled: !!initialEmployee && !editing,
    retry: false,
    refetchOnWindowFocus: false,
    staleTime: 0,
  });
  useEffect(() => {
    if (editing) document.getElementById("person-name")?.focus();
  }, [editing]);
  const [selectedRoles, setSelectedRoles] = useState(employee?.roles || [employee?.role || "PA"]);
  const [assignments, setAssignments] = useState(employee?.assignedOfficerIds || []);
  const [profileOrganization, setProfileOrganization] = useState({
    directorate: employee?.directorate || "",
    deputy: employee?.deputy || "",
  });
  const [phoneRows, setPhoneRows] = useState([{ id: 0, number: "" }]);
  const nextPhoneId = useRef(1);
  const phoneFocus = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (phoneFocus.current) {
      document.getElementById(phoneFocus.current)?.focus();
      phoneFocus.current = undefined;
    }
  }, [phoneRows]);
  const isPA = selectedRoles.includes("PA");
  const assignedOfficer =
    assignments.length === 1
      ? catalogs.data?.officers.find((officer) => officer.id === assignments[0])
      : undefined;
  const assignedOrganization = officerOrganization(assignedOfficer);
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
  const title = employee
    ? editing
      ? "Edit pegawai"
      : "Detail pegawai"
    : preview
      ? "Tambah pegawai demo"
      : "Tambah pegawai";
  async function submit(form: HTMLFormElement) {
    if (lock.current) return;
    const values = new FormData(form);
    const input: EmployeeInput = {
      name: String(values.get("name") || ""),
      email: String(values.get("email") || ""),
      roles: selectedRoles,
      jobTitle: String(values.get("jobTitle") || ""),
      deputy: isPA ? assignedOrganization.deputy : profileOrganization.deputy,
      directorate: isPA ? assignedOrganization.directorate : profileOrganization.directorate,
      phoneNumbers: values.getAll("phoneNumbers").map(String),
      nip: String(values.get("nip") || ""),
      active: own || values.get("active") === "on",
      assignedOfficerIds: selectedRoles.includes("PA") ? assignments : [],
      initialPassword: employee ? undefined : String(values.get("initialPassword") || ""),
      requirePasswordChange: employee ? undefined : values.get("requirePasswordChange") === "on",
    };
    const parsed = employeeInputSchema(minimumPassword, !!employee).safeParse(input);
    if (!parsed.success) {
      setFields(
        Object.fromEntries(
          parsed.error.issues.map((issue) => [String(issue.path[0]), issue.message]),
        ),
      );
      const [field, index] = parsed.error.issues[0].path;
      const inputs = form.querySelectorAll<HTMLElement>(`[name="${CSS.escape(String(field))}"]`);
      const invalidInput = inputs[typeof index === "number" ? index : 0];
      if (field === "phoneNumbers") invalidInput?.closest("details")?.setAttribute("open", "");
      invalidInput?.focus();
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
      if (error instanceof ApiError && Object.keys(error.fieldErrors ?? {})[0] === "phoneNumbers")
        form.querySelector('[name="phoneNumbers"]')?.closest("details")?.setAttribute("open", "");
      focusServerField(form, error);
    } finally {
      lock.current = false;
      setBusy(false);
    }
  }
  return (
    <ManagementDialog
      title={title}
      className="employee-editor-modal form-density-compact"
      description={
        !editing
          ? "Informasi profil dan akses akun pegawai."
          : preview
            ? "Kelola informasi dan akses pegawai pada data demo."
            : "Kelola informasi pegawai dan akses akunnya."
      }
      dirty={dirty}
      busy={busy}
      error={editing ? error : detail.error}
      onClose={onClose}
      onReload={onReload}
      returnFocusTo={returnFocusTo}
    >
      {(requestClose) =>
        !editing && employee ? (
          <>
            {detail.isPending ? (
              <Loading />
            ) : (
              <EmployeeDetails
                employee={detail.data || employee}
                officers={catalogs.data?.officers}
              />
            )}
            {detail.error && (
              <Button type="button" kind="ghost" onClick={() => void detail.refetch()}>
                Coba lagi
              </Button>
            )}
            {!!(detail.data || employee).assignedOfficerIds.length && (
              <ReferenceDataNotice
                loading={catalogs.isPending}
                error={catalogs.error}
                onRetry={() => void catalogs.refetch()}
              />
            )}
            <div className="form-actions employee-editor-actions">
              <Button type="button" kind="ghost" onClick={requestClose}>
                Tutup
              </Button>
              <Button
                type="button"
                disabled={!detail.data || detail.isFetching || !!detail.error}
                onClick={() => {
                  if (!detail.data) return;
                  setEmployee(detail.data);
                  const numbers = employeePhoneNumbers(detail.data);
                  setPhoneRows((numbers.length ? numbers : [""]).map((number, id) => ({ id, number })));
                  nextPhoneId.current = Math.max(1, numbers.length);
                  setSelectedRoles(detail.data.roles || [detail.data.role || "PA"]);
                  setAssignments(detail.data.assignedOfficerIds);
                  setProfileOrganization({
                    directorate: detail.data.directorate || "",
                    deputy: detail.data.deputy || "",
                  });
                  setEditing(true);
                }}
              >
                <Pencil size={15} aria-hidden="true" /> Edit pegawai
              </Button>
            </div>
          </>
        ) : (
          <form
            className="employee-editor-form"
            onChange={() => setDirty(true)}
            onSubmit={(event) => {
              event.preventDefault();
              void submit(event.currentTarget);
            }}
          >
            <section className="employee-editor-section" aria-labelledby="person-information-title">
              <div className="employee-section-heading">
                <h3 id="person-information-title">Informasi pegawai</h3>
                <RequiredFieldsHint />
              </div>
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
              </div>
            </section>
            <details className="employee-profile-details employee-editor-section">
              <summary>
                <span>
                  <strong>Profil & organisasi</strong>
                </span>
                <span className="employee-profile-toggle">
                  <span>Opsional</span>
                  <ChevronDown size={18} aria-hidden="true" />
                </span>
              </summary>
              <div className="form-grid">
                {(
                  [
                    ["jobTitle", "Jabatan"],
                    ["deputy", "Kedeputian"],
                    ["directorate", "Direktorat"],
                    ["nip", "NIP"],
                  ] as const
                )
                  .filter(([key]) => !isPA || (key !== "deputy" && key !== "directorate"))
                  .map(([key, label]) => (
                    <div className="field" key={key}>
                      <label htmlFor={`person-${key}`}>{label}</label>
                      <input
                        id={`person-${key}`}
                        name={key}
                        defaultValue={
                          key === "deputy" || key === "directorate"
                            ? undefined
                            : employee?.[key] || ""
                        }
                        value={
                          key === "deputy" || key === "directorate"
                            ? profileOrganization[key]
                            : undefined
                        }
                        onChange={(event) => {
                          if (key === "deputy" || key === "directorate")
                            setProfileOrganization((current) => ({
                              ...current,
                              [key]: event.target.value,
                            }));
                        }}
                        type="text"
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
                <div className="field span-2" role="group" aria-label="Nomor telepon">
                  {phoneRows.map(({ id, number }, index) => (
                    <div className="field" key={id}>
                      <label htmlFor={`person-phone-${id}`}>
                        {index === 0 ? "Nomor telepon" : `Nomor telepon ${index + 1}`}
                      </label>
                      <input
                        id={`person-phone-${id}`}
                        name="phoneNumbers"
                        type="tel"
                        defaultValue={number}
                        maxLength={255}
                        aria-invalid={!!fields.phoneNumbers}
                        aria-describedby={fields.phoneNumbers ? "person-phones-error" : undefined}
                      />
                      <div>
                        <Button
                          type="button"
                          kind="ghost"
                          aria-label={`Hapus nomor telepon ${index + 1}`}
                          onClick={() => {
                            const remaining = phoneRows.filter((row) => row.id !== id);
                            const next = remaining[index] ?? remaining[index - 1];
                            phoneFocus.current = next ? `person-phone-${next.id}` : "person-add-phone";
                            setPhoneRows(remaining);
                            setDirty(true);
                          }}
                        >
                          Hapus
                        </Button>
                      </div>
                    </div>
                  ))}
                  {fields.phoneNumbers && (
                    <span id="person-phones-error" className="field-error" role="alert">
                      {fields.phoneNumbers}
                    </span>
                  )}
                  <div>
                    <Button
                      id="person-add-phone"
                      type="button"
                      kind="secondary"
                      onClick={() => {
                        const id = nextPhoneId.current++;
                        phoneFocus.current = `person-phone-${id}`;
                        setPhoneRows([...phoneRows, { id, number: "" }]);
                        setDirty(true);
                      }}
                    >
                      Tambah nomor telepon
                    </Button>
                  </div>
                </div>
              </div>
            </details>

            <section className="employee-editor-section" aria-labelledby="person-access-title">
              <div className="employee-section-heading">
                <h3 id="person-access-title">Akses akun</h3>
              </div>
              <div className="employee-access-fields">
                <div className="field">
                  <RoleAssignments
                    selected={selectedRoles}
                    disabled={own || busy}
                    error={fields.roles || fields.role}
                    onChange={(value) => {
                      setSelectedRoles(value);
                      setDirty(true);
                    }}
                  />
                </div>
                {selectedRoles.includes("PA") && (
                  <div className="employee-officer-assignments">
                    <ReferenceDataNotice
                      loading={catalogs.isPending}
                      error={catalogs.error}
                      onRetry={() => void catalogs.refetch()}
                    />
                    <SearchableMultiSelect
                      multiple={false}
                      id="person-officers"
                      name="assignedOfficerIds"
                      label="Pejabat yang didampingi"
                      itemLabel="pejabat"
                      options={[
                        ...(catalogs.data?.officers || [])
                          .filter(
                            (officer) =>
                              !officer.vip && (officer.active || assignments.includes(officer.id)),
                          )
                          .map((officer) => ({
                            value: officer.id,
                            label: officer.name + (officer.active ? "" : " (tidak aktif)"),
                          })),
                        ...unavailable.map((id) => ({
                          value: id,
                          label: "Pejabat tidak tersedia",
                        })),
                      ]}
                      selected={assignments.length === 1 ? assignments : []}
                      onChange={(values) => {
                        setAssignments(values);
                        setDirty(true);
                      }}
                      placeholder="Cari dan pilih pejabat…"
                      disabled={busy || !catalogs.data}
                      error={
                        fields.assignedOfficerIds ||
                        (assignments.length > 1
                          ? "Pilih satu pejabat untuk menggantikan penugasan lama."
                          : undefined)
                      }
                    />
                    {assignments.length === 1 && (
                      <dl
                        className="employee-assignment-organization"
                        aria-label="Unit pejabat terpilih"
                      >
                        <div>
                          <dt>Direktorat</dt>
                          <dd>{assignedOrganization.directorate || "Belum tersedia"}</dd>
                        </div>
                        <div>
                          <dt>Kedeputian</dt>
                          <dd>{assignedOrganization.deputy || "Belum tersedia"}</dd>
                        </div>
                      </dl>
                    )}
                    {assignments.length > 1 && (
                      <Button
                        type="button"
                        kind="ghost"
                        onClick={() => {
                          setAssignments([]);
                          setDirty(true);
                        }}
                      >
                        Hapus penugasan lama
                      </Button>
                    )}
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
                  </div>
                )}
              </div>
              <label className="employee-account-status">
                <span>
                  <strong id="person-active-label">Akun aktif</strong>
                  <small id="person-active-hint">Izinkan pegawai masuk dan menggunakan akun.</small>
                </span>
                <input
                  type="checkbox"
                  name="active"
                  defaultChecked={employee?.active ?? true}
                  disabled={own}
                  aria-labelledby="person-active-label"
                  aria-describedby="person-active-hint"
                />
              </label>
              {own && (
                <p className="hint">
                  Peran dan status akun sendiri tidak dapat diubah dari formulir ini.
                </p>
              )}
            </section>
            {!employee && (
              <section className="employee-editor-section" aria-labelledby="person-security-title">
                <div className="employee-section-heading">
                  <h3 id="person-security-title">Keamanan akun</h3>
                </div>
                <div className="field">
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
                  <small id="person-password-hint">Minimal {minimumPassword} karakter.</small>
                  <ErrorNotice error={fields.initialPassword} />
                  <label className="check-label">
                    <input name="requirePasswordChange" type="checkbox" defaultChecked />
                    <span>Minta penggantian kata sandi saat masuk pertama</span>
                  </label>
                </div>
              </section>
            )}
            <div className="form-actions employee-editor-actions">
              <Button type="button" kind="ghost" onClick={requestClose} disabled={busy}>
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
        )
      }
    </ManagementDialog>
  );
}
