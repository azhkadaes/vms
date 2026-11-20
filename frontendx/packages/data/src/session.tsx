import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { apiBase, preview } from "./config.ts";
import { roles, sessionFor, type Role, type Session, type PendingSession } from "./domain.ts";
import { ApiError } from "./errors.ts";
import { createHttpClient, onAuthenticationLoss } from "./http.ts";
import { parseSession, sessionKey } from "./session-contract.ts";
import { canSwitchRole } from "./unsaved.ts";
import { csrfToken, setStaffAccessToken, staffAccessToken } from "./access-token.ts";

const request = createHttpClient(apiBase);
function adoptSession(value: unknown) {
  const result = value as { accessToken?: unknown; session?: unknown } | null;
  if (typeof result?.accessToken !== "string" || !result.accessToken)
    throw new ApiError(502, "Sesi masuk tidak valid. Coba lagi.");
  const session = parseSession(result.session);
  setStaffAccessToken(result.accessToken);
  return session;
}

let refreshRequestId: string | undefined;
let refreshInFlight: Promise<Session | PendingSession> | undefined;
async function readSession() {
  if (staffAccessToken())
    return parseSession(await request("/staff/session", { authenticated: true }));
  if (!csrfToken()) throw new ApiError(401, "Silakan masuk.");
  if (!refreshInFlight) {
    refreshRequestId ||= crypto.randomUUID();
    refreshInFlight = request("/staff/session/refresh", { body: { refreshRequestId } })
      .then((value) => {
        const session = adoptSession(value);
        refreshRequestId = undefined;
        return session;
      })
      .catch((error: unknown) => {
        if (error instanceof ApiError && error.status < 500) refreshRequestId = undefined;
        throw error;
      })
      .finally(() => {
        refreshInFlight = undefined;
      });
  }
  return refreshInFlight;
}
function isPendingSession(value: Session | PendingSession): value is PendingSession {
  return value.role === "";
}
function isActiveSession(value: Session | PendingSession): value is Session {
  return value.role !== "";
}
export async function changePassword(currentPassword: string, newPassword: string) {
  if (preview) throw new ApiError(400, "Penggantian kata sandi tidak tersedia pada pratinjau.");
  adoptSession(
    await request("/staff/password", {
      body: { currentPassword, newPassword },
      authenticated: true,
    }),
  );
}
const SessionContext = createContext<{
  session: Session | null;
  pendingSession: PendingSession | null;
  loading: boolean;
  error: Error | null;
  role: Role | null;
  setRole: (role: Role) => void;
  switchRole: (role: Role) => Promise<void>;
  signOut: () => Promise<void>;
  refreshSession: () => Promise<unknown>;
}>({
  session: null,
  pendingSession: null,
  loading: true,
  error: null,
  role: null,
  setRole() {},
  async switchRole() {},
  async signOut() {},
  async refreshSession() {},
});

export async function signIn(credentials: { email: string; password: string }) {
  if (preview) return;
  try {
    adoptSession(await request("/staff/sessions", { body: credentials }));
  } catch (error) {
    if (error instanceof ApiError && error.status === 401)
      throw new ApiError(401, "Email atau kata sandi tidak sesuai.");
    throw error;
  }
}
function initialPreviewRole(): Role {
  try {
    return roles.find((role) => role === sessionStorage.getItem("vms-preview-role")) || "PA";
  } catch {
    return "PA";
  }
}
export function SessionProvider({ children }: { children: ReactNode }) {
  const [previewRole, setPreviewRole] = useState(initialPreviewRole);
  const [expired, setExpired] = useState(false);
  const client = useQueryClient();
  const key = ["session", preview ? previewRole : "current"];
  const query = useQuery({
    queryKey: key,
    queryFn: async () => (preview ? sessionFor(previewRole) : readSession()),
    enabled: !expired,
    refetchOnWindowFocus: "always",
    refetchOnReconnect: "always",
    retry: (count, error) => count < 1 && !(error instanceof ApiError && error.status < 500),
  });
  const denied =
    expired || (query.error instanceof ApiError && [401, 403].includes(query.error.status));
  const principal = denied ? null : query.data || null;
  const pendingSession = principal && isPendingSession(principal) ? principal : null;
  const session = principal && isActiveSession(principal) ? principal : null;
  const fingerprint = sessionKey(principal);
  const previousKey = useRef<string | null>(null);
  useEffect(() => {
    if (previousKey.current && previousKey.current !== fingerprint) {
      const old = previousKey.current;
      void client.cancelQueries({ predicate: (query) => query.queryKey[1] === old });
      client.removeQueries({ predicate: (query) => query.queryKey[1] === old });
    }
    previousKey.current = fingerprint;
  }, [client, fingerprint]);
  useEffect(
    () =>
      onAuthenticationLoss(() => {
        setExpired(true);
        void client.cancelQueries({ predicate: (query) => query.queryKey[0] !== "session" });
        client.removeQueries({ predicate: (query) => query.queryKey[0] !== "session" });
      }),
    [client],
  );
  function setRole(role: Role) {
    if (!preview) return;
    client.clear();
    try {
      sessionStorage.setItem("vms-preview-role", role);
    } catch {
      /* Role fixtures can still work without storage. */
    }
    setPreviewRole(role);
  }
  const switchLock = useRef(false);
  async function switchRole(next: Role) {
    if (switchLock.current || next === session?.role) return;
    if (!canSwitchRole()) return;
    switchLock.current = true;
    try {
      if (preview) {
        setRole(next);
      } else {
        const selected = adoptSession(
          await request("/staff/session/role", {
            body: { role: next },
            authenticated: true,
          }),
        );
        if (selected.roleSelectionRequired || selected.role !== next)
          throw new ApiError(502, "Pilihan peran belum dikonfirmasi.");
      }
      await client.cancelQueries();
      client.clear();
      location.assign("/admin/dashboard");
    } finally {
      switchLock.current = false;
    }
  }
  async function signOut() {
    if (!preview) {
      try {
        await request("/staff/session", { method: "DELETE" });
      } catch (error) {
        if (!(error instanceof ApiError && error.status === 401)) throw error;
      }
    }
    setStaffAccessToken();
    client.clear();
    location.assign("/admin/login");
  }
  return (
    <SessionContext.Provider
      value={{
        session,
        pendingSession,
        role: session?.role || null,
        loading: query.isPending && !expired,
        error: expired ? new ApiError(401, "Sesi berakhir. Silakan masuk kembali.") : query.error,
        setRole,
        switchRole,
        signOut,
        refreshSession: () => query.refetch(),
      }}
    >
      {children}
    </SessionContext.Provider>
  );
}
export function useSession() {
  return useContext(SessionContext);
}
