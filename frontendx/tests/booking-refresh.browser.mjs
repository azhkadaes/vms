import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";

const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || "playwright");
const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3000";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);
const evidence = new URL("../anti-slop/evidence-2026-10-30/refresh/", import.meta.url);
let browser;
before(async () => {
  await mkdir(evidence, { recursive: true });
  browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
});
after(async () => browser?.close());

async function open(role = "PA", route = "/admin/booking", width = 1280) {
  const context = await browser.newContext({
    viewport: { width, height: 900 },
    timezoneId: "Asia/Makassar",
    reducedMotion: "reduce",
  });
  await context.addInitScript((role) => sessionStorage.setItem("vms-preview-role", role), role);
  const page = await context.newPage();
  page.setDefaultTimeout(12000);
  await page.clock.install();
  const requests = [];
  page.on("request", (request) => {
    if (new URL(request.url()).pathname === "/__preview/staff/bookings" && request.method() === "GET")
      requests.push(request.url());
  });
  await page.goto(origin + route);
  await page.locator(".bookings-refresh time").waitFor();
  await page.getByRole("button", { name: "Perbarui data kunjungan", exact: true }).waitFor();
  await page.waitForLoadState("networkidle");
  return { context, page, requests };
}
async function waitUntil(check) {
  for (let attempt = 0; attempt < 120; attempt++) {
    if (await check()) return;
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  assert.fail("Expected refresh state did not arrive");
}
async function visibility(page, hidden) {
  await page.evaluate((hidden) => {
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      get: () => (hidden ? "hidden" : "visible"),
    });
    window.dispatchEvent(new Event("visibilitychange"));
  }, hidden);
}

test("staff polling uses one shared request every 30 seconds and preserves URL/scroll", async () => {
  const { context, page, requests } = await open(
    "PA",
    "/admin/booking?q=AuditRefresh&status=pending&sort=name",
  );
  try {
    const baseline = requests.length;
    await page.clock.runFor(29000);
    assert.equal(requests.length, baseline);
    const name = `AuditRefresh ${crypto.randomUUID().slice(0, 8)}`;
    const response = await context.request.post(origin + "/__preview/bookings", {
      data: {
        ack: true,
        officerId: "o2",
        date: "2099-12-12",
        time: "10:00",
        purpose: "Synthetic refresh verification",
        name,
        organization: "Audit Example",
        email: "refresh@example.test",
        phone: "08123456789",
        identityType: "KTP",
        visitors: [name],
        idempotencyKey: crypto.randomUUID(),
      },
    });
    assert.equal(response.status(), 201);
    const url = page.url();
    await page.clock.runFor(1100);
    await page.locator(".booking-desktop-table").getByText(name, { exact: true }).waitFor();
    assert.equal(requests.length, baseline + 1);
    assert.equal(page.url(), url);
    assert.equal(
      await page.getByRole("textbox", { name: "Cari kunjungan", exact: true }).inputValue(),
      "AuditRefresh",
    );
  } finally {
    await context.close();
  }
});

test("active Receptionist view polls every 15 seconds", async () => {
  const { context, page, requests } = await open("Receptionist", "/admin/check-in");
  try {
    const baseline = requests.length;
    await page.clock.runFor(14000);
    assert.equal(requests.length, baseline);
    await page.clock.runFor(1100);
    await waitUntil(() => requests.length === baseline + 1);
  } finally {
    await context.close();
  }
});

test("hidden tabs pause polling and returning refreshes even a recently fetched result", async () => {
  const { context, page, requests } = await open();
  try {
    const baseline = requests.length;
    await visibility(page, true);
    await page.clock.runFor(65000);
    assert.equal(requests.length, baseline);
    await visibility(page, false);
    await waitUntil(() => requests.length === baseline + 1);
    await page.waitForLoadState("networkidle");
    await visibility(page, true);
    await page.clock.runFor(1000);
    await visibility(page, false);
    await waitUntil(() => requests.length === baseline + 2);
  } finally {
    await context.close();
  }
});

test("manual refresh retains records on error, reports stale data and recovers", async () => {
  const { context, page } = await open();
  try {
    const before = await page.locator(".booking-desktop-table tbody").innerText();
    const lastSuccess = await page.locator(".bookings-refresh time").getAttribute("datetime");
    await page.route("**/__preview/staff/bookings?*", (route) =>
      route.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({ message: "Synthetic refresh outage" }),
      }),
    );
    await page.getByRole("button", { name: "Perbarui data kunjungan", exact: true }).focus();
    await page.keyboard.press("Enter");
    await page.locator('.bookings-refresh[data-freshness="stale"]').waitFor();
    assert.equal(await page.locator(".booking-desktop-table tbody").innerText(), before);
    assert.equal(
      await page.locator(".bookings-refresh time").getAttribute("datetime"),
      lastSuccess,
    );
    assert.match(
      await page.locator(".bookings-refresh [role=status]").innerText(),
      /Menampilkan data terakhir/,
    );
    await page.screenshot({
      path: new URL("staff-stale.png", evidence).pathname.replace(/^\/(\w:)/, "$1"),
    });
    await page.unroute("**/__preview/staff/bookings?*");
    await page.getByRole("button", { name: "Perbarui data kunjungan", exact: true }).click();
    await page.locator('.bookings-refresh[data-freshness="current"]').waitFor();
    await page.getByRole("button", { name: "Perbarui data kunjungan", exact: true }).waitFor();
    assert.equal(
      await page.locator(".bookings-refresh [role=status]").innerText(),
      "Data kunjungan diperbarui.",
    );
  } finally {
    await context.close();
  }
});

