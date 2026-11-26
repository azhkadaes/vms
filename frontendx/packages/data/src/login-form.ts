import { ApiError } from "./errors.ts";

export async function withLoginFormRecovery<T>(
  login: () => Promise<T>,
): Promise<T> {
  try {
    return await login();
  } catch (error) {
    if (!(
      error instanceof ApiError &&
      error.status === 403 &&
      error.code === "csrf_expired"
    ))
      throw error;
  }

  // The staff document gateway renews the signed anonymous form cookies.
  const response = await fetch("/admin/login", {
    credentials: "same-origin",
    cache: "no-store",
    headers: { Accept: "text/html" },
    signal: AbortSignal.timeout(15_000),
  });
  if (!response.ok)
    throw new ApiError(
      response.status,
      "Formulir masuk belum dapat diperbarui. Coba lagi.",
    );
  await response.text();
  return login();
}
