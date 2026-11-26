import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { startPreview } from "./helpers/preview-server.mjs";

const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || "playwright");
const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3000";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);
const evidence = new URL("../anti-slop/evidence-2026-10-31/public-history/", import.meta.url);
let browser;
before(async () => {
  await mkdir(evidence, { recursive: true });
  browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
});
after(async () => browser?.close());

async function post(api, path, body) {
  const response = await fetch(api.base + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return { status: response.status, data: await response.json() };
}
async function open({
  width = 390,
  path = "/status",
  time = "2099-08-19T09:00:00+08:00",
  browserTime = time,
  lifecycle = false,
  rejected = false,
  unknown = false,
  language = "id",
} = {}) {
  const api = await startPreview({ time });
  let booking = (
    await post(api, "/bookings", {
      ack: true,
      idempotencyKey: crypto.randomUUID(),
      officerId: "o2",
      date: "2099-08-19",
      time: "10:00",
      name: "Public History Example",
      organization: "History Example Organization",
      email: "history@example.test",
      phone: "00123456789",
      purpose: "Public history verification",
      visitors: ["Public History Example"],
      identityType: "KTP",
    })
  ).data;
  if (rejected)
    booking = (
      await post(api, `/bookings/${booking.id}/reject?role=PA`, {
        version: booking.version,
        rejectionReason: "Alasan publik asli.",
        internalRejectionNote: "PRIVATE INTERNAL NOTE",
      })
    ).data;
  if (lifecycle) {
    booking = (
      await post(api, `/bookings/${booking.id}/approve?role=PA`, {
        version: booking.version,
        roomId: "room-1",
        duration: 60,
      })
    ).data;
    booking = (
      await post(api, `/bookings/${booking.id}/reschedule?role=PA`, {
        version: booking.version,
        date: "2099-08-19",
        time: "11:00",
        roomId: "room-2",
        duration: 60,
        reason: "Alasan publik asli.\nTidak diterjemahkan.",
      })
    ).data;
  }
  const context = await browser.newContext({
    viewport: { width, height: 900 },
    timezoneId: "America/Los_Angeles",
    reducedMotion: "reduce",
  });
  await context.addInitScript(
    (language) => localStorage.setItem("vms-visitor-language", language),
    language,
  );
  const requests = [],
    errors = [];
  await context.route("**/__preview/**", async (route) => {
    const url = new URL(route.request().url());
    const response = await route.fetch({
      url: api.base + url.pathname.split("/__preview")[1] + url.search,
    });
    if (unknown && url.pathname.endsWith("/lookup")) {
      const value = await response.json();
      value.history.push({
        at: "invalid",
        action: "future-private-action",
        message: "PRIVATE RAW MESSAGE",
        publicMessage: "PRIVATE FUTURE COPY",
      });
      await route.fulfill({ response, json: value });
    } else await route.fulfill({ response });
  });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => requests.push(request.url()));
  await page.clock.install({ time: new Date(browserTime) });
  await page.goto(origin + path);
  return {
    api,
    page,
    context,
    booking,
    requests,
    errors,
    async close() {
      await context.close();
      await api.close();
    },
  };
}
async function lookup(env) {
  await env.page.locator("#status-code").fill(env.booking.code);
  await env.page.locator("#status-email").fill(env.booking.email);
  await env.page.getByRole("button", { name: /^(Lihat kunjungan|View visit)$/ }).focus();
  await env.page.keyboard.press("Enter");
  await env.page.getByRole("heading", { name: /^(Rincian kunjungan|Visit summary)$/ }).waitFor();
}
async function outline(page) {
  const levels = await page
    .getByRole("heading")
    .evaluateAll((nodes) => nodes.map((node) => Number(node.tagName[1])));
  assert.equal(levels.filter((level) => level === 1).length, 1, `One active h1: ${levels}`);
  let previous = 0;
  for (const level of levels) {
    assert.ok(level <= previous + 1, `No heading skip: ${levels}`);
    previous = level;
  }
}
async function snapshot(env, name) {
  assert.equal(
    await env.page.evaluate(() => document.documentElement.scrollWidth > innerWidth),
    false,
  );
  assert.deepEqual(env.errors, []);
  await env.page.screenshot({
    path: fileURLToPath(new URL(name + ".png", evidence)),
    fullPage: true,
  });
  await writeFile(
    new URL(name + ".aria.txt", evidence),
    await env.page.locator("body").ariaSnapshot(),
  );
}

