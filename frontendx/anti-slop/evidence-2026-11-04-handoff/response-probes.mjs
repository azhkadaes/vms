import assert from "node:assert/strict";
import { staffBookingResponse } from "../../packages/data/src/wire.ts";

const schedule = {
  id: "fixture-returned-id",
  date: "2099-01-01",
  time: "09:00",
  duration: 60,
  room: "Fixture room",
  status: "pending",
  version: 1,
  timing: {
    serverTime: 1,
    opensAt: 2,
    endsAt: 3,
    displayStatus: "pending",
    canCheckIn: false,
    canCheckOut: false,
    canCancel: true,
  },
};
const ordinary = {
  ...schedule,
  visitKind: "standard",
  code: "12345678",
  name: "Fixture visitor",
  organization: "Fixture organization",
  email: "visitor@example.test",
  phone: "081234567890",
  officerId: "fixture-officer",
  host: "Fixture host",
  vip: false,
  purpose: "Fixture visit",
  visitors: ["Fixture visitor"],
  identityType: "KTP",
  history: [],
};
assert.equal(staffBookingResponse.safeParse(schedule).success, true);
console.log("CONFIRMED: schedule-only ordinary staff record is accepted");
assert.equal(
  staffBookingResponse.safeParse({ ...schedule, restrictedView: "reception", name: "Fixture visitor" }).success,
  true,
);
console.log("CONFIRMED: restricted marker without visitKind accepts identity field");
const originalFetch = globalThis.fetch;
let payload;
globalThis.fetch = async () => Response.json(payload);
try {
  const { readStaffBooking, readStaffBookings } = await import("../../packages/data/src/read-model.ts");
  payload = ordinary;
  assert.equal((await readStaffBooking("fixture-requested-id")).id, "fixture-returned-id");
  console.log("CONFIRMED: detail accepts a different id from the requested id");
  payload = { items: [ordinary], total: 1, totalScoped: 1, page: 2, pageSize: 25 };
  const result = await readStaffBookings({ view: "list", page: 1, pageSize: 7 });
  assert.equal(result.page, 2);
  assert.equal(result.pageSize, 25);
  console.log("CONFIRMED: list accepts mismatched response page/pageSize");
} finally {
  globalThis.fetch = originalFetch;
}
