import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { startPreview } from "./helpers/preview-server.mjs";

const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || "playwright");
const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3000";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);
const evidence = new URL("../anti-slop/evidence-2026-10-30/restricted-arrival/", import.meta.url);
const protocol = encodeURIComponent("Protocol Officer");
let browser, api;
before(async () => {
  await mkdir(evidence, { recursive: true });
  browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
});
after(async () => browser?.close());
async function post(path, body) {
  const response = await fetch(api.base + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const data = await response.json();
  assert.ok(response.ok, JSON.stringify(data));
  return data;
}
async function fixture(extra = {}) {
  return post(`/staff/bookings?role=${protocol}`, {
    visitKind: "restricted",
    date: "2099-08-19",
    time: "10:00",
    name: "Terbatas Browser Name",
    organization: "Terbatas Browser Org",
    serviceId: "0012345678",
    officerId: "o2",
    idempotencyKey: crypto.randomUUID(),
    ...extra,
  });
}
async function open(role, route, width = 390) {
  const context = await browser.newContext({
    viewport: { width, height: 900 },
    timezoneId: "Asia/Makassar",
    reducedMotion: "reduce",
  });
  await context.addInitScript((role) => sessionStorage.setItem("vms-preview-role", role), role);
  await context.route("**/__preview/**", async (route) => {
    const url = new URL(route.request().url());
    const response = await route.fetch({
      url: api.base + url.pathname.split("/__preview")[1] + url.search,
    });
    await route.fulfill({ response });
  });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  await page.clock.install({ time: new Date("2099-08-19T09:00:00+08:00") });
  await page.goto(origin + route);
  return { context, page };
}
async function save(page, name) {
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
  await page.screenshot({ path: fileURLToPath(new URL(`${name}.png`, evidence)), fullPage: true });
  await writeFile(new URL(`${name}.aria.txt`, evidence), await page.locator("body").ariaSnapshot());
}
async function withApi(run) {
  api = await startPreview({ time: "2099-08-19T09:00:00+08:00" });
  try {
    await run();
  } finally {
    await api.close();
  }
}

test("Terbatas four-field form preserves leading zeroes, dirty draft and keyboard focus at 320px", async () =>
  withApi(async () => {
    const { page, context } = await open(
      "Protocol Officer",
      "/admin/booking?new=1&kind=restricted",
      320,
    );
    try {
      await page.getByRole("heading", { name: "Buat kunjungan terbatas", exact: true }).waitFor();
      for (const id of [
        "#restricted-name",
        "#restricted-organization",
        "#restricted-serviceId",
        "#restricted-host",
      ])
        assert.equal(await page.locator(id).count(), 1);
      assert.equal(
        await page
          .locator('.modal input[type="email"], .modal input[type="file"], .modal textarea')
          .count(),
        0,
      );
      await page.locator("#restricted-serviceId").fill("0012345678");
      await page.keyboard.press("Escape");
      await page.getByRole("heading", { name: "Buang perubahan?" }).waitFor();
      await page.waitForFunction(
        () =>
          document.activeElement?.tagName === "BUTTON" &&
          document.activeElement.textContent === "Lanjutkan mengisi",
      );
      assert.equal(
        await page
          .getByRole("button", { name: "Lanjutkan mengisi" })
          .evaluate((node) => node === document.activeElement),
        true,
      );
      await page.keyboard.press("Enter");
      assert.equal(await page.locator("#restricted-serviceId").inputValue(), "0012345678");
      await page.locator(".date-picker-trigger").click();
      await page.getByRole("combobox", { name: "Pilih tahun", exact: true }).selectOption("2099");
      await page.getByRole("combobox", { name: "Pilih bulan", exact: true }).selectOption("7");
      await page
        .locator('.shadcn-calendar-day[data-day="2099-08-19"]:not(:disabled)')
        .first()
        .click();
      await page.locator(".time-picker-trigger").click();
      await page
        .getByRole("listbox", { name: "Jam", exact: true })
        .getByRole("option", { name: "10", exact: true })
        .click();
      await page
        .getByRole("listbox", { name: "Menit", exact: true })
        .getByRole("option", { name: "00", exact: true })
        .click();
      await page.locator(".date-picker > small").click();
      await page.locator("#restricted-name").fill("Terbatas Form Name");
      await page.locator("#restricted-organization").fill("Terbatas Form Org");
      await page.locator("#restricted-host").selectOption("o2");
      await save(page, "form-320");
      const [response] = await Promise.all([
        page.waitForResponse(
          (r) =>
            r.request().method() === "POST" &&
            new URL(r.url()).pathname.endsWith("/staff/bookings"),
        ),
        page.getByRole("button", { name: "Buat kunjungan terbatas", exact: true }).click(),
      ]);
      assert.equal(response.status(), 201);
      const b = await response.json();
      assert.equal(b.status, "approved");
      assert.equal(b.serviceId, "0012345678");
      await page.getByRole("heading", { name: "Detail kunjungan", exact: true }).waitFor();
      await page.getByRole("button", { name: "Edit detail", exact: true }).click();
      assert.equal(await page.locator("#restricted-serviceId").inputValue(), "0012345678");
      await page.locator("#restricted-serviceId").fill("");
      await page.getByRole("button", { name: "Simpan perubahan", exact: true }).click();
      await page.locator(".modal").waitFor({ state: "hidden" });
    } finally {
      await context.close();
    }
  }));

test("PA and Admin read permitted Terbatas details without write controls; general exports exclude it", async () =>
  withApi(async () => {
    const b = await fixture();
    for (const role of ["PA", "Super Admin"]) {
      const { page, context } = await open(
        role,
        `/admin/booking?selected=${b.id}`,
        role === "PA" ? 390 : 1280,
      );
      try {
        await page.getByRole("heading", { name: "Detail kunjungan", exact: true }).waitFor();
        const text = await page.locator(".modal").innerText();
        assert.ok(text.includes(b.name));
        assert.ok(text.includes(b.organization));
        assert.ok(text.includes(role === "PA" ? "5678" : "0012345678"));
        if (role === "PA") assert.equal(text.includes("0012345678"), false);
        assert.equal(
          await page
            .locator(".modal")
            .getByRole("button", { name: /^(Setujui|Tolak|Ubah jadwal|Edit detail)$/ })
            .count(),
          0,
        );
        await save(page, role === "PA" ? "pa-read-390" : "admin-read-1280");
      } finally {
        await context.close();
      }
    }
    await post("/bookings", {
      ack: true,
      idempotencyKey: crypto.randomUUID(),
      officerId: "o1",
      date: "2099-08-20",
      time: "10:00",
      name: "Ordinary Export Name",
      organization: "Ordinary Export Org",
      email: "ordinary@example.test",
      phone: "08123456789",
      purpose: "General report fixture",
      visitors: ["Ordinary Export Name"],
      identityType: "KTP",
    });
    const { page, context } = await open("Protocol Officer", "/admin/logs", 1280);
    try {
      await page.getByTestId("report-total").filter({ hasText: "1" }).waitFor();
      assert.equal((await page.locator("body").innerText()).includes(b.name), false);
      const [download] = await Promise.all([
        page.waitForEvent("download"),
        page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).click(),
      ]);
      const content = await readFile(await download.path(), "utf8");
      assert.ok(content.includes('"Tanggal awal","Tanggal akhir","Cakupan","Status","Jumlah"'));
      assert.equal(content.includes("Ordinary Export Name"), false);
      for (const secret of [b.name, b.organization, b.serviceId, b.reference])
        assert.equal(content.includes(secret), false, secret);
    } finally {
      await context.close();
    }
  }));

