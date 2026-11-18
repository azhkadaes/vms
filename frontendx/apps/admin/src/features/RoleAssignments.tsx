import { roles, type Role } from "@vms/data";
import { SearchableMultiSelect } from "@vms/ui";

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
  return (
    <SearchableMultiSelect
      id="person-roles"
      name="roles"
      label="Peran pegawai"
      itemLabel="peran"
      required
      options={roles.map((role) => ({ value: role, label: role }))}
      selected={selected}
      onChange={onChange}
      placeholder="Cari dan pilih peran…"
      disabled={disabled}
      error={error}
    />
  );
}
