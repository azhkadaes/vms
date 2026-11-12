import test from "node:test";
import assert from "node:assert/strict";
import { publicVisitHistory } from "../packages/data/src/history.ts";
import { canVisitorCancel, getVisitTiming } from "../packages/data/src/attendance.ts";

test("public events preserve order, action, time and designated public reasons without staff metadata", () => {
  const stored = [
    {
      at: "2099-08-19T01:00:00Z",
      action: "received",
      message: "PRIVATE creator",
      actor: "PRIVATE actor",
    },
    {
      at: "2099-08-19T01:01:00Z",
      action: "approved",
      message: "PA: approve",
      internalNote: "PRIVATE",
    },
    {
      at: "2099-08-19T01:02:00Z",
      action: "rescheduled",
      message: "PRIVATE metadata",
      publicMessage: "Alasan publik\nTetap sesuai aslinya.",
    },
  ];
  assert.deepEqual(publicVisitHistory(stored), [
    { at: stored[0].at, action: "received" },
    { at: stored[1].at, action: "approved" },
    { at: stored[2].at, action: "rescheduled", publicMessage: stored[2].publicMessage },
  ]);
  assert.equal(stored[1].message, "PA: approve");
});

test("legacy/unknown actions become a generic event and never fall back to raw messages", () => {
  const events = publicVisitHistory([
    { at: "2099-08-19T01:00:00Z", message: "PRIVATE legacy actor" },
    {
      at: "2099-08-19T01:01:00Z",
      action: "future-internal-action",
      message: "PRIVATE",
      publicMessage: "PRIVATE future field",
    },
  ]);
  assert.deepEqual(
    events.map((event) => event.action),
    ["updated", "updated"],
  );
  assert.equal(JSON.stringify(events).includes("PRIVATE"), false);
  assert.deepEqual(publicVisitHistory([]), []);
});

test("visitor cancellation closes exactly at start for Pending and Approved", () => {
  const visit = { date: "2099-08-19", time: "10:00", duration: 60, status: "pending" as const };
  const start = Date.parse("2099-08-19T10:00:00+08:00");
  for (const status of ["pending", "approved"] as const) {
    const current = { ...visit, status };
    assert.equal(canVisitorCancel(current, start - 1), true);
    assert.equal(canVisitorCancel(current, start), false);
    assert.equal(canVisitorCancel(current, start + 1), false);
    assert.equal(getVisitTiming(current, start).canCancel, false);
    assert.equal(getVisitTiming(current, start - 1).canCancel, true);
  }
});

test("checked-in/terminal or invalid-date visits never allow visitor cancellation", () => {
  for (const status of ["checked_in", "checked_out", "cancelled", "rejected"] as const)
    assert.equal(canVisitorCancel({ date: "2099-08-19", time: "10:00", status }, 0), false);
  assert.equal(canVisitorCancel({ date: "invalid", time: "10:00", status: "approved" }, 0), false);
});
