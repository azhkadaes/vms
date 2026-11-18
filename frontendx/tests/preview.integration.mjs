import test, { before, after, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";

import { startPreview } from "./helpers/preview-server.mjs";
let api, base;
before(async () => {
  api = await startPreview();
  base = api.base;
});
beforeEach(() => api.setNow("2099-01-01T08:00:00+08:00"));
after(async () => api?.close());
async function post(path, body) {
  const res = await fetch(base + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return { status: res.status, data: await res.json() };
}
test("preview retries, fixed rescheduling, and stale writes preserve booking state", async () => {
  const payload = {
    ack: true,
    idempotencyKey: randomUUID(),
    officerId: "o2",
    name: "QA Synthetic Visitor",
    organization: "QA Example",
    email: "qa@example.test",
    phone: "08000000000",
    date: "2099-10-20",
    time: "10:00",
    purpose: "Synthetic UI integration test",
    identityType: "SIM",
    visitors: ["QA Synthetic Visitor"],
  };
  const first = await post("/bookings", payload);
  const retry = await post("/bookings", payload);
  assert.equal(first.status, 201);
  assert.equal(first.data.id, retry.data.id);
  const approved = await post(`/bookings/${first.data.id}/approve?role=PA`, {
    version: 1,
    room: "Ruang Rapat Nusantara",
    duration: 60,
  });
  assert.equal(approved.data.status, "approved");
  const changed = await post(`/bookings/${first.data.id}/reschedule?role=PA`, {
    version: 2,
    date: "2099-10-21",
    time: "11:00",
    duration: 90,
    room: "Ruang Rapat Meranti",
    reason: "Fixed synthetic schedule adjustment",
  });
  assert.equal(changed.status, 200);
  assert.equal(changed.data.status, "approved");
  assert.equal(changed.data.rescheduled, true);
  assert.equal(changed.data.room, "Ruang Rapat Meranti");
  assert.match(changed.data.history.at(-1).message, /Fixed synthetic/);
  const stale = await post(`/bookings/${first.data.id}/reschedule?role=PA`, {
    version: 2,
    date: "2099-10-22",
    time: "11:00",
    duration: 60,
    room: "Ruang Rapat Meranti",
    reason: "Stale change",
  });
  assert.equal(stale.status, 409);
  const denied = await post(`/bookings/${first.data.id}/reschedule?role=Receptionist`, {
    version: 3,
    date: "2099-10-22",
    time: "11:00",
    duration: 60,
    room: "Ruang Rapat Meranti",
    reason: "Wrong role",
  });
  assert.equal(denied.status, 403);
  const list = await fetch(base + "/bookings?role=PA").then((r) => r.json());
  assert.equal(list.filter((b) => b.id === first.data.id).length, 1);
});

test("visitor lookup and cancellation require the email and booking code together, without OTP", async () => {
  const created = await post("/bookings", {
    ack: true,
    idempotencyKey: randomUUID(),
    officerId: "o2",
    name: "QA Email Pair",
    organization: "QA Example",
    email: "pair@example.test",
    phone: "08000000000",
    date: "2099-11-01",
    time: "10:00",
    purpose: "Synthetic pair lookup check",
    identityType: "KTP",
    visitors: ["QA Email Pair"],
    ndName: "qa-example.pdf",
  });
  assert.equal(created.status, 201);
  const { code, id, version } = created.data;
  const found = await post("/lookup", { code, email: " PAIR@example.test " });
  assert.equal(found.status, 200);
  assert.equal(found.data.id, id);
  assert.equal(found.data.ndName, "qa-example.pdf");
  for (const credentials of [
    { code },
    { email: "pair@example.test" },
    { code, email: "wrong@example.test" },
    { code: "00000000", email: "pair@example.test" },
  ]) {
    const rejected = await post("/lookup", credentials);
    assert.equal(rejected.status, 401);
    assert.equal(rejected.data.id, undefined);
    assert.match(rejected.data.message, /Email atau kode booking tidak cocok/);
  }
  const denied = await post(`/bookings/${id}/cancel`, { code, version });
  assert.equal(denied.status, 403);
  const cancelled = await post(`/bookings/${id}/cancel`, {
    code,
    email: "pair@example.test",
    version,
  });
  assert.equal(cancelled.status, 200);
  assert.equal(cancelled.data.status, "cancelled");
});

for (const [role, officerId, otherRole] of [
  ["PA", "o2", "Protocol Officer"],
  ["Protocol Officer", "o1", "PA"],
]) {
  test(`${role} rejection separates public and internal remarks across every visitor response`, async () => {
    const payload = {
      ack: true,
      idempotencyKey: randomUUID(),
      officerId,
      name: `QA Rejection ${role}`,
      organization: "QA Example",
      email: "rejection@example.test",
      phone: "08000000000",
      date: "2099-12-01",
      time: "09:00",
      purpose: "Synthetic rejection privacy test",
      identityType: "KTP",
      visitors: [`QA Rejection ${role}`],
      internalRejectionNote: "FORGED_VISITOR_NOTE",
    };
    const created = await post("/bookings", payload);
    assert.equal(created.status, 201);
    assert.equal("internalRejectionNote" in created.data, false);
    const { id, code } = created.data;
    const path = `/bookings/${id}/reject?role=${encodeURIComponent(role)}`;
    for (const fields of [
      { reason: "Old single field" },
      { rejectionReason: "    " },
      {
        rejectionReason: "Valid public message",
        internalRejectionNote: "x".repeat(2001),
      },
    ]) {
      const invalid = await post(path, { version: 1, ...fields });
      assert.equal(invalid.status, 422);
      assert.ok(invalid.data.fieldErrors);
    }
    for (const deniedRole of ["Receptionist", "Super Admin", otherRole]) {
      const denied = await post(`/bookings/${id}/reject?role=${encodeURIComponent(deniedRole)}`, {
        version: 1,
        rejectionReason: "Wrong role rejection",
        internalRejectionNote: "denied",
      });
      assert.equal(denied.status, 403);
    }
    const rejectionReason = "Silakan ajukan jadwal lain untuk pertemuan ini.";
    const internalRejectionNote = `INTERNAL_ONLY_${randomUUID()}`;
    const rejected = await post(path, {
      version: 1,
      rejectionReason: `  ${rejectionReason}  `,
      internalRejectionNote,
    });
    assert.equal(rejected.status, 200);
    assert.equal(rejected.data.status, "rejected");
    assert.equal(rejected.data.version, 2);
    assert.equal(rejected.data.rejectionReason, rejectionReason);
    assert.equal(rejected.data.internalRejectionNote, internalRejectionNote);
    assert.equal(JSON.stringify(rejected.data.history).includes(internalRejectionNote), false);
    const stale = await post(path, {
      version: 1,
      rejectionReason: "Stale public reason",
      internalRejectionNote: "stale",
    });
    assert.equal(stale.status, 409);
    const ownerList = await fetch(base + `/bookings?role=${encodeURIComponent(role)}`).then((r) =>
      r.json(),
    );
    assert.equal(ownerList.find((b) => b.id === id).internalRejectionNote, internalRejectionNote);
    const otherList = await fetch(base + `/bookings?role=${encodeURIComponent(otherRole)}`).then(
      (r) => r.json(),
    );
    assert.equal(JSON.stringify(otherList).includes(internalRejectionNote), false);
    const found = await post("/lookup", { code, email: payload.email });
    const retried = await post("/bookings", payload);
    const reception = await post("/resolve?role=Receptionist", {
      credential: code,
    });
    assert.equal(reception.status, 404);
    assert.equal(reception.data.id, undefined);
    assert.equal(JSON.stringify(reception.data).includes(rejectionReason), false);
    for (const response of [found, retried]) {
      assert.equal(response.data.rejectionReason, rejectionReason);
      assert.equal("internalRejectionNote" in response.data, false);
      assert.equal(JSON.stringify(response.data).includes(internalRejectionNote), false);
      assert.equal(JSON.stringify(response.data).includes("FORGED_VISITOR_NOTE"), false);
    }
  });
}

test("Receptionist list and code/QR lookup exclude unapproved requests and retain completed visits", async () => {
  api.setNow("2099-12-25T09:45:00+08:00");
  const payload = {
    ack: true,
    officerId: "o2",
    name: "QA Reception Scope",
    organization: "QA Example",
    email: "reception-scope@example.test",
    phone: "081234567890",
    date: "2099-12-25",
    time: "10:00",
    purpose: "Synthetic receptionist status scope",
    identityType: "KTP",
    visitors: ["QA Reception Scope"],
  };
  async function create() {
    const response = await post("/bookings", {
      ...payload,
      idempotencyKey: randomUUID(),
    });
    assert.equal(response.status, 201);
    return response.data;
  }
  async function hidden(booking) {
    const unknown = await post("/resolve?role=Receptionist", {
      credential: "00000000",
    });
    for (const credential of [booking.code, `VMS-DEMO:${booking.code}`]) {
      const resolved = await post("/resolve?role=Receptionist", { credential });
      assert.equal(resolved.status, 404);
      assert.deepEqual(resolved.data, unknown.data);
      assert.equal(resolved.data.id, undefined);
      assert.equal(JSON.stringify(resolved.data).includes(payload.name), false);
    }
  }
  const pending = await create();
  await hidden(pending);
  const premature = await post(`/bookings/${pending.id}/checkin?role=Receptionist`, { version: 1 });
  assert.equal(premature.status, 403);
  const approved = await post(`/bookings/${pending.id}/approve?role=PA`, {
    version: 1,
    room: "Ruang Rapat Meranti",
    duration: 60,
  });
  assert.equal(approved.status, 200);
  const receptionList = await fetch(base + "/bookings?role=Receptionist");
  const preview = (await receptionList.json()).find((booking) => booking.id === pending.id);
  assert.equal(preview.name, pending.name);
  assert.equal(preview.visitorCount, payload.visitors.length);
  for (const field of ["code", "email", "phone", "purpose", "visitors", "history", "identityType"])
    assert.equal(Object.hasOwn(preview, field), false, field);
  assert.equal((await fetch(base + `/bookings/${pending.id}?role=Receptionist`)).status, 404);
  for (const credential of [pending.code, `VMS-DEMO:${pending.code}`]) {
    const resolved = await post("/resolve?role=Receptionist", { credential });
    assert.equal(resolved.status, 200);
    assert.equal(resolved.data.status, "approved");
  }
  assert.equal(
    (
      await post(`/bookings/${pending.id}/checkout?role=Receptionist`, {
        version: 2,
      })
    ).status,
    403,
  );
  const checkedIn = await post(`/bookings/${pending.id}/checkin?role=Receptionist`, {
    version: 2,
    credential: pending.code,
  });
  assert.equal(checkedIn.status, 200);
  assert.equal(checkedIn.data.status, "checked_in");
  assert.equal(
    (await post("/resolve?role=Receptionist", { credential: pending.code })).data.status,
    "checked_in",
  );
  assert.equal(
    (
      await post(`/bookings/${pending.id}/checkin?role=Receptionist`, {
        version: 3,
      })
    ).status,
    403,
  );
  const checkedOut = await post(`/bookings/${pending.id}/checkout?role=Receptionist`, {
    version: 3,
    credential: pending.code,
  });
  assert.equal(checkedOut.status, 200);
  assert.equal(checkedOut.data.status, "checked_out");
  assert.equal(
    (await post("/resolve?role=Receptionist", { credential: pending.code })).data.status,
    "checked_out",
  );
  assert.equal(
    (
      await post(`/bookings/${pending.id}/checkout?role=Receptionist`, {
        version: 4,
      })
    ).status,
    403,
  );
  const rejected = await create();
  assert.equal(
    (
      await post(`/bookings/${rejected.id}/reject?role=PA`, {
        version: 1,
        rejectionReason: "Synthetic rejection scope check",
        internalRejectionNote: "Internal example",
      })
    ).status,
    200,
  );
  await hidden(rejected);
  const cancelled = await create();
  assert.equal(
    (
      await post(`/bookings/${cancelled.id}/cancel`, {
        code: cancelled.code,
        email: payload.email,
        version: 1,
      })
    ).status,
    200,
  );
  await hidden(cancelled);
  const list = await fetch(base + "/bookings?role=Receptionist").then((r) => r.json());
  assert.ok(list.length);
  assert.ok(list.every((b) => ["approved", "checked_in", "checked_out"].includes(b.status)));
  assert.equal(list.find((b) => b.id === pending.id).status, "checked_out");
  assert.equal(
    list.some((b) => b.id === rejected.id || b.id === cancelled.id),
    false,
  );
});

test("rejection can be completed without an internal note", async () => {
  const created = await post("/bookings", {
    ack: true,
    idempotencyKey: randomUUID(),
    officerId: "o2",
    name: "QA Optional Note",
    organization: "QA Example",
    email: "optional@example.test",
    phone: "08000000000",
    date: "2099-12-02",
    time: "09:00",
    purpose: "Synthetic optional note test",
    identityType: "SIM",
    visitors: ["QA Optional Note"],
  });
  const result = await post(`/bookings/${created.data.id}/reject?role=PA`, {
    version: 1,
    rejectionReason: "Silakan lengkapi permohonan kunjungan Anda.",
  });
  assert.equal(result.status, 200);
  assert.equal(result.data.status, "rejected");
  assert.equal("internalRejectionNote" in result.data, false);
});

for (const [role, officerId, forbiddenOfficer] of [
  ["PA", "o2", "o1"],
  ["Protocol Officer", "o1", "o2"],
]) {
  test(`${role} creates scoped Approved appointments with safe idempotent retries`, async () => {
    const path = `/staff/bookings?role=${encodeURIComponent(role)}`;
    const input = {
      idempotencyKey: randomUUID(),
      officerId,
      name: `QA Staff ${role}`,
      organization: "QA Example",
      email: "staff-created@example.test",
      phone: "081234567890",
      date: role === "PA" ? "2099-12-10" : "2099-12-11",
      time: "10:00",
      purpose: "Synthetic staff appointment",
      identityType: "KTP",
      visitors: [`QA Staff ${role}`, "QA Group Member"],
      ndName: "qa-staff.pdf",
      status: "approved",
      vip: officerId !== "o1",
      createdBy: { name: "FORGED_ACTOR", role: "Super Admin" },
      internalRejectionNote: "FORGED_NOTE",
    };
    const deniedTarget = await post(path, {
      ...input,
      officerId: forbiddenOfficer,
    });
    assert.equal(deniedTarget.status, 403);
    const badInput = await post(path, { ...input, phone: "08abc1234" });
    assert.equal(badInput.status, 422);
    assert.ok(badInput.data.fieldErrors.phone);
    const missingKey = await post(path, { ...input, idempotencyKey: "" });
    assert.equal(missingKey.status, 422);
    for (const deniedRole of ["Receptionist", "Super Admin"]) {
      assert.equal(
        (await post(`/staff/bookings?role=${encodeURIComponent(deniedRole)}`, input)).status,
        403,
      );
    }
    const created = await post(path, input);
    assert.equal(created.status, 201);
    assert.equal(created.data.status, "approved");
    assert.match(created.data.history.at(-1).message, /otomatis disetujui/);
    assert.equal(created.data.version, 1);
    assert.equal(created.data.vip, officerId === "o1");
    assert.equal(created.data.duration, 60);
    assert.equal(created.data.createdBy.role, role);
    assert.notEqual(created.data.createdBy.name, "FORGED_ACTOR");
    assert.equal(created.data.internalRejectionNote, undefined);
    assert.deepEqual(created.data.visitors, input.visitors);
    const retry = await post(path, input);
    assert.equal(retry.status, 201);
    assert.equal(retry.data.id, created.data.id);
    const conflict = await post(path, {
      ...input,
      purpose: "Changed data under the same key",
    });
    assert.equal(conflict.status, 409);
    const list = await fetch(base + `/bookings?role=${encodeURIComponent(role)}`).then((r) =>
      r.json(),
    );
    assert.equal(list.filter((b) => b.id === created.data.id).length, 1);
    const found = await post("/lookup", {
      code: created.data.code,
      email: input.email,
    });
    assert.equal(found.status, 200);
    assert.equal(found.data.status, "approved");
    assert.equal("createdBy" in found.data, false);
    assert.equal(JSON.stringify(found.data).includes("FORGED"), false);
    const approved = await post(
      `/bookings/${created.data.id}/approve?role=${encodeURIComponent(role)}`,
      { version: 1, room: "Ruang Rapat Nusantara", duration: 60 },
    );
    assert.equal(approved.status, 403);
    const occupied = await post(path, {
      ...input,
      idempotencyKey: randomUUID(),
      time: "10:30",
    });
    assert.equal(occupied.status, 409);
  });
}
