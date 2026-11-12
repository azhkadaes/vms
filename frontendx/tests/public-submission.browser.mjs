import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { startPreview } from "./helpers/preview-server.mjs";

const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || "playwright");
const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3000";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);
const evidence = new URL("../anti-slop/evidence-2026-10-30/public-submission/", import.meta.url);
const keyName = "vms-pending-submission";
let browser;
before(async () => {
  await mkdir(evidence, { recursive: true });
  browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
});
after(async () => browser?.close());
async function open({ mode = "normal", width = 390, language = "id" } = {}) {
  const api = await startPreview({ time: "2099-08-19T09:00:00+08:00" });
  const context = await browser.newContext({
    viewport: { width, height: 900 },
    timezoneId: "Asia/Makassar",
    reducedMotion: "reduce",
  });
  await context.addInitScript(
    (language) => localStorage.setItem("vms-visitor-language", language),
    language,
  );
  const posts = [],
    errors = [];
  let recovered;
  await context.route("**/__preview/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const target = api.base + url.pathname.split("/__preview")[1] + url.search;
    const creation = url.pathname.endsWith("/__preview/bookings") && request.method() === "POST";
    if (creation) {
      posts.push(request.postDataJSON());
      if (mode === "validation" && posts.length === 1) {
        await route.fulfill({
          status: 422,
          contentType: "application/json",
          body: JSON.stringify({
            message: "Periksa data permohonan.",
            fieldErrors: { email: ["Periksa nama email dan domain Anda."] },
          }),
        });
        return;
      }
      if (mode === "missing" && posts.length <= 2) {
        await route.abort("failed");
        return;
      }
      if (mode === "retry-rejected" && posts.length === 2) {
        await route.fulfill({
          status: 429,
          contentType: "application/json",
          body: JSON.stringify({ message: "Coba lagi nanti." }),
        });
        return;
      }
    }
    const response = await route.fetch({ url: target });
    if (creation) {
      recovered = await response.json();
      if (
        ((mode === "lost-once" || mode === "retry-rejected") && posts.length === 1) ||
        mode === "lost-twice"
      ) {
        await route.abort("failed");
        return;
      }
      if (mode === "malformed") {
        await route.fulfill({ status: 201, contentType: "application/json", body: "{}" });
        return;
      }
    }
    await route.fulfill({ response });
  });
  const page = await context.newPage();
  page.setDefaultTimeout(12000);
  page.on("pageerror", (error) => errors.push(error.message));
  await page.clock.install({ time: new Date("2099-08-19T09:00:00+08:00") });
  await page.goto(origin + "/booking");
  return {
    page,
    context,
    api,
    posts,
    errors,
    get received() {
      return recovered;
    },
    async close() {
      await context.close();
      await api.close();
    },
  };
}
async function review(env) {
  const { page, posts } = env;
  await page.locator("#officerId").selectOption("o2");
  await page.locator(".date-picker-trigger").click();
  await page
    .getByRole("combobox", { name: /Pilih tahun|Choose year/, exact: true })
    .selectOption("2099");
  await page
    .getByRole("combobox", { name: /Pilih bulan|Choose month/, exact: true })
    .selectOption("7");
  await page.locator('.shadcn-calendar-day[data-day="2099-08-19"]:not(:disabled)').first().click();
  await page.locator(".time-picker-trigger").click();
  await page
    .getByRole("listbox", { name: /^(Jam|Hours?)$/ })
    .getByRole("option", { name: "10", exact: true })
    .click();
  await page
    .getByRole("listbox", { name: /^(Menit|Minutes?)$/ })
    .getByRole("option", { name: "00", exact: true })
    .click();
  await page.locator(".date-picker > small").click();
  await page.locator("#purpose").fill("Original Browser Purpose");
  await page.locator('.booking-form .form-actions button[type="button"]').click();
  await page.locator("#name").fill("Submission Browser Name");
  await page.locator("#organization").fill("Submission Browser Organization");
  await page.locator("#email").fill("original@example.test");
  await page.locator("#phone").fill("00123456789");
  await page.locator('.booking-form .form-actions button[type="button"]').click();
  await page
    .getByRole("heading", { name: /Periksa sekali lagi|Review once more/, exact: false })
    .waitFor();
  assert.equal(posts.length, 0, "Entering review cannot submit a booking");
  await page.locator('.booking-form input[type="checkbox"]').check();
}
async function send(page) {
  await page.locator('.booking-form button[type="submit"]').focus();
  await page.keyboard.press("Enter");
}
async function success(env) {
  await env.page.locator(".booking-code-display strong").waitFor();
  assert.equal(
    await env.page.locator(".booking-code-display strong").innerText(),
    env.received.code,
  );
  assert.equal(await env.page.evaluate((key) => sessionStorage.getItem(key), keyName), null);
  assert.deepEqual(env.errors, []);
}
async function snapshot(page, name) {
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
  await page.screenshot({ path: fileURLToPath(new URL(`${name}.png`, evidence)), fullPage: true });
  await writeFile(new URL(`${name}.aria.txt`, evidence), await page.locator("body").ariaSnapshot());
}
async function records(env) {
  return fetch(env.api.base + "/bookings?role=PA")
    .then((response) => response.json())
    .then((records) => records.filter((record) => record.name === "Submission Browser Name"));
}

