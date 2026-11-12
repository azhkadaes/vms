import test from "node:test";
import assert from "node:assert/strict";
import { can, sessionFor, type StaffBooking } from "../packages/data/src/domain.ts";
import { staffCancellationSchema } from "../packages/data/src/validation.ts";

const visit: StaffBooking = {
  id: "staff-cancel",
  code: "88002001",
  date: "2099-08-19",
  time: "10:00",
  duration: 60,
  room: "Example Room",
  officerId: "o2",
  vip: false,
  status: "approved",
  version: 2,
};
test("staff cancellation requires explicit permission and existing decision scope, including after the arrival window", () => {
  const late = Date.parse("2099-08-19T12:00:00+08:00");
  const pa = sessionFor("PA");
  assert.equal(can(pa, visit, "cancel", late), true);
  assert.equal(
    can(
      { ...pa, permissions: pa.permissions.filter((permission) => permission !== "cancel") },
      visit,
      "cancel",
      late,
    ),
    false,
  );
  assert.equal(can({ ...pa, assignedOfficerIds: [] }, visit, "cancel", late), false);
  assert.equal(can(sessionFor("Protocol Officer"), visit, "cancel", late), false);
  assert.equal(
    can(sessionFor("Protocol Officer"), { ...visit, visitKind: "restricted" }, "cancel", late),
    true,
  );
  for (const role of ["Receptionist", "Super Admin"] as const)
    assert.equal(
      can(
        { ...sessionFor(role), permissions: ["bookings", "cancel"] },
        { ...visit, visitKind: "restricted" },
        "cancel",
        late,
      ),
      false,
    );
  for (const status of ["checked_in", "checked_out", "cancelled", "rejected"] as const)
    assert.equal(can(pa, { ...visit, status }, "cancel", late), false);
});
test("staff cancellation reasons are required, trimmed and bounded", () => {
  assert.equal(
    staffCancellationSchema.parse({ reason: "Valid reason" }).internalCancellationNote,
    "",
  );
  assert.equal(
    staffCancellationSchema.parse({
      reason: "Valid reason",
      internalCancellationNote: "  Staff only  ",
    }).internalCancellationNote,
    "Staff only",
  );
  assert.equal(
    staffCancellationSchema.safeParse({
      reason: "Valid reason",
      internalCancellationNote: "x".repeat(2001),
    }).success,
    false,
  );
  for (const reason of ["", "     ", "abcd", "x".repeat(1001)])
    assert.equal(staffCancellationSchema.safeParse({ reason }).success, false);
  assert.equal(
    staffCancellationSchema.parse({ reason: "  Required staff reason.  " }).reason,
    "Required staff reason.",
  );
});
