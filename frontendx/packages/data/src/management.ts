import { apiBase, preview } from "./config.ts";
import { roles, sessionFor } from "./domain.ts";
import { ApiError } from "./errors.ts";
import { createHttpClient } from "./http.ts";
import type {
  Employee,
  EmployeeFilter,
  EmployeeInput,
  Integration,
  IntegrationInput,
  RoleDefinition,
} from "./management-contract.ts";

export interface ManagementRepository {
  employees(
    filter: EmployeeFilter,
    signal?: AbortSignal,
  ): Promise<{
    items: Employee[];
    total: number;
    totalAll: number;
    page: number;
    pageSize: number;
    passwordMinLength: number;
  }>;
  employee(id: string): Promise<Employee>;
  saveEmployee(id: string | null, input: EmployeeInput, version?: number): Promise<Employee>;
  roles(): Promise<RoleDefinition[]>;
  saveRole(record: RoleDefinition, permissions: string[]): Promise<RoleDefinition>;
  integrations(): Promise<Integration[]>;
  saveIntegration(record: Integration, input: IntegrationInput): Promise<Integration>;
}
export function createManagementRepository(
  base = "/v1",
  fetcher: typeof fetch = fetch,
): ManagementRepository {
  const request = createHttpClient(base, fetcher);
  async function parse<T>(input: unknown, schema: { parse: (input: unknown) => T }) {
    try {
      return schema.parse(input);
    } catch {
      throw new ApiError(502, "Data administrasi tidak valid. Coba lagi.");
    }
  }
  return {
    employees: async (filter, signal) => {
      const { employeeListSchema } = await import("./management-contract.ts");
      return parse(
        await request(
          `/staff/employees?${new URLSearchParams({ q: filter.q, page: String(filter.page), pageSize: String(filter.pageSize) })}`,
          { authenticated: true, signal },
        ),
        employeeListSchema,
      );
    },
    employee: async (id) => {
      const { employeeSchema } = await import("./management-contract.ts");
      return parse(
        await request(`/staff/employees/${encodeURIComponent(id)}`, { authenticated: true }),
        employeeSchema,
      );
    },
    saveEmployee: async (id, input, version) => {
      const { employeeSchema } = await import("./management-contract.ts");
      return parse(
        await request(`/staff/employees${id ? "/" + encodeURIComponent(id) : ""}`, {
          body: { ...input, version },
          method: id ? "PATCH" : "POST",
          authenticated: true,
        }),
        employeeSchema,
      );
    },
    roles: async () => {
      const { roleDefinitionSchema } = await import("./management-contract.ts");
      return parse(
        await request("/staff/roles", { authenticated: true }),
        roleDefinitionSchema.array(),
      );
    },
    saveRole: async (record, permissions) => {
      const { roleDefinitionSchema } = await import("./management-contract.ts");
      return parse(
        await request(`/staff/roles/${encodeURIComponent(record.id)}`, {
          body: { permissions, version: record.version },
          method: "PATCH",
          authenticated: true,
        }),
        roleDefinitionSchema,
      );
    },
    integrations: async () => {
      const { integrationSchema } = await import("./management-contract.ts");
      return parse(
        await request("/staff/integrations", { authenticated: true }),
        integrationSchema.array(),
      );
    },
    saveIntegration: async (record, input) => {
      const { integrationSchema } = await import("./management-contract.ts");
      return parse(
        await request(`/staff/integrations/${encodeURIComponent(record.id)}`, {
          body: input,
          method: "PATCH",
          authenticated: true,
        }),
        integrationSchema,
      );
    },
  };
}

