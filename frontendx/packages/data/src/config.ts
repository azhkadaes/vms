export const preview =
  import.meta.env?.DEV === true &&
  typeof location !== "undefined" &&
  ["localhost", "127.0.0.1", "[::1]"].includes(location.hostname);

const configuredBase = import.meta.env?.VITE_API_BASE_PATH || "/v1";
if (!/^\/(?!\/)[a-zA-Z0-9/_-]+$/.test(configuredBase))
  throw new Error("VITE_API_BASE_PATH harus berupa path pada origin yang sama.");
export const apiBase = configuredBase.replace(/\/$/, "");