test("offline status retains records and reconnection triggers refresh", async () => {
  const { context, page, requests } = await open();
  try {
    const baseline = requests.length;
    const before = await page.locator(".booking-desktop-table tbody").innerText();
    await page.evaluate(() => window.dispatchEvent(new Event("offline")));
    await page.locator('.bookings-refresh[data-freshness="offline"]').waitFor();
    assert.equal(await page.locator(".booking-desktop-table tbody").innerText(), before);
    assert.equal(
      await page.getByRole("button", { name: "Perbarui data kunjungan", exact: true }).isDisabled(),
      true,
    );
    await page.evaluate(() => window.dispatchEvent(new Event("online")));
    await waitUntil(() => requests.length === baseline + 1);
    await page.locator('.bookings-refresh[data-freshness="current"]').waitFor();
  } finally {
    await context.close();
  }
});

test("an access-denied refresh hides the cached list instead of presenting it as stale", async () => {
  const { context, page } = await open();
  try {
    await page.locator(".booking-desktop-table tbody").waitFor();
    await page.route("**/__preview/staff/bookings?*", (route) =>
      route.fulfill({
        status: 403,
        contentType: "application/json",
        body: JSON.stringify({ message: "Akses tidak diizinkan." }),
      }),
    );
    await page.getByRole("button", { name: "Perbarui data kunjungan", exact: true }).click();
    await page.locator('.bookings-refresh[data-freshness="access-error"]').waitFor();
    assert.equal(await page.locator(".booking-desktop-table").count(), 0);
    assert.match(
      await page.locator(".bookings-refresh [role=status]").innerText(),
      /Akses data berubah/,
    );
    await page.getByRole("button", { name: "Coba lagi", exact: true }).waitFor();
  } finally {
    await context.close();
  }
});

test("background updates preserve an unsaved staff form and mobile controls reflow", async () => {
  const { context, page, requests } = await open(
    "PA",
    "/admin/booking?new=1&kind=standard&q=keep&sort=name",
    320,
  );
  try {
    await page.locator("#appointment-purpose").fill("Keep this unsaved purpose during refresh.");
    await page.locator(".modal-scroll").evaluate((element) => {
      element.scrollTop = 160;
    });
    const scroll = await page.locator(".modal-scroll").evaluate((element) => element.scrollTop);
    const url = page.url();
    const baseline = requests.length;
    await page.clock.runFor(30100);
    await waitUntil(() => requests.length === baseline + 1);
    assert.equal(
      await page.locator("#appointment-purpose").inputValue(),
      "Keep this unsaved purpose during refresh.",
    );
    assert.equal(
      await page.locator(".modal-scroll").evaluate((element) => element.scrollTop),
      scroll,
    );
    assert.equal(page.url(), url);
    await page.getByRole("button", { name: "Batal", exact: true }).click();
    await page.getByRole("button", { name: "Buang isian", exact: true }).click();
    const layout = await page.locator(".bookings-refresh").evaluate((element) => {
      const rect = element.getBoundingClientRect();
      return {
        left: rect.left,
        right: rect.right,
        viewport: innerWidth,
        documentWidth: document.documentElement.scrollWidth,
      };
    });
    assert.ok(layout.left >= 0 && layout.right <= layout.viewport);
    assert.equal(layout.documentWidth, layout.viewport);
    await page.screenshot({
      path: new URL("staff-current-320.png", evidence).pathname.replace(/^\/(\w:)/, "$1"),
    });
    await writeFile(new URL("mobile-layout.json", evidence), JSON.stringify(layout, null, 2));
  } finally {
    await context.close();
  }
});

test("a background update does not silently advance the version of an open decision draft", async () => {
  const { context, page } = await open("PA", "/admin/booking?status=pending");
  try {
    await page.locator('.booking-desktop-table button[aria-label^="Lihat detail"]').first().click();
    const id = new URL(page.url()).searchParams.get("selected");
    const response = await context.request.get(origin + "/__preview/bookings?role=PA");
    const original = (await response.json()).find((booking) => booking.id === id);
    assert.ok(original);
    await page.getByRole("button", { name: "Tolak", exact: true }).click();
    const reason = page.getByRole("textbox", {
      name: "Pesan penolakan untuk pengaju",
      exact: true,
    });
    await reason.fill("Keep this draft while another staff member updates the visit.");
    await page.route("**/__preview/staff/bookings?*", async (route) => {
      const response = await route.fetch();
      const records = await response.json();
      await route.fulfill({
        response,
        json: records.map((booking) =>
          booking.id === id ? { ...booking, version: booking.version + 1 } : booking,
        ),
      });
    });
    await page.clock.runFor(30100);
    await page.getByText("Kunjungan ini diperbarui oleh petugas lain.", { exact: false }).waitFor();
    assert.equal(
      await reason.inputValue(),
      "Keep this draft while another staff member updates the visit.",
    );
    let submitted;
    await page.route(`**/__preview/staff/bookings/${id}/reject?*`, async (route) => {
      submitted = route.request().postDataJSON();
      await route.fulfill({
        status: 409,
        contentType: "application/json",
        body: JSON.stringify({ message: "Data telah berubah. Tinjau ulang sebelum melanjutkan." }),
      });
    });
    await page.getByRole("button", { name: "Tolak permohonan", exact: true }).click();
    await page.getByRole("alert").first().waitFor();
    assert.equal(submitted.version, original.version);
    assert.equal(
      await reason.inputValue(),
      "Keep this draft while another staff member updates the visit.",
    );
  } finally {
    await context.close();
  }
});
