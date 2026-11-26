import test from "node:test";
import { can } from "../tooling/preview-policy.ts";
import assert from "node:assert/strict";
import {
  restrictedAppointmentSchema,
  staffAppointmentInputSchema,
  bookingSchema,
} from "../packages/data/src/validation.ts";
import {
  scope,
  sessionFor,
  canCreateAppointment,
  toVisitorBooking,
  toStaffBooking,
  generalReportRecords,
  officers,
  type StoredBooking,
} from "../packages/data/src/domain.ts";

const minimal = { visitKind: "restricted" as const, date: "2099-08-01", time: "10:00" };
const arrival = Date.parse("2099-08-01T09:30:00+08:00");
function booking(): StoredBooking {
  return {
    ...minimal,
    id: "private",
    code: "88000900",
    name: "Private Name",
    organization: "Private Organization",
    serviceId: "0012345678",
    officerId: "o2",
    email: "",
    phone: "",
    purpose: "",
    identityType: "",
    visitors: [],
    host: "Private Host",
    vip: false,
    room: "",
    duration: 60,
    status: "approved",
    version: 1,
    history: [],
  };
}

test("Terbatas requires date/time and permits only four optional fields, preserving leading zeroes", () => {
  assert.deepEqual(restrictedAppointmentSchema.parse(minimal), minimal);
  assert.equal(staffAppointmentInputSchema.safeParse(minimal).success, true);
  assert.equal(bookingSchema.safeParse(minimal).success, false);
  for (const extra of [
    { date: "" },
    { time: "" },
    { date: "2099-02-30" },
    { time: "24:00" },
    { email: "private@example.test" },
    { phone: "08123456789" },
    { purpose: "Former field" },
    { visitors: ["Former roster"] },
    { ndName: "private.pdf" },
    { identityType: "KTP" },
    { serviceId: "x".repeat(101) },
  ])
    assert.equal(
      restrictedAppointmentSchema.safeParse({ ...minimal, ...extra }).success,
      false,
      JSON.stringify(extra),
    );
  assert.equal(restrictedAppointmentSchema.parse({ ...minimal, name: "   " }).name, undefined);
  assert.equal(
    restrictedAppointmentSchema.parse({ ...minimal, serviceId: " 001234 " }).serviceId,
    "001234",
  );
});
test("only Protocol creates Terbatas for any active host or no host", () => {
  const protocol = sessionFor("Protocol Officer");
  for (const id of [undefined, "", ...officers.map((o) => o.id)])
    assert.equal(canCreateAppointment(protocol, id, "restricted"), true);
  assert.equal(canCreateAppointment(protocol, "o2"), false);
  assert.equal(canCreateAppointment(protocol, "missing", "restricted"), false);
  const previous = officers[1].active;
  try {
    officers[1].active = false;
    assert.equal(canCreateAppointment(protocol, "o2", "restricted"), false);
  } finally {
    officers[1].active = previous;
  }
  for (const role of ["PA", "Receptionist", "Super Admin"] as const)
    assert.equal(
      canCreateAppointment(
        { ...sessionFor(role), permissions: ["bookings", "create"] },
        "o2",
        "restricted",
      ),
      false,
    );
});
test("assigned PA and Super Admin read Terbatas without decision authority; reception retains its lifecycle scope", () => {
  for (const vip of [false, true])
    for (const officerId of ["", "o1", "o2"]) {
      const b = { ...booking(), vip, officerId };
      assert.equal(scope([b], sessionFor("PA")).length, officerId === "o2" ? 1 : 0);
      assert.equal(scope([b], sessionFor("Super Admin")).length, 1);
      assert.equal(
        can(sessionFor("Protocol Officer"), { ...b, status: "pending" }, "approve"),
        true,
      );
      for (const role of ["PA", "Super Admin"] as const) {
        const actor = {
          ...sessionFor(role),
          permissions: ["bookings", "approve", "reschedule", "checkin"],
        };
        assert.equal(can(actor, { ...b, status: "pending" }, "approve", arrival), false);
        assert.equal(can(actor, b, "reschedule", arrival), false);
        assert.equal(can(actor, b, "checkin", arrival), false);
      }
      assert.equal(can(sessionFor("Receptionist"), b, "checkin", arrival), true);
      assert.throws(() => toVisitorBooking(b));
    }
  for (const status of ["pending", "rejected", "cancelled"] as const)
    assert.equal(scope([{ ...booking(), status }], sessionFor("Receptionist")).length, 0);
});
test("Terbatas projections omit private fields instead of sending blanks or hiding them in the UI", () => {
  const b = {
    ...booking(),
    email: "private@example.test",
    phone: "08123456789",
    purpose: "PRIVATE_PURPOSE",
    ndName: "private.pdf",
    internalRejectionNote: "PRIVATE_NOTE",
    futureSecret: "FUTURE_SECRET",
    history: [{ at: "2099-08-01", message: "PRIVATE_HISTORY" }],
  };
  const reception = toStaffBooking(b, sessionFor("Receptionist"));
  for (const key of [
    "name",
    "organization",
    "serviceId",
    "serviceIdLast4",
    "host",
    "officerId",
    "code",
    "history",
    "createdBy",
    "email",
    "phone",
    "purpose",
    "ndName",
    "internalRejectionNote",
    "futureSecret",
  ])
    assert.equal(key in reception, false, key);
  const arrivalView = toStaffBooking(b, sessionFor("Receptionist"), { arrival: true });
  assert.equal(arrivalView.name, b.name);
  assert.equal(arrivalView.host, b.host);
  assert.equal(arrivalView.serviceIdLast4, "5678");
  for (const key of [
    "organization",
    "serviceId",
    "code",
    "history",
    "email",
    "internalRejectionNote",
  ])
    assert.equal(key in arrivalView, false, key);
  const pa = toStaffBooking(b, sessionFor("PA"));
  assert.equal(pa.organization, b.organization);
  assert.equal(pa.serviceIdLast4, "5678");
  for (const key of ["serviceId", "code", "history", "internalRejectionNote"])
    assert.equal(key in pa, false, key);
  assert.equal(toStaffBooking(b, sessionFor("Protocol Officer")).serviceId, "0012345678");
  assert.equal(toStaffBooking(b, sessionFor("Super Admin")).serviceId, "0012345678");
  assert.equal("internalRejectionNote" in toStaffBooking(b, sessionFor("Super Admin")), false);
  assert.throws(() => toStaffBooking({ ...b, officerId: "o1" }, sessionFor("PA")));
  assert.deepEqual(
    generalReportRecords([b, { ...b, id: "ordinary", visitKind: "standard" as const }]).map(
      (b) => b.id,
    ),
    ["ordinary"],
  );
});
