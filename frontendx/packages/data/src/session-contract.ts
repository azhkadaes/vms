import { roles, type Session, type PendingSession } from "./domain.ts";
import { ApiError } from "./errors.ts";

export function parseSession(value: unknown): Session | PendingSession {
  const raw = value && typeof value === "object" ? (value as Record<string, unknown>) : {};
  if (raw.active !== true) throw new ApiError(401, "Sesi tidak aktif.");
  const role = roles.find((role) => role === raw.role);
  const pending = raw.role === "" && raw.roleSelectionRequired === true;
  const strings = (input: unknown): input is string[] =>
    Array.isArray(input) && input.every((item) => typeof item === "string" && item.length > 0);
  if (
    (!role && !pending) ||
    typeof raw.id !== "string" ||
    !raw.id ||
    typeof raw.name !== "string" ||
    !raw.name.trim() ||
    !strings(raw.permissions) ||
    !strings(raw.assignedOfficerIds)
  )
    throw new ApiError(502, "Data sesi tidak valid. Coba lagi.");
  const available = raw.availableRoles === undefined && role ? [role] : raw.availableRoles;
  if (
    !Array.isArray(available) ||
    !available.length ||
    available.some((item) => !roles.includes(item)) ||
    new Set(available).size !== available.length ||
    (role && !available.includes(role)) ||
    (raw.roleSelectionRequired !== undefined && raw.roleSelectionRequired !== pending) ||
    (raw.roleRevision !== undefined &&
      (!Number.isSafeInteger(raw.roleRevision) || Number(raw.roleRevision) < 0)) ||
    (pending && (raw.permissions.length > 0 || raw.assignedOfficerIds.length > 0))
  )
    throw new ApiError(502, "Data peran sesi tidak valid.");
  if (raw.mustChangePassword !== undefined && typeof raw.mustChangePassword !== "boolean")
    throw new ApiError(502, "Aturan kata sandi tidak valid.");
  if (
    raw.mustChangePassword &&
    (!Number.isInteger(raw.passwordMinLength) ||
      Number(raw.passwordMinLength) < 8 ||
      Number(raw.passwordMinLength) > 128)
  )
    throw new ApiError(502, "Aturan kata sandi tidak valid.");
  const common = {
    id: raw.id,
    name: raw.name,
    permissions: [...new Set(raw.permissions)],
    assignedOfficerIds: [...new Set(raw.assignedOfficerIds)],
    mustChangePassword: raw.mustChangePassword === true,
    passwordMinLength:
      typeof raw.passwordMinLength === "number" ? raw.passwordMinLength : undefined,
    availableRoles: available as Session["availableRoles"],
    roleRevision: Number(raw.roleRevision || 0),
  };
  return pending
    ? {
        ...common,
        role: "",
        roleSelectionRequired: true,
        availableRoles: available as PendingSession["availableRoles"],
      }
    : { ...common, role: role!, roleSelectionRequired: false };
}
export function sessionKey(session: Session | PendingSession | null) {
  return session
    ? JSON.stringify([
        session.id || session.name,
        session.role,
        session.roleRevision || 0,
        session.availableRoles || [session.role],
        [...session.permissions].sort(),
        [...session.assignedOfficerIds].sort(),
        !!session.mustChangePassword,
      ])
    : null;
}
