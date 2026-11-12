import test, { before, after, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";
import { request } from "node:http";
import { officers } from "../packages/data/src/domain.ts";
import { startPreview } from "./helpers/preview-server.mjs";

let api;
before(async () => {
  api = await startPreview({ time: "2099-08-19T09:00:00+08:00" });
});
beforeEach(() => api.setNow("2099-08-19T09:00:00+08:00"));
after(async () => api?.close());
const input = () => ({
  idempotencyKey: randomUUID(),
  officerId: "o2",
  date: "2099-08-19",
  time: "10:00",
  purpose: "Public validation fixture",
  name: `Validation ${randomUUID().slice(0, 8)}`,
  organization: "Example Agency",
  email: "validation@example.test",
  phone: "00123456789",
  identityType: "KTP",
  ack: true,
});
function draft() {
  const value = input();
  return { ...value, visitors: [value.name] };
}
async function post(path, body) {
  const response = await fetch(api.base + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return { status: response.status, data: await response.json() };
}
async function list() {
  return fetch(api.base + "/bookings?role=PA").then((r) => r.json());
}

test("public creation rejects missing/invalid fields with field errors and creates no partial record", async () => {
  const before = (await list()).length;
  const invalid = [
    ["officerId", ""],
    ["officerId", "missing"],
    ["date", "2099-02-30"],
    ["date", "2099-08-18"],
    ["time", "09:00"],
    ["time", "16:01"],
    ["name", "x"],
    ["name", "x".repeat(121)],
    ["organization", "x"],
    ["purpose", "x"],
    ["purpose", "x".repeat(1001)],
    ["email", "invalid"],
    ["phone", "+62123456789"],
    ["phone", "123"],
    ["identityType", "other"],
    ["visitors", []],
    ["visitors", ["Wrong Representative"]],
    ["visitors", ["", "x"]],
    ["ack", false],
    ["ack", undefined],
    ["ndName", "x".repeat(256)],
    ["ndName", ""],
  ];
  for (const [field, value] of invalid) {
    const result = await post("/bookings", { ...draft(), [field]: value });
    assert.equal(result.status, 422, `${field}: ${JSON.stringify(result.data)}`);
    assert.ok(result.data.fieldErrors[field]?.length, field);
  }
  const tooMany = draft();
  tooMany.visitors = Array(21).fill(tooMany.name);
  assert.equal((await post("/bookings", tooMany)).status, 422);
  const active = officers[1].active;
  try {
    officers[1].active = false;
    assert.equal((await post("/bookings", draft())).data.fieldErrors.officerId.length, 1);
  } finally {
    officers[1].active = active;
  }
  for (const idempotencyKey of [undefined, "", {}, "x".repeat(129)])
    assert.equal((await post("/bookings", { ...draft(), idempotencyKey })).status, 422);
  assert.equal((await list()).length, before);
});
test("public input normalization binds replays, derives server fields and rejects changed contents", async () => {
  const value = draft();
  const first = await post("/bookings", {
    ...value,
    name: ` ${value.name} `,
    visitors: [` ${value.name} `],
    email: " VALIDATION@EXAMPLE.TEST ",
    status: "approved",
    vip: true,
    code: "00000000",
    internalRejectionNote: "SECRET",
    createdBy: { role: "Super Admin" },
    history: [{ message: "FORGED" }],
  });
  assert.equal(first.status, 201);
  assert.equal(first.data.status, "pending");
  assert.equal(first.data.vip, false);
  assert.notEqual(first.data.code, "00000000");
  assert.equal(first.data.phone, "00123456789");
  for (const secret of ["SECRET", "FORGED", "Super Admin"])
    assert.equal(JSON.stringify(first.data).includes(secret), false);
  const replay = await post("/bookings", { ...value, visitKind: "standard" });
  assert.equal(replay.status, 201);
  assert.equal(replay.data.id, first.data.id);
  assert.equal(replay.data.history.length, 1);
  for (const changed of [
    { purpose: "Changed content" },
    { email: "other@example.test" },
    { ack: false },
  ])
    assert.equal((await post("/bookings", { ...value, ...changed })).status, 409);
  api.setNow("2099-08-20T15:00:00+08:00");
  const historical = await post("/bookings", value);
  assert.equal(historical.status, 201);
  assert.equal(historical.data.id, first.data.id);
  assert.equal((await list()).filter((b) => b.id === first.data.id).length, 1);
});
test("concurrent duplicate POSTs commit one booking and one logical submission event", async () => {
  const value = draft();
  const responses = await Promise.all(Array.from({ length: 8 }, () => post("/bookings", value)));
  assert.ok(responses.every((response) => response.status === 201));
  assert.equal(new Set(responses.map((response) => response.data.id)).size, 1);
  const records = (await list()).filter((b) => b.name === value.name);
  assert.equal(records.length, 1);
  assert.equal(records[0].history.length, 1);
});
test("recovery requires the original email and returns only the visitor projection", async () => {
  const value = draft();
  const created = await post("/bookings", value);
  const recovery = { idempotencyKey: value.idempotencyKey, email: "wrong@example.test" };
  assert.equal((await post("/submissions/resolve", recovery)).status, 401);
  assert.equal((await post("/submissions/resolve", { ...recovery, email: "invalid" })).status, 422);
  const resolved = await post("/submissions/resolve", {
    ...recovery,
    email: " VALIDATION@EXAMPLE.TEST ",
  });
  assert.equal(resolved.status, 200);
  assert.equal(resolved.data.outcome, "committed");
  assert.equal(resolved.data.booking.id, created.data.id);
  for (const key of ["createdBy", "internalRejectionNote", "idempotencyKey", "ack"])
    assert.equal(key in resolved.data.booking, false, key);
  assert.equal((await post("/bookings", value)).data.id, created.data.id);
});
test("resolving an absent reference seals it against late original creation", async () => {
  const value = draft();
  const before = (await list()).length;
  const result = await post("/submissions/resolve", {
    idempotencyKey: value.idempotencyKey,
    email: value.email,
  });
  assert.deepEqual(result.data, { outcome: "not_committed" });
  assert.equal((await post("/bookings", value)).status, 409);
  assert.equal((await list()).length, before);
  assert.equal((await post("/bookings", { ...value, idempotencyKey: randomUUID() })).status, 201);
});
test("an original POST still receiving its body cannot commit after conclusive missing recovery", async () => {
  const value = draft();
  const body = JSON.stringify(value);
  const url = new URL(api.base + "/bookings");
  let finish;
  const response = new Promise((resolve, reject) => {
    const req = request(
      url,
      {
        method: "POST",
        headers: { "Content-Type": "application/json", "Content-Length": Buffer.byteLength(body) },
      },
      (res) => {
        let text = "";
        res.on("data", (chunk) => (text += chunk));
        res.on("end", () => resolve({ status: res.statusCode, data: JSON.parse(text) }));
      },
    );
    req.on("error", reject);
    req.write(body.slice(0, -1));
    finish = () => req.end(body.slice(-1));
  });
  assert.equal(
    (
      await post("/submissions/resolve", {
        idempotencyKey: value.idempotencyKey,
        email: value.email,
      })
    ).data.outcome,
    "not_committed",
  );
  finish();
  assert.equal((await response).status, 409);
  assert.equal(
    (await list()).some((b) => b.name === value.name),
    false,
  );
});
