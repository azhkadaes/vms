import test from "node:test";
import assert from "node:assert/strict";
import { createHttpClient } from "../packages/data/src/http.ts";
import { createStaffSession } from "../packages/data/src/staff-session.ts";
import { parseSession, sessionKey } from "../packages/data/src/session-contract.ts";
import { createRepository } from "../packages/data/src/repositories.ts";
import { ApiError } from "../packages/data/src/errors.ts";
import type { VisitCommand } from "../packages/data/src/contracts.ts";
import {
  sessionFor,
  toStaffBooking,
  toVisitorBooking,
  type Booking,
} from "../packages/data/src/domain.ts";
import { bookingSchemaFor, visitSchemaFor } from "../packages/data/src/validation.ts";
import { createManagementRepository } from "../packages/data/src/management.ts";
import { integrationFieldSchema } from "../packages/data/src/management-contract.ts";
import { staffReadModelResponse } from "../packages/data/src/wire.ts";

const record: Booking = {
  id: "contract",
  code: "88001001",
  name: "Contract Example",
  organization: "Example Organization",
  email: "contract@example.test",
  phone: "00123456789",
  officerId: "o2",
  host: "Example Host",
  vip: false,
  date: "2099-08-19",
  time: "10:00",
  duration: 60,
  room: "Example Room",
  purpose: "Contract verification",
  visitors: ["Contract Example"],
  identityType: "KTP",
  status: "approved",
  version: 2,
  history: [{ at: "2099-08-19T01:00:00Z", action: "approved", message: "PA: approve" }],
};
const response = (value: unknown, status = 200, headers = {}) =>
  new Response(JSON.stringify(value), {
    status,
    headers: { "Content-Type": "application/json", ...headers },
  });

test("a session has one explicit canonical role and scope follows identity, permissions and assignments", () => {
  const canonical = { id: "account-1", active: true, ...sessionFor("Protocol Officer") };
  assert.equal(parseSession(canonical).role, "Protocol Officer");
  const reordered = { ...canonical, roles: [{ name: "PA" }, { name: "Super Admin" }] };
  assert.equal(parseSession(reordered).role, "Protocol Officer");
  assert.throws(
    () => parseSession({ ...canonical, role: undefined, roles: [{ name: "PA" }] }),
    ApiError,
  );
  assert.throws(() => parseSession({ ...canonical, permissions: [42] }), ApiError);
  assert.throws(
    () => parseSession({ ...canonical, active: false }),
    (error: unknown) => error instanceof ApiError && error.status === 401,
  );
  assert.notEqual(
    sessionKey(parseSession(canonical)),
    sessionKey(parseSession({ ...canonical, id: "account-2" })),
  );
  assert.notEqual(
    sessionKey(parseSession(canonical)),
    sessionKey(parseSession({ ...canonical, assignedOfficerIds: ["another"] })),
  );
});

test("HTTP failures keep status, fields, rate-limit advice and only authenticated 401 signals session loss", async () => {
  let lost = 0;
  const session = createStaffSession({ csrf: () => undefined, locks: undefined });
  const stop = session.onLoss(() => lost++);
  const client = createHttpClient("/v1", async () =>
    response(
      {
        message: "Wait",
        code: "RATE_LIMITED",
        fieldErrors: { email: ["Check email"], invalid: [42] },
      },
      429,
      { "Retry-After": "30" },
    ),
  );
  await assert.rejects(
    client("/public/bookings", { body: {} }),
    (error: unknown) =>
      error instanceof ApiError &&
      error.status === 429 &&
      error.retryAfter === 30 &&
      error.code === "RATE_LIMITED" &&
      error.fieldErrors?.email[0] === "Check email" &&
      !error.fieldErrors.invalid,
  );
  const unauthorized = createHttpClient("/v1", async () => response({}, 401), session);
  await assert.rejects(unauthorized("/public/lookup", { body: {} }));
  assert.equal(lost, 0);
  await assert.rejects(unauthorized("/staff/bookings", { authenticated: true }));
  assert.equal(lost, 1);
  stop();
});

