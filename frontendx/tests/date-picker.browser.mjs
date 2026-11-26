import { startPreview } from "./helpers/preview-server.mjs";
import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";

const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3000";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || "playwright");
let browser;
const apis = [];
const fixedTime = "2096-02-28T08:00:00+08:00";
const evidence = fileURLToPath(new URL("../.impeccable/review/", import.meta.url));
before(async () => {
  await mkdir(evidence, { recursive: true });
  browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
});
after(async () => {
  await browser?.close();
  await Promise.all(apis.map((api) => api.close()));
});
async function testPage(viewport = { width: 1280, height: 1000 }, timezoneId = "Asia/Makassar") {
  const api = await startPreview({ time: fixedTime });
  apis.push(api);
  async function post(path, data) {
    const response = await fetch(api.base + path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    });
    assert.ok(response.ok);
    return response.json();
  }
  const booking = await post("/staff/bookings?role=PA", {
    idempotencyKey: crypto.randomUUID(),
    officerId: "o2",
    date: "2096-03-01",
    time: "10:00",
    name: "Calendar Example",
    organization: "Calendar Organization",
    email: "calendar@example.test",
    phone: "00123456789",
    purpose: "Calendar verification",
    visitors: ["Calendar Example"],
    identityType: "KTP",
  });
  assert.equal(booking.status, "approved");
  const page = await browser.newPage({ viewport, timezoneId, reducedMotion: "reduce" });
  if (timezoneId !== "America/Los_Angeles")
    await page.addInitScript((time) => {
      const NativeDate = Date;
      window.Date = class extends NativeDate {
        constructor(...args) {
          super(...(args.length ? args : [time]));
        }
        static now() {
          return time;
        }
      };
    }, Date.parse(fixedTime));
  await page.route("**/__preview/**", async (route) => {
    const url = new URL(route.request().url());
    await route.fulfill({
      response: await route.fetch({
        url: api.base + url.pathname.split("/__preview")[1] + url.search,
      }),
    });
  });
  page.setDefaultTimeout(12000);
  return page;
}
async function open(path, viewport, timezoneId) {
  const page = await testPage(viewport, timezoneId);
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto(origin + path);
  await page.locator(".date-picker-trigger").waitFor();
  if (path === "/booking") await page.locator("#officerId").selectOption("o2");
  return { page, errors };
}
async function calendar(page) {
  await page.locator(".date-picker-trigger").click();
  await page.locator(".date-picker-popover").waitFor();
  await page.locator('[data-slot="calendar"]').waitFor();
}
async function pickDate(page, iso) {
  const [year, month] = iso.split("-");
  await calendar(page);
  await page
    .getByRole("combobox", { name: "Pilih tahun", exact: true })
    .selectOption(String(Number(year)));
  await page
    .getByRole("combobox", { name: "Pilih bulan", exact: true })
    .selectOption(String(Number(month) - 1));
  await page.locator(`.shadcn-calendar-day[data-day="${iso}"]:not(:disabled)`).first().click();
  await page.locator(".date-picker-popover").waitFor({ state: "hidden" });
}
async function timePicker(page) {
  await page.locator(".time-picker-trigger").click();
  await page.locator(".time-picker-popover").waitFor();
  await page.getByRole("listbox", { name: "Jam", exact: true }).waitFor();
}
async function pickTime(page, value) {
  const [hour, minute] = value.split(":");
  await timePicker(page);
  await page
    .getByRole("listbox", { name: "Jam", exact: true })
    .getByRole("option", { name: hour, exact: true })
    .click();
  await page
    .getByRole("listbox", { name: "Menit", exact: true })
    .getByRole("option", { name: minute, exact: true })
    .click();
  assert.equal(await page.locator('[data-slot="time-picker-confirm"]').count(), 0);
  await page.locator(".date-picker > small").click();
  await page.locator(".time-picker-popover").waitFor({ state: "hidden" });
  assert.equal(await page.locator('[data-slot="time-picker-field"]').inputValue(), value);
}
function today() {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Makassar",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(fixedTime));
}

