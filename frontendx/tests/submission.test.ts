import test from "node:test";
import assert from "node:assert/strict";
import { ApiError } from "../packages/data/src/errors.ts";
import {
  BookingSubmission,
  type SubmissionPhase,
  type SubmissionResolution,
  type SubmissionStore,
} from "../packages/data/src/submission.ts";
import {
  publicBookingInputSchema,
  publicBookingCreateSchema,
} from "../packages/data/src/validation.ts";
import { officers, type PublicBooking } from "../packages/data/src/domain.ts";

const input = () =>
  publicBookingInputSchema.parse({
    officerId: "o2",
    date: "2099-08-19",
    time: "10:00",
    purpose: "Visit for coordination",
    name: "Visitor Example",
    organization: "Example Agency",
    email: "visitor@example.test",
    phone: "00123456789",
    identityType: "KTP",
    visitors: ["Visitor Example"],
    ack: true,
  });
const booking = (): PublicBooking => ({
  ...input(),
  id: "booking-1",
  code: "88000100",
  host: "Host Example",
  vip: false,
  room: "",
  duration: 60,
  status: "pending",
  version: 1,
  history: [],
});
function store(initial: string | null = null) {
  let value = initial;
  const adapter: SubmissionStore = {
    read: () => value,
    write: (key) => {
      value = key;
    },
    clear: () => {
      value = null;
    },
  };
  return { adapter, read: () => value };
}
const ignored = () => {};

test("public wire normalization preserves identifiers and rejects invalid roster, consent and optional filename", () => {
  const normalized = publicBookingInputSchema.parse({
    ...input(),
    name: " Visitor Example ",
    visitors: [" Visitor Example "],
    email: " VISITOR@EXAMPLE.TEST ",
    forged: "secret",
  });
  assert.equal(normalized.email, "visitor@example.test");
  assert.equal(normalized.phone, "00123456789");
  assert.equal(normalized.name, "Visitor Example");
  assert.equal("forged" in normalized, false);
  for (const changed of [
    { ack: false },
    { ack: undefined },
    { visitors: [] },
    { visitors: ["Other Representative"] },
    { visitors: ["Visitor Example", ""] },
    { visitors: Array(21).fill("Visitor Example") },
    { ndName: "" },
    { ndName: "x".repeat(256) },
  ])
    assert.equal(
      publicBookingInputSchema.safeParse({ ...input(), ...changed }).success,
      false,
      JSON.stringify(changed),
    );
});
test("new creation checks service time and active officer while static normalization permits historical replay", () => {
  const now = Date.parse("2099-08-19T10:00:00+08:00");
  assert.equal(publicBookingCreateSchema(now).safeParse(input()).success, false);
  assert.equal(publicBookingInputSchema.safeParse(input()).success, true);
  const previous = officers[1].active;
  try {
    officers[1].active = false;
    assert.equal(publicBookingCreateSchema(now - 60_000).safeParse(input()).success, false);
    assert.equal(publicBookingInputSchema.safeParse(input()).success, true);
  } finally {
    officers[1].active = previous;
  }
});
test("a lost response retries once with the same normalized snapshot and reference", async () => {
  const saved = store();
  const calls: { key: string; payload: string }[] = [];
  const phases: SubmissionPhase[] = [];
  const form = input();
  const controller = new BookingSubmission(
    {
      async create(value, key) {
        calls.push({ key, payload: JSON.stringify(value) });
        if (calls.length === 1) {
          form.name = "Edited during request";
          throw new TypeError("fetch failed");
        }
        return booking();
      },
      async resolveSubmission() {
        assert.fail("Successful retry needs no resolution request");
      },
    },
    saved.adapter,
  );
  assert.equal((await controller.submit(form, (phase) => phases.push(phase))).id, "booking-1");
  assert.equal(calls.length, 2);
  assert.deepEqual(calls[0], calls[1]);
  assert.equal(calls[0].payload.includes("Edited during request"), false);
  assert.deepEqual(phases, ["sending", "retrying", "received"]);
  assert.ok(saved.read());
  controller.confirmReceipt((phase) => phases.push(phase));
  assert.equal(saved.read(), null);
});
test("two transport failures preserve only the reference and prohibit another send, including after reload", async () => {
  const saved = store();
  let calls = 0;
  const api = {
    async create() {
      calls++;
      throw new TypeError("fetch failed");
    },
    async resolveSubmission(): Promise<SubmissionResolution> {
      return { outcome: "committed", booking: booking() };
    },
  };
  const first = new BookingSubmission(api, saved.adapter);
  await assert.rejects(first.submit(input(), ignored));
  assert.equal(calls, 2);
  assert.equal(first.phase, "uncertain");
  assert.match(saved.read()!, /^[a-f0-9-]{36}$/);
  await assert.rejects(first.submit({ ...input(), purpose: "Edited details" }, ignored));
  const restored = new BookingSubmission(api, saved.adapter);
  assert.equal(restored.phase, "uncertain");
  await assert.rejects(restored.submit(input(), ignored));
  assert.equal(calls, 2);
  const resolved = await restored.recover("visitor@example.test", ignored);
  assert.equal(resolved.outcome, "committed");
  assert.equal(calls, 2);
  assert.ok(saved.read());
  restored.confirmReceipt(ignored);
  assert.equal(saved.read(), null);
});
for (const status of [401, 403, 409, 422, 429])
  test(`HTTP ${status} is not retried and permits a corrected explicit submission`, async () => {
    const saved = store();
    let calls = 0;
    const controller = new BookingSubmission(
      {
        async create() {
          calls++;
          throw new ApiError(status, "rejected");
        },
        async resolveSubmission() {
          assert.fail("Definitive rejection needs no recovery");
        },
      },
      saved.adapter,
    );
    await assert.rejects(controller.submit(input(), ignored));
    assert.equal(calls, 1);
    assert.equal(controller.phase, "idle");
    assert.equal(saved.read(), null);
  });
