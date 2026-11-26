import {
  adoptStaffAccess,
  clearStaffAccess,
  csrfToken,
  sameStaffAuthority,
  staffAccess,
} from "./access-token.ts";
import { apiBase } from "./config.ts";
import { ApiError, isTemporaryNetworkError } from "./errors.ts";

const authenticationLoss = new Set<() => void>();
type StaffAccess = NonNullable<ReturnType<typeof staffAccess>>;
const authorityChanges = new Set<(session: StaffAccess["session"]) => void>();
export function onAuthenticationLoss(listener: () => void) {
  authenticationLoss.add(listener);
  return () => {
    authenticationLoss.delete(listener);
  };
}

export function onStaffAuthorityChange(listener: (session: StaffAccess["session"]) => void) {
  authorityChanges.add(listener);
  return () => {
    authorityChanges.delete(listener);
  };
}

function loseStaffAccess() {
  clearStaffAccess();
  authenticationLoss.forEach((listener) => listener());
}

async function withStaffCookieLock<T>(run: () => Promise<T>): Promise<T> {
  // Refresh and password changes rotate a cookie shared by all tabs on this origin.
  const locks = globalThis.navigator?.locks;
  return locks ? await locks.request("vms-staff-session", run) : run();
}

let refreshInFlight: Promise<StaffAccess> | undefined;
let refreshRequestId: string | undefined;

export function refreshStaffAccess() {
  if (!refreshInFlight) {
    refreshInFlight = withStaffCookieLock(async () => {
      const previous = staffAccess();
      if (!csrfToken()) {
        if (previous) loseStaffAccess();
        throw new ApiError(401, "Silakan masuk.");
      }
      const request = createHttpClient(apiBase);
      const body = { refreshRequestId: (refreshRequestId ||= crypto.randomUUID()) };
      const refresh = async () => {
        const started = performance.now();
        const value = await request("/staff/session/refresh", { body });
        return adoptStaffAccess(value, started);
      };
      try {
        let next: StaffAccess;
        try {
          next = await refresh();
        } catch (error) {
          // Recover an uncertain rotation using the same ID while the server can recover it.
          if (
            !isTemporaryNetworkError(error) &&
            !(error instanceof ApiError && error.status >= 500)
          )
            throw error;
          next = await refresh();
        }
        refreshRequestId = undefined;
        if (previous && !sameStaffAuthority(previous, next))
          authorityChanges.forEach((listener) => listener(next.session));
        return next;
      } catch (error) {
        if (error instanceof ApiError && error.status < 500) {
          refreshRequestId = undefined;
          if (error.status === 401) loseStaffAccess();
        }
        throw error;
      }
    }).finally(() => {
      refreshInFlight = undefined;
    });
  }
  return refreshInFlight;
}

type RequestOptions = {
  body?: unknown;
  signal?: AbortSignal;
  method?: string;
  authenticated?: boolean;
  responseType?: "blob";
};
function fieldErrors(value: unknown): Record<string, string[]> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) return;
  const entries = Object.entries(value).filter(
    (entry): entry is [string, string[]] =>
      Array.isArray(entry[1]) && entry[1].every((item) => typeof item === "string"),
  );
  return Object.fromEntries(entries);
}

