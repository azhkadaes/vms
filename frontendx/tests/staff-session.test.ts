import test from "node:test";
import assert from "node:assert/strict";
import { sessionFor, type Role } from "../packages/data/src/domain.ts";
import { ApiError } from "../packages/data/src/errors.ts";
import { createHttpClient } from "../packages/data/src/http.ts";
import { createStaffSession } from "../packages/data/src/staff-session.ts";

const serverTime = Date.parse("2099-08-19T01:00:00Z");
const json = (value: unknown, status = 200) =>
  new Response(JSON.stringify(value), {
    status,
    headers: { "Content-Type": "application/json" },
  });
function grant(token: string, role: Role = "PA", lifetimeMs = 5 * 60_000) {
  return {
    accessToken: token,
    accessExpiresAt: new Date(serverTime + lifetimeMs).toISOString(),
    session: { id: "account-1", active: true, ...sessionFor(role), timing: { serverTime } },
  };
}

type Call = { path: string; method: string; token: string | null; csrf: string | null; body: unknown };
/** A fake backend: `routes` answers by path; every call is recorded. */
function harness(routes: Record<string, (call: Call, count: number) => Response | Promise<Response>>) {
  let clock = 0;
  let cookie: string | undefined = "csrf-1";
  const calls: Call[] = [];
  const counts = new Map<string, number>();
  const fetcher = (async (url: string, options: RequestInit = {}) => {
    const headers = new Headers(options.headers);
    const call: Call = {
      path: String(url).replace(/^\/v1/, ""),
      method: options.method ?? "GET",
      token: headers.get("Authorization"),
      csrf: headers.get("X-CSRF-Token"),
      body: options.body ? JSON.parse(String(options.body)) : undefined,
    };
    calls.push(call);
    const count = (counts.get(call.path) ?? 0) + 1;
    counts.set(call.path, count);
    const route = routes[call.path];
    if (!route) throw new Error(`unexpected ${call.path}`);
    return route(call, count);
  }) as typeof fetch;
  let id = 0;
  const session = createStaffSession({
    fetcher,
    now: () => clock,
    csrf: () => cookie,
    locks: undefined,
    newId: () => `refresh-${++id}`,
  });
  const request = createHttpClient("/v1", fetcher, session);
  return {
    session,
    request,
    calls,
    advance: (ms: number) => void (clock += ms),
    setCookie: (value: string | undefined) => void (cookie = value),
  };
}

test("concurrent requests without access share one refresh and both carry its token", async () => {
  const env = harness({
    "/staff/session/refresh": () => json(grant("token-1")),
    "/staff/bookings": () => json([]),
  });
  await Promise.all([
    env.request("/staff/bookings", { authenticated: true }),
    env.request("/staff/bookings", { authenticated: true }),
  ]);
  const refreshes = env.calls.filter((call) => call.path === "/staff/session/refresh");
  assert.equal(refreshes.length, 1);
  assert.equal(refreshes[0].csrf, "csrf-1");
  assert.deepEqual(
    env.calls.filter((call) => call.path === "/staff/bookings").map((call) => call.token),
    ["Bearer token-1", "Bearer token-1"],
  );
});

test("access about to expire is refreshed before the request, on the injected clock", async () => {
  let issued = 0;
  const env = harness({
    "/staff/sessions": () => json(grant(`token-${++issued}`, "PA", 60_000)),
    "/staff/session/refresh": () => json(grant(`token-${++issued}`, "PA", 60_000)),
    "/staff/bookings": () => json([]),
  });
  await env.session.signIn({ email: "pa@example.test", password: "secret" });
  await env.request("/staff/bookings", { authenticated: true });
  env.advance(59_500);
  await env.request("/staff/bookings", { authenticated: true });
  assert.deepEqual(
    env.calls.map((call) => [call.path, call.token]),
    [
      ["/staff/sessions", null],
      ["/staff/bookings", "Bearer token-1"],
      ["/staff/session/refresh", null],
      ["/staff/bookings", "Bearer token-2"],
    ],
  );
});

test("a rejected bearer refreshes and retries once; a second rejection is not session loss", async () => {
  let lost = 0;
  const env = harness({
    "/staff/sessions": () => json(grant("stale")),
    "/staff/session/refresh": () => json(grant("fresh")),
    "/staff/bookings": () => json({ message: "expired" }, 401),
  });
  env.session.onLoss(() => lost++);
  await env.session.signIn({ email: "pa@example.test", password: "secret" });
  await assert.rejects(
    env.request("/staff/bookings", { authenticated: true }),
    (error: unknown) => error instanceof ApiError && error.status === 503,
  );
  assert.deepEqual(
    env.calls.filter((call) => call.path === "/staff/bookings").map((call) => call.token),
    ["Bearer stale", "Bearer fresh"],
  );
  assert.equal(lost, 0);
});

