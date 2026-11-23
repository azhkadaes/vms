import test from "node:test";
import assert from "node:assert/strict";
import { ApiError } from "../packages/data/src/errors.ts";
import { withLoginFormRecovery } from "../packages/data/src/login-form.ts";

function expired() {
  const error = new ApiError(403, "Formulir sudah kedaluwarsa.");
  error.code = "csrf_expired";
  return error;
}

test("expired login initializes signed form cookies before one retry", async (t) => {
  const calls: string[] = [];
  const original = globalThis.fetch;
  t.after(() => {
    globalThis.fetch = original;
  });
  globalThis.fetch = async (url, options) => {
    assert.equal(url, "/admin/login");
    assert.equal(options?.credentials, "same-origin");
    assert.equal(options?.cache, "no-store");
    calls.push("initialize");
    return new Response("staff document");
  };
  const result = await withLoginFormRecovery(async () => {
    calls.push("login");
    if (calls.length === 1) throw expired();
    return "signed in";
  });
  assert.equal(result, "signed in");
  assert.deepEqual(calls, ["login", "initialize", "login"]);
});

test("login never loops or retries credentials, forbidden and rate-limit failures", async (t) => {
  const original = globalThis.fetch;
  t.after(() => {
    globalThis.fetch = original;
  });
  let initializations = 0;
  globalThis.fetch = async () => {
    initializations++;
    return new Response("staff document");
  };
  for (const status of [401, 403, 429]) {
    const error = new ApiError(status, "Rejected");
    let attempts = 0;
    await assert.rejects(
      withLoginFormRecovery(async () => {
        attempts++;
        throw error;
      }),
      error,
    );
    assert.equal(attempts, 1);
  }
  assert.equal(initializations, 0);
  let attempts = 0;
  await assert.rejects(
    withLoginFormRecovery(async () => {
      attempts++;
      throw expired();
    }),
    { code: "csrf_expired" },
  );
  assert.equal(attempts, 2);
  assert.equal(initializations, 1);
});

test("unavailable form initialization does not submit credentials again", async (t) => {
  const original = globalThis.fetch;
  t.after(() => {
    globalThis.fetch = original;
  });
  globalThis.fetch = async () => new Response("Unavailable", { status: 502 });
  let attempts = 0;
  await assert.rejects(
    withLoginFormRecovery(async () => {
      attempts++;
      throw expired();
    }),
    { status: 502 },
  );
  assert.equal(attempts, 1);
});