export function createHttpClient(base = "/v1", fetcher: typeof fetch = fetch) {
  async function send(
    path: string,
    {
      body,
      signal,
      method = body === undefined ? "GET" : "POST",
      responseType,
    }: RequestOptions = {},
    token?: string,
  ): Promise<unknown> {
    const controller = new AbortController();
    const abort = () => controller.abort(signal?.reason);
    if (signal?.aborted) abort();
    else signal?.addEventListener("abort", abort, { once: true });
    const timeout = setTimeout(
      () =>
        controller.abort(
          new DOMException("Permintaan terlalu lama. Periksa koneksi Anda.", "TimeoutError"),
        ),
      path === "/staff/session/refresh" ? 4000 : 15000,
    );
    try {
      const lifecycle =
        path === "/staff/sessions" ||
        path === "/staff/session/refresh" ||
        (path === "/staff/session" && method === "DELETE");
      const csrf = lifecycle ? csrfToken() : undefined;
      const response = await fetcher(base + path, {
        method,
        credentials: "include",
        signal: controller.signal,
        headers: {
          ...(body === undefined || body instanceof FormData
            ? {}
            : { "Content-Type": "application/json" }),
          ...(["GET", "HEAD"].includes(method) ? {} : { "X-VMS-Request": "browser" }),
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
          ...(csrf ? { "X-CSRF-Token": csrf } : {}),
        },
        body:
          body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body),
      });
      if (response.status === 204 && response.ok) return null;
      if (response.ok && responseType === "blob") return await response.blob();
      const value: unknown = await response.json().catch(() => null);
      if (!response.ok) {
        const error = value && typeof value === "object" ? (value as Record<string, unknown>) : {};
        const fallback =
          response.status === 401
            ? "Sesi berakhir. Silakan masuk kembali."
            : response.status === 403
              ? "Akses tidak diizinkan."
              : response.status === 429
                ? "Terlalu banyak permintaan. Tunggu sebelum mencoba lagi."
                : "Layanan belum tersedia. Coba lagi.";
        const failure = new ApiError(
          response.status,
          typeof error.message === "string" ? error.message : fallback,
          fieldErrors(error.fieldErrors),
        );
        failure.code = typeof error.code === "string" ? error.code : undefined;
        const retryAfter = Number(response.headers.get("Retry-After"));
        if (retryAfter > 0) failure.retryAfter = retryAfter;
        throw failure;
      }
      if (value === null) throw new ApiError(502, "Respons layanan tidak valid. Coba lagi.");
      return value;
    } catch (error) {
      if (controller.signal.aborted) throw controller.signal.reason;
      if (error instanceof TypeError)
        throw new TypeError("Tidak dapat terhubung. Periksa koneksi Anda.", { cause: error });
      throw error;
    } finally {
      clearTimeout(timeout);
      signal?.removeEventListener("abort", abort);
    }
  }

  return async function request(path: string, options: RequestOptions = {}) {
    if (!options.authenticated) {
      const changesCookies =
        base === apiBase &&
        (path === "/staff/sessions" || (path === "/staff/session" && options.method === "DELETE"));
      return changesCookies ? withStaffCookieLock(() => send(path, options)) : send(path, options);
    }
    options.signal?.throwIfAborted();

    const protectedStaff = base === apiBase && path.startsWith("/staff/");
    const currentSession = path === "/staff/session";
    const previous = protectedStaff ? staffAccess() : undefined;
    let credential = previous;
    const verifyAuthority = (next: StaffAccess) => {
      options.signal?.throwIfAborted();
      if (!currentSession && previous && !sameStaffAuthority(previous, next))
        throw new ApiError(
          409,
          "Sesi atau peran telah berubah. Periksa kembali sebelum melanjutkan.",
        );
    };
    const sendProtected = (access?: StaffAccess) =>
      protectedStaff && path === "/staff/password"
        ? withStaffCookieLock(() => send(path, options, access?.token))
        : send(path, options, access?.token);

    if (
      protectedStaff &&
      (previous
        ? previous.usableUntil <= performance.now() + 1000 || previous.csrf !== csrfToken()
        : !!csrfToken())
    ) {
      credential = await refreshStaffAccess();
      verifyAuthority(credential);
    }

    try {
      return await sendProtected(credential);
    } catch (error) {
      if (!(error instanceof ApiError) || (error.status !== 401 && error.code !== "role_changed"))
        throw error;
      if (!protectedStaff || (!credential && !csrfToken())) {
        if (error.status === 401) loseStaffAccess();
        throw error;
      }

      // A rejected bearer does not establish that the server session has expired.
      options.signal?.throwIfAborted();
      const latest = staffAccess();
      const next = latest && latest !== credential ? latest : await refreshStaffAccess();
      verifyAuthority(next);
      if (error.code === "role_changed" && !currentSession) throw error;
      try {
        return await sendProtected(next);
      } catch (retryError) {
        if (retryError instanceof ApiError && retryError.status === 401)
          throw new ApiError(503, "Sesi belum dapat diverifikasi. Coba lagi.");
        throw retryError;
      }
    }
  };
}