function createPreviewManagement(): ManagementRepository {
  let employees: Employee[] = [
    {
      id: "employee-1",
      name: "Aulia Rahma",
      email: "aulia@example.test",
      role: "PA",
      active: true,
      assignedOfficerIds: ["o2"],
      version: 1,
    },
    {
      id: "employee-2",
      name: "Nadia Putri",
      email: "nadia@example.test",
      role: "Protocol Officer",
      active: true,
      assignedOfficerIds: [],
      version: 1,
    },
    {
      id: "employee-3",
      name: "Dimas Pratama",
      email: "dimas@example.test",
      role: "Receptionist",
      active: true,
      assignedOfficerIds: [],
      version: 1,
    },
  ];
  let definitions: RoleDefinition[] = roles.map((name) => ({
    id: name,
    name,
    description: "Kewenangan mengikuti peran dan penugasan yang sah.",
    permissions: sessionFor(name).permissions,
    assignablePermissions: sessionFor(name).permissions,
    editable: true,
    version: 1,
  }));
  let integrations: Integration[] = [
    {
      id: "email",
      name: "Email",
      description: "Pemberitahuan kunjungan melalui layanan email backend.",
      enabled: false,
      health: "not_configured",
      version: 1,
      fields: [
        { key: "endpoint", label: "Alamat layanan", kind: "url", required: true, value: "" },
        { key: "sender", label: "Email pengirim", kind: "email", required: true, value: "" },
        {
          key: "credential",
          label: "Kredensial layanan",
          kind: "secret",
          required: true,
          configured: false,
        },
      ],
    },
    {
      id: "storage",
      name: "Penyimpanan dokumen",
      description: "Penyimpanan berkas terlindungi dikelola backend.",
      enabled: false,
      health: "not_configured",
      version: 1,
      fields: [
        { key: "endpoint", label: "Alamat layanan", kind: "url", required: true, value: "" },
        { key: "bucket", label: "Lokasi penyimpanan", kind: "text", required: true, value: "" },
        {
          key: "credential",
          label: "Kredensial layanan",
          kind: "secret",
          required: true,
          configured: false,
        },
      ],
    },
  ];
  function conflict(actual: number, expected?: number) {
    if (actual !== expected)
      throw new ApiError(409, "Data telah berubah. Muat ulang sebelum melanjutkan.");
  }
  return {
    async employees(filter) {
      const needle = filter.q.trim().toLowerCase();
      const filtered = employees.filter((record) =>
        [record.name, record.email, ...(record.roles || [record.role || ""])].some((value) =>
          value.toLowerCase().includes(needle),
        ),
      );
      return {
        items: filtered.slice((filter.page - 1) * filter.pageSize, filter.page * filter.pageSize),
        total: filtered.length,
        totalAll: employees.length,
        page: filter.page,
        pageSize: filter.pageSize,
        passwordMinLength: 12,
      };
    },
    async employee(id) {
      const record = employees.find((record) => record.id === id);
      if (!record) throw new ApiError(404, "Pegawai tidak ditemukan.");
      return structuredClone(record);
    },
    async saveEmployee(id, input, version) {
      const current = employees.find((record) => record.id === id);
      if (id && !current) throw new ApiError(404, "Pegawai tidak ditemukan.");
      if (current) conflict(current.version, version);
      const record: Employee = {
        id: id || crypto.randomUUID(),
        name: input.name.trim(),
        email: input.email.trim(),
        role: input.role,
        roles: input.roles || (input.role ? [input.role] : []),
        jobTitle: input.jobTitle,
        deputy: input.deputy,
        directorate: input.directorate,
        phone: input.phone,
        nip: input.nip,
        assignedOfficerIds: (input.roles || [input.role]).includes("PA")
          ? input.assignedOfficerIds
          : [],
        active: input.active,
        version: (current?.version || 0) + 1,
      };
      employees = [record, ...employees.filter((value) => value.id !== id)];
      return record;
    },
    async roles() {
      return structuredClone(definitions);
    },
    async saveRole(record, permissions) {
      const current = definitions.find((value) => value.id === record.id)!;
      conflict(current.version, record.version);
      const updated = {
        ...current,
        permissions: permissions.filter((value) => current.assignablePermissions.includes(value)),
        version: current.version + 1,
      };
      definitions = definitions.map((value) => (value.id === record.id ? updated : value));
      return structuredClone(updated);
    },
    async integrations() {
      return structuredClone(integrations);
    },
    async saveIntegration(record, input) {
      const current = integrations.find((value) => value.id === record.id)!;
      conflict(current.version, input.version);
      const updated: Integration = {
        ...current,
        enabled: input.enabled,
        version: current.version + 1,
        fields: current.fields.map((field) =>
          field.kind === "secret"
            ? {
                ...field,
                configured: input.clearSecrets.includes(field.key)
                  ? false
                  : !!input.values[field.key] || field.configured,
              }
            : { ...field, value: input.values[field.key] ?? field.value },
        ),
      };
      integrations = integrations.map((value) => (value.id === record.id ? updated : value));
      return structuredClone(updated);
    },
  };
}
export const management: ManagementRepository = preview
  ? createPreviewManagement()
  : createManagementRepository(apiBase);
