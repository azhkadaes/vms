import test from "node:test";
import assert from "node:assert/strict";

test("reception file reads send the resolved credential by POST for both verification files", async (t) => {
  const original = globalThis.fetch;
  t.after(() => {
    globalThis.fetch = original;
  });
  let kind: "attachment" | "visitor-photo" = "attachment";
  globalThis.fetch = async (url, options) => {
    assert.equal(url, `/v1/staff/bookings/42/${kind}`);
    assert.equal(options?.method, "POST");
    assert.deepEqual(JSON.parse(String(options?.body)), { credential: "00123456" });
    assert.equal(options?.credentials, "include");
    assert.equal(new Headers(options?.headers).get("X-VMS-Request"), "browser");
    return new Response(
      kind === "attachment"
        ? "%PDF-1.7 fixture"
        : new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]),
      {
        headers: { "Content-Type": kind === "attachment" ? "application/pdf" : "image/png" },
      },
    );
  };
  const { loadArrivalFile } = await import("../packages/data/src/attachments.ts");
  for (kind of ["attachment", "visitor-photo"] as const)
    assert.ok(
      (await loadArrivalFile(
        "42",
        "Receptionist",
        "00123456",
        kind,
        new AbortController().signal,
      )) instanceof Blob,
    );
});