test("production repository uses audience paths, cookie credentials and no caller-selected role", async () => {
  const calls: { url: string; options?: RequestInit }[] = [];
  const api = createRepository({
    fetcher: async (url, options) => {
      calls.push({ url: String(url), options });
      if (String(url).includes("/admission"))
        return response({
          payload: "server-issued-opaque-credential",
          imagePng: "cG5nLWZpeHR1cmU=",
          expiresAt: "2099-08-19T03:00:00Z",
          serverTime: Date.parse("2099-08-19T01:00:00Z"),
        });
      if (String(url).includes("/staff/"))
        return response(toStaffBooking(record, sessionFor("PA")));
      return response(toVisitorBooking(record));
    },
  });
  const publicBooking = await api.lookup(record.code, record.email);
  await api.perform(record, { type: "approve", roomId: "room-1", duration: 60 }, "PA");
  assert.equal((await api.admission(publicBooking)).imagePng, "cG5nLWZpeHR1cmU=");
  assert.deepEqual(
    calls.map((call) => call.url),
    [
      "/v1/public/lookup",
      "/v1/staff/bookings/contract/approve",
      "/v1/public/bookings/contract/admission",
    ],
  );
  for (const call of calls) {
    assert.equal(call.options?.credentials, "include");
    assert.equal(new Headers(call.options?.headers).get("X-VMS-Request"), "browser");
    assert.equal(call.url.includes("role="), false);
    assert.equal("role" in JSON.parse(String(call.options?.body)), false);
  }
  assert.equal(JSON.stringify(publicBooking.history).includes("PA:"), false);
});

test("staff writes use the booking and attendance API request shapes", async () => {
  const requests: { url: string; method?: string; body: unknown }[] = [];
  const api = createRepository({
    fetcher: async (url, options) => {
      requests.push({
        url: String(url),
        method: options?.method,
        body: JSON.parse(String(options?.body)),
      });
      return response(toStaffBooking(record, sessionFor("PA")));
    },
  });
  const restricted = {
    visitKind: "restricted" as const,
    officerId: "2",
    date: "2099-08-19",
    time: "10:00",
    roomId: "7",
  };
  await api.createStaff(restricted, "booking-reference", "Protocol Officer");
  await api.perform({ id: "12", version: 2 }, { type: "attend", direction: "in", credential: "fixture" }, "Receptionist");
  await api.perform({ id: "12", version: 3 }, { type: "attend", direction: "out", credential: "fixture" }, "Receptionist");
  assert.deepEqual(requests, [
    {
      url: "/v1/staff/bookings",
      method: "POST",
      body: { ...restricted, idempotencyKey: "booking-reference" },
    },
    {
      url: "/v1/staff/bookings/12/attendance",
      method: "POST",
      body: { action: "check_in", version: 2, credential: "fixture" },
    },
    {
      url: "/v1/staff/bookings/12/attendance",
      method: "POST",
      body: { action: "check_out", version: 3, credential: "fixture" },
    },
  ]);
});

test("malformed successful responses remain unavailable and cannot masquerade as a received submission", async () => {
  const api = createRepository({ fetcher: async () => response({ id: "only-an-id" }) });
  await assert.rejects(
    api.lookup(record.code, record.email),
    (error: unknown) => error instanceof ApiError && error.status === 502,
  );
  await assert.rejects(api.resolveSubmission("reference", record.email), ApiError);
  const invalidJson = createHttpClient("/v1", async () => new Response("<html>Unavailable</html>"));
  await assert.rejects(invalidJson("/public/bookings", { body: {} }), ApiError);
});

test("production attendance uses the attendance endpoint with an explicit action and credential", async () => {
  const api = createRepository({
    fetcher: async (url, options) => {
      assert.equal(String(url), "/v1/staff/bookings/contract/attendance");
      assert.equal(options?.method, "POST");
      const body = JSON.parse(String(options?.body));
      assert.equal(body.credential, "00123456");
      assert.equal(body.version, 2);
      assert.ok(["check_in", "check_out"].includes(body.action));
      return response(toStaffBooking(record, sessionFor("Receptionist")));
    },
  });
  for (const direction of ["in", "out"] as const)
    await api.perform(
      { id: record.id, version: 2 },
      { type: "attend", direction, credential: "00123456" },
      "Receptionist",
    );
});

test("wire validation rejects accidental restricted identity in the general reception projection", async () => {
  const api = createRepository({
    fetcher: async () =>
      response({ total: 1, totalScoped: 1, page: 1, pageSize: 25, items: [
        {
          id: "restricted",
          visitKind: "restricted",
          reference: "TRB-example",
          date: "2099-08-19",
          time: "10:00",
          duration: 60,
          room: "",
          status: "approved",
          version: 1,
          restrictedView: "reception",
          timing: toVisitorBooking(record).timing,
          name: "PRIVATE name",
        },
      ] }),
  });
  await assert.rejects(api.list("Receptionist"), ApiError);
});

