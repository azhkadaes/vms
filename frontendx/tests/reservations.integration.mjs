import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";

import { startPreview } from "./helpers/preview-server.mjs";
import { sessionFor } from "../packages/data/src/domain.ts";
let api, base;
before(async () => {
  api = await startPreview({
    time: "2098-01-01T08:00:00+08:00",
    // Exercise conflicts across two resources with an explicitly authorized test actor.
    session: (role) => ({ ...sessionFor(role), assignedOfficerIds: role === "PA" ? ["o2", "o3"] : [] }),
  });
  base = api.base;
});
after(async () => api?.close());
const protocol = encodeURIComponent("Protocol Officer");
test("default PA sees only the assigned officer in lists and direct detail access", async () => {
  const scopedApi = await startPreview();
  try {
    const send = async (path, input) => fetch(scopedApi.base + path, {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    const ids = [];
    for (const officerId of ["o2", "o3"]) {
      const response = await send("/bookings", {
        officerId, date: "2099-08-20", time: "09:00", purpose: "PA scope check",
        name: "Scope QA", organization: "QA Example", email: "scope@example.test",
        phone: "081234567890", identityType: "KTP", visitors: ["Scope QA"],
        ack: true, idempotencyKey: randomUUID(),
      });
      assert.equal(response.status, 201);
      ids.push((await response.json()).id);
    }
    const response = await fetch(scopedApi.base + "/bookings?role=PA&officerId=o3");
    const items = await response.json();
    assert.ok(items.length > 0);
    assert.ok(items.every((item) => item.officerId === "o2"));
    assert.ok(items.some((item) => item.id === ids[0]));
    assert.equal((await fetch(scopedApi.base + `/bookings/${ids[1]}?role=PA`)).status, 404);
  } finally {
    await scopedApi.close();
  }
});
async function post(path, input) {
  const response = await fetch(base + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return { status: response.status, data: await response.json() };
}
async function create(date, time, officerId = "o2") {
  const name = `Reservation QA ${randomUUID().slice(0, 8)}`;
  const result = await post("/bookings", {
    ack: true,
    idempotencyKey: randomUUID(),
    officerId,
    date,
    time,
    purpose: "Synthetic resource reservation test",
    name,
    organization: "QA Example",
    email: "reservations@example.test",
    phone: "081234567890",
    identityType: "KTP",
    visitors: [name],
  });
  assert.equal(result.status, 201, JSON.stringify(result.data));
  assert.equal(result.data.status, "pending");
  return result.data;
}
function approve(booking, roomId = "room-1") {
  return post(`/bookings/${booking.id}/approve?role=PA`, {
    version: booking.version,
    roomId,
    duration: 60,
  });
}
test("PA creation saves the chosen room and reserves the full duration", async () => {
  const input = {
    officerId: "o2",
    date: "2098-08-15",
    time: "09:00",
    duration: 120,
    roomId: "room-1",
    purpose: "Verify custom visit duration",
    name: "Duration QA",
    organization: "QA Example",
    email: "duration@example.test",
    phone: "081234567890",
    identityType: "KTP",
    visitors: ["Duration QA"],
  };
  const save = (changes = {}) => post("/staff/bookings?role=PA", {
    ...input, ...changes, idempotencyKey: randomUUID(),
  });
  const created = await save();
  assert.equal(created.status, 201, JSON.stringify(created.data));
  assert.equal(created.data.duration, 120);
  const saved = await read(created.data.id);
  assert.equal(saved.duration, 120);
  assert.equal(saved.roomId, input.roomId);
  assert.equal(saved.room, "Ruang Rapat Nusantara");
  const invalidRoom = await save({ roomId: "room-unknown" });
  assert.equal(invalidRoom.status, 422);
  assert.ok(invalidRoom.data.fieldErrors.roomId);
  const protocolBooking = (time) => post(`/staff/bookings?role=${protocol}`, {
    visitKind: "restricted",
    date: input.date,
    time,
    roomId: input.roomId,
    officerId: "o2",
    idempotencyKey: randomUUID(),
  });
  assert.equal((await protocolBooking("10:30")).status, 409);
  const restricted = await protocolBooking("12:00");
  assert.equal(restricted.status, 201);
  assert.equal((await read(restricted.data.id, "Protocol Officer")).room, "Ruang Rapat Nusantara");
  assert.equal((await save({ time: "10:30", duration: 30 })).status, 409);
  assert.equal((await save({ time: "11:00", duration: 30 })).status, 201);
  for (const duration of [14, 241, 30.5]) {
    const invalid = await save({ duration });
    assert.equal(invalid.status, 422);
    assert.ok(invalid.data.fieldErrors.duration);
  }
});
async function read(id, role = "PA") {
  const response = await fetch(`${base}/bookings/${id}?role=${encodeURIComponent(role)}`);
  assert.equal(response.status, 200);
  return response.json();
}

test("assigned PA can edit all approved visit fields and internal notes without reapproval", async () => {
  const pending = await create("2098-08-16", "09:00");
  const approved = await approve(pending);
  assert.equal(approved.status, 200);
  const changes = {
    date: "2098-08-17",
    time: "11:00",
    duration: 90,
    roomId: "room-2",
    visitorInstructions: "Datang melalui lobi utama.",
    coordinationNote: "Siapkan ruang untuk koordinasi internal.",
    reason: "Penyesuaian agenda pejabat tujuan.",
  };
  const saved = await post(`/bookings/${pending.id}/reschedule?role=PA`, {
    version: approved.data.version,
    ...changes,
  });
  assert.equal(saved.status, 200, JSON.stringify(saved.data));
  const persisted = await read(pending.id);
  assert.equal(persisted.status, "approved");
  assert.equal(persisted.version, approved.data.version + 1);
  for (const key of ["date", "time", "duration", "roomId", "visitorInstructions", "coordinationNote"])
    assert.equal(persisted[key], changes[key]);
  const reception = await fetch(`${base}/bookings/${pending.id}?role=Receptionist`);
  assert.equal(reception.status, 404);
});

test("Pending requests coexist; approval reserves only matching officer or room and leaves conflicts unchanged", async () => {
  const date = "2098-06-11";
  const first = await create(date, "09:00");
  const sameOfficer = await create(date, "09:00");
  const differentOfficer = await create(date, "09:00", "o3");
  assert.equal((await approve(first)).status, 200);
  assert.equal((await approve(sameOfficer, "room-2")).status, 409);
  assert.equal((await approve(differentOfficer)).status, 409);
  assert.equal((await read(sameOfficer.id)).status, "pending");
  assert.equal((await read(sameOfficer.id)).version, 1);
  assert.equal((await read(differentOfficer.id)).version, 1);
  assert.equal((await approve(differentOfficer, "room-2")).status, 200);

  const insideExistingCommitment = await create(date, "09:15");
  assert.equal((await approve(insideExistingCommitment, "room-3")).status, 409);
  const adjacent = await create(date, "10:00");
  assert.equal((await approve(adjacent)).status, 200);

  const longer = await post(`/bookings/${first.id}/reschedule?role=PA`, {
    version: 2,
    date,
    time: "09:00",
    duration: 90,
    roomId: "room-1",
    reason: "Synthetic duration extension",
  });
  assert.equal(longer.status, 409);
  const unchanged = await read(first.id);
  assert.equal(unchanged.version, 2);
  assert.equal(unchanged.duration, 60);
  const earlier = await post(`/bookings/${first.id}/reschedule?role=PA`, {
    version: 2,
    date,
    time: "08:00",
    duration: 60,
    roomId: "room-1",
    reason: "Move to an available adjacent interval",
  });
  assert.equal(earlier.status, 200);
  assert.equal(earlier.data.status, "approved");
  assert.equal((await approve(sameOfficer, "room-3")).status, 200);
});

test("competing approvals cannot both reserve the same resource", async () => {
  const first = await create("2098-06-12", "10:00");
  const second = await create("2098-06-12", "10:00");
  const results = await Promise.all([approve(first), approve(second, "room-2")]);
  assert.deepEqual(results.map((result) => result.status).sort(), [200, 409]);
  const records = await Promise.all([read(first.id), read(second.id)]);
  assert.equal(records.filter((record) => record.status === "approved").length, 1);
  assert.equal(
    records.filter((record) => record.status === "pending" && record.version === 1).length,
    1,
  );
});

test("approved restricted-host changes use the same policy and a blank room creates no phantom reservation", async () => {
  const date = "2098-06-13";
  const occupied = await create(date, "10:00");
  assert.equal((await approve(occupied)).status, 200);
  const draft = { visitKind: "restricted", date, time: "10:00" };
  const first = await post(`/staff/bookings?role=${protocol}`, {
    ...draft,
    officerId: "o1",
    idempotencyKey: randomUUID(),
  });
  const second = await post(`/staff/bookings?role=${protocol}`, {
    ...draft,
    officerId: "o3",
    idempotencyKey: randomUUID(),
  });
  assert.equal(first.status, 201);
  assert.equal(second.status, 201);
  for (const record of [first.data, second.data]) {
    assert.equal(record.status, "approved");
  }
  const conflict = await post(`/bookings/${first.data.id}/update?role=${protocol}`, {
    ...draft,
    officerId: "o2",
    version: first.data.version,
  });
  assert.equal(conflict.status, 409);
  const unchanged = await read(first.data.id, "Protocol Officer");
  assert.equal(unchanged.officerId, "o1");
  assert.equal(unchanged.version, 1);
  // Production requires an officer on every restricted visit (ADR-0001).
  const blank = await post(`/bookings/${first.data.id}/update?role=${protocol}`, {
    ...draft,
    officerId: "",
    version: 1,
  });
  assert.equal(blank.status, 422);
  assert.ok(blank.data.fieldErrors.officerId);
  const sameOfficer = await create(date, "10:15");
  api.setNow("2098-06-13T09:45:00+08:00");
  const checkedIn = await post(`/bookings/${occupied.id}/attendance?role=Receptionist`, { action: "check_in",
    version: 2,
    credential: occupied.code,
  });
  assert.equal(checkedIn.status, 200);
  assert.equal((await approve(sameOfficer, "room-3")).status, 409);
});
