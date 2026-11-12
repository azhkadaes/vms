import test from "node:test";
import assert from "node:assert/strict";
import { parseSession, sessionKey } from "../packages/data/src/session-contract.ts";
import { employeeSchema, employeeInputSchema } from "../packages/data/src/management-contract.ts";
import {
  toStaffBooking,
  toVisitorBooking,
  sessionFor,
  type StoredBooking,
} from "../packages/data/src/domain.ts";
import { publicBookingResponse, staffBookingResponse } from "../packages/data/src/wire.ts";

test("multi-role sessions stay pending without authority until server selection", () => {
  const pending = {
    active: true,
    id: "1",
    name: "Synthetic employee",
    role: "",
    permissions: [],
    assignedOfficerIds: [],
    availableRoles: ["PA", "Receptionist"],
    roleSelectionRequired: true,
    roleRevision: 0,
  };
  assert.equal(parseSession(pending).roleSelectionRequired, true);
  assert.throws(() => parseSession({ ...pending, permissions: ["approve"] }));
  assert.throws(() => parseSession({ ...pending, assignedOfficerIds: ["1"] }));
  assert.throws(() => parseSession({ ...pending, availableRoles: ["invented"] }));
  const selected = { ...pending, role: "PA", roleSelectionRequired: false, roleRevision: 1 };
  assert.equal(parseSession(selected).role, "PA");
  assert.throws(() => parseSession({ ...selected, availableRoles: ["Receptionist"] }));
  assert.notEqual(
    sessionKey(parseSession(selected)),
    sessionKey(parseSession({ ...selected, roleRevision: 2 })),
  );
});

test("employee contracts retain multiple roles and optional profile fields without requiring a singular role", () => {
  const employee = {
    id: "1",
    name: "Synthetic employee",
    email: "synthetic@example.test",
    roles: ["PA", "Receptionist"],
    active: true,
    assignedOfficerIds: ["1"],
    version: 1,
    nip: "000123",
    jobTitle: "Synthetic title",
  };
  const parsed = employeeSchema.parse({ ...employee, initialPassword: "private" });
  assert.deepEqual(parsed.roles, employee.roles);
  assert.equal(parsed.nip, "000123");
  assert.equal("initialPassword" in parsed, false);
  const input = employeeInputSchema(12, true);
  assert.equal(input.safeParse(employee).success, true);
  assert.equal(input.safeParse({ ...employee, roles: [] }).success, false);
});

test("note projections retain visitor instructions and exclude coordination from public and reception", () => {
  const record: StoredBooking = {
    id: "1",
    code: "00123456",
    name: "Synthetic visitor",
    organization: "Synthetic org",
    email: "visitor@example.test",
    phone: "0812345678",
    officerId: "o2",
    host: "Synthetic host",
    vip: false,
    date: "2099-01-01",
    time: "10:00",
    duration: 60,
    room: "Synthetic room",
    purpose: "Synthetic visit",
    visitors: ["Synthetic visitor"],
    identityType: "KTP",
    status: "approved",
    version: 1,
    history: [],
    visitorInstructions: "Enter through reception",
    coordinationNote: "Staff coordination only",
  };
  const visitor = publicBookingResponse.parse({
    ...toVisitorBooking(record),
    coordinationNote: record.coordinationNote,
  });
  assert.equal(visitor.visitorInstructions, record.visitorInstructions);
  assert.equal("coordinationNote" in visitor, false);
  assert.equal(toStaffBooking(record, sessionFor("Receptionist")).coordinationNote, undefined);
  assert.equal(toStaffBooking(record, sessionFor("PA")).coordinationNote, record.coordinationNote);
  const restricted = { ...record, visitKind: "restricted" as const };
  const limited = toStaffBooking(restricted, sessionFor("Receptionist"));
  assert.equal(
    staffBookingResponse.safeParse({ ...limited, coordinationNote: "leak" }).success,
    false,
  );
});