test("form rules accept active backend catalog IDs and reject inactive/unknown destinations", () => {
  const catalogs = [
    { id: "real-officer-42", active: true },
    { id: "inactive-42", active: false },
  ];
  const visit = {
    officerId: "real-officer-42",
    date: "2099-08-19",
    time: "10:00",
    purpose: "Contract verification",
  };
  assert.equal(visitSchemaFor(catalogs).safeParse(visit).success, true);
  assert.equal(visitSchemaFor(catalogs).safeParse({ ...visit, officerId: "o2" }).success, false);
  assert.equal(
    visitSchemaFor(catalogs).safeParse({ ...visit, officerId: "inactive-42" }).success,
    false,
  );
  assert.equal(
    bookingSchemaFor(catalogs).safeParse({ ...record, ...visit, visitors: [], ack: true }).success,
    true,
  );
});

test("multipart requests preserve file bytes, use browser-generated boundaries and protect authenticated downloads", async () => {
  const form = new FormData();
  form.set("file", new File(["%PDF-1.7 test bytes"], "test.pdf", { type: "application/pdf" }));
  let options: RequestInit | undefined;
  const request = createHttpClient("/v1", async (_url, input) => {
    options = input;
    return response({ token: "opaque-upload-token" });
  });
  await request("/public/attachments", { body: form });
  assert.equal(options?.body, form);
  assert.equal(new Headers(options?.headers).has("Content-Type"), false);
  assert.equal(new Headers(options?.headers).get("X-VMS-Request"), "browser");
  const download = createHttpClient(
    "/v1",
    async () =>
      new Response("%PDF-1.7 test bytes", { headers: { "Content-Type": "application/pdf" } }),
  );
  assert.equal(
    (
      (await download("/staff/bookings/id/attachment", {
        authenticated: true,
        responseType: "blob",
      })) as Blob
    ).type,
    "application/pdf",
  );
});

test("photo uploads keep their tokens and validation errors separate from supporting documents", async (context) => {
  const forms: FormData[] = [];
  context.mock.method(globalThis, "fetch", async (_url: unknown, options: RequestInit) => {
    forms.push(options.body as FormData);
    if (forms.length === 3)
      return response(
        { message: "Invalid image", fieldErrors: { ndName: ["Invalid image"] } },
        422,
      );
    return response({
      token: `opaque-upload-token-${forms.length}`,
      expiresAt: new Date(Date.now() + 60_000).toISOString(),
      serverTime: Date.now(),
    });
  });
  const { createVisitFiles } = await import("../packages/data/src/visit-files.ts");
  const files = createVisitFiles();
  const memo = new File(["%PDF-1.7 memo"], "memo.pdf", { type: "application/pdf" });
  const pngSignature = Uint8Array.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
  const photo = new File([pngSignature], "visitor.png", { type: "image/png" });
  await assert.rejects(
    files.upload(memo, "public", "visitor-photo"),
    (error: unknown) => error instanceof ApiError && !!error.fieldErrors?.visitorPhotoName,
  );
  await assert.rejects(
    files.upload(
      new File([pngSignature], "memo.pdf", { type: "application/pdf" }),
      "staff",
      "attachment",
    ),
    (error: unknown) => error instanceof ApiError && !!error.fieldErrors?.ndName,
  );
  assert.equal(forms.length, 0);
  assert.equal(await files.upload(memo, "public", "attachment"), "opaque-upload-token-1");
  assert.equal(await files.upload(photo, "public", "visitor-photo"), "opaque-upload-token-2");
  assert.equal(await files.upload(photo, "public", "visitor-photo"), "opaque-upload-token-2");
  assert.equal(forms.length, 2, "retrying the same photo reuses its unexpired token");
  assert.equal((forms[0].get("file") as File).name, "memo.pdf");
  assert.equal((forms[1].get("file") as File).name, "visitor.png");
  assert.notEqual(forms[0].get("reference"), forms[1].get("reference"));
  await assert.rejects(
    files.upload(
      new File([pngSignature, "rejected"], "rejected.png", { type: "image/png" }),
      "public",
      "visitor-photo",
    ),
    (error: unknown) =>
      error instanceof ApiError &&
      error.fieldErrors?.visitorPhotoName?.[0] === "Invalid image" &&
      !error.fieldErrors?.ndName,
  );
});

test("employee updates send writable fields and a version without creation credentials", async () => {
  let submitted: unknown;
  const employee = {
    id: "staff-42",
    name: "Example Person",
    email: "person@example.test",
    role: "PA" as const,
    active: true,
    assignedOfficerIds: ["server-officer"],
    version: 4,
  };
  const api = createManagementRepository("/v1", async (_url, options) => {
    submitted = options?.body;
    return response({ ...employee, initialPassword: "NEVER RETURN THIS PASSWORD" });
  });
  const result = await api.saveEmployee(employee.id, {
    ...employee,
    phoneNumbers: ["0812345678", "+62 542 123456"],
    initialPassword: "CREATE-ONLY-PASSWORD",
    requirePasswordChange: true,
  }, 3);
  assert.equal("initialPassword" in result, false);
  assert.deepEqual(JSON.parse(String(submitted)), {
    name: employee.name,
    email: employee.email,
    roles: ["PA"],
    phoneNumbers: ["0812345678", "+62 542 123456"],
    active: true,
    assignedOfficerIds: employee.assignedOfficerIds,
    version: 3,
  });
});

