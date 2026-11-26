import test from "node:test";
import assert from "node:assert/strict";
import { startPreview } from "./helpers/preview-server.mjs";

async function post(api, path, body) {
  const response = await fetch(api.base + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return { status: response.status, data: await response.json() };
}
function input() {
  return {
    idempotencyKey: crypto.randomUUID(),
    officerId: "o2",
    date: "2099-08-19",
    time: "10:00",
    name: "History Example",
    organization: "History Example Organization",
    email: "history@example.test",
    phone: "00123456789",
    purpose: "History verification",
    visitors: ["History Example"],
    identityType: "KTP",
    ack: true,
  };
}
async function create(api, staff = false) {
  const payload = input();
  const response = await post(api, staff ? "/staff/bookings?role=PA" : "/bookings", payload);
  assert.equal(response.status, 201);
  return { payload, booking: response.data };
}
async function lookup(api, booking) {
  const response = await post(api, "/lookup", { code: booking.code, email: booking.email });
  assert.equal(response.status, 200);
  return response.data;
}
function assertPublic(booking) {
  for (const event of booking.history) {
    assert.ok(Object.keys(event).every((key) => ["at", "action", "publicMessage"].includes(key)));
    assert.ok(
      [
        "received",
        "approved",
        "rejected",
        "rescheduled",
        "cancelled",
        "checked_in",
        "checked_out",
        "updated",
      ].includes(event.action),
    );
    assert.equal(Number.isFinite(Date.parse(event.at)), true);
  }
  assert.equal(JSON.stringify(booking.history).includes("PA:"), false);
}

test("staff-created lifecycle becomes typed public history while staff messages remain separate", async () => {
  const api = await startPreview({ time: "2099-08-19T09:00:00+08:00" });
  try {
    let { booking } = await create(api, true);
    let publicBooking = await lookup(api, booking);
    assertPublic(publicBooking);
    assert.deepEqual(publicBooking.history, [
      { at: "2099-08-19T01:00:00.000Z", action: "received" },
      { at: "2099-08-19T01:00:00.000Z", action: "approved" },
    ]);
    assert.equal(booking.status, "approved");
    assert.match(booking.history.at(-1).message, /otomatis disetujui/);
    const reason = "Alasan publik dalam bahasa asli.";
    booking = (
      await post(api, `/bookings/${booking.id}/reschedule?role=PA`, {
        version: booking.version,
        date: "2099-08-19",
        time: "11:00",
        duration: 60,
        roomId: "room-2",
        reason,
      })
    ).data;
    publicBooking = await lookup(api, booking);
    assert.deepEqual(publicBooking.history.at(-1), {
      at: "2099-08-19T01:00:00.000Z",
      action: "rescheduled",
      publicMessage: reason,
    });
    api.setNow("2099-08-19T11:00:00+08:00");
    booking = (
      await post(api, `/bookings/${booking.id}/attendance?role=Receptionist`, { action: "check_in",
        version: booking.version,
        credential: booking.code,
      })
    ).data;
    api.setNow("2099-08-19T12:30:00+08:00");
    booking = (
      await post(api, `/bookings/${booking.id}/attendance?role=Receptionist`, { action: "check_out",
        version: booking.version,
        credential: booking.code,
      })
    ).data;
    publicBooking = await lookup(api, booking);
    assertPublic(publicBooking);
    assert.deepEqual(
      publicBooking.history.map((event) => event.action),
      ["received", "approved", "rescheduled", "checked_in", "checked_out"],
    );
    assert.equal(publicBooking.history.at(-1).at, "2099-08-19T04:30:00.000Z");
  } finally {
    await api.close();
  }
});

test("lookup, submission replay and recovery keep public rejection copy and exclude internal notes", async () => {
  const api = await startPreview({ time: "2099-08-19T09:00:00+08:00" });
  try {
    const { payload, booking } = await create(api);
    const publicReason = "Silakan ajukan kembali.\nPesan publik asli.";
    await post(api, `/bookings/${booking.id}/reject?role=PA`, {
      version: 1,
      rejectionReason: publicReason,
      internalRejectionNote: "PRIVATE NOTE",
    });
    const recovered = await post(api, "/submissions/resolve", {
      idempotencyKey: payload.idempotencyKey,
      email: payload.email,
    });
    const replay = await post(api, "/bookings", payload);
    for (const result of [await lookup(api, booking), recovered.data.booking, replay.data]) {
      assertPublic(result);
      assert.equal(result.rejectionReason, publicReason);
      assert.deepEqual(
        result.history.map((event) => event.action),
        ["received", "rejected"],
      );
      assert.equal(JSON.stringify(result).includes("PRIVATE"), false);
      assert.equal("createdBy" in result, false);
    }
  } finally {
    await api.close();
  }
});

test("cancellation is permitted just before start but rejected at/after start without changing state", async () => {
  for (const approved of [false, true]) {
    for (const time of ["09:59:59.999", "10:00:00.000", "10:00:00.001", "11:00:00.001"]) {
      const api = await startPreview({ time: "2099-08-19T09:00:00+08:00" });
      try {
        let { booking } = await create(api);
        if (approved)
          booking = (
            await post(api, `/bookings/${booking.id}/approve?role=PA`, {
              version: 1,
              duration: 60,
              roomId: "room-1",
            })
          ).data;
        api.setNow(`2099-08-19T${time}+08:00`);
        const result = await post(api, `/bookings/${booking.id}/cancel`, {
          code: booking.code,
          email: booking.email,
          version: booking.version,
        });
        const expected = time.startsWith("09:") ? 200 : 409;
        assert.equal(result.status, expected);
        const current = await lookup(api, booking);
        assertPublic(current);
        assert.equal(current.status, expected === 200 ? "cancelled" : booking.status);
        assert.equal(current.version, booking.version + (expected === 200 ? 1 : 0));
        if (expected === 200) assert.equal(current.history.at(-1).action, "cancelled");
        else assert.equal(current.history.length, booking.history.length);
      } finally {
        await api.close();
      }
    }
  }
});