test("a response lost after commit retries the same request once and shows one receipt", async () => {
  const env = await open({ mode: "lost-once" });
  try {
    await review(env);
    await send(env.page);
    await success(env);
    assert.equal(env.posts.length, 2);
    assert.deepEqual(env.posts[0], env.posts[1]);
    assert.equal((await records(env)).length, 1);
    assert.equal((await records(env))[0].history.length, 1);
  } finally {
    await env.close();
  }
});
test("uncertain recovery returns the original booking after draft edits without sending those edits", async () => {
  const env = await open({ mode: "lost-twice" });
  try {
    await review(env);
    await send(env.page);
    await env.page.locator(".submission-recovery").waitFor();
    assert.equal(env.posts.length, 2);
    await env.page.getByRole("link", { name: /Langkah sebelumnya|Previous step/ }).click();
    await env.page.locator("#email").fill("edited@example.test");
    await env.page.locator("#name").fill("Edited Browser Name");
    assert.equal(await env.page.locator("#submission-email").inputValue(), "original");
    await snapshot(env.page, "uncertain-edited-390");
    await env.page.getByRole("button", { name: "Periksa hasil pengiriman", exact: true }).click();
    await success(env);
    assert.equal(env.posts.length, 2);
    assert.equal((await records(env)).length, 1);
    assert.equal(env.received.email, "original@example.test");
  } finally {
    await env.close();
  }
});
test("reload retains only the reference; wrong email preserves recovery and correct email restores receipt at 320px", async () => {
  const env = await open({ mode: "lost-twice", width: 320 });
  try {
    await review(env);
    await send(env.page);
    await env.page.locator(".submission-recovery").waitFor();
    const stored = await env.page.evaluate((key) => sessionStorage.getItem(key), keyName);
    assert.equal(stored, env.posts[0].idempotencyKey);
    await env.page.reload();
    await env.page.locator(".submission-recovery").waitFor();
    assert.equal(await env.page.locator("#purpose").inputValue(), "");
    assert.equal(await env.page.locator("#submission-email").inputValue(), "");
    await env.page.locator("#submission-email").fill("wrong@example.test");
    await env.page.getByRole("button", { name: "Periksa hasil pengiriman", exact: true }).click();
    await env.page
      .getByRole("alert")
      .filter({ hasText: "Email pengajuan sebelumnya tidak cocok" })
      .waitFor();
    assert.equal(await env.page.evaluate((key) => sessionStorage.getItem(key), keyName), stored);
    await snapshot(env.page, "recovery-wrong-email-320");
    const approved = await fetch(env.api.base + `/bookings/${env.received.id}/approve?role=PA`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ version: 1, room: "Ruang Rapat Nusantara", duration: 60 }),
    });
    assert.equal(approved.status, 200);
    await env.page.locator("#submission-email").fill("original@example.test");
    await env.page.getByRole("button", { name: "Periksa hasil pengiriman", exact: true }).click();
    await success(env);
    assert.ok(
      (await env.page.locator(".submission").innerText()).includes(
        "Permohonan Anda sudah tercatat.",
      ),
    );
    assert.equal(env.posts.length, 2);
  } finally {
    await env.close();
  }
});
test("conclusive missing recovery retains the draft and requires explicit send with a new reference", async () => {
  const env = await open({ mode: "missing" });
  try {
    await review(env);
    await send(env.page);
    await env.page.locator(".submission-recovery").waitFor();
    await env.page.getByRole("button", { name: "Periksa hasil pengiriman", exact: true }).click();
    await env.page.getByRole("status").filter({ hasText: "referensinya sudah ditutup" }).waitFor();
    assert.equal(env.posts.length, 2);
    assert.equal((await records(env)).length, 0);
    await env.page.getByRole("link", { name: /Langkah sebelumnya|Previous step/ }).click();
    assert.equal(await env.page.locator("#phone").inputValue(), "00123456789");
    await env.page.locator("#organization").fill("Corrected Browser Organization");
    await env.page.locator('.booking-form .form-actions button[type="button"]').click();
    await send(env.page);
    await success(env);
    assert.equal(env.posts.length, 3);
    assert.notEqual(env.posts[2].idempotencyKey, env.posts[0].idempotencyKey);
    assert.equal(env.received.organization, "Corrected Browser Organization");
  } finally {
    await env.close();
  }
});
test("HTTP field rejection is not retried and returns focus to the preserved field", async () => {
  const env = await open({ mode: "validation", width: 1280 });
  try {
    await review(env);
    await send(env.page);
    await env.page.locator("#email[aria-invalid=true]").waitFor();
    await env.page.waitForFunction(() => document.activeElement?.id === "email");
    assert.equal(
      await env.page.locator("#email").evaluate((node) => node === document.activeElement),
      true,
    );
    assert.equal(await env.page.locator("#name").inputValue(), "Submission Browser Name");
    assert.equal(await env.page.locator("#email").inputValue(), "original");
    assert.equal(env.posts.length, 1);
    assert.equal(await env.page.evaluate((key) => sessionStorage.getItem(key), keyName), null);
    await snapshot(env.page, "server-field-error-1280");
    await env.page.locator("#email").fill("corrected@example.test");
    await env.page.locator('.booking-form .form-actions button[type="button"]').click();
    await send(env.page);
    await success(env);
    assert.equal(env.posts.length, 2);
    assert.notEqual(env.posts[0].idempotencyKey, env.posts[1].idempotencyKey);
  } finally {
    await env.close();
  }
});
test("a malformed successful response stays recoverable and the recovery panel is localized in English", async () => {
  const env = await open({ mode: "malformed", language: "en" });
  try {
    await review(env);
    await send(env.page);
    await env.page
      .getByRole("heading", { name: "Check the previous submission", exact: true })
      .waitFor();
    assert.equal(env.posts.length, 1);
    await snapshot(env.page, "recovery-english-390");
    await env.page.getByRole("button", { name: "Check submission result", exact: true }).click();
    await success(env);
    assert.equal(env.posts.length, 1);
  } finally {
    await env.close();
  }
});
test("a rate-limit rejection on retry preserves recovery of the already committed first request", async () => {
  const env = await open({ mode: "retry-rejected", width: 768 });
  try {
    await review(env);
    await send(env.page);
    await env.page.locator(".submission-recovery").waitFor();
    assert.equal(env.posts.length, 2);
    assert.ok(await env.page.evaluate((key) => sessionStorage.getItem(key), keyName));
    await snapshot(env.page, "recovery-tablet-768");
    await env.page.getByRole("button", { name: "Periksa hasil pengiriman", exact: true }).click();
    await success(env);
    assert.equal((await records(env)).length, 1);
    assert.equal(env.posts.length, 2);
  } finally {
    await env.close();
  }
});
