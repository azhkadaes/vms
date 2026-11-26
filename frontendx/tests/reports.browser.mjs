import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { startPreview } from "./helpers/preview-server.mjs";

const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || "playwright");
const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3000";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);
const evidence = new URL("../anti-slop/evidence-2026-10-30/reports/", import.meta.url);
let browser;
before(async () => {
  await mkdir(evidence, { recursive: true });
  browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
});
after(async () => browser?.close());
async function open({ role = "PA", width = 390, mode = "normal" } = {}) {
  let summaryMode = mode;
  const api = await startPreview({ time: "2099-08-19T09:00:00+08:00" });
  async function create(officerId = "o2", kind = "standard") {
    const body = {
      visitKind: kind,
      idempotencyKey: crypto.randomUUID(),
      date: "2099-08-19",
      time: "10:00",
      officerId,
      name: `PRIVATE ${crypto.randomUUID()}`,
    };
    if (kind === "standard")
      Object.assign(body, {
        organization: "PRIVATE ORG",
        email: "private@example.test",
        phone: "00123456789",
        purpose: "PRIVATE PURPOSE",
        visitors: [body.name],
        identityType: "KTP",
        ack: true,
      });
    else body.serviceId = "0012345678";
    const response = await fetch(
      api.base + (kind === "standard" ? "/bookings" : "/staff/bookings?role=Protocol%20Officer"),
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    );
    assert.equal(response.status, 201);
    return response.json();
  }
  const fixtures = [
    await create("o2"),
    await create("o3"),
    await create("o1"),
    await create("o2", "restricted"),
  ];
  const context = await browser.newContext({
    viewport: { width, height: 900 },
    timezoneId: "Asia/Makassar",
    reducedMotion: "reduce",
  });
  await context.addInitScript((role) => sessionStorage.setItem("vms-preview-role", role), role);
  await context.route("**/__preview/**", async (route) => {
    const url = new URL(route.request().url());
    const summary = url.pathname.endsWith("/reports/summary");
    if (summary && ["error", "denied"].includes(summaryMode)) {
      await route.fulfill({
        status: summaryMode === "denied" ? 403 : 503,
        contentType: "application/json",
        body: JSON.stringify({
          message:
            summaryMode === "denied"
              ? "Akses laporan tidak diizinkan."
              : "Gangguan ringkasan sintetis.",
        }),
      });
      return;
    }
    const response = await route.fetch({
      url: api.base + url.pathname.split("/__preview")[1] + url.search,
    });
    if (summary && summaryMode === "malformed") {
      const data = await response.json();
      data.total += 1;
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(data),
      });
      return;
    }
    await route.fulfill({ response });
  });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.clock.install({ time: new Date("2099-08-19T09:00:00+08:00") });
  await page.goto(origin + "/admin/dashboard");
  return {
    api,
    context,
    page,
    fixtures,
    errors,
    setMode(value) {
      summaryMode = value;
    },
    async close() {
      await context.close();
      await api.close();
    },
  };
}
function totalCard(page) {
  return page.locator(".analytics-card-total strong");
}
async function total(page, value) {
  await totalCard(page).filter({ hasText: new RegExp(`^${value}$`) }).waitFor();
}
async function snapshot(env, name) {
  assert.equal(
    await env.page.evaluate(() => document.documentElement.scrollWidth > innerWidth),
    false,
  );
  assert.equal(
    await env.page
      .locator(".analytics-card")
      .evaluateAll((cards) =>
        cards.every(
          (card) =>
            card.getBoundingClientRect().right <= innerWidth &&
            card.getBoundingClientRect().left >= 0,
        ),
      ),
    true,
    "Analytics cards must fit within the viewport",
  );
  assert.deepEqual(env.errors, []);
  await env.page.screenshot({
    path: fileURLToPath(new URL(`${name}.png`, evidence)),
    fullPage: true,
  });
  await writeFile(
    new URL(`${name}.aria.txt`, evidence),
    await env.page.locator("body").ariaSnapshot(),
  );
}
test("dashboard analytics applies two checked statuses and removes a filter tag on mobile", async () => {
  const env = await open({ role: "Protocol Officer", width: 320 });
  const { page } = env;
  try {
    const status = page.getByRole("combobox", { name: "Status", exact: true });
    const filters = page.getByRole("button", { name: "Filter statistik", exact: true });
    const analytics = page.locator(".dashboard-analytics");
    await filters.click();
    await status.click();
    await page.getByRole("option", { name: "Pending", exact: true }).click();
    await status.fill("Disetujui");
    await status.press("Enter");
    assert.equal(await page.getByRole("option", { name: "Disetujui", exact: true }).getAttribute("aria-selected"), "true");
    await status.press("Escape");
    await page.getByRole("button", { name: "Terapkan", exact: true }).click();
    await analytics.getByText("Ditolak", { exact: true }).waitFor({ state: "hidden" });
    await analytics.getByText("Pending", { exact: true }).waitFor();
    await analytics.getByText("Disetujui", { exact: true }).waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await filters.click();
    await page.getByRole("button", { name: "Hapus status Disetujui", exact: true }).press("Enter");
    await page.getByRole("button", { name: "Terapkan", exact: true }).click();
    await analytics.getByText("Disetujui", { exact: true }).waitFor({ state: "hidden" });
    await analytics.getByText("Pending", { exact: true }).waitFor();
  } finally {
    await env.close();
  }
});
test("dashboard analytics excludes restricted visits and fits mobile and desktop widths", async () => {
  const env = await open({ role: "Protocol Officer", width: 320 });
  try {
    await total(env.page, 3);
    const analytics = env.page.getByRole("region", { name: "Analitik kunjungan" });
    assert.ok((await analytics.innerText()).includes("Tidak termasuk kunjungan terbatas"));
    for (const fixture of env.fixtures)
      assert.equal((await analytics.innerText()).includes(fixture.name), false);
    await snapshot(env, "summary-admin-320");
    await env.page.setViewportSize({ width: 768, height: 900 });
    await snapshot(env, "summary-admin-768");
    await env.page.setViewportSize({ width: 1280, height: 900 });
    await snapshot(env, "summary-admin-1280");
  } finally {
    await env.close();
  }
});

