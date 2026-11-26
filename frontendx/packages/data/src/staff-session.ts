import { apiBase } from "./config.ts";
import type { PendingSession, Role, Session } from "./domain.ts";
import { ApiError, isTemporaryNetworkError } from "./errors.ts";
import { createTransport, type RequestOptions, type Send } from "./http.ts";
import { withLoginFormRecovery } from "./login-form.ts";
import { parseSession, sessionKey } from "./session-contract.ts";

export type Principal = Session | PendingSession;

type Access = {
  token: string;
  /** On the `now()` clock. */
  usableUntil: number;
  session: Principal;
  authority: string | null;
  csrf: string | undefined;
};

export function readCsrfCookie() {
  if (typeof document === "undefined") return;
  const prefix = "__Host-vms-csrf=";
  return document.cookie
    .split("; ")
    .find((entry) => entry.startsWith(prefix))
    ?.slice(prefix.length);
}

export type StaffSessionOptions = {
  base?: string;
  fetcher?: typeof fetch;
  /** Monotonic milliseconds. */
  now?: () => number;
  csrf?: () => string | undefined;
  /** Serialises cookie rotation across tabs; omit to run unlocked. */
  locks?: Pick<LockManager, "request">;
  newId?: () => string;
};

/**
 * A staff member's access token, refresh cycle and current authority (CONTEXT.md, "Staff
 * session"). It is the only module that knows how staff credentials are obtained, rotated and
 * attached to requests.
 */
export interface StaffSession {
  signIn(credentials: { email: string; password: string }): Promise<Principal>;
  signOut(): Promise<void>;
  /** The server's view of the current session, refreshing access first if there is none. */
  current(): Promise<Principal>;
  selectRole(role: Role): Promise<Session>;
  changePassword(currentPassword: string, newPassword: string): Promise<void>;
  /** Reports real user activity so the server can extend the session. */
  renew(signal?: AbortSignal): Promise<void>;
  /**
   * Sends an authenticated request: refreshes access when it is missing, about to expire or
   * bound to another CSRF cookie; retries once after a rejected bearer; and fails with 409
   * when the refresh revealed a different authority than the request was made under.
   */
  authorized(send: Send, path: string, options: RequestOptions): Promise<unknown>;
  /** Fires when the server confirms the session is gone. */
  onLoss(listener: () => void): () => void;
  /** Fires when a refresh reveals a different role, permissions or account. */
  onAuthorityChange(listener: (session: Principal) => void): () => void;
}

