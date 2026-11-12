import { ApiError } from "./errors.ts";

const authenticationLoss = new Set<() => void>();
export function onAuthenticationLoss(listener: () => void) {
  authenticationLoss.add(listener);
  return () => {
    authenticationLoss.delete(listener);
  };
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
  return async function request(
    path: string,
    {
      body,
      signal,
      method = body === undefined ? "GET" : "POST",
      authenticated = false,
      responseType,
    }: RequestOptions = {},
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
      15000,
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
        },
        body:
          body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body),
      });
      if (authenticated && response.status === 401)
        authenticationLoss.forEach((listener) => listener());
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
