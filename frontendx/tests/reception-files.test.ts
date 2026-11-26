import test from "node:test";
import assert from "node:assert/strict";
import { ApiError } from "../packages/data/src/errors.ts";
import { createVisitFiles, type VisitFileKind } from "../packages/data/src/visit-files.ts";

const pdf = () => new Blob(["%PDF-1.7 fixture"], { type: "application/pdf" });
const png = () =>
  new Blob([new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10])], { type: "image/png" });
const valid: Record<VisitFileKind, () => Blob> = { attachment: pdf, "visitor-photo": png };

type Call = { url: string; method?: string; body?: unknown; credentials?: string; marker: string | null };
function serving(blob: () => Blob) {
  const calls: Call[] = [];
  const files = createVisitFiles({
    fetcher: async (url, options) => {
      calls.push({
        url: String(url),
        method: options?.method,
        body: options?.body === undefined ? undefined : JSON.parse(String(options.body)),
        credentials: options?.credentials,
        marker: new Headers(options?.headers).get("X-VMS-Request"),
      });
      const file = blob();
      return new Response(file, { headers: { "Content-Type": file.type } });
    },
  });
  return { files, calls };
}

for (const kind of ["attachment", "visitor-photo"] as const) {
  for (const credential of [undefined, "00123456"]) {
    test(`${kind} ${credential ? "with a reception credential is read by POST" : "is read by GET"}`, async () => {
      const { files, calls } = serving(valid[kind]);
      const blob = await files.open("42", kind, credential ? "Receptionist" : "PA", { credential });
      assert.equal(blob.type, valid[kind]().type);
      assert.deepEqual(calls, [
        {
          url: `/v1/staff/bookings/42/${kind}`,
          method: credential ? "POST" : "GET",
          body: credential ? { credential } : undefined,
          credentials: "include",
          marker: credential ? "browser" : null,
        },
      ]);
    });
  }

  test(`${kind} is rejected when the served bytes are not what the kind allows`, async () => {
    const other = kind === "attachment" ? png : pdf;
    const forged = () => new Blob(["not really"], { type: valid[kind]().type });
    for (const blob of [other, forged]) {
      const { files } = serving(blob);
      await assert.rejects(
        files.open("42", kind, "PA"),
        (error: unknown) => error instanceof ApiError && error.status === 502,
      );
    }
  });
}

test("preview reads go to the preview server with the impersonated role", async () => {
  const urls: string[] = [];
  const files = createVisitFiles({
    development: true,
    fetcher: async (url) => {
      urls.push(String(url));
      return new Response(pdf(), { headers: { "Content-Type": "application/pdf" } });
    },
  });
  await files.open("42", "attachment", "Receptionist", { credential: "00123456" });
  assert.deepEqual(urls, ["/__preview/staff/bookings/42/attachment?role=Receptionist"]);
});
