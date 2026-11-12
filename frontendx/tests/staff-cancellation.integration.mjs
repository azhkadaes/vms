import test from "node:test";
import assert from "node:assert/strict";
import { sessionFor } from "../packages/data/src/domain.ts";
import { startPreview } from "./helpers/preview-server.mjs";

async function request(api, path, body) {
  const response = await fetch(api.base + path, {
    method: body ? "POST" : "GET",
    headers: { "Content-Type": "application/json" },
    body: body && JSON.stringify(body),
  });
  return { status: response.status, data: await response.json() };
}
async function create(api, extra = {}) {
  return (
    await request(api, "/bookings", {
      ack: true,
      idempotencyKey: crypto.randomUUID(),
      officerId: "o2",
      date: "2099-08-19",
      time: "10:00",
      name: "Cancellation Example",
      organization: "Example Organization",
      email: "cancel@example.test",
      phone: "00123456789",
      purpose: "Cancellation verification",
      visitors: ["Cancellation Example"],
      identityType: "KTP",
      ...extra,
    })
  ).data;
}

test("late staff cancellation exposes the visitor reason, protects internal notes and releases reservations", async () => {
  const api = await startPreview({ time: "2099-08-19T09:00:00+08:00" });
  try {
    let visit = await create(api);
    visit = (
      await request(api, `/bookings/${visit.id}/approve?role=PA`, {
        version: visit.version,
        room: "Ruang Rapat Nusantara",
        duration: 60,
      })
    ).data;
    api.setNow("2099-08-19T10:30:00+08:00");
    assert.equal(
      (
        await request(api, `/bookings/${visit.id}/cancel`, {
          code: visit.code,
          email: visit.email,
          version: visit.version,
        })
      ).status,
      409,
    );
    const next = await create(api, { time: "10:45" });
    assert.equal(
      (
        await request(api, `/bookings/${next.id}/approve?role=PA`, {
          version: next.version,
          room: "Ruang Rapat Nusantara",
          duration: 60,
        })
      ).status,
      409,
    );
    assert.equal(
      (
        await request(api, `/staff/bookings/${visit.id}/cancel?role=PA`, {
          version: visit.version,
          reason: "     ",
        })
      ).status,
      422,
    );
    const cancelled = await request(api, `/staff/bookings/${visit.id}/cancel?role=PA`, {
      version: visit.version,
      reason: "The host is unavailable.",
      internalCancellationNote: "PRIVATE staff follow-up.",
    });
    assert.equal(cancelled.status, 200);
    assert.equal(cancelled.data.status, "cancelled");
    assert.equal(cancelled.data.cancellationReason, "The host is unavailable.");
    assert.equal(cancelled.data.internalCancellationNote, "PRIVATE staff follow-up.");
    const detail = await request(api, `/bookings/${visit.id}?role=PA`);
    assert.equal(detail.data.internalCancellationNote, "PRIVATE staff follow-up.");
    assert.equal(cancelled.data.history.length, visit.history.length + 1);
    assert.equal(
      (
        await request(api, `/staff/bookings/${visit.id}/cancel?role=PA`, {
          version: visit.version,
          reason: "Repeated cancellation.",
        })
      ).status,
      409,
    );
    const publicVisit = await request(api, "/lookup", { code: visit.code, email: visit.email });
    assert.equal(publicVisit.data.status, "cancelled");
    assert.equal(publicVisit.data.cancellationReason, "The host is unavailable.");
    assert.equal(JSON.stringify(publicVisit.data).includes("PRIVATE"), false);
    assert.equal("internalCancellationNote" in publicVisit.data, false);
    assert.equal(JSON.stringify(publicVisit.data.history).includes("PRIVATE"), false);
    assert.equal(publicVisit.data.history.at(-1).action, "cancelled");
    assert.equal(
      (await request(api, "/resolve?role=Receptionist", { credential: visit.code })).status,
      404,
    );
    assert.equal(
      (
        await request(api, `/bookings/${next.id}/approve?role=PA`, {
          version: next.version,
          room: "Ruang Rapat Nusantara",
          duration: 60,
        })
      ).status,
      200,
    );
  } finally {
    await api.close();
  }
});
test("restricted staff cancellation remains Protocol-only and never opens public cancellation", async () => {
  const api = await startPreview({ time: "2099-08-19T09:00:00+08:00" });
  try {
    const visit = (
      await request(api, "/staff/bookings?role=Protocol%20Officer", {
        visitKind: "restricted",
        date: "2099-08-19",
        time: "10:00",
        officerId: "o2",
        idempotencyKey: crypto.randomUUID(),
      })
    ).data;
    for (const role of ["PA", "Receptionist", "Super Admin"])
      assert.ok(
        [403, 404].includes(
          (
            await request(
              api,
              `/staff/bookings/${visit.id}/cancel?role=${encodeURIComponent(role)}`,
              { version: visit.version, reason: "Unauthorized reason." },
            )
          ).status,
        ),
      );
    assert.equal(
      (
        await request(api, `/bookings/${visit.id}/cancel`, {
          version: visit.version,
          reason: "Public attempt.",
        })
      ).status,
      404,
    );
    assert.equal(
      (
        await request(api, `/staff/bookings/${visit.id}/cancel?role=Protocol%20Officer`, {
          version: visit.version,
          reason: "Authorized restricted cancellation.",
        })
      ).data.status,
      "cancelled",
    );
  } finally {
    await api.close();
  }
});
test("staff cancellation cannot bypass missing permission or an already checked-in visit", async () => {
  let permit = false;
  const api = await startPreview({
    time: "2099-08-19T09:00:00+08:00",
    session: (role) => {
      const session = sessionFor(role);
      return {
        ...session,
        permissions: permit
          ? session.permissions
          : session.permissions.filter((permission) => permission !== "cancel"),
      };
    },
  });
  try {
    let visit = await create(api);
    assert.equal(
      (
        await request(api, `/staff/bookings/${visit.id}/cancel?role=PA`, {
          version: visit.version,
          reason: "No permission reason.",
        })
      ).status,
      403,
    );
    permit = true;
    visit = (
      await request(api, `/bookings/${visit.id}/approve?role=PA`, {
        version: visit.version,
        room: "Ruang Rapat Nusantara",
        duration: 60,
      })
    ).data;
    api.setNow("2099-08-19T09:30:00+08:00");
    visit = (
      await request(api, `/bookings/${visit.id}/checkin?role=Receptionist`, {
        version: visit.version,
        credential: visit.code,
        verified: true,
      })
    ).data;
    assert.equal(visit.status, "checked_in");
    assert.equal(
      (
        await request(api, `/staff/bookings/${visit.id}/cancel?role=PA`, {
          version: visit.version,
          reason: "Too late after check-in.",
        })
      ).status,
      403,
    );
  } finally {
    await api.close();
  }
});