test("deferred integration settings issue no production request and strip secret values", async () => {
  const api = createManagementRepository("/v1", async () => {
    assert.fail("The backend does not expose integration settings");
  });
  await assert.rejects(api.integrations(), (error) => error instanceof ApiError && error.status === 503);
  const secret = integrationFieldSchema.parse({
    key: "credential",
    label: "Credential",
    kind: "secret",
    required: true,
    configured: true,
    value: "NEVER RETURN THIS VALUE",
  });
  assert.equal(secret.value, undefined);
  assert.equal(secret.configured, true);
});

test("bounded staff responses reject overflow, impossible dates and invalid private projections", () => {
  const staff = toStaffBooking(record, sessionFor("PA"));
  const page = { items: [staff], total: 1, totalScoped: 4, page: 1, pageSize: 7 };
  assert.equal(staffReadModelResponse.safeParse(page).success, true);
  assert.equal(staffReadModelResponse.safeParse({ ...page, total: 0 }).success, false);
  assert.equal(
    staffReadModelResponse.safeParse({
      ...page,
      items: Array(101).fill(staff),
      total: 101,
      totalScoped: 101,
      pageSize: 101,
    }).success,
    false,
  );
  assert.equal(
    staffReadModelResponse.safeParse({ ...page, items: [{ ...staff, date: "2099-02-30" }] })
      .success,
    false,
  );
  assert.equal(
    staffReadModelResponse.safeParse({
      ...page,
      items: [
        {
          ...staff,
          visitKind: "restricted",
          restrictedView: "assigned-pa",
          code: undefined,
          email: undefined,
          phone: undefined,
          purpose: "PRIVATE",
          history: undefined,
        },
      ],
    }).success,
    false,
  );
});

test("visit commands map onto the production wire contract with the visit's version", async () => {
  const calls: { url: string; body: Record<string, unknown> }[] = [];
  const api = createRepository({
    fetcher: async (url, options) => {
      calls.push({ url: String(url), body: JSON.parse(String(options?.body)) });
      return response(toStaffBooking(record, sessionFor("PA")));
    },
  });
  const visit = { id: "v1", version: 7 };
  const cases: [VisitCommand, string, Record<string, unknown>][] = [
    [
      { type: "approve", roomId: "room-2", duration: 45 },
      "approve",
      { roomId: "room-2", duration: 45, version: 7 },
    ],
    [
      { type: "reschedule", date: "2099-09-01", time: "09:00", roomId: null, reason: "Rapat" },
      "reschedule",
      { date: "2099-09-01", time: "09:00", roomId: null, reason: "Rapat", version: 7 },
    ],
    [
      { type: "reject", rejectionReason: "Tidak sesuai" },
      "reject",
      { rejectionReason: "Tidak sesuai", version: 7 },
    ],
    [{ type: "cancel", reason: "Batal rapat" }, "cancel", { reason: "Batal rapat", version: 7 }],
    [
      { type: "attend", direction: "out", credential: "00123456" },
      "attendance",
      { action: "check_out", credential: "00123456", version: 7 },
    ],
  ];
  for (const [command, operation, body] of cases) {
    await api.perform(visit, command, "PA");
    const call = calls.at(-1)!;
    assert.equal(call.url, `/v1/staff/bookings/v1/${operation}`);
    assert.deepEqual(call.body, body);
  }
});

test("staff visits carry the server's allowedActions and nothing else decides them", async () => {
  const visit = toStaffBooking(record, sessionFor("PA"), { allowedActions: ["approve", "reject"] });
  const parse = (body: unknown) =>
    createRepository({ fetcher: async () => response(body) }).resolve("00123456", "PA");
  assert.deepEqual((await parse(visit)).allowedActions, ["approve", "reject"]);
  const { allowedActions: _omitted, ...missing } = visit;
  await assert.rejects(parse(missing), (error: unknown) => error instanceof ApiError && error.status === 502);
  await assert.rejects(
    parse({ ...visit, allowedActions: ["delete"] }),
    (error: unknown) => error instanceof ApiError && error.status === 502,
  );
});
