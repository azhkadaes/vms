import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { sessionFor } from "../packages/data/src/domain.ts";
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
async function open({ role = "PA", width = 390, mode = "normal", path = "/admin/logs" } = {}) {
  let summaryMode = mode,
    viewOnly = false;
  const api = await startPreview({
    time: "2099-08-19T09:00:00+08:00",
    session: (role) => {
      const session = sessionFor(role);
      return {
        ...session,
        permissions: viewOnly
          ? session.permissions.filter((permission) => permission !== "reportExport")
          : session.permissions,
      };
    },
  });
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
  const errors = [],
    requests = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => requests.push(new URL(request.url()).pathname));
  await page.clock.install({ time: new Date("2099-08-19T09:00:00+08:00") });
  await page.goto(origin + path);
  return {
    api,
    context,
    page,
    fixtures,
    errors,
    requests,
    create,
    setMode(value) {
      summaryMode = value;
    },
    revokeExport() {
      viewOnly = true;
    },
    async close() {
      await context.close();
      await api.close();
    },
  };
}
async function total(page, value) {
  await page
    .getByTestId("report-total")
    .filter({ hasText: String(value) })
    .waitFor();
  assert.equal(await page.getByTestId("report-total").innerText(), String(value));
}
async function snapshot(env, name) {
  assert.equal(
    await env.page.evaluate(() => document.documentElement.scrollWidth > innerWidth),
    false,
  );
  assert.equal(
    await env.page
      .locator(".report-summary-table td")
      .evaluateAll((cells) =>
        cells.every(
          (cell) =>
            cell.getBoundingClientRect().right <= innerWidth &&
            cell.getBoundingClientRect().left >= 0,
        ),
      ),
    true,
    "Count cells must not be clipped by a containing panel",
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
async function csv(page) {
  const [download] = await Promise.all([
    page.waitForEvent("download"),
    page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).click(),
  ]);
  return readFile(await download.path(), "utf8");
}
test("dashboard analytics applies two checked statuses and removes a filter tag on mobile", async () => {
  const env = await open({ role: "Protocol Officer", width: 320, path: "/admin/dashboard" });
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
test("Protocol Officer keeps aggregate reports without booking rows; summary CSV and controls reflow from 320px to desktop", async () => {
  const env = await open({ role: "Protocol Officer", width: 320 });
  try {
    await total(env.page, 3);
    assert.equal(env.requests.filter((path) => path === "/__preview/bookings").length, 0);
    assert.ok(
      (await env.page.locator(".report-counts").innerText()).includes("Lingkup kunjungan yang diizinkan"),
    );
    for (const fixture of env.fixtures)
      assert.equal((await env.page.locator("body").innerText()).includes(fixture.name), false);
    const content = await csv(env.page);
    assert.ok(content.includes('"Tanggal awal","Tanggal akhir","Cakupan","Status","Jumlah"'));
    for (const fixture of env.fixtures)
      for (const secret of [fixture.name, fixture.code, fixture.email, fixture.serviceId].filter(
        Boolean,
      ))
        assert.equal(content.includes(secret), false, secret);
    await snapshot(env, "summary-admin-320");
    await env.page.setViewportSize({ width: 768, height: 900 });
    await snapshot(env, "summary-admin-768");
    await env.page.setViewportSize({ width: 1280, height: 900 });
    await snapshot(env, "summary-admin-1280");
  } finally {
    await env.close();
  }
});
test("invalid filter focuses its field; applying a successful empty range shows zero and URL history restores filters", async () => {
  const env = await open();
  try {
    await total(env.page, 2);
    await env.page.locator("#report-from").fill("2099-09-01");
    assert.equal(
      await env.page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).isDisabled(),
      true,
    );
    await env.page.getByRole("button", { name: "Terapkan filter", exact: true }).click();
    assert.equal(
      await env.page.locator("#report-to").evaluate((node) => node === document.activeElement),
      true,
    );
    assert.equal(await env.page.locator("#report-to").getAttribute("aria-invalid"), "true");
    await env.page.locator("#report-to").fill("2099-09-30");
    await env.page.getByRole("button", { name: "Terapkan filter", exact: true }).click();
    await total(env.page, 0);
    assert.ok(env.page.url().includes("from=2099-09-01"));
    await env.page
      .getByRole("status")
      .filter({ hasText: "Tidak ada kunjungan sesuai filter" })
      .waitFor();
    await snapshot(env, "summary-empty-390");
    await env.page.goBack();
    await total(env.page, 2);
    assert.equal(await env.page.locator("#report-from").inputValue(), "2099-08-01");
  } finally {
    await env.close();
  }
});
test("initial errors never show zero; refresh failure marks old results; denial hides them; malformed data stays unavailable", async () => {
  const env = await open({ mode: "error" });
  try {
    await env.page.getByRole("button", { name: "Coba lagi", exact: true }).waitFor();
    assert.equal(await env.page.getByTestId("report-total").count(), 0);
    env.setMode("normal");
    await env.page.getByRole("button", { name: "Coba lagi", exact: true }).click();
    await total(env.page, 2);
    env.setMode("error");
    await env.page.getByRole("button", { name: "Perbarui laporan", exact: true }).click();
    await env.page
      .getByText("Angka terakhir tetap ditampilkan. Perbarui laporan sebelum mengekspor.", {
        exact: true,
      })
      .waitFor();
    await total(env.page, 2);
    assert.equal(
      await env.page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).isDisabled(),
      true,
    );
    await snapshot(env, "summary-stale-390");
    env.setMode("denied");
    await env.page.getByRole("button", { name: "Coba lagi", exact: true }).click();
    await env.page
      .getByRole("alert")
      .filter({ hasText: "Akses laporan tidak diizinkan." })
      .waitFor();
    assert.equal(await env.page.getByTestId("report-total").count(), 0);
    await snapshot(env, "summary-denied-390");
    env.setMode("malformed");
    await env.page.getByRole("button", { name: "Coba lagi", exact: true }).click();
    await env.page
      .getByRole("alert")
      .filter({ hasText: /Data ringkasan belum dapat ditampilkan|Respons layanan tidak valid/ })
      .waitFor();
    assert.equal(
      await env.page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).isDisabled(),
      true,
    );
  } finally {
    await env.close();
  }
});
test("changed summaries require review before export; revoked export rights are reflected after refresh", async () => {
  const env = await open({ width: 1280 });
  try {
    await total(env.page, 2);
    await env.create();
    await env.page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).click();
    await env.page
      .getByRole("status")
      .filter({ hasText: "Ringkasan diperbarui. Periksa angka terbaru" })
      .waitFor();
    await total(env.page, 3);
    const content = await csv(env.page);
    assert.ok(content.includes('"Pending","3"'));
    env.revokeExport();
    await env.page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).click();
    await env.page
      .getByRole("alert")
      .filter({ hasText: "Akses laporan tidak diizinkan." })
      .waitFor();
    assert.equal(await env.page.getByTestId("report-total").count(), 0);
    await env.page.getByRole("button", { name: "Coba lagi", exact: true }).click();
    await total(env.page, 3);
    assert.equal(
      await env.page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).isDisabled(),
      true,
    );
    await env.page
      .getByText("Akun Anda dapat membaca ringkasan; izin ekspor diperlukan untuk mengunduhnya.", {
        exact: true,
      })
      .waitFor();
    assert.deepEqual(env.errors, []);
  } finally {
    await env.close();
  }
});
test("an offline pending range shows unavailable data until a successful connection returns", async () => {
  const env = await open();
  try {
    await total(env.page, 2);
    await env.page.evaluate(() => {
      Object.defineProperty(navigator, "onLine", { configurable: true, get: () => false });
      window.dispatchEvent(new Event("offline"));
    });
    await env.page.locator("#report-from").fill("2099-07-01");
    await env.page.locator("#report-to").fill("2099-07-31");
    await env.page.getByRole("button", { name: "Terapkan filter", exact: true }).click();
    await env.page.getByRole("status").filter({ hasText: "Anda sedang offline" }).waitFor();
    assert.equal(await env.page.getByTestId("report-total").count(), 0);
    assert.equal(
      await env.page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).isDisabled(),
      true,
    );
    await env.page.evaluate(() => {
      Object.defineProperty(navigator, "onLine", { configurable: true, get: () => true });
      window.dispatchEvent(new Event("online"));
    });
    await total(env.page, 0);
    await env.page
      .getByRole("status")
      .filter({ hasText: "Tidak ada kunjungan sesuai filter" })
      .waitFor();
  } finally {
    await env.close();
  }
});