test("expanded, inline and standalone registration use logical headings while Back preserves the form", async () => {
  const env = await open({ width: 320, path: "/?view=registration#registration" });
  try {
    await env.page.locator("#purpose").fill("Retained across choices");
    await outline(env.page);
    await snapshot(env, "registration-expanded-320");
    await env.page.getByRole("link", { name: "Kembali", exact: true }).click();
    await env.page
      .locator("#visit-options")
      .getByRole("link", { name: "Daftar kunjungan", exact: true })
      .click();
    assert.equal(await env.page.locator("#purpose").inputValue(), "Retained across choices");
    await outline(env.page);
    for (const width of [768, 1280]) {
      await env.page.setViewportSize({ width, height: 900 });
      await env.page.goto(origin + "/");
      await env.page.locator("#registration-title").waitFor();
      assert.equal(
        await env.page.locator("#registration-title").evaluate((node) => node.tagName),
        "H2",
      );
      await outline(env.page);
      await snapshot(env, `registration-inline-${width}`);
    }
    await env.page.setViewportSize({ width: 390, height: 900 });
    for (const path of ["/booking", "/booking/visitors", "/booking/review"]) {
      await env.page.goto(origin + path);
      await env.page.locator("#registration-title").waitFor();
      assert.equal(
        await env.page.locator("#registration-title").evaluate((node) => node.tagName),
        "H1",
      );
      await outline(env.page);
    }
  } finally {
    await env.close();
  }
});

test("public history translates actions, keeps public reasons and shows service timestamps in WITA", async () => {
  const env = await open({ lifecycle: true });
  try {
    await lookup(env);
    const result = env.page.getByRole("heading", { name: "Rincian kunjungan", exact: true });
    assert.equal(await result.evaluate((node) => node === document.activeElement), true);
    await outline(env.page);
    const history = env.page.locator(".public-visit-history");
    assert.ok((await history.innerText()).includes("Jadwal kunjungan diubah"));
    assert.equal((await history.innerText()).includes("PA:"), false);
    assert.equal(
      await history.locator("p").textContent(),
      "Alasan publik asli.\nTidak diterjemahkan.",
    );
    assert.equal(
      await history.locator("time").first().getAttribute("datetime"),
      "2099-08-19T01:00:00.000Z",
    );
    assert.match(await history.locator("time").first().innerText(), /09[.:]00.*WITA/);
    await env.page
      .getByRole("combobox", { name: "Bahasa halaman", exact: true })
      .selectOption("en");
    assert.ok((await history.innerText()).includes("Request received"));
    assert.ok((await history.innerText()).includes("Visit approved"));
    assert.ok((await history.innerText()).includes("Visit rescheduled"));
    assert.equal(
      await history.locator("p").textContent(),
      "Alasan publik asli.\nTidak diterjemahkan.",
    );
    const update = env.page.getByRole("button", { name: "Update", exact: true });
    await update.focus();
    await env.page.keyboard.press("Enter");
    await update.waitFor({ state: "visible" });
    await env.page.waitForFunction(
      () => document.activeElement === document.querySelector(".status-heading button"),
    );
    assert.equal(await update.evaluate((node) => node === document.activeElement), true);
    await snapshot(env, "history-en-390");
    await env.page.getByRole("button", { name: "Check another visit", exact: true }).click();
    assert.equal(
      await env.page.locator("#status-code").evaluate((node) => node === document.activeElement),
      true,
    );
    assert.equal(await env.page.locator("#status-code").inputValue(), env.booking.code);
  } finally {
    await env.close();
  }
});

test("expanded phone and inline desktop status results have one active main heading and preserve rejection copy", async () => {
  const env = await open({ width: 320, path: "/?view=status#booking-status", rejected: true });
  try {
    await lookup(env);
    await outline(env.page);
    assert.equal(
      await env.page
        .getByRole("heading", { name: "Rincian kunjungan", exact: true })
        .evaluate((node) => node.tagName),
      "H2",
    );
    assert.equal(await env.page.locator(".decision-note p").innerText(), "Alasan publik asli.");
    assert.equal((await env.page.locator("body").innerText()).includes("PRIVATE INTERNAL"), false);
    await snapshot(env, "status-expanded-320");
    await env.page.locator("#visit-history-title").scrollIntoViewIfNeeded();
    await snapshot(env, "status-history-expanded-320");
    await env.page.setViewportSize({ width: 1280, height: 900 });
    await outline(env.page);
    assert.equal(
      await env.page
        .getByRole("heading", { name: "Rincian kunjungan", exact: true })
        .evaluate((node) => node.tagName),
      "H3",
    );
    assert.equal(
      await env.page
        .getByRole("heading", { name: env.booking.name, exact: true })
        .evaluate((node) => node.tagName),
      "H4",
    );
    await snapshot(env, "status-inline-1280");
  } finally {
    await env.close();
  }
});

