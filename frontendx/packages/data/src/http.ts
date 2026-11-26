import { ApiError } from "./errors.ts";

export type RequestOptions = {
  body?: unknown;
  signal?: AbortSignal;
  method?: string;
  /** Sends the request through the client's staff session, if it has one. */
  authenticated?: boolean;
  responseType?: "blob";
};
/** What a session may add to a request on its way out. */
export type TransportOptions = RequestOptions & {
  token?: string;
  csrf?: string;
  timeoutMs?: number;
};
export type Send = (path: string, options?: TransportOptions) => Promise<unknown>;
/** Attaches staff credentials to authenticated requests (see staff-session.ts). */
export interface Authorizer {
  authorized(send: Send, path: string, options: RequestOptions): Promise<unknown>;
}

function fieldErrors(value: unknown): Record<string, string[]> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) return;
  const entries = Object.entries(value).filter(
    (entry): entry is [string, string[]] =>
      Array.isArray(entry[1]) && entry[1].every((item) => typeof item === "string"),
  );
  return Object.fromEntries(entries);
}

/** A session-agnostic JSON/blob transport with timeouts and Indonesian error copy. */
export function createTransport(base = "/v1", fetcher: typeof fetch = fetch): Send {
  return async function send(
    path,
    {
      body,
      signal,
      method = body === undefined ? "GET" : "POST",
      responseType,
      token,
      csrf,
      timeoutMs = 15000,
    } = {},
  ) {
    const controller = new AbortController();
    const abort = () => controller.abort(signal?.reason);
    if (signal?.aborted) abort();
    else signal?.addEventListener("abort", abort, { once: true });
    const timeout = setTimeout(
      () =>
        controller.abort(
          new DOMException("Permintaan terlalu lama. Periksa koneksi Anda.", "TimeoutError"),
        ),
      timeoutMs,
    );
    try {
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
  };
}

/**
 * A request function. Authenticated requests go through `session`; a client without one (the
 * preview adapter, public callers) sends every request as-is.
 */
export function createHttpClient(
  base = "/v1",
  fetcher: typeof fetch = fetch,
  session?: Authorizer,
) {
  const send = createTransport(base, fetcher);
  return function request(path: string, options: RequestOptions = {}) {
    return options.authenticated && session
      ? session.authorized(send, path, options)
      : send(path, options);
  };
}
