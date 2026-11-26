import test from "node:test";
import assert from "node:assert/strict";
import { chromium } from "playwright";

const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3002";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);

const base = {
  code: "88000001",
  name: "Upcoming",
  organization: "Example",
  email: "test@example.test",
  phone: "081234567890",
  officerId: "o2",
  host: "Host",
  vip: false,
  date: "2099-10-20",
  time: "10:00",
  duration: 60,
  room: "Room",
  purpose: "Visit",
  visitors: ["Upcoming"],
  identityType: "KTP",
  status: "approved",
  version: 1,
  history: [],
  allowedActions: [],
};

async function open(browser, role, rows) {
  const context = await browser.newContext({
    timezoneId: "Asia/Makassar",
    viewport: { width: 1440, height: 900 },
  });
  await context.addInitScript((role) => sessionStorage.setItem("vms-preview-role", role), role);
  const page = await context.newPage();
  await page.route("**/__preview/staff/bookings?*", (route) =>
    route.fulfill({
      json: rows.map((booking) => ({
        // Restricted rows carry only their own fields; the rest start from a regular visit.
        ...(booking.visitKind === "restricted"
          ? { date: base.date, time: base.time, duration: 60, room: "", status: "approved", version: 1, history: [], allowedActions: [], roomId: null, code: "88000090" }
          : base),
        ...booking,
        timing: {
          serverTime: Date.now(),
          opensAt: Date.parse(`${booking.date || base.date}T09:30:00+08:00`),
          endsAt: Date.parse(`${booking.date || base.date}T11:00:00+08:00`),
          displayStatus: booking.status || base.status,
          canCheckIn: false,
          canCheckOut: false,
          canCancel: false,
        },
      })),
    }),
  );
  await page.goto(`${origin}/admin/booking`);
  await page.locator("tbody tr").first().waitFor();
  return { context, page };
}

async function choose(page, trigger, option) {
  await trigger.click();
  await page.locator(".checkbox-dropdown-popover").waitFor();
  await page.getByRole("checkbox", { name: option }).check();
  await page.keyboard.press("Escape");
  await page.locator(".checkbox-dropdown-popover").waitFor({ state: "detached" });
}

const names = (page, selector) =>
  page.locator(selector).allTextContents().then((all) => all.map((name) => name.trim()));

test("column headers sort and the status dropdown filters several statuses at once", async () => {
  const browser = await chromium.launch({ channel: "chrome", headless: true });
  try {
    const { context, page } = await open(browser, "PA", [
      { id: "past", name: "Past", status: "checked_out", date: "2000-01-01" },
      { id: "upcoming" },
      { id: "pending", name: "Pending", status: "pending" },
    ]);
    const rows = () => names(page, "tbody .row-link");
    assert.deepEqual(await rows(), ["Pending", "Upcoming", "Past"]);
    assert.equal(await page.getByRole("combobox", { name: "Urutan kunjungan" }).count(), 0);
    assert.equal(await page.getByLabel("VIP saja").count(), 0);
    assert.equal(await page.getByRole("button", { name: /^Tipe kunjungan/ }).count(), 0);

    await page.getByRole("button", { name: "Jadwal · WITA", exact: true }).click();
    await page.waitForFunction(() => document.querySelector("tbody .row-link")?.textContent.trim() === "Past");

    const status = page.getByRole("button", { name: /^Status / });
    assert.equal(await status.innerText(), "Semua status");
    await choose(page, status, /^Pending/);
    await choose(page, status, /^Disetujui/);
    assert.equal(new URL(page.url()).searchParams.get("status"), "pending,approved");
    assert.equal(await status.innerText(), "2 dipilih");
    await page.waitForFunction(() => document.querySelectorAll("tbody .row-link").length === 2);
    assert.deepEqual((await rows()).sort(), ["Pending", "Upcoming"]);

    await page.reload();
    await page.locator("tbody tr").first().waitFor();
    assert.equal(await status.innerText(), "2 dipilih");
    await status.click();
    await page.getByRole("button", { name: "Tampilkan semua", exact: true }).click();
    assert.equal(new URL(page.url()).searchParams.has("status"), false);
    await page.waitForFunction(() => document.querySelectorAll("tbody .row-link").length === 3);
    await context.close();
  } finally {
    await browser.close();
  }
});

test("Protocol filters by visit type instead of a VIP-only checkbox", async () => {
  const browser = await chromium.launch({ channel: "chrome", headless: true });
  try {
    const { context, page } = await open(browser, "Protocol Officer", [
      { id: "regular", name: "Regular" },
      { id: "vip", name: "Important", vip: true },
      { id: "restricted", visitKind: "restricted", reference: "TRB-1", restrictedView: "full" },
    ]);
    assert.equal(await page.getByLabel("VIP saja").count(), 0);
    const kind = page.getByRole("button", { name: /^Tipe kunjungan / });
    assert.equal(await kind.innerText(), "Semua tipe");
    await choose(page, kind, /^VIP/);
    assert.equal(new URL(page.url()).searchParams.get("list-kind"), "vip");
    assert.equal(await kind.innerText(), "VIP");
    await page.waitForFunction(() => document.querySelectorAll("tbody tr").length === 1);
    await choose(page, kind, /^Reguler/);
    assert.equal(new URL(page.url()).searchParams.get("list-kind"), "standard,vip");
    await page.waitForFunction(() => document.querySelectorAll("tbody tr").length === 2);
    await context.close();
  } finally {
    await browser.close();
  }
});