test("reception gates identity by code, blocks early arrival, then permits check-in and next-day checkout", async () =>
  withApi(async () => {
    const b = await fixture();
    assert.equal(b.status, "approved");
    const { page, context } = await open("Receptionist", `/admin/booking?selected=${b.id}`, 390);
    try {
      await page.getByRole("heading", { name: "Detail kunjungan", exact: true }).waitFor();
      let text = await page.locator(".modal").innerText();
      for (const secret of [b.name, b.organization, b.serviceId, b.code, "5678"])
        assert.equal(text.includes(secret), false, secret);
      await save(page, "reception-redacted-390");
      await page.goto(origin + "/admin/check-in");
      await page.getByRole("button", { name: /Kode kunjungan Masukkan/ }).click();
      await page.locator("#booking-code").fill(b.code);
      await page.getByRole("button", { name: "Periksa kunjungan", exact: true }).click();
      await page.getByRole("heading", { name: "Validasi pengunjung", exact: true }).waitFor();
      text = await page.locator(".modal").innerText();
      assert.ok(text.includes(b.name));
      assert.ok(text.includes("5678"));
      assert.equal(text.includes(b.organization), false);
      assert.equal(text.includes(b.serviceId), false);
      await page.getByRole("checkbox").check();
      assert.equal(
        await page.getByRole("button", { name: "Konfirmasi check in", exact: true }).isDisabled(),
        true,
      );
      await save(page, "reception-early-390");
      api.setNow("2099-08-19T09:30:00+08:00");
      await page.getByRole("button", { name: "Periksa kode lain", exact: true }).click();
      await page.locator("#booking-code").fill(b.code);
      await page.getByRole("button", { name: "Periksa kunjungan", exact: true }).click();
      await page.getByRole("checkbox").check();
      await page.getByRole("button", { name: "Konfirmasi check in", exact: true }).focus();
      await page.keyboard.press("Enter");
      await page.locator(".modal").waitFor({ state: "hidden" });
      api.setNow("2099-08-20T15:00:00+08:00");
      await page.goto(origin + "/admin/check-out");
      await page.getByText(b.reference, { exact: true }).first().waitFor();
      await page.getByRole("button", { name: /Kode kunjungan Masukkan/ }).click();
      await page.locator("#booking-code").fill(b.code);
      await page.getByRole("button", { name: "Periksa kunjungan", exact: true }).click();
      await page.getByRole("checkbox").check();
      await page.getByRole("button", { name: "Konfirmasi check out", exact: true }).click();
      await page.locator(".modal").waitFor({ state: "hidden" });
    } finally {
      await context.close();
    }
  }));