export function createStaffSession({
  base = "/v1",
  fetcher = fetch,
  now = () => performance.now(),
  csrf = readCsrfCookie,
  locks = globalThis.navigator?.locks,
  newId = () => crypto.randomUUID(),
}: StaffSessionOptions = {}): StaffSession {
  const send = createTransport(base, fetcher);
  const lossListeners = new Set<() => void>();
  const authorityListeners = new Set<(session: Principal) => void>();
  let access: Access | undefined;
  let refreshing: Promise<Access> | undefined;
  let refreshRequestId: string | undefined;

  // Refresh, sign-in, sign-out and password changes rotate a cookie shared by all tabs.
  const exclusive = <T>(run: () => Promise<T>): Promise<T> =>
    locks ? (locks.request("vms-staff-session", run) as Promise<T>) : run();

  function adopt(value: unknown, started: number): Access {
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
      csrf: csrf(),
    };
    return access;
  }
  const sameAuthority = (a: Access, b: Access) => a.authority === b.authority && a.csrf === b.csrf;
  function lose() {
    access = undefined;
    lossListeners.forEach((listener) => listener());
  }

  function refresh(): Promise<Access> {
    refreshing ??= exclusive(async () => {
      const previous = access;
      if (!csrf()) {
        if (previous) lose();
        throw new ApiError(401, "Silakan masuk.");
      }
      const body = { refreshRequestId: (refreshRequestId ||= newId()) };
      const attempt = async () => {
        const started = now();
        return adopt(
          await send("/staff/session/refresh", { body, csrf: csrf(), timeoutMs: 4000 }),
          started,
        );
      };
      try {
        let next: Access;
        try {
          next = await attempt();
        } catch (error) {
          // Recover an uncertain rotation using the same ID while the server can recover it.
          if (!isTemporaryNetworkError(error) && !(error instanceof ApiError && error.status >= 500))
            throw error;
          next = await attempt();
        }
        refreshRequestId = undefined;
        if (previous && !sameAuthority(previous, next))
          authorityListeners.forEach((listener) => listener(next.session));
        return next;
      } catch (error) {
        if (error instanceof ApiError && error.status < 500) {
          refreshRequestId = undefined;
          if (error.status === 401) lose();
        }
        throw error;
      }
    }).finally(() => {
      refreshing = undefined;
    });
    return refreshing;
  }

  async function authorized(
    transport: Send,
    path: string,
    options: RequestOptions,
    { allowAuthorityChange = false, exclusiveSend = false } = {},
  ): Promise<unknown> {
    options.signal?.throwIfAborted();
    const previous = access;
    let credential = previous;
    const verifyAuthority = (next: Access) => {
      options.signal?.throwIfAborted();
      if (!allowAuthorityChange && previous && !sameAuthority(previous, next))
        throw new ApiError(
          409,
          "Sesi atau peran telah berubah. Periksa kembali sebelum melanjutkan.",
        );
    };
    const sendWith = (current?: Access) => {
      const run = () => transport(path, { ...options, token: current?.token });
      return exclusiveSend ? exclusive(run) : run();
    };

    if (previous ? previous.usableUntil <= now() + 1000 || previous.csrf !== csrf() : !!csrf()) {
      credential = await refresh();
      verifyAuthority(credential);
    }

    try {
      return await sendWith(credential);
    } catch (error) {
      if (!(error instanceof ApiError) || (error.status !== 401 && error.code !== "role_changed"))
        throw error;
      if (!credential && !csrf()) {
        if (error.status === 401) lose();
        throw error;
      }
      // A rejected bearer does not establish that the server session has expired.
      options.signal?.throwIfAborted();
      const latest = access;
      const next = latest && latest !== credential ? latest : await refresh();
      verifyAuthority(next);
      if (error.code === "role_changed" && !allowAuthorityChange) throw error;
      try {
        return await sendWith(next);
      } catch (retryError) {
        if (retryError instanceof ApiError && retryError.status === 401)
          throw new ApiError(503, "Sesi belum dapat diverifikasi. Coba lagi.");
        throw retryError;
      }
    }
  }

  return {
    async signIn(credentials) {
      try {
        const started = now();
        const value = await withLoginFormRecovery(() =>
          exclusive(() => send("/staff/sessions", { body: credentials, csrf: csrf() })),
        );
        return adopt(value, started).session;
      } catch (error) {
        if (error instanceof ApiError && error.status === 401)
          throw new ApiError(401, "Email atau kata sandi tidak sesuai.");
        throw error;
      }
    },
    async signOut() {
      try {
        await exclusive(() => send("/staff/session", { method: "DELETE", csrf: csrf() }));
      } catch (error) {
        if (!(error instanceof ApiError && error.status === 401)) throw error;
      }
      access = undefined;
    },
    async current() {
      if (!access) return (await refresh()).session;
      return parseSession(
        await authorized(send, "/staff/session", { authenticated: true }, {
          allowAuthorityChange: true,
        }),
      );
    },
    async selectRole(role) {
      const started = now();
      const selected = adopt(
        await authorized(send, "/staff/session/role", { body: { role }, authenticated: true }),
        started,
      ).session;
      if (selected.roleSelectionRequired || selected.role !== role)
        throw new ApiError(502, "Pilihan peran belum dikonfirmasi.");
      return selected as Session;
    },
    async changePassword(currentPassword, newPassword) {
      const started = now();
      adopt(
        await authorized(
          send,
          "/staff/password",
          { body: { currentPassword, newPassword }, authenticated: true },
          { exclusiveSend: true },
        ),
        started,
      );
    },
    async renew(signal) {
      await authorized(send, "/staff/session/renew", { body: {}, authenticated: true, signal });
    },
    authorized: (transport, path, options) => authorized(transport, path, options),
    onLoss(listener) {
      lossListeners.add(listener);
      return () => void lossListeners.delete(listener);
    },
    onAuthorityChange(listener) {
      authorityListeners.add(listener);
      return () => void authorityListeners.delete(listener);
    },
  };
}

export const staffSession = createStaffSession({ base: apiBase });