test("unknown event codes never render raw messages and invalid times have safe feedback", async () => {
  const env = await open({ unknown: true, language: "en" });
  try {
    await lookup(env);
    const history = await env.page.locator(".public-visit-history").innerText();
    assert.ok(history.includes("Visit updated"));
    assert.ok(history.includes("Time unavailable"));
    assert.equal(history.includes("PRIVATE"), false);
    await snapshot(env, "history-unknown-en-390");
  } finally {
    await env.close();
  }
});

test("an open cancellation dialog expires using service time even when the browser clock is wrong", async () => {
  const env = await open({
    time: "2099-08-19T09:59:55+08:00",
    browserTime: "2030-01-01T01:00:00Z",
  });
  try {
    await lookup(env);
    const cancel = env.page.getByRole("button", { name: "Batalkan kunjungan", exact: true });
    assert.equal(await cancel.isDisabled(), false);
    await cancel.click();
    const dialog = env.page.getByRole("dialog", { name: "Batalkan kunjungan?", exact: true });
    await dialog.waitFor();
    await env.page.clock.runFor(6000);
    assert.equal(
      await dialog.getByRole("button", { name: "Ya, batalkan", exact: true }).isDisabled(),
      true,
    );
    assert.equal(
      await env.page
        .getByRole("button", { name: "Batalkan kunjungan", exact: true, includeHidden: true })
        .isDisabled(),
      true,
    );
    await dialog.getByRole("button", { name: "Tetap simpan kunjungan", exact: true }).click();
    assert.equal(
      await env.page
        .getByRole("button", { name: "Perbarui", exact: true })
        .evaluate((node) => node === document.activeElement),
      true,
    );
    assert.equal(env.requests.filter((url) => url.includes("/cancel")).length, 0);
    await snapshot(env, "cancel-cutoff-390");
  } finally {
    await env.close();
  }
});

test("the service rejects stale open confirmation at start and the UI retains the visit and explains it", async () => {
  const env = await open({ time: "2099-08-19T09:50:00+08:00", language: "en" });
  try {
    await lookup(env);
    await env.page.getByRole("button", { name: "Cancel visit", exact: true }).click();
    env.api.setNow("2099-08-19T10:00:00+08:00");
    const dialog = env.page.getByRole("dialog", { name: "Cancel visit?", exact: true });
    await dialog.getByRole("button", { name: "Yes, cancel", exact: true }).click();
    await dialog
      .getByText(
        "Self-service cancellation is only available before the visit start time. Contact staff for help.",
        { exact: true },
      )
      .waitFor();
    await env.page.waitForFunction(
      () => document.querySelector(".status-panel .toolbar button")?.disabled,
    );
    assert.equal(
      await dialog.getByRole("button", { name: "Yes, cancel", exact: true }).isDisabled(),
      true,
    );
    assert.equal(await dialog.isVisible(), true);
    const current = await post(env.api, "/lookup", {
      code: env.booking.code,
      email: env.booking.email,
    });
    assert.equal(current.data.status, "pending");
    assert.equal(current.data.version, env.booking.version);
    await snapshot(env, "cancel-stale-en-390");
  } finally {
    await env.close();
  }
});

test("confirmed pre-start cancellation shows its localized event, closes confirmation and returns focus", async () => {
  const env = await open();
  try {
    await lookup(env);
    await env.page.getByRole("button", { name: "Batalkan kunjungan", exact: true }).click();
    const dialog = env.page.getByRole("dialog", { name: "Batalkan kunjungan?", exact: true });
    await dialog.getByRole("button", { name: "Ya, batalkan", exact: true }).click();
    await dialog.waitFor({ state: "hidden" });
    await env.page.waitForFunction(
      () => document.activeElement === document.querySelector(".status-heading button"),
    );
    assert.ok(
      (await env.page.locator(".public-visit-history").innerText()).includes(
        "Kunjungan dibatalkan",
      ),
    );
    assert.equal(await env.page.locator(".qr-display").count(), 0);
    await outline(env.page);
    await snapshot(env, "cancel-confirmed-390");
  } finally {
    await env.close();
  }
});