test("a refused refresh ends the session and notifies listeners", async () => {
  let lost = 0;
  const env = harness({
    "/staff/sessions": () => json(grant("token-1", "PA", 60_000)),
    "/staff/session/refresh": () => json({ message: "gone" }, 401),
  });
  env.session.onLoss(() => lost++);
  await env.session.signIn({ email: "pa@example.test", password: "secret" });
  env.advance(60_000);
  await assert.rejects(
    env.request("/staff/bookings", { authenticated: true }),
    (error: unknown) => error instanceof ApiError && error.status === 401,
  );
  assert.equal(lost, 1);
});

test("a refresh that reveals another role reports it and refuses the pending request", async () => {
  const changes: string[] = [];
  const env = harness({
    "/staff/sessions": () => json(grant("token-1", "PA", 60_000)),
    "/staff/session/refresh": () => json(grant("token-2", "Protocol Officer")),
    "/staff/session": () =>
      json({ id: "account-1", active: true, ...sessionFor("Protocol Officer") }),
    "/staff/bookings": () => json([]),
  });
  env.session.onAuthorityChange((session) => changes.push(session.role));
  await env.session.signIn({ email: "pa@example.test", password: "secret" });
  env.advance(60_000);
  await assert.rejects(
    env.request("/staff/bookings", { authenticated: true }),
    (error: unknown) => error instanceof ApiError && error.status === 409,
  );
  assert.deepEqual(changes, ["Protocol Officer"]);
  assert.equal(env.calls.filter((call) => call.path === "/staff/bookings").length, 0);
  assert.equal((await env.session.current()).role, "Protocol Officer");
});

test("an uncertain refresh is retried with the same refresh request ID", async () => {
  const env = harness({
    "/staff/session/refresh": (_call, count) =>
      count === 1 ? json({ message: "unavailable" }, 503) : json(grant("token-1")),
  });
  assert.equal((await env.session.current()).role, "PA");
  assert.deepEqual(
    env.calls.map((call) => call.body),
    [{ refreshRequestId: "refresh-1" }, { refreshRequestId: "refresh-1" }],
  );
});

test("a CSRF cookie rotated by another tab forces a refresh and refuses the request it interrupted", async () => {
  const env = harness({
    "/staff/sessions": () => json(grant("token-1")),
    "/staff/session/refresh": () => json(grant("token-2")),
    "/staff/bookings": () => json([]),
  });
  await env.session.signIn({ email: "pa@example.test", password: "secret" });
  env.setCookie("csrf-2");
  await assert.rejects(
    env.request("/staff/bookings", { authenticated: true }),
    (error: unknown) => error instanceof ApiError && error.status === 409,
  );
  await env.request("/staff/bookings", { authenticated: true });
  assert.deepEqual(
    env.calls.slice(1).map((call) => [call.path, call.csrf ?? call.token]),
    [
      ["/staff/session/refresh", "csrf-2"],
      ["/staff/bookings", "Bearer token-2"],
    ],
  );
});

test("sign-out sends the CSRF cookie and forgets access so the next request refreshes", async () => {
  const env = harness({
    "/staff/sessions": () => json(grant("token-1")),
    "/staff/session": () => new Response(null, { status: 204 }),
    "/staff/session/refresh": () => json(grant("token-2")),
    "/staff/bookings": () => json([]),
  });
  await env.session.signIn({ email: "pa@example.test", password: "secret" });
  await env.session.signOut();
  await env.request("/staff/bookings", { authenticated: true });
  assert.deepEqual(
    env.calls.map((call) => [call.method, call.path, call.csrf]),
    [
      ["POST", "/staff/sessions", "csrf-1"],
      ["DELETE", "/staff/session", "csrf-1"],
      ["POST", "/staff/session/refresh", "csrf-1"],
      ["GET", "/staff/bookings", null],
    ],
  );
});

test("a client without a session sends authenticated requests unchanged", async () => {
  const seen: (string | null)[] = [];
  const request = createHttpClient("/__preview", (async (_url: string, options: RequestInit) => {
    seen.push(new Headers(options.headers).get("Authorization"));
    return json({ ok: true });
  }) as typeof fetch);
  await request("/staff/bookings", { authenticated: true });
  assert.deepEqual(seen, [null]);
});
