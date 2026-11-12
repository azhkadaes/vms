import { bookingSchema, visitSchema } from "../packages/data/src/validation.ts";
import test from "node:test";
import assert from "node:assert/strict";
import { emailDomains, splitEmail, joinEmail, phoneDigits } from "../packages/data/src/contact.ts";
import {
  emailSchema,
  rejectionSchema,
  staffAppointmentSchema,
  staffAppointmentInputSchema,
} from "../packages/data/src/validation.ts";
import {
  can,
  statuses,
  bookingStatusesFor,
  scope,
  selectBookings,
  sessionFor,
  validateND,
  validateReschedule,
  today,
  canReadInternalRejection,
  canCreateAppointment,
  toVisitorBooking,
  toStaffBooking,
  type Booking,
} from "../packages/data/src/domain.ts";
const base: Booking = {
  id: "a",
  code: "88000001",
  name: "Citra Anggraini",
  organization: "Instansi Demo",
  email: "demo@example.test",
  phone: "08000000000",
  officerId: "o2",
  host: "Host Demo",
  vip: false,
  date: "2099-10-20",
  time: "10:00",
  duration: 60,
  room: "Ruang Rapat Nusantara",
  purpose: "Kunjungan demo",
  visitors: ["Citra Anggraini"],
  identityType: "KTP",
  status: "approved",
  version: 1,
  history: [],
};
test("staff creation requires explicit permission and respects PA assignment and Protocol VIP scope", () => {
  const pa = sessionFor("PA");
  const protocol = sessionFor("Protocol Officer");
  assert.equal(canCreateAppointment(pa), true);
  assert.equal(canCreateAppointment(protocol), true);
  assert.equal(canCreateAppointment(pa, "o2"), true);
  assert.equal(canCreateAppointment(pa, "o1"), false);
  assert.equal(canCreateAppointment({ ...pa, assignedOfficerIds: [] }, "o2"), false);
  assert.equal(canCreateAppointment({ ...pa, permissions: [] }, "o2"), false);
  assert.equal(canCreateAppointment(protocol, "o1"), true);
  assert.equal(canCreateAppointment(protocol, "o2"), false);
  assert.equal(canCreateAppointment(protocol, "missing"), false);
  for (const role of ["Receptionist", "Super Admin"] as const) {
    assert.equal(canCreateAppointment(sessionFor(role)), false);
    assert.equal(
      canCreateAppointment({ ...sessionFor(role), permissions: ["create"] }, "o2"),
      false,
    );
  }
});
test("staff drafts share contact and schedule rules without visitor consent impersonation", () => {
  const form = {
    officerId: "o2",
    date: "2099-12-01",
    time: "10:00",
    purpose: "Koordinasi kunjungan",
    name: "Nama Contoh",
    organization: "Instansi Contoh",
    email: "nama@agency.example",
    phone: "081234567890",
    identityType: "SIM",
    visitors: [{ name: "Anggota Contoh" }],
  };
  assert.equal(staffAppointmentSchema.safeParse(form).success, true);
  assert.equal(bookingSchema.safeParse(form).success, false);
  assert.equal(
    staffAppointmentSchema.safeParse({ ...form, phone: "+628123456789" }).success,
    false,
  );
  assert.equal(staffAppointmentSchema.safeParse({ ...form, date: "2000-01-01" }).success, false);
  assert.equal(
    staffAppointmentSchema.safeParse({ ...form, visitors: [{ name: "" }] }).success,
    false,
  );
  const input = {
    ...form,
    visitors: [form.name, "Anggota Contoh"],
    ndName: "contoh.pdf",
  };
  assert.equal(staffAppointmentInputSchema.safeParse(input).success, true);
  assert.equal(staffAppointmentInputSchema.safeParse({ ...input, visitors: [] }).success, false);
  assert.equal(
    staffAppointmentInputSchema.safeParse({
      ...input,
      visitors: ["Nama berbeda"],
    }).success,
    false,
  );
  assert.equal(
    staffAppointmentInputSchema.safeParse({
      ...input,
      visitors: Array(21).fill(form.name),
    }).success,
    false,
  );
  assert.equal(
    "createdBy" in
      toVisitorBooking({
        ...base,
        createdBy: { name: "Staf Contoh", role: "PA" },
      } as Booking),
    false,
  );
});
test("rejection requires a trimmed public message and accepts a bounded optional internal note", () => {
  assert.equal(rejectionSchema.safeParse({ rejectionReason: "    " }).success, false);
  assert.equal(rejectionSchema.safeParse({ rejectionReason: "abcd" }).success, false);
  assert.equal(rejectionSchema.safeParse({ rejectionReason: "a".repeat(1001) }).success, false);
  assert.equal(
    rejectionSchema.safeParse({
      rejectionReason: "Pesan pengaju",
      internalRejectionNote: "a".repeat(2001),
    }).success,
    false,
  );
  assert.deepEqual(
    rejectionSchema.parse({
      rejectionReason: "  Silakan ajukan jadwal lain.  ",
    }),
    {
      rejectionReason: "Silakan ajukan jadwal lain.",
      internalRejectionNote: "",
    },
  );
  assert.deepEqual(
    rejectionSchema.parse({
      rejectionReason: "a".repeat(1000),
      internalRejectionNote: "b".repeat(2000),
    }),
    {
      rejectionReason: "a".repeat(1000),
      internalRejectionNote: "b".repeat(2000),
    },
  );
});
test("internal rejection notes are limited to the PA or Protocol decision scope", () => {
  const pa = sessionFor("PA");
  assert.equal(canReadInternalRejection(pa, base), true);
  assert.equal(canReadInternalRejection(pa, { ...base, officerId: "unassigned" }), false);
  assert.equal(canReadInternalRejection(pa, { ...base, vip: true }), false);
  assert.equal(canReadInternalRejection({ ...pa, permissions: [] }, base), false);
  assert.equal(
    canReadInternalRejection(sessionFor("Protocol Officer"), {
      ...base,
      vip: true,
    }),
    true,
  );
  assert.equal(canReadInternalRejection(sessionFor("Protocol Officer"), base), false);
  assert.equal(canReadInternalRejection(sessionFor("Receptionist"), base), false);
  assert.equal(canReadInternalRejection(sessionFor("Super Admin"), base), false);
});
test("visitor projection removes private and future fields without modifying staff data", () => {
  const record = {
    ...base,
    rejectionReason: "Silakan ajukan jadwal lain.",
    internalRejectionNote: "Internal synthetic context",
    futureStaffSecret: "private",
    history: [
      {
        at: "2099-01-01",
        message: "Permohonan ditolak",
        internalNote: "private history metadata",
      },
    ],
  };
  const publicRecord = toVisitorBooking(record);
  assert.equal(publicRecord.rejectionReason, record.rejectionReason);
  assert.equal(JSON.stringify(publicRecord).includes("private"), false);
  assert.equal("internalRejectionNote" in publicRecord, false);
  assert.equal(record.internalRejectionNote, "Internal synthetic context");
  assert.equal(
    toStaffBooking(record, sessionFor("PA")).internalRejectionNote,
    record.internalRejectionNote,
  );
  assert.equal(
    "internalRejectionNote" in toStaffBooking(record, sessionFor("Receptionist")),
    false,
  );
});
test("email dropdown uses complete suffixes and preserves local parts and custom addresses", () => {
  assert.ok(emailDomains.includes("ikn.go.id"));
  assert.deepEqual(splitEmail(""), { username: "", domain: "" });
  assert.deepEqual(splitEmail("visitor"), { username: "visitor", domain: "" });
  assert.equal(joinEmail("", ""), "");
  assert.equal(joinEmail("", "gmail.com"), "@gmail.com");
  assert.equal(joinEmail("nama.anda+visit", "GMAIL.COM"), "nama.anda+visit@gmail.com");
  for (const email of ["visitor@gmail.com", "visitor@agency.example", "visitor@@gmail.com"]) {
    const parts = splitEmail(email);
    assert.equal(joinEmail(parts.username, parts.domain), email);
  }
});
test("composed email requires a valid username and complete domain before lookup or booking", () => {
  for (const email of [
    "",
    "visitor",
    "@gmail.com",
    "visitor@",
    "visitor@agency",
    "visitor@@gmail.com",
    "a..b@gmail.com",
    "a b@gmail.com",
  ]) {
    assert.equal(emailSchema.safeParse(email).success, false, email);
  }
  assert.equal(emailSchema.safeParse(joinEmail("visitor", "agency.example")).success, true);
});
test("phone paste removes non-digits and preserves leading zeroes", () => {
  assert.equal(phoneDigits("0812-3456 7890"), "081234567890");
  assert.equal(phoneDigits("+62 (812) 345-6789"), "628123456789");
  assert.equal(phoneDigits("abc08e12.34"), "081234");
});
test("booking validation accepts custom email domains and rejects non-numeric phone values", () => {
  const draft = {
    officerId: "o2",
    date: "2099-10-20",
    time: "10:00",
    purpose: "Diskusi kerja sama",
    name: "Nama Demo",
    organization: "Instansi Demo",
    email: "visitor@agency.example",
    phone: "081234567890",
    identityType: "KTP",
    visitors: [],
    ack: true,
  };
  const parsed = bookingSchema.safeParse(draft);
  assert.equal(parsed.success, true);
  if (parsed.success) assert.equal(parsed.data.phone, "081234567890");
  for (const phone of [
    "+628123456789",
    "0812 3456 7890",
    "0812abc3456",
    "0812-3456",
    "1e12345678",
  ]) {
    assert.equal(bookingSchema.safeParse({ ...draft, phone }).success, false);
  }
});
test("past start is rejected on the first step and before privacy acknowledgement", () => {
  const visit = {
    officerId: "o2",
    date: today(),
    time: "00:00",
    purpose: "Synthetic past schedule",
  };
  const first = visitSchema.safeParse(visit);
  assert.equal(first.success, false);
  if (!first.success) assert.ok(first.error.issues.some((issue) => issue.path[0] === "time"));
  const final = bookingSchema.safeParse({
    ...visit,
    name: "Nama Demo",
    organization: "Instansi Demo",
    email: "demo@example.test",
    phone: "08000000000",
    identityType: "KTP",
    visitors: [],
    ack: false,
  });
  assert.equal(final.success, false);
  if (!final.success) {
    assert.ok(final.error.issues.some((issue) => issue.path[0] === "time"));
    assert.ok(final.error.issues.some((issue) => issue.path[0] === "ack"));
  }
});
test("PA scope excludes VIP and unassigned officers; Super Admin has no implicit visit access", () => {
  const list = [base, { ...base, id: "b", vip: true }, { ...base, id: "c", officerId: "unknown" }];
  assert.deepEqual(
    scope(list, sessionFor("PA")).map((b) => b.id),
    ["a"],
  );
  assert.deepEqual(scope(list, sessionFor("Super Admin")), []);
});
test("Receptionist visibility and filters include only approved, checked-in and checked-out bookings", () => {
  const matrix = statuses.map((status) => ({ ...base, id: status, status }));
  const receptionist = sessionFor("Receptionist");
  assert.deepEqual(bookingStatusesFor(receptionist), [
    "approved",
    "checked_in",
    "checked_out",
    "missed",
  ]);
  assert.deepEqual(
    scope(matrix, receptionist).map((b) => b.status),
    ["approved", "checked_in", "checked_out"],
  );
  assert.deepEqual(scope([{ ...base, rescheduled: true, vip: true }], receptionist).length, 1);
  for (const role of ["PA", "Protocol Officer"] as const) {
    assert.deepEqual(bookingStatusesFor(sessionFor(role)), [...statuses, "missed"]);
    assert.equal(scope(matrix, sessionFor(role)).length, 6);
  }
  assert.equal(matrix.length, 6);
});
test("Receptionist check-in and checkout transitions remain restricted to the preceding state", () => {
  const receptionist = sessionFor("Receptionist");
  for (const status of statuses) {
    assert.equal(
      can(receptionist, { ...base, status }, "checkin", Date.parse("2099-10-20T09:30:00+08:00")),
      status === "approved",
      status,
    );
    assert.equal(
      can(receptionist, { ...base, status }, "checkout"),
      status === "checked_in",
      status,
    );
  }
});
test("fixed rescheduling stays available only to correct operational role on approved bookings", () => {
  assert.equal(can(sessionFor("PA"), base, "reschedule"), true);
  assert.equal(can(sessionFor("PA"), { ...base, vip: true }, "reschedule"), false);
  assert.equal(can(sessionFor("Protocol Officer"), { ...base, vip: true }, "reschedule"), true);
  assert.equal(can(sessionFor("PA"), { ...base, status: "checked_in" }, "reschedule"), false);
  assert.equal(can(sessionFor("Super Admin"), base, "reschedule"), false);
});
test("staff-created approved visits retain cancellation and rescheduling within decision scope", () => {
  const manual = {
    ...base,
    status: "approved" as const,
    createdBy: { name: "Staff Example", role: "PA" as const },
  };
  for (const action of ["cancel", "reschedule"] as const) {
    assert.equal(can(sessionFor("PA"), manual, action), true);
    assert.equal(can({ ...sessionFor("PA"), assignedOfficerIds: [] }, manual, action), false);
    assert.equal(can(sessionFor("Protocol Officer"), manual, action), false);
    assert.equal(can(sessionFor("Protocol Officer"), { ...manual, vip: true }, action), true);
    assert.equal(can(sessionFor("Super Admin"), manual, action), false);
    assert.equal(can(sessionFor("PA"), { ...manual, status: "cancelled" }, action), false);
  }
  assert.equal(can(sessionFor("PA"), { ...manual, status: "pending" }, "reschedule"), false);
  assert.equal(can(sessionFor("PA"), manual, "approve"), false);
  assert.equal(can(sessionFor("PA"), manual, "reject"), false);
});
test("default order prioritizes pending and then ascending visit time", () => {
  const list = [
    base,
    { ...base, id: "late", status: "pending" as const, time: "13:00" },
    { ...base, id: "early", status: "pending" as const, time: "09:00" },
  ];
  assert.deepEqual(
    selectBookings(list, "", "all", "priority").map((b) => b.id),
    ["early", "late", "a"],
  );
});
test("default order keeps upcoming arrivals ahead of past and closed visits", () => {
  const list: Booking[] = [
    { ...base, id: "completed", status: "checked_out", date: "2000-01-02" },
    { ...base, id: "missed", date: "2000-01-01" },
    { ...base, id: "cancelled", status: "cancelled", date: "2099-10-21" },
    { ...base, id: "later", time: "14:00" },
    { ...base, id: "inside", status: "checked_in", time: "09:00" },
    { ...base, id: "pending", status: "pending", date: "2099-10-22" },
    base,
  ];
  const expected = ["pending", "inside", "a", "later", "cancelled", "completed", "missed"];
  assert.deepEqual(
    selectBookings(list, "", "all", "priority").map((b) => b.id),
    expected,
  );
  assert.deepEqual(
    selectBookings(list, "", "all", "priority", true).map((b) => b.id),
    expected,
  );
  assert.deepEqual(
    selectBookings(list, "", "approved", "priority").map((b) => b.id),
    ["a", "later"],
  );
  assert.equal(list[0].id, "completed");
});
test("explicit time sorting remains chronological across statuses", () => {
  const list: Booking[] = [
    { ...base, id: "pending", status: "pending", time: "13:00" },
    base,
    { ...base, id: "past", date: "2000-01-01" },
  ];
  assert.deepEqual(
    selectBookings(list, "", "all", "time").map((b) => b.id),
    ["past", "a", "pending"],
  );
  assert.deepEqual(
    selectBookings(list, "", "all", "time", true).map((b) => b.id),
    ["pending", "a", "past"],
  );
});
test("search ranks representative names before organization matches", () => {
  const list = [
    { ...base, id: "organization", name: "Dewi", organization: "Citra Studio" },
    { ...base, id: "name", name: "Citra Anggraini" },
  ];
  assert.deepEqual(
    selectBookings(list, "citra", "all", "priority").map((b) => b.id),
    ["name", "organization"],
  );
  assert.equal(selectBookings(list, "citra", "rejected", "name").length, 0);
});
test("named roster and required privacy acknowledgment validated without ID photo/number", () => {
  const draft = {
    officerId: "o2",
    date: "2099-10-20",
    time: "10:00",
    purpose: "Diskusi kerja sama",
    name: "Nama Demo",
    organization: "Instansi Demo",
    email: "demo@example.test",
    phone: "08000000000",
    identityType: "Paspor",
    visitors: [],
    ack: true,
  };
  assert.equal(bookingSchema.safeParse(draft).success, true);
  assert.equal(bookingSchema.safeParse({ ...draft, visitors: [{ name: "" }] }).success, false);
  assert.equal(bookingSchema.safeParse({ ...draft, ack: false }).success, false);
  assert.equal(bookingSchema.safeParse({ ...draft, date: "2020-01-01" }).success, false);
});
test("ND rejects unsupported types, empty documents and oversize uploads", () => {
  assert.equal(validateND(null), null);
  assert.equal(validateND({ type: "application/pdf", size: 1000 }), null);
  assert.ok(validateND({ type: "image/svg+xml", size: 100 }));
  assert.ok(validateND({ type: "application/pdf", size: 0 }));
  assert.ok(validateND({ type: "application/pdf", size: 6 * 1024 * 1024 }));
});
test("rescheduling requires future time, valid duration, place and reason", () => {
  assert.equal(
    validateReschedule("2099-10-20", "10:00", 60, "Ruang Rapat Nusantara", "Koordinasi agenda"),
    null,
  );
  assert.ok(validateReschedule("2020-01-01", "10:00", 60, "Ruang Rapat Nusantara", "Alasan"));
  assert.ok(validateReschedule("2099-10-20", "10:00", 0, "Ruang Rapat Nusantara", "Alasan"));
  assert.ok(validateReschedule("2099-10-20", "10:00", 60, "Tempat tidak valid", "Alasan"));
  assert.ok(validateReschedule("2099-10-20", "10:00", 60, "Ruang Rapat Nusantara", ""));
});
