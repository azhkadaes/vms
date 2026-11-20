let accessToken: string | undefined;

export function staffAccessToken() {
  return accessToken;
}

export function setStaffAccessToken(value?: string) {
  accessToken = value;
}

export function csrfToken() {
  if (typeof document === "undefined") return;
  const prefix = "__Host-vms-csrf=";
  return document.cookie
    .split("; ")
    .find((entry) => entry.startsWith(prefix))
    ?.slice(prefix.length);
}
