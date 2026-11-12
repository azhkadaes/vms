import test from "node:test";
import assert from "node:assert/strict";
import { hasReservationConflict } from "../packages/data/src/scheduling.ts";
import type { Booking } from "../packages/data/src/domain.ts";

const candidate = {
  date: "2098-06-11",
  time: "10:00",
  duration: 60,
  officerId: "o2",
  room: "Room A",
};
const existing = { ...candidate, id: "existing", status: "approved" as Booking["status"] };
const cases = [
  {
    name: "same officer conflicts across different rooms",
    other: { room: "Room B" },
    expected: true,
  },
  {
    name: "same room conflicts across different officers",
    other: { officerId: "o3" },
    expected: true,
  },
  {
    name: "different officers and rooms can overlap",
    other: { officerId: "o3", room: "Room B" },
    expected: false,
  },
  { name: "earlier adjacent interval is allowed", other: { time: "09:00" }, expected: false },
  { name: "later adjacent interval is allowed", other: { time: "11:00" }, expected: false },
  { name: "different dates do not conflict", other: { date: "2098-06-12" }, expected: false },
  { name: "Pending does not reserve", other: { status: "pending" as const }, expected: false },
  { name: "Rejected does not reserve", other: { status: "rejected" as const }, expected: false },
  { name: "Cancelled does not reserve", other: { status: "cancelled" as const }, expected: false },
  {
    name: "Checked out releases its reservation",
    other: { status: "checked_out" as const },
    expected: false,
  },
  {
    name: "Checked in retains its reservation",
    other: { status: "checked_in" as const },
    expected: true,
  },
  {
    name: "longer commitment covers the candidate",
    other: { time: "09:30", duration: 120 },
    expected: true,
  },
  { name: "partial interval overlap conflicts", other: { time: "10:45" }, expected: true },
];
for (const entry of cases)
  test(entry.name, () => {
    assert.equal(
      hasReservationConflict(candidate, [{ ...existing, ...entry.other }]),
      entry.expected,
    );
  });
test("the record being changed is excluded", () => {
  assert.equal(hasReservationConflict(candidate, [existing], existing.id), false);
});
test("empty rooms do not create a shared resource", () => {
  assert.equal(
    hasReservationConflict({ ...candidate, room: "" }, [
      { ...existing, officerId: "o3", room: "" },
    ]),
    false,
  );
});
test("empty officers do not create a shared resource", () => {
  assert.equal(
    hasReservationConflict({ ...candidate, officerId: "" }, [
      { ...existing, officerId: "", room: "Room B" },
    ]),
    false,
  );
});
test("resourceless restricted slots can coexist", () => {
  assert.equal(
    hasReservationConflict({ ...candidate, officerId: "", room: "" }, [
      { ...existing, officerId: "", room: "" },
    ]),
    false,
  );
});
test("extending a duration checks the new end", () => {
  const next = { ...existing, time: "11:00" };
  assert.equal(hasReservationConflict(candidate, [next]), false);
  assert.equal(hasReservationConflict({ ...candidate, duration: 90 }, [next]), true);
});
