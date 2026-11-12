import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";

import { startPreview } from "./helpers/preview-server.mjs";
let api, base;
before(async () => {
  api = await startPreview({ time: "2098-01-01T08:00:00+08:00" });
  base = api.base;
});
after(async () => api?.close());
const protocol = encodeURIComponent("Protocol Officer");
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
function approve(booking, room = "Ruang Rapat Nusantara") {
  return post(`/bookings/${booking.id}/approve?role=PA`, {
    version: booking.version,
    room,
    duration: 60,
  });
}
async function read(id, role = "PA") {
  const response = await fetch(`${base}/bookings/${id}?role=${encodeURIComponent(role)}`);
  assert.equal(response.status, 200);
  return response.json();
}

test("Pending requests coexist; approval reserves only matching officer or room and leaves conflicts unchanged", async () => {
  const date = "2098-06-11";
  const first = await create(date, "09:00");
  const sameOfficer = await create(date, "09:00");
  const differentOfficer = await create(date, "09:00", "o3");
  assert.equal((await approve(first)).status, 200);
  assert.equal((await approve(sameOfficer, "Ruang Rapat Meranti")).status, 409);
  assert.equal((await approve(differentOfficer)).status, 409);
  assert.equal((await read(sameOfficer.id)).status, "pending");
  assert.equal((await read(sameOfficer.id)).version, 1);
  assert.equal((await read(differentOfficer.id)).version, 1);
  assert.equal((await approve(differentOfficer, "Ruang Rapat Meranti")).status, 200);

  const insideExistingCommitment = await create(date, "09:15");
  assert.equal((await approve(insideExistingCommitment, "Ruang Rapat Mahakam")).status, 409);
  const adjacent = await create(date, "10:00");
  assert.equal((await approve(adjacent)).status, 200);

  const longer = await post(`/bookings/${first.id}/reschedule?role=PA`, {
    version: 2,
    date,
    time: "09:00",
    duration: 90,
    room: "Ruang Rapat Nusantara",
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
    room: "Ruang Rapat Nusantara",
    reason: "Move to an available adjacent interval",
  });
  assert.equal(earlier.status, 200);
  assert.equal(earlier.data.status, "approved");
  assert.equal((await approve(sameOfficer, "Ruang Rapat Mahakam")).status, 200);
});

test("competing approvals cannot both reserve the same resource", async () => {
  const first = await create("2098-06-12", "10:00");
  const second = await create("2098-06-12", "10:00");
  const results = await Promise.all([approve(first), approve(second, "Ruang Rapat Meranti")]);
  assert.deepEqual(results.map((result) => result.status).sort(), [200, 409]);
  const records = await Promise.all([read(first.id), read(second.id)]);
  assert.equal(records.filter((record) => record.status === "approved").length, 1);
  assert.equal(
    records.filter((record) => record.status === "pending" && record.version === 1).length,
    1,
  );
});

test("approved restricted-host changes use the same policy and blank resources create no phantom reservation", async () => {
  const date = "2098-06-13";
  const occupied = await create(date, "10:00");
  assert.equal((await approve(occupied)).status, 200);
  const draft = { visitKind: "restricted", date, time: "10:00" };
  const first = await post(`/staff/bookings?role=${protocol}`, {
    ...draft,
    idempotencyKey: randomUUID(),
  });
  const second = await post(`/staff/bookings?role=${protocol}`, {
    ...draft,
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
  assert.equal(unchanged.officerId, undefined);
  assert.equal(unchanged.version, 1);
  assert.equal(
    (
      await post(`/bookings/${first.data.id}/update?role=${protocol}`, {
        ...draft,
        officerId: "o3",
        version: 1,
      })
    ).status,
    200,
  );
  assert.equal(
    (
      await post(`/bookings/${first.data.id}/update?role=${protocol}`, {
        ...draft,
        officerId: "",
        version: 2,
      })
    ).status,
    200,
  );
  const sameOfficer = await create(date, "10:15");
  api.setNow("2098-06-13T09:45:00+08:00");
  const checkedIn = await post(`/bookings/${occupied.id}/checkin?role=Receptionist`, {
    version: 2,
    credential: occupied.code,
  });
  assert.equal(checkedIn.status, 200);
  assert.equal((await approve(sameOfficer, "Ruang Rapat Mahakam")).status, 409);
});
