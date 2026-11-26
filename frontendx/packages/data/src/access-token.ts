import { ApiError } from "./errors.ts";
import { parseSession, sessionKey } from "./session-contract.ts";

type StaffAccess = {
  token: string;
  usableUntil: number;
  session: ReturnType<typeof parseSession>;
  authority: string | null;
  csrf: string | undefined;
};
let access: StaffAccess | undefined;

export function staffAccess() {
  return access;
}

export function adoptStaffAccess(value: unknown, started: number) {
  const result = value as {
    accessToken?: unknown;
    accessExpiresAt?: unknown;
    session?: { timing?: { serverTime?: unknown } };
  } | null;
  const serverTime = result?.session?.timing?.serverTime;
  const expiry =
    typeof result?.accessExpiresAt === "string" ? Date.parse(result.accessExpiresAt) : NaN;
  if (
    typeof result?.accessToken !== "string" ||
    !result.accessToken ||
    typeof serverTime !== "number" ||
    !Number.isFinite(serverTime) ||
    !Number.isFinite(expiry) ||
    expiry <= serverTime
  )
    throw new ApiError(502, "Sesi masuk tidak valid. Coba lagi.");
  const session = parseSession(result.session);
  access = {
    token: result.accessToken,
    usableUntil: started + expiry - serverTime,
    session,
    authority: sessionKey(session),
    csrf: csrfToken(),
  };
  return access;
}

export function sameStaffAuthority(a: StaffAccess, b: StaffAccess) {
  return a.authority === b.authority && a.csrf === b.csrf;
}

export function clearStaffAccess() {
  access = undefined;
}

export function csrfToken() {
  if (typeof document === "undefined") return;
  const prefix = "__Host-vms-csrf=";
  return document.cookie
    .split("; ")
    .find((entry) => entry.startsWith(prefix))
    ?.slice(prefix.length);
}
