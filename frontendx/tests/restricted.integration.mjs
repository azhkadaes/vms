import test, { before, after, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";
import { sessionFor } from "../packages/data/src/domain.ts";
import { startPreview } from "./helpers/preview-server.mjs";

let api,
  base,
  assigned = ["o2", "o3"];
before(async () => {
  api = await startPreview({
    session: (role) => ({
      ...sessionFor(role),
      ...(role === "PA" ? { assignedOfficerIds: assigned } : {}),
    }),
  });
  base = api.base;
});
beforeEach(() => {
  api.setNow("2099-08-19T09:00:00+08:00");
  assigned = ["o2", "o3"];
});
after(async () => api?.close());
const protocol = encodeURIComponent("Protocol Officer");
const input = () => ({
  visitKind: "restricted",
  date: "2099-08-19",
  time: "10:00",
  idempotencyKey: randomUUID(),
});
async function post(path, body) {
  const r = await fetch(base + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return { status: r.status, data: await r.json() };
}
async function read(path) {
  const r = await fetch(base + path);
  return { status: r.status, data: await r.json() };
}
async function list(role) {
  return (await read(`/bookings?role=${encodeURIComponent(role)}`)).data;
}
async function create(extra = {}) {
  const result = await post(`/staff/bookings?role=${protocol}`, { ...input(), ...extra });
  assert.equal(result.status, 201, JSON.stringify(result.data));
  return result.data;
}

test("minimal Terbatas creation, payload-bound retry, edits and read-only role projections", async () => {
  const draft = input();
  for (const role of ["PA", "Receptionist", "Super Admin"])
    assert.equal(
      (await post(`/staff/bookings?role=${encodeURIComponent(role)}`, draft)).status,
      403,
    );
  assert.equal((await post("/bookings", draft)).status, 403);
  const b = await create(draft);
  assert.equal(b.status, "approved");
  for (const key of ["name", "organization", "serviceId", "officerId", "email", "visitors"])
    assert.equal(key in b, false, key);
  assert.equal((await post(`/staff/bookings?role=${protocol}`, draft)).data.id, b.id);
  assert.equal(
    (await post(`/staff/bookings?role=${protocol}`, { ...draft, time: "11:00" })).status,
    409,
  );
  assert.equal(
    (await list("Receptionist")).some((r) => r.id === b.id),
    true,
  );
  assert.equal(
    (await list("PA")).some((r) => r.id === b.id),
    false,
  );
  const edit = {
    visitKind: "restricted",
    date: b.date,
    time: b.time,
    version: 1,
    name: "PRIVATE_NAME",
    organization: "PRIVATE_ORG",
    serviceId: "0012345678",
    officerId: "o2",
  };
  const edited = await post(`/bookings/${b.id}/update?role=${protocol}`, edit);
  assert.equal(edited.status, 200);
  assert.equal(edited.data.serviceId, "0012345678");
  assert.equal((await post(`/bookings/${b.id}/update?role=${protocol}`, edit)).status, 409);
  for (const role of ["PA", "Super Admin"]) {
    const detail = await read(`/bookings/${b.id}?role=${encodeURIComponent(role)}`);
    assert.equal(detail.status, 200);
    assert.equal(detail.data.name, edit.name);
    assert.equal(detail.data.organization, edit.organization);
    assert.equal(detail.data.serviceId, role === "Super Admin" ? edit.serviceId : undefined);
    assert.equal(detail.data.serviceIdLast4, role === "PA" ? "5678" : undefined);
    for (const action of ["approve", "reject", "reschedule", "checkin", "checkout", "update"])
      assert.equal(
        (await post(`/bookings/${b.id}/${action}?role=${encodeURIComponent(role)}`, { version: 2 }))
          .status,
        403,
        action,
      );
  }
  assert.equal((await post("/lookup", { code: b.code, email: "" })).status, 401);
  assert.equal(
    (await post(`/bookings/${b.id}/cancel`, { code: b.code, email: "", version: 2 })).status,
    404,
  );
  assigned = [];
  const hidden = await read(`/bookings/${b.id}?role=PA`);
  const unknown = await read("/bookings/missing?role=PA");
  assert.equal(hidden.status, 404);
  assert.deepEqual(hidden.data, unknown.data);
  assert.equal(
    (await list("PA")).some((r) => r.id === b.id),
    false,
  );
  for (const action of ["approve", "reject", "reschedule", "checkin", "checkout", "update"])
    assert.equal((await post(`/bookings/${b.id}/${action}?role=PA`, { version: 2 })).status, 404);
  assert.equal((await post("/resolve?role=PA", { credential: b.code })).status, 404);
});

test("reception list redacts identity and only code/QR lookup yields arrival details", async () => {
  const b = await create({
    name: "RECEPTION_SECRET_NAME",
    organization: "RECEPTION_SECRET_ORG",
    serviceId: "0099887766",
  });
  const record = (await list("Receptionist")).find((r) => r.id === b.id);
  assert.equal((await read(`/bookings/${b.id}?role=Receptionist`)).status, 404);
  assert.equal(record.reference, b.reference);
  for (const key of [
    "code",
    "name",
    "organization",
    "serviceId",
    "serviceIdLast4",
    "host",
    "officerId",
    "history",
    "createdBy",
  ])
    assert.equal(key in record, false, key);
  const unknown = await post("/resolve?role=Receptionist", { credential: "00000000" });
  assert.deepEqual(
    (await post("/resolve?role=Receptionist", { credential: b.reference })).data,
    unknown.data,
  );
  assert.equal((await post("/resolve?role=Receptionist", { credential: b.id })).status, 404);
  for (const credential of [b.code, `VMS-DEMO:${b.code}`]) {
    const resolved = await post("/resolve?role=Receptionist", { credential });
    assert.equal(resolved.status, 200);
    assert.equal(resolved.data.name, b.name);
    assert.equal(resolved.data.serviceIdLast4, "7766");
    for (const key of ["code", "organization", "serviceId", "history", "email", "phone"])
      assert.equal(key in resolved.data, false, key);
  }
  api.setNow("2099-08-19T09:30:00+08:00");
  assert.equal(
    (await post(`/bookings/${b.id}/checkin?role=Receptionist`, { version: b.version })).status,
    404,
  );
  const checkedIn = await post(`/bookings/${b.id}/checkin?role=Receptionist`, {
    version: b.version,
    credential: b.code,
  });
  assert.equal(checkedIn.status, 200);
  assert.equal(checkedIn.data.status, "checked_in");
  assert.equal("name" in checkedIn.data, false);
  api.setNow("2099-08-20T15:00:00+08:00");
  const checkedOut = await post(`/bookings/${b.id}/checkout?role=Receptionist`, {
    version: checkedIn.data.version,
    credential: `VMS-DEMO:${b.code}`,
  });
  assert.equal(checkedOut.status, 200);
  assert.equal(checkedOut.data.status, "checked_out");
});

test("Terbatas edits clear optional values, immediately revoke former PA access and reject legacy fields", async () => {
  const b = await create({
    name: "Clear Me",
    time: "11:00",
    organization: "Clear Org",
    serviceId: "00004444",
    officerId: "o2",
  });
  const update = {
    visitKind: "restricted",
    date: b.date,
    time: b.time,
    version: 1,
    name: "",
    organization: "",
    serviceId: "",
    officerId: "",
  };
  const cleared = await post(`/bookings/${b.id}/update?role=${protocol}`, update);
  assert.equal(cleared.status, 200);
  for (const key of ["name", "organization", "serviceId", "officerId"])
    assert.equal(key in cleared.data, false, key);
  assert.equal((await read(`/bookings/${b.id}?role=PA`)).status, 404);
  for (const extra of [
    { email: "test@example.test" },
    { phone: "08123456789" },
    { purpose: "Old field" },
    { ndName: "test.pdf" },
    { identityType: "KTP" },
    { visitors: ["Old roster"] },
    { date: "2099-02-30" },
    { time: "29:00" },
    { serviceId: "x".repeat(101) },
  ])
    assert.equal(
      (await post(`/staff/bookings?role=${protocol}`, { ...input(), ...extra })).status,
      422,
      JSON.stringify(extra),
    );
  assert.equal(
    (await post(`/staff/bookings?role=${protocol}`, { ...input(), officerId: "missing" })).status,
    403,
  );
  assert.equal(
    (
      await post(`/bookings/${b.id}/update?role=${protocol}`, {
        ...update,
        version: 2,
        email: "old@example.test",
      })
    ).status,
    422,
  );
  assert.equal(
    (await post(`/staff/bookings?role=${protocol}`, { ...input(), visitKind: "standard" })).status,
    422,
  );
});

for (const clock of ["09:29:59.999", "09:30:00.000", "11:00:00.000", "11:00:00.001"]) {
  test(`preview enforces arrival cutoff ${clock} on every code/QR request`, async () => {
    const b = await create();
    api.setNow(`2099-08-19T${clock}+08:00`);
    const allowed = clock === "09:30:00.000" || clock === "11:00:00.000";
    for (const credential of [b.code, `VMS-DEMO:${b.code}`]) {
      const resolved = await post("/resolve?role=Receptionist", { credential });
      assert.equal(resolved.data.timing.canCheckIn, allowed);
    }
    const response = await post(`/bookings/${b.id}/checkin?role=Receptionist`, {
      version: b.version,
      credential: b.code,
    });
    assert.equal(response.status, allowed ? 200 : 409);
    const stored = (await read(`/bookings/${b.id}?role=${protocol}`)).data;
    assert.equal(stored.status, allowed ? "checked_in" : "approved");
    assert.equal(stored.version, allowed ? b.version + 1 : b.version);
    if (clock === "11:00:00.001") {
      assert.equal(stored.timing.displayStatus, "missed");
      const reschedule = () =>
        post(`/bookings/${b.id}/reschedule?role=${protocol}`, {
          version: b.version,
          date: "2099-08-27",
          time: "10:00",
        });
      api.setNow("2099-08-26T11:00:00.001+08:00");
      assert.equal((await reschedule()).status, 403);
      api.setNow("2099-08-26T11:00:00.000+08:00");
      const updated = await reschedule();
      assert.equal(updated.status, 200);
      assert.equal(updated.data.date, "2099-08-27");
      assert.equal(updated.data.timing.displayStatus, "approved");
    }
  });
}
