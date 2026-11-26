import test from "node:test";
import { can } from "../tooling/preview-policy.ts";
import assert from "node:assert/strict";
import {
  getVisitTiming,
  sessionFor,
  statuses,
  selectBookings,
  type StaffBooking,
} from "../packages/data/src/domain.ts";

const visit: StaffBooking = {
  allowedActions: [],
  id: "timed",
  date: "2099-08-19",
  time: "10:00",
  duration: 60,
  room: "",
  status: "approved",
  version: 2,
  officerId: "o2",
};
const time = (clock: string) => Date.parse(`2099-08-19T${clock}+08:00`);
for (const [clock, allowed, display] of [
  ["09:29:59.999", false, "approved"],
  ["09:30:00.000", true, "approved"],
  ["10:00:00.000", true, "approved"],
  ["11:00:00.000", true, "approved"],
  ["11:00:00.001", false, "missed"],
] as const)
  test(`arrival at ${clock} WITA: ${display}, allowed=${allowed}`, () => {
    const timing = getVisitTiming(visit, time(clock));
    assert.equal(timing.canCheckIn, allowed);
    assert.equal(timing.displayStatus, display);
    assert.equal(can(sessionFor("Receptionist"), visit, "checkin", time(clock)), allowed);
  });
test("Missed visits can be rescheduled by authorized staff for seven days after the scheduled end", () => {
  const missed = { ...visit, timing: getVisitTiming(visit, time("11:00:00.001")) };
  assert.equal(visit.status, "approved");
  assert.equal(statuses.length, 6);
  assert.equal(can(sessionFor("PA"), missed, "reschedule"), true);
  assert.equal(can(sessionFor("Receptionist"), missed, "reschedule"), false);
  const deadline = Date.parse("2099-08-26T11:00:00.000+08:00");
  assert.equal(can(sessionFor("PA"), visit, "reschedule", deadline), true);
  assert.equal(can(sessionFor("PA"), visit, "reschedule", deadline + 1), false);
  assert.deepEqual(selectBookings([missed], "", "approved", "time"), []);
  assert.equal(selectBookings([missed], "", "missed", "time").length, 1);
});
test("checked-in visitors may check out after the scheduled end and never become Missed", () => {
  const checkedIn = { ...visit, status: "checked_in" as const };
  const nextDay = time("11:00:00") + 86400_000;
  assert.equal(getVisitTiming(checkedIn, nextDay).displayStatus, "checked_in");
  assert.equal(can(sessionFor("Receptionist"), checkedIn, "checkout", nextDay), true);
  assert.equal(can(sessionFor("Receptionist"), checkedIn, "checkin", nextDay), false);
});
test("invalid schedule data never grants check-in", () => {
  assert.equal(getVisitTiming({ ...visit, time: "invalid" }, time("10:00:00")).canCheckIn, false);
});
