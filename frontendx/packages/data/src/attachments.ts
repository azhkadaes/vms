import { apiBase, preview } from "./config.ts";
import {
  matchesAttachmentSignature,
  validateND,
  validateVisitorPhoto,
  type Role,
} from "./domain.ts";
import { ApiError } from "./errors.ts";
import { createHttpClient } from "./http.ts";

const request = createHttpClient(apiBase);
const previewUploads = new WeakMap<File, string>();
type Uploaded = { token: string; expiresAt: string; serverTime: number; usableUntil: number };
const uploads = {
  public: new WeakMap<File, { reference: string; result?: Uploaded }>(),
  staff: new WeakMap<File, { reference: string; result?: Uploaded }>(),
};
export async function uploadAttachment(
  file: File | null,
  audience: "public" | "staff",
  kind: "document" | "visitorPhoto" = "document",
): Promise<string | undefined> {
  if (!file) return;
  const field = kind === "visitorPhoto" ? "visitorPhotoName" : "ndName";
  const invalid = kind === "visitorPhoto" ? validateVisitorPhoto(file) : validateND(file);
  if (invalid) throw new ApiError(422, invalid, { [field]: [invalid] });
  if (
    !matchesAttachmentSignature(file.type, new Uint8Array(await file.slice(0, 8).arrayBuffer()))
  ) {
    const message = kind === "visitorPhoto" ? "Gunakan foto JPEG atau PNG." : "Gunakan berkas PDF.";
    throw new ApiError(422, message, { [field]: [message] });
  }
  if (preview) {
    const existing = previewUploads.get(file);
    if (existing) return existing;
    const path = kind === "visitorPhoto" ? "/visitor-photos" : "/attachments";
    const response = await fetch(`/__preview${path}`, {
      method: "POST",
      headers: { "Content-Type": file.type },
      body: file,
    });
    if (!response.ok)
      throw new ApiError(response.status, "Berkas belum dapat diunggah. Coba lagi.");
    const result = await response.json();
    if (typeof result.token !== "string") throw new ApiError(502, "Unggahan berkas tidak valid.");
    previewUploads.set(file, result.token);
    return result.token;
  }
  const entries = uploads[audience];
  let entry = entries.get(file);
  if (entry?.result && entry.result.usableUntil > performance.now() + 1000)
    return entry.result.token;
  if (!entry || entry.result) {
    entry = { reference: crypto.randomUUID() };
    entries.set(file, entry);
  }
  const form = new FormData();
  form.set("file", file);
  form.set("reference", entry.reference);
  form.set("purpose", kind);
  const started = performance.now();
  let result: unknown;
  try {
    result = await request(`/${audience}/attachments`, {
      body: form,
      authenticated: audience === "staff",
    });
  } catch (error) {
    if (error instanceof ApiError && error.status === 422)
      error.fieldErrors = {
        [field]: error.fieldErrors?.[field] || error.fieldErrors?.ndName || [error.message],
      };
    throw error;
  }
  const raw = result as Partial<Uploaded> | null;
  if (
    !raw ||
    typeof raw.token !== "string" ||
    raw.token.length < 16 ||
    raw.token.length > 4096 ||
    typeof raw.expiresAt !== "string" ||
    typeof raw.serverTime !== "number" ||
    !(Date.parse(raw.expiresAt) - raw.serverTime > performance.now() - started)
  )
    throw new ApiError(502, "Dokumen belum dapat dikonfirmasi. Coba kirim kembali.");
  entry.result = {
    token: raw.token,
    expiresAt: raw.expiresAt,
    serverTime: raw.serverTime,
    usableUntil: started + Date.parse(raw.expiresAt) - raw.serverTime,
  };
  return raw.token;
}
export async function downloadAttachment(id: string, name: string) {
  const blob = await request(`/staff/bookings/${encodeURIComponent(id)}/attachment`, {
    authenticated: true,
    responseType: "blob",
  });
  if (
    !(blob instanceof Blob) ||
    (validateND({ type: blob.type, size: blob.size, name }) &&
      validateVisitorPhoto({ type: blob.type, size: blob.size, name }))
  )
    throw new ApiError(502, "Berkas tidak valid atau belum tersedia.");
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name.replace(/[\\/\u0000-\u001f]/g, "_");
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export async function loadVisitorPhoto(
  id: string,
  role: string,
  signal?: AbortSignal,
): Promise<Blob> {
  const path = `/bookings/${encodeURIComponent(id)}/visitor-photo`;
  const blob = preview
    ? await createHttpClient("/__preview")(`${path}?role=${encodeURIComponent(role)}`, {
        authenticated: true,
        responseType: "blob",
        signal,
      })
    : await request(`/staff${path}`, { authenticated: true, responseType: "blob", signal });
  if (!(blob instanceof Blob) || validateVisitorPhoto({ type: blob.type, size: blob.size }))
    throw new ApiError(502, "Foto pengunjung tidak valid atau belum tersedia.");
  return blob;
}

export async function loadArrivalFile(
  id: string,
  role: Role,
  credential: string,
  kind: "attachment" | "visitor-photo",
  signal: AbortSignal,
): Promise<Blob> {
  const path = `/bookings/${encodeURIComponent(id)}/${kind}`;
  const options = { authenticated: true, responseType: "blob", signal } as const;
  const blob = preview
    ? await createHttpClient("/__preview")(`${path}?role=${encodeURIComponent(role)}`, {
        ...options,
        body: { credential },
      })
    : await request(`/staff${path}`, options);
  const validate = kind === "attachment" ? validateND : validateVisitorPhoto;
  if (
    !(blob instanceof Blob) ||
    validate({ size: blob.size, type: blob.type }) ||
    !matchesAttachmentSignature(blob.type, new Uint8Array(await blob.slice(0, 8).arrayBuffer()))
  )
    throw new ApiError(502, "Berkas tidak valid atau belum tersedia.");
  return blob;
}
