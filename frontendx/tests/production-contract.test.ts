import test from "node:test";
import assert from "node:assert/strict";
import { createHttpClient, onAuthenticationLoss } from "../packages/data/src/http.ts";
import { parseSession, sessionKey } from "../packages/data/src/session-contract.ts";
import { createRepository } from "../packages/data/src/repositories.ts";
import { ApiError } from "../packages/data/src/errors.ts";
import {
  sessionFor,
  toStaffBooking,
  toVisitorBooking,
  type Booking,
} from "../packages/data/src/domain.ts";
import { bookingSchemaFor, visitSchemaFor } from "../packages/data/src/validation.ts";
import { createManagementRepository } from "../packages/data/src/management.ts";
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
  const stop = onAuthenticationLoss(() => lost++);
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
  const unauthorized = createHttpClient("/v1", async () => response({}, 401));
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
          expiresAt: "2099-08-19T03:00:00Z",
          serverTime: Date.parse("2099-08-19T01:00:00Z"),
        });
      if (String(url).includes("/staff/"))
        return response(toStaffBooking(record, sessionFor("PA")));
      return response(toVisitorBooking(record));
    },
  });
  const publicBooking = await api.lookup(record.code, record.email);
  await api.action(record.id, "approve", { version: 1, room: "Example Room", duration: 60 }, "PA");
  await api.admission(publicBooking);
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

test("wire validation rejects accidental restricted identity in the general reception projection", async () => {
  const api = createRepository({
    fetcher: async () =>
      response([
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
      ]),
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

test("management contracts strip returned passwords and secret values and retain optimistic versions", async () => {
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
  const api = createManagementRepository("/v1", async (url, options) => {
    submitted = options?.body;
    return response(
      String(url).endsWith("/integrations")
        ? [
            {
              id: "email",
              name: "Email",
              description: "Email service",
              enabled: true,
              health: "ready",
              version: 1,
              fields: [
                {
                  key: "credential",
                  label: "Credential",
                  kind: "secret",
                  required: true,
                  configured: true,
                  value: "NEVER RETURN THIS VALUE",
                },
              ],
            },
          ]
        : { ...employee, initialPassword: "NEVER RETURN THIS PASSWORD" },
    );
  });
  const result = await api.saveEmployee(employee.id, { ...employee }, 3);
  assert.equal("initialPassword" in result, false);
  assert.equal(JSON.parse(String(submitted)).version, 3);
  const integrations = await api.integrations();
  assert.equal(integrations[0].fields[0].value, undefined);
  assert.equal(integrations[0].fields[0].configured, true);
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
