import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { preview } from "./config.ts";
import { roles, sessionFor, type Role, type Session, type PendingSession } from "./domain.ts";
import { ApiError } from "./errors.ts";
import { sessionKey } from "./session-contract.ts";
import { staffSession } from "./staff-session.ts";
import { canSwitchRole } from "./unsaved.ts";

const staffSessionQueryKey = ["session", "current"];
const activityEvents = ["pointerdown", "keydown", "input", "wheel"];
function isPendingSession(value: Session | PendingSession): value is PendingSession {
  return value.role === "";
}
function isActiveSession(value: Session | PendingSession): value is Session {
  return value.role !== "";
}
export async function changePassword(currentPassword: string, newPassword: string) {
  if (preview) throw new ApiError(400, "Penggantian kata sandi tidak tersedia pada pratinjau.");
  await staffSession.changePassword(currentPassword, newPassword);
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
  await staffSession.signIn(credentials);
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
  const key = preview ? ["session", previewRole] : staffSessionQueryKey;
  const query = useQuery({
    queryKey: key,
    queryFn: async () => (preview ? sessionFor(previewRole) : staffSession.current()),
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
      staffSession.onLoss(() => {
        setExpired(true);
        void client.cancelQueries({ predicate: (query) => query.queryKey[0] !== "session" });
        client.removeQueries({ predicate: (query) => query.queryKey[0] !== "session" });
      }),
    [client],
  );
  useEffect(
    () =>
      staffSession.onAuthorityChange((next) => {
        client.setQueryData(staffSessionQueryKey, next);
      }),
    [client],
  );
  useEffect(() => {
    if (preview || !fingerprint) return;
    const controller = new AbortController();
    let lastRenewal = -Infinity;
    function reportActivity(event: Event) {
      const now = performance.now();
      if (!event.isTrusted || now - lastRenewal < 60_000) return;
      lastRenewal = now;
      void staffSession.renew(controller.signal).catch(() => {
        // Activity checks are retried on a later interaction. Confirmed expiry is handled by HTTP.
      });
    }
    for (const type of activityEvents)
      window.addEventListener(type, reportActivity, { capture: true, passive: true });
    return () => {
      controller.abort();
      for (const type of activityEvents) window.removeEventListener(type, reportActivity, true);
    };
  }, [fingerprint]);
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
        await staffSession.selectRole(next);
      }
      await client.cancelQueries();
      client.clear();
      location.assign("/admin/dashboard");
    } finally {
      switchLock.current = false;
    }
  }
  async function signOut() {
    if (!preview) await staffSession.signOut();
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