test("visitor shadcn calendar and ReUI time picker preserve ISO values, required validation, leap days and keyboard focus", async () => {
  const { page, errors } = await open("/booking");
  try {
    assert.equal(await page.locator(".date-picker-trigger").getAttribute("data-slot"), "button");
    assert.equal(
      await page.locator(".time-picker-trigger").getAttribute("data-slot"),
      "time-picker-trigger",
    );
    await page.locator("#officerId").selectOption("o2");
    await page.getByRole("button", { name: /^Lanjut/, exact: false }).click();
    await page.locator(".date-picker .field-error").waitFor();
    assert.equal(
      await page.locator(".date-picker-trigger").evaluate((e) => e === document.activeElement),
      true,
    );
    await calendar(page);
    assert.equal(
      await page.getByRole("button", { name: "Bulan sebelumnya", exact: true }).isDisabled(),
      true,
    );
    await page.getByRole("button", { name: "Bulan berikutnya", exact: true }).click();
    assert.equal(
      await page.getByRole("button", { name: "Bulan sebelumnya", exact: true }).isEnabled(),
      true,
    );
    await page.getByRole("button", { name: "Bulan sebelumnya", exact: true }).click();
    await page
      .locator(`.shadcn-calendar-day[data-day="${today()}"]:not(:disabled)`)
      .first()
      .click();
    assert.equal(await page.locator(".date-picker-value").inputValue(), today());
    await pickDate(page, "2096-02-29");
    assert.equal(await page.locator(".date-picker-value").inputValue(), "2096-02-29");
    await page.locator(".date-picker-trigger").press("Enter");
    await page.locator(".date-picker-popover").waitFor();
    await page.keyboard.press("Escape");
    await page.locator(".date-picker-popover").waitFor({ state: "hidden" });
    await page.waitForFunction(() =>
      document.activeElement?.classList.contains("date-picker-trigger"),
    );
    await pickTime(page, "15:45");
    await page.locator("#purpose").fill("Synthetic date and time input verification.");
    await page.getByRole("button", { name: /^Lanjut/, exact: false }).click();
    await page.waitForURL(/\/booking\/visitors$/);
    assert.deepEqual(errors, []);
  } finally {
    await page.close();
  }
});

test("staff calendar closes before its dialog and date/time changes retain the discard prompt", async () => {
  const { page, errors } = await open("/admin/booking?new=1");
  try {
    await pickDate(page, "2096-02-29");
    await pickTime(page, "16:59");
    assert.equal(await page.locator('[data-slot="time-picker-field"]').inputValue(), "16:59");
    await calendar(page);
    assert.equal(
      await page.locator(".date-picker-popover").getAttribute("data-slot"),
      "popover-content",
    );
    await page.screenshot({ path: `${evidence}/shadcn-date-time-staff-desktop.png` });
    await page.keyboard.press("Escape");
    await page.locator(".date-picker-popover").waitFor({ state: "hidden" });
    assert.equal(await page.locator(".modal").count(), 1, "Escape closed the outer staff dialog");
    await page.waitForFunction(() =>
      document.activeElement?.classList.contains("date-picker-trigger"),
    );
    assert.equal(await page.locator(".modal").count(), 1);
    await page.getByRole("button", { name: "Batal", exact: true }).click();
    await page.getByRole("button", { name: "Buang isian", exact: true }).waitFor();
    assert.deepEqual(errors, []);
  } finally {
    await page.close();
  }
});

test("rescheduling submits ISO date and minute-precision time through native FormData", async () => {
  const page = await testPage();
  try {
    await page.goto(origin + "/admin/booking?status=approved");
    await page.locator('button[aria-label^="Lihat detail"]:visible').first().click();
    await page.getByRole("button", { name: "Edit kunjungan", exact: true }).click();
    await page.locator(".date-picker-trigger").waitFor();
    const current = await page.locator(".date-picker-value").inputValue();
    assert.match(current, /^\d{4}-\d{2}-\d{2}$/);
    const invalid = await page.locator(".date-picker-value").evaluate((e) => {
      e.value = "2020-01-01";
      return e.checkValidity();
    });
    assert.equal(invalid, false);
    const save = page.getByRole("button", { name: "Simpan perubahan", exact: true });
    assert.equal(await save.count(), 0);
    await pickDate(page, "2096-02-29");
    await save.waitFor();
    await pickDate(page, current);
    assert.equal(await save.count(), 0);
    await pickTime(page, "16:59");
    await save.waitFor();
    await pickDate(page, "2096-02-29");
    const data = await page
      .locator(".modal form")
      .evaluate((form) => Object.fromEntries(new FormData(form)));
    assert.equal(data.date, "2096-02-29");
    assert.equal(data.time, "16:59");
    assert.equal(await page.locator(".modal").count(), 1);
  } finally {
    await page.close();
  }
});

