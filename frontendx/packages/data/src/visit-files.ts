import { apiBase, preview } from "./config.ts";
import {
  matchesAttachmentSignature,
  validateND,
  validateVisitorPhoto,
  type Role,
} from "./domain.ts";
import { ApiError } from "./errors.ts";
import { createApi, type ApiOptions } from "./repositories.ts";

/** A binary that belongs to a visit (see CONTEXT.md, "Visit file"). */
export type VisitFileKind = "attachment" | "visitor-photo";

const rules = {
  attachment: {
    purpose: "document",
    field: "ndName",
    validate: validateND,
    invalid: "Gunakan berkas PDF.",
  },
  "visitor-photo": {
    purpose: "visitorPhoto",
    field: "visitorPhotoName",
    validate: validateVisitorPhoto,
    invalid: "Gunakan foto JPEG atau PNG.",
  },
} as const;

async function verified(kind: VisitFileKind, file: Blob): Promise<string | null> {
  const rule = rules[kind];
  const issue = rule.validate({ size: file.size, type: file.type });
  if (issue) return issue;
  const head = new Uint8Array(await file.slice(0, 8).arrayBuffer());
  return matchesAttachmentSignature(file.type, head) ? null : rule.invalid;
}

type Uploaded = { token: string; usableUntil: number };

export interface VisitFiles {
  /**
   * Uploads a file for a visit that is about to be submitted and returns its token. A file
   * whose token is still usable is not sent again. Invalid files fail with a 422 whose field
   * error names the form field (`ndName` or `visitorPhotoName`).
   */
  upload(
    file: File | null,
    audience: "public" | "staff",
    kind: VisitFileKind,
  ): Promise<string | undefined>;
  /**
   * Reads a visit file and returns it only after its type, size and signature bytes check out.
   * Receptionists pass the visitor's resolved `credential`; other staff read without one.
   */
  open(
    visitId: string,
    kind: VisitFileKind,
    role: Role,
    options?: { credential?: string; signal?: AbortSignal },
  ): Promise<Blob>;
}

export function createVisitFiles(options: ApiOptions = {}): VisitFiles {
  const { request, staff } = createApi(options);
  const uploads = {
    public: new WeakMap<File, { reference: string; result?: Uploaded }>(),
    staff: new WeakMap<File, { reference: string; result?: Uploaded }>(),
  };
  return {
    async upload(file, audience, kind) {
      if (!file) return;
      const { field, purpose } = rules[kind];
      const issue = await verified(kind, file);
      if (issue) throw new ApiError(422, issue, { [field]: [issue] });
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
      form.set("purpose", purpose);
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
      const raw = result as { token?: unknown; expiresAt?: unknown; serverTime?: unknown } | null;
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
        usableUntil: started + Date.parse(raw.expiresAt) - raw.serverTime,
      };
      return raw.token;
    },
    async open(visitId, kind, role, { credential, signal } = {}) {
      const blob = await request(staff(`/bookings/${encodeURIComponent(visitId)}/${kind}`, role), {
        authenticated: true,
        responseType: "blob",
        signal,
        ...(credential === undefined ? {} : { body: { credential } }),
      });
      if (!(blob instanceof Blob) || (await verified(kind, blob)))
        throw new ApiError(
          502,
          kind === "visitor-photo"
            ? "Foto pengunjung tidak valid atau belum tersedia."
            : "Berkas tidak valid atau belum tersedia.",
        );
      return blob;
    },
  };
}

export const visitFiles = createVisitFiles({ development: preview, base: apiBase });

/** Opens a visit file and hands it to the browser as a download named `name`. */
export async function saveVisitFile(visitId: string, kind: VisitFileKind, role: Role, name: string) {
  const blob = await visitFiles.open(visitId, kind, role);
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name.replace(/[\\/\u0000-\u001f]/g, "_");
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