for (const error of [
  new ApiError(503, "server failure"),
  new ApiError(408, "timeout"),
  new SyntaxError("invalid response"),
  new DOMException("aborted", "AbortError"),
])
  test(`${error.name}/${error.message} remains uncertain without an automatic network retry`, async () => {
    const saved = store();
    let calls = 0;
    const controller = new BookingSubmission(
      {
        async create() {
          calls++;
          throw error;
        },
        async resolveSubmission() {
          return { outcome: "not_committed" };
        },
      },
      saved.adapter,
    );
    await assert.rejects(controller.submit(input(), ignored));
    assert.equal(calls, 1);
    assert.equal(controller.phase, "uncertain");
    const result = await controller.recover("visitor@example.test", ignored);
    assert.equal(result.outcome, "not_committed");
    assert.equal(controller.phase, "idle");
    assert.equal(calls, 1, "Recovery must not automatically send the current draft");
  });
test("wrong recovery email retains the pending reference; timeout retries and repeated clicks cannot multiply sends", async () => {
  const saved = store();
  let calls = 0,
    resolutions = 0;
  let release!: () => void;
  const pending = new Promise<void>((resolve) => {
    release = resolve;
  });
  const controller = new BookingSubmission(
    {
      async create() {
        calls++;
        await pending;
        throw new DOMException("timed out", "TimeoutError");
      },
      async resolveSubmission() {
        if (++resolutions === 1) throw new ApiError(401, "wrong email");
        return { outcome: "not_committed" };
      },
    },
    saved.adapter,
  );
  const first = controller.submit(input(), ignored);
  await assert.rejects(controller.submit(input(), ignored));
  release();
  await assert.rejects(first);
  assert.equal(calls, 2);
  const key = saved.read();
  await assert.rejects(controller.recover("wrong@example.test", ignored));
  assert.equal(saved.read(), key);
  assert.equal(controller.phase, "uncertain");
  await controller.recover("visitor@example.test", ignored);
  assert.equal(saved.read(), null);
});
test("unavailable reference storage fails before the first POST", async () => {
  let calls = 0;
  const controller = new BookingSubmission(
    {
      async create() {
        calls++;
        return booking();
      },
      async resolveSubmission() {
        return { outcome: "not_committed" };
      },
    },
    {
      read: () => null,
      write() {
        throw new Error("storage unavailable");
      },
      clear() {},
    },
  );
  await assert.rejects(controller.submit(input(), ignored));
  assert.equal(calls, 0);
  assert.equal(controller.phase, "idle");
});
test("a completed response retains recovery until the receipt is actually displayed", async () => {
  const saved = store();
  const api = {
    async create() {
      return booking();
    },
    async resolveSubmission(): Promise<SubmissionResolution> {
      return { outcome: "committed", booking: booking() };
    },
  };
  const original = new BookingSubmission(api, saved.adapter);
  await original.submit(input(), ignored);
  assert.equal(original.phase, "received");
  assert.equal(original.canSubmit, false);
  const restored = new BookingSubmission(api, saved.adapter);
  assert.equal(restored.phase, "uncertain");
  assert.equal((await restored.recover("visitor@example.test", ignored)).outcome, "committed");
  restored.confirmReceipt(ignored);
  assert.equal(saved.read(), null);
});
for (const status of [401, 403, 409, 422, 429])
  test(`HTTP ${status} on retry cannot erase the uncertain first attempt`, async () => {
    let calls = 0;
    const saved = store();
    const controller = new BookingSubmission(
      {
        async create() {
          if (++calls === 1) throw new TypeError("response lost");
          throw new ApiError(status, "retry rejected");
        },
        async resolveSubmission() {
          return { outcome: "not_committed" };
        },
      },
      saved.adapter,
    );
    await assert.rejects(controller.submit(input(), ignored));
    assert.equal(calls, 2);
    assert.equal(controller.phase, "uncertain");
    assert.ok(saved.read());
    await assert.rejects(
      controller.submit({ ...input(), purpose: "Edited after retry rejection" }, ignored),
    );
    await controller.recover("visitor@example.test", ignored);
    assert.equal(calls, 2);
    assert.equal(controller.canSubmit, true);
  });
test("a retired form cannot clear a newer form's pending reference", async () => {
  const saved = store();
  let rejectOld!: (error: Error) => void;
  const controller = new BookingSubmission(
    {
      create: () =>
        new Promise<PublicBooking>((_, reject) => {
          rejectOld = reject;
        }),
      async resolveSubmission() {
        return { outcome: "not_committed" };
      },
    },
    saved.adapter,
  );
  const original = controller.submit(input(), ignored);
  saved.adapter.write("newer-reference");
  rejectOld(new ApiError(409, "original reference closed"));
  await assert.rejects(original);
  assert.equal(saved.read(), "newer-reference");
});
test("two mounted form controllers cannot start independent attempts in the same session", async () => {
  const saved = store();
  let calls = 0,
    finish!: (value: PublicBooking) => void;
  const api = {
    create: () => {
      calls++;
      return new Promise<PublicBooking>((resolve) => {
        finish = resolve;
      });
    },
    async resolveSubmission(): Promise<SubmissionResolution> {
      return { outcome: "not_committed" };
    },
  };
  const first = new BookingSubmission(api, saved.adapter);
  const second = new BookingSubmission(api, saved.adapter);
  const pending = first.submit(input(), ignored);
  await assert.rejects(second.submit(input(), ignored));
  assert.equal(second.phase, "uncertain");
  assert.equal(calls, 1);
  finish(booking());
  await pending;
  first.confirmReceipt(ignored);
});
