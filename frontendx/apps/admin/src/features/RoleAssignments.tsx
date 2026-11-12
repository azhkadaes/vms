import { roles, type Role } from "@vms/data";
import { useRef, useState } from "react";

export default function RoleAssignments({
  selected,
  onChange,
  disabled,
  error,
}: {
  selected: Role[];
  onChange: (value: Role[]) => void;
  disabled?: boolean;
  error?: string;
}) {
  const [search, setSearch] = useState("");
  const details = useRef<HTMLDetailsElement>(null);
  const visible = roles.filter((role) => role.toLowerCase().includes(search.trim().toLowerCase()));
  return (
    <fieldset className="management-fieldset role-assignments" disabled={disabled}>
      <legend>
        Peran pegawai <span aria-hidden="true">*</span>
      </legend>
      <div className="role-tags" aria-label="Peran terpilih">
        {selected.map((role) => (
          <span className="role-tag" key={role}>
            {role}
            <button
              type="button"
              disabled={disabled}
              aria-label={`Hapus peran ${role}`}
              onClick={() => onChange(selected.filter((value) => value !== role))}
            >
              ×
            </button>
          </span>
        ))}
        {!selected.length && <span className="hint">Belum ada peran dipilih.</span>}
      </div>
      <details
        id="person-roles-picker"
        ref={details}
        onKeyDown={(event) => {
          if (event.key === "Escape" && details.current?.open) {
            event.preventDefault();
            event.stopPropagation();
            details.current.open = false;
            details.current.querySelector("summary")?.focus();
          }
        }}
      >
        <summary aria-describedby={error ? "person-roles-error" : undefined}>Pilih peran</summary>
        <div className="role-options">
          <label htmlFor="person-role-search">Cari peran</label>
          <input
            id="person-role-search"
            type="search"
            value={search}
            disabled={disabled}
            onChange={(event) => setSearch(event.target.value)}
          />
          {visible.map((role) => (
            <label className="check-label" key={role}>
              <input
                type="checkbox"
                name="roles"
                value={role}
                checked={selected.includes(role)}
                onChange={(event) =>
                  onChange(
                    event.target.checked
                      ? [...selected, role]
                      : selected.filter((value) => value !== role),
                  )
                }
              />
              {role}
            </label>
          ))}
          {!visible.length && <p role="status">Tidak ada peran yang cocok.</p>}
        </div>
      </details>
      {error && (
        <span id="person-roles-error" role="alert" className="field-error">
          {error}
        </span>
      )}
    </fieldset>
  );
}
