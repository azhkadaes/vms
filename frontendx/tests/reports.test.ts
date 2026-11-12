import test from "node:test";
import assert from "node:assert/strict";
import {
  aggregateVisits,
  summaryCsv,
  summaryFilterSchema,
  summaryResponseSchema,
  reportStatuses,
  type VisitSummary,
} from "../packages/data/src/reports.ts";
import { sessionFor, type Booking } from "../packages/data/src/domain.ts";

const now = Date.parse("2099-08-19T09:00:00+08:00");
const filter = { from: "2099-08-01", to: "2099-08-31", status: "all" as const };
const base: Booking = {
  id: "ordinary",
  code: "88000999",
  name: "PRIVATE NAME",
  organization: "PRIVATE ORG",
  email: "private@example.test",
  phone: "00123456789",
  officerId: "o2",
  host: "PRIVATE HOST",
  vip: false,
  date: "2099-08-19",
  time: "10:00",
  duration: 60,
  room: "",
  purpose: "PRIVATE PURPOSE",
  visitors: ["PRIVATE ROSTER"],
  identityType: "KTP",
  status: "approved",
  version: 1,
  history: [],
};
const records = [
  base,
  { ...base, id: "vip", vip: true, officerId: "o1", status: "pending" as const },
  { ...base, id: "unassigned", officerId: "o4", status: "checked_in" as const },
  { ...base, id: "restricted", visitKind: "restricted" as const },
  { ...base, id: "missed", date: "2099-08-18" },
];
test("report ranges validate calendar, order, supported status and the inclusive 366-day bound", () => {
  for (const extra of [
    { from: "invalid" },
    { to: "2099-02-30" },
    { from: "2099-09-01" },
    { status: "private" },
    { from: "2099-01-01", to: "2100-01-02" },
  ])
    assert.equal(summaryFilterSchema.safeParse({ ...filter, ...extra }).success, false);
  assert.equal(
    summaryFilterSchema.safeParse({ ...filter, from: "2099-01-01", to: "2100-01-01" }).success,
    true,
  );
});
test("Admin organization summaries do not require ordinary visit detail permission and exclude restricted records", () => {
  const session = { ...sessionFor("Super Admin"), permissions: ["reports"] };
  const summary = aggregateVisits(records, session, filter, now);
  assert.equal(summary.total, 4);
  assert.equal(summary.counts.approved, 1);
  assert.equal(summary.counts.missed, 1);
  assert.equal(summary.scope, "organization");
  for (const secret of [
    base.name,
    base.email,
    base.organization,
    base.code,
    base.host,
    base.purpose,
    "PRIVATE ROSTER",
  ])
    assert.equal(JSON.stringify(summary).includes(secret), false, secret);
});
test("PA aggregate scope follows current assignments without requiring booking-detail permission", () => {
  const session = { ...sessionFor("PA"), permissions: ["reports"], assignedOfficerIds: ["o2"] };
  assert.equal(aggregateVisits(records, session, filter, now).total, 2);
  assert.equal(
    aggregateVisits(records, { ...session, assignedOfficerIds: [] }, filter, now).total,
    0,
  );
  assert.equal(
    aggregateVisits(records, { ...session, assignedOfficerIds: ["o4"] }, filter, now).total,
    1,
  );
  assert.throws(() =>
    aggregateVisits(records, { ...session, permissions: ["bookings"] }, filter, now),
  );
});
test("Protocol preserves its authorized ordinary scope; explicitly authorized reception remains limited to attendance states", () => {
  assert.equal(aggregateVisits(records, sessionFor("Protocol Officer"), filter, now).total, 4);
  assert.equal(
    aggregateVisits(
      records,
      { ...sessionFor("Receptionist"), permissions: ["reports"] },
      filter,
      now,
    ).total,
    3,
  );
});
test("Missed is a mutually exclusive display count and status/date filters cannot duplicate visits", () => {
  const summary = aggregateVisits(
    records,
    sessionFor("Super Admin"),
    { ...filter, status: "missed" },
    now,
  );
  assert.equal(summary.total, 1);
  assert.equal(summary.counts.missed, 1);
  assert.equal(summary.counts.approved, 0);
  assert.equal(
    reportStatuses.reduce((sum, status) => sum + summary.counts[status], 0),
    1,
  );
  assert.equal(
    aggregateVisits(records, sessionFor("Super Admin"), { ...filter, to: "2099-08-17" }, now).total,
    0,
  );
});
test("summary CSV contains only period/status/counts and schema rejects inconsistent totals or unexpected record fields", () => {
  const summary: VisitSummary = {
    ...aggregateVisits(records, sessionFor("Super Admin"), filter, now),
    revision: "a".repeat(64),
  };
  assert.equal(summaryResponseSchema.safeParse(summary).success, true);
  assert.equal(summaryResponseSchema.safeParse({ ...summary, total: 999 }).success, false);
  assert.equal(summaryResponseSchema.safeParse({ ...summary, records: [base] }).success, false);
  const csv = summaryCsv(summary);
  assert.ok(csv.includes('"Tanggal awal","Tanggal akhir","Cakupan","Status","Jumlah"'));
  assert.equal(csv.split("\r\n").length, 8);
  for (const secret of [base.name, base.email, base.organization, base.code, base.host])
    assert.equal(csv.includes(secret), false, secret);
});