test("restricted creation accepts date/time and editing keeps both controls read-only", async () => {
  const page = await testPage();
  page.setDefaultTimeout(12000);
  try {
    await page.goto(origin + "/admin/booking");
    await page.locator("#active-role").selectOption("Protocol Officer");
    await page.waitForURL(/\/admin\/dashboard$/);
    await page.goto(origin + "/admin/booking");
    await page.locator(".appointment-type-menu summary").click();
    await page.getByRole("button", { name: /Kunjungan Terbatas/ }).click();
    await pickDate(page, "2096-02-29");
    await pickTime(page, "16:59");
    await page.locator("#restricted-host").selectOption("o1");
    await page.getByRole("button", { name: "Buat kunjungan terbatas", exact: true }).click();
    const edit = page.getByRole("button", { name: "Edit kunjungan", exact: true });
    await edit.waitFor();
    const visitor = page.getByRole("region", { name: "Detail pengunjung", exact: true });
    const visit = page.getByRole("region", { name: "Detail kunjungan", exact: true });
    const editVisitor = visitor.getByRole("button", { name: "Edit detail pengunjung" });
    const editSchedule = visit.getByRole("button", { name: "Edit jadwal kunjungan" });
    assert.equal(await editVisitor.count(), 0);
    assert.equal(await editSchedule.count(), 0);
    await edit.press("Enter");
    await editSchedule.click();
    await page.getByRole("dialog", { name: "Edit jadwal kunjungan" }).waitFor();
    assert.equal(await page.locator('[data-slot="time-picker-field"]').inputValue(), "16:59");
    await page.getByRole("button", { name: "Kembali", exact: true }).click();
    await editSchedule.and(page.locator(":focus")).waitFor();
    await editVisitor.press("Enter");
    await page.getByRole("heading", { name: "Edit kunjungan terbatas", exact: true }).waitFor();
    assert.equal(await page.locator(".date-picker-value").inputValue(), "2096-02-29");
    assert.equal(await page.locator(".date-picker-trigger").isDisabled(), true);
    assert.equal(await page.locator('[data-slot="time-picker-field"]').inputValue(), "16:59");
    assert.equal(await page.locator("#restricted-time").isDisabled(), true);
    assert.equal(
      await page.locator('[data-slot="time-picker-field"]').getAttribute("readonly"),
      "",
    );
    await page.screenshot({ path: `${evidence}/shadcn-date-time-readonly.png` });
    const save = page.getByRole("button", { name: "Simpan perubahan", exact: true });
    assert.equal(await save.count(), 0);
    const name = page.locator("#restricted-name");
    await name.fill("Nama pengunjung");
    await save.waitFor();
    await name.fill("");
    assert.equal(await save.count(), 0);
  } finally {
    await page.close();
  }
});