test("a stale arrival confirmation clears previously resolved identity and requires another lookup", async () =>
  withApi(async () => {
    const b = await fixture();
    assert.equal(b.status, "approved");
    api.setNow("2099-08-19T09:30:00+08:00");
    const { page, context } = await open("Receptionist", "/admin/check-in");
    try {
      const trigger = page.getByRole("button", { name: /Kode kunjungan Masukkan/ });
      await trigger.click();
      await page.locator("#booking-code").fill(b.code);
      await page.getByRole("button", { name: "Periksa kunjungan", exact: true }).click();
      await page.getByRole("checkbox").check();
      await post(`/bookings/${b.id}/update?role=${protocol}`, {
        visitKind: "restricted",
        date: b.date,
        time: b.time,
        version: b.version,
        name: "Updated Arrival Name",
        officerId: "o1",
      });
      await page.getByRole("button", { name: "Konfirmasi check in", exact: true }).click();
      await page.locator("#booking-code").waitFor();
      assert.equal((await page.locator(".modal").innerText()).includes(b.name), false);
      await page.getByRole("alert").filter({ hasText: "Data telah berubah" }).waitFor();
      await page.locator("#booking-code").fill(b.code);
      await page.getByRole("button", { name: "Periksa kunjungan", exact: true }).click();
      await page.getByText("Updated Arrival Name", { exact: true }).waitFor();
      assert.equal(await page.getByRole("checkbox").isChecked(), false);
      await page.keyboard.press("Escape");
      assert.equal(await trigger.evaluate((node) => node === document.activeElement), true);
    } finally {
      await context.close();
    }
  }));

test("Missed appears in staff filtering and visitor ticket with no entry QR", async () =>
  withApi(async () => {
    const ordinary = await post("/bookings", {
      ack: true,
      idempotencyKey: crypto.randomUUID(),
      officerId: "o2",
      date: "2099-08-19",
      time: "10:00",
      name: "Missed Browser Name",
      organization: "Missed Browser Org",
      email: "missed@example.test",
      phone: "08123456789",
      purpose: "Arrival expiry fixture",
      visitors: ["Missed Browser Name"],
      identityType: "KTP",
    });
    await post(`/bookings/${ordinary.id}/approve?role=PA`, {
      version: 1,
      room: "Ruang Rapat Meranti",
      duration: 60,
    });
    api.setNow("2099-08-19T11:00:00.001+08:00");
    const staff = await open("PA", `/admin/booking?status=missed&selected=${ordinary.id}`, 390);
    try {
      await staff.page.getByRole("heading", { name: "Detail kunjungan", exact: true }).waitFor();
      assert.ok((await staff.page.locator(".modal").innerText()).includes("Terlewat"));
      assert.equal(
        await staff.page.getByRole("button", { name: "Ubah jadwal", exact: true }).count(),
        0,
      );
      await save(staff.page, "missed-staff-390");
    } finally {
      await staff.context.close();
    }
    const { page, context } = await open("PA", "/status", 390);
    try {
      await page.locator("#status-email").fill("missed@example.test");
      await page.locator(".input-otp input").fill(ordinary.code);
      await page.getByRole("button", { name: "Lihat kunjungan", exact: true }).click();
      await page.locator(".visit-ticket--missed").waitFor();
      assert.equal(await page.locator(".visit-ticket canvas, .visit-ticket img").count(), 0);
      await save(page, "missed-visitor-390");
    } finally {
      await context.close();
    }
  }));
