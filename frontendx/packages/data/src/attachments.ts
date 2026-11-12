import { apiBase, preview } from "./config.ts";
import { validateND } from "./domain.ts";
import { ApiError } from "./errors.ts";
import { createHttpClient } from "./http.ts";

const request = createHttpClient(apiBase);
type Uploaded = { token: string; expiresAt: string; serverTime: number; usableUntil: number };
const uploads = {
  public: new WeakMap<File, { reference: string; result?: Uploaded }>(),
  staff: new WeakMap<File, { reference: string; result?: Uploaded }>(),
};
export async function uploadAttachment(
  file: File | null,
  audience: "public" | "staff",
): Promise<string | undefined> {
  if (!file) return;
  const invalid = validateND(file);
  if (invalid) throw new ApiError(422, invalid, { ndName: [invalid] });
  if (preview) return;
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
  const started = performance.now();
  const result = await request(`/${audience}/attachments`, {
    body: form,
    authenticated: audience === "staff",
  });
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
  if (!(blob instanceof Blob) || validateND({ type: blob.type, size: blob.size, name }))
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
