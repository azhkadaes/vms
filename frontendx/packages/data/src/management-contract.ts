import { z } from "zod";
import { roles } from "./domain.ts";

const role = z.enum(roles);
export const employeeSchema = z
  .object({
    id: z.string().min(1),
    name: z.string().min(2),
    email: z.string().email(),
    role: role.optional(),
    roles: z.array(role).min(1).optional(),
    jobTitle: z.string().optional(),
    deputy: z.string().optional(),
    directorate: z.string().optional(),
    phone: z.string().optional(),
    phoneNumbers: z.array(z.string()).optional(),
    nip: z.string().optional(),
    active: z.boolean(),
    assignedOfficerIds: z.array(z.string()),
    version: z.number().int().positive(),
  })
  .refine((value) => !!value.roles?.length || !!value.role, "Peran pegawai wajib diisi.");
export type Employee = z.infer<typeof employeeSchema>;
export function employeePhoneNumbers(employee?: Employee | null): string[] {
  return employee?.phoneNumbers ?? (employee?.phone ? [employee.phone] : []);
}
export const employeeListSchema = z.object({
  items: z.array(employeeSchema).max(250),
  total: z.number().int().nonnegative(),
  totalAll: z.number().int().nonnegative(),
  page: z.number().int().positive(),
  pageSize: z.number().int().min(1).max(250),
  passwordMinLength: z.number().int().min(8).max(128),
});
export function employeeInputSchema(minimum: number, editing: boolean) {
  return z
    .object({
      name: z.string().trim().min(2, "Masukkan nama lengkap.").max(255),
      email: z.string().trim().email("Periksa email pegawai."),
      role: role.optional(),
      roles: z.array(role).min(1, "Pilih minimal satu peran.").optional(),
      jobTitle: z.string().trim().max(255).optional(),
      deputy: z.string().trim().max(255).optional(),
      directorate: z.string().trim().max(255).optional(),
      phone: z.string().trim().max(255).optional(),
      phoneNumbers: z
        .array(z.string().trim().max(255, "Maksimal 255 karakter."))
        .transform((numbers) => numbers.filter(Boolean))
        .optional(),
      nip: z.string().trim().max(255).optional(),
      active: z.boolean(),
      assignedOfficerIds: z
        .array(z.string())
        .max(1, "Pegawai hanya dapat ditugaskan ke satu pejabat."),
      initialPassword: editing
        ? z.string().optional()
        : z.string().min(minimum, `Kata sandi minimal ${minimum} karakter.`).max(256),
      requirePasswordChange: editing ? z.boolean().optional() : z.boolean(),
    })
    .refine((value) => !!value.roles?.length || !!value.role, {
      path: ["roles"],
      message: "Pilih minimal satu peran.",
    });
}
export type EmployeeInput = Omit<Employee, "id" | "version"> & {
  initialPassword?: string;
  requirePasswordChange?: boolean;
};
export const roleDefinitionSchema = z.object({
  id: z.string().min(1),
  name: role,
  description: z.string(),
  permissions: z.array(z.string()),
  assignablePermissions: z.array(z.string()),
  editable: z.boolean(),
  version: z.number().int().positive(),
});
export type RoleDefinition = z.infer<typeof roleDefinitionSchema>;
export const integrationFieldSchema = z
  .object({
    key: z.string().regex(/^[a-zA-Z][a-zA-Z0-9_]*$/),
    label: z.string().min(1),
    kind: z.enum(["text", "secret", "url", "email", "number"]),
    required: z.boolean(),
    value: z.string().optional(),
    configured: z.boolean().optional(),
  })
  .transform((field) => (field.kind === "secret" ? { ...field, value: undefined } : field));
export const integrationSchema = z.object({
  id: z.string().min(1),
  name: z.string(),
  description: z.string(),
  enabled: z.boolean(),
  health: z.enum(["not_configured", "ready", "degraded"]),
  version: z.number().int().positive(),
  fields: z.array(integrationFieldSchema).max(30),
});
export type Integration = z.infer<typeof integrationSchema>;
export type IntegrationInput = {
  enabled: boolean;
  values: Record<string, string>;
  clearSecrets: string[];
  version: number;
};
export type EmployeeFilter = { q: string; page: number; pageSize: number };