test("analytics validates a reversed date range, shows zero for an empty range and resets to this month", async () => {
  const env = await open();
  const { page } = env;
  try {
    await total(page, 1);
    const filters = page.getByRole("button", { name: "Filter statistik", exact: true });
    const from = page.getByLabel("Dari tanggal", { exact: true });
    const to = page.getByLabel("Sampai tanggal", { exact: true });
    const apply = page.getByRole("button", { name: "Terapkan", exact: true });
    await filters.click();
    await from.fill("2099-09-01");
    await apply.click();
    assert.equal(await to.getAttribute("aria-invalid"), "true");
    assert.equal(await to.evaluate((node) => node === document.activeElement), true);
    await to.fill("2099-09-30");
    await apply.click();
    await total(page, 0);
    await snapshot(env, "summary-empty-390");
    await filters.click();
    await page.getByRole("button", { name: "Bulan ini", exact: true }).click();
    await total(page, 1);
  } finally {
    await env.close();
  }
});

test("analytics keeps last counts after an outage and hides them when access is denied", async () => {
  const env = await open({ mode: "error" });
  const { page } = env;
  const refresh = page.getByRole("button", { name: "Perbarui statistik", exact: true });
  try {
    await page.getByRole("alert").filter({ hasText: "Gangguan ringkasan sintetis." }).waitFor();
    assert.equal(await totalCard(page).count(), 0);
    env.setMode("normal");
    await refresh.click();
    await total(page, 1);
    env.setMode("error");
    await refresh.click();
    await page.getByText("Menampilkan data terakhir", { exact: false }).waitFor();
    await total(page, 1);
    await snapshot(env, "summary-stale-390");
    env.setMode("denied");
    await refresh.click();
    await page.getByRole("alert").filter({ hasText: "Akses laporan tidak diizinkan." }).waitFor();
    assert.equal(await totalCard(page).count(), 0);
    await snapshot(env, "summary-denied-390");
    env.setMode("malformed");
    await refresh.click();
    await page.getByRole("alert").filter({ hasText: "Respons layanan tidak valid" }).waitFor();
  } finally {
    await env.close();
  }
});

test("analytics requests a new date range after the browser reconnects", async () => {
  const env = await open();
  const { page } = env;
  try {
    await total(page, 1);
    await page.evaluate(() => {
      Object.defineProperty(navigator, "onLine", { configurable: true, get: () => false });
      window.dispatchEvent(new Event("offline"));
    });
    await page.getByRole("button", { name: "Filter statistik", exact: true }).click();
    await page.getByLabel("Dari tanggal", { exact: true }).fill("2099-07-01");
    await page.getByLabel("Sampai tanggal", { exact: true }).fill("2099-07-31");
    await page.getByRole("button", { name: "Terapkan", exact: true }).click();
    await page.getByRole("status").filter({ hasText: "Anda sedang offline" }).waitFor();
    assert.equal(await totalCard(page).count(), 0);
    await page.evaluate(() => {
      Object.defineProperty(navigator, "onLine", { configurable: true, get: () => true });
      window.dispatchEvent(new Event("online"));
    });
    await total(page, 0);
  } finally {
    await env.close();
  }
});