for (const width of [320, 390, 768]) {
  test(`shadcn date/time fields and calendars fit visitor/staff at ${width}px`, async () => {
    for (const [path, kind] of [
      ["/booking", "visitor"],
      ["/admin/booking?new=1", "staff"],
    ]) {
      const { page, errors } = await open(path, { width, height: 900 });
      try {
        await calendar(page);
        const rect = await page.locator(".date-picker-popover").boundingBox();
        assert.ok(rect.width <= 285, "Calendar is not compact");
        assert.ok(rect.x >= 0 && rect.x + rect.width <= width + 1, JSON.stringify(rect));
        assert.ok(rect.y >= 0 && rect.y + rect.height <= 901, "Calendar clipped vertically");
        assert.equal(
          await page.evaluate(() => document.documentElement.scrollWidth > innerWidth),
          false,
        );
        await page.screenshot({ path: `${evidence}/shadcn-date-time-${kind}-${width}.png` });
        await page.getByRole("button", { name: "Tutup kalender", exact: true }).click();
        if (kind === "visitor") await pickDate(page, "2096-02-29");
        await pickTime(page, "15:45");
        assert.equal(await page.locator('[data-slot="time-picker-field"]').inputValue(), "15:45");
        await timePicker(page);
        const timeRect = await page.locator(".time-picker-popover").boundingBox();
        assert.ok(
          timeRect.width <= 217 && timeRect.x >= 0 && timeRect.x + timeRect.width <= width + 1,
          "Time picker is not compact or is clipped",
        );
        assert.ok(
          timeRect.y >= 0 && timeRect.y + timeRect.height <= 901,
          "Time picker is clipped vertically",
        );
        await page.screenshot({ path: `${evidence}/compact-time-${kind}-${width}.png` });
        await page.keyboard.press("Escape");
        await page.locator(".time-picker-popover").waitFor({ state: "hidden" });
        await page.waitForFunction(() =>
          document.activeElement?.classList.contains("time-picker-trigger"),
        );
        if (kind === "staff") assert.equal(await page.locator(".modal").count(), 1);
        assert.deepEqual(errors, []);
      } finally {
        await page.close();
      }
    }
  });
}

test("calendar selection stays on the same WITA date in a different browser timezone", async () => {
  const { page, errors } = await open(
    "/booking",
    { width: 1280, height: 1000 },
    "America/Los_Angeles",
  );
  try {
    await pickDate(page, "2096-02-29");
    assert.equal(await page.locator(".date-picker-value").inputValue(), "2096-02-29");
    await page.waitForFunction(() =>
      document.querySelector(".date-picker-trigger")?.textContent.includes("29 Februari 2096"),
    );
    assert.match(await page.locator(".date-picker-trigger").innerText(), /29 Februari 2096/);
    assert.deepEqual(errors, []);
  } finally {
    await page.close();
  }
});

test("ReUI time columns save immediately, retain selections on close, and support keyboard, clear and required errors", async () => {
  const { page, errors } = await open("/booking");
  try {
    await pickDate(page, "2096-02-29");
    await timePicker(page);
    await page
      .getByRole("listbox", { name: "Jam", exact: true })
      .getByRole("option", { name: "14", exact: true })
      .click();
    await page
      .getByRole("listbox", { name: "Menit", exact: true })
      .getByRole("option", { name: "35", exact: true })
      .click();
    assert.equal(await page.locator('[data-slot="time-picker-field"]').inputValue(), "14:35");
    await page.keyboard.press("Escape");
    await page.locator(".time-picker-popover").waitFor({ state: "hidden" });
    assert.equal(await page.locator('[data-slot="time-picker-field"]').inputValue(), "14:35");
    await timePicker(page);
    const hours = page.getByRole("listbox", { name: "Jam", exact: true });
    await hours.focus();
    await hours.press("Home");
    await hours.press("ArrowRight");
    const minutes = page.getByRole("listbox", { name: "Menit", exact: true });
    assert.equal(await minutes.evaluate((e) => e === document.activeElement), true);
    await minutes.pressSequentially("45");
    await minutes.press("Enter");
    await page.locator(".time-picker-popover").waitFor({ state: "hidden" });
    assert.equal(await page.locator('[data-slot="time-picker-field"]').inputValue(), "08:45");
    await timePicker(page);
    await page.locator('[data-slot="time-picker-clear"]').click();
    await page.locator(".date-picker > small").click();
    await page.locator(".time-picker-popover").waitFor({ state: "hidden" });
    assert.equal(await page.locator('[data-slot="time-picker-field"]').inputValue(), "");
    assert.equal(
      await page.locator('[data-slot="time-picker-field"]').evaluate((e) => e.checkValidity()),
      false,
    );
    await page.locator(".time-picker .field-error").waitFor();
    assert.deepEqual(errors, []);
  } finally {
    await page.close();
  }
});
