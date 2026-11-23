import { employeePhoneNumbers, type Employee } from "@vms/data/management-contract";
import { officerOrganization, type Catalogs } from "@vms/data";

export default function EmployeeDetails({
  employee,
  officers,
  showAssignments = true,
}: {
  employee: Employee;
  officers?: Catalogs["officers"];
  showAssignments?: boolean;
}) {
  const roles = employee.roles || (employee.role ? [employee.role] : []);
  const assignedOfficer =
    employee.assignedOfficerIds.length === 1
      ? officers?.find((officer) => officer.id === employee.assignedOfficerIds[0])
      : undefined;
  const organization = roles.includes("PA") ? officerOrganization(assignedOfficer) : undefined;
  const phoneNumbers = employeePhoneNumbers(employee);
  const profile = (
    [
      ["Email", employee.email],
      ["NIP", employee.nip],
      ["Jabatan", employee.jobTitle],
      ["Kedeputian", organization?.deputy || employee.deputy],
      ["Direktorat", organization?.directorate || employee.directorate],
    ] as const
  ).filter(([, value]) => value?.trim());
  return (
    <div className="employee-details">
      <div className="employee-detail-identity">
        <span className="employee-detail-avatar" aria-hidden="true">
          {employee.name
            .trim()
            .split(/\s+/)
            .slice(0, 2)
            .map((part) => part[0])
            .join("")
            .toUpperCase()}
        </span>
        <div>
          <h3>{employee.name}</h3>
          <span className={`employee-account-badge ${employee.active ? "is-active" : ""}`}>
            {employee.active ? "Aktif" : "Nonaktif"}
          </span>
        </div>
      </div>
      <dl className="employee-detail-grid">
        {profile.map(([label, value]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
        {phoneNumbers.length > 0 && (
          <div>
            <dt>Nomor telepon</dt>
            {phoneNumbers.map((number, index) => <dd key={index}>{number}</dd>)}
          </div>
        )}
        <div className="span-2">
          <dt>Peran pegawai</dt>
          <dd className="employee-detail-tags">
            {roles.map((role) => (
              <span key={role}>{role}</span>
            ))}
          </dd>
        </div>
        {showAssignments && !!employee.assignedOfficerIds.length && (
          <div className="span-2">
            <dt>Penugasan pejabat</dt>
            <dd>
              <ul className="employee-detail-assignments">
                {employee.assignedOfficerIds.map((id) => (
                  <li key={id}>
                    {officers?.find((officer) => officer.id === id)?.name ||
                      "Pejabat tidak tersedia"}
                  </li>
                ))}
              </ul>
            </dd>
          </div>
        )}
      </dl>
    </div>
  );
}
