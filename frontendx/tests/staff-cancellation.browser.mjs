import test from "node:test";
import assert from "node:assert/strict";
import { chromium } from "playwright";
import { startPreview } from "./helpers/preview-server.mjs";
import { mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";

const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3000";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);
test("late staff cancellation keeps its reason through a conflict and submits only after explicit review", async () => {
  const browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
  const api = await startPreview({ time: "2099-08-19T09:00:00+08:00" });
  async function post(path, data) {
    const response = await fetch(api.base + path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    });
    assert.ok(response.ok);
    return response.json();
  }
  try {
    let visit = await post("/staff/bookings?role=PA", {
      idempotencyKey: crypto.randomUUID(),
      officerId: "o2",
      date: "2099-08-19",
      time: "10:00",
      name: "Cancellation Browser Example",
      organization: "Example Organization",
      email: "cancel@example.test",
      phone: "00123456789",
      purpose: "Cancellation browser verification",
      visitors: ["Cancellation Browser Example"],
      identityType: "KTP",
    });
    assert.equal(visit.status, "approved");
    api.setNow("2099-08-19T10:30:00+08:00");
    const context = await browser.newContext({
      viewport: { width: 390, height: 900 },
      timezoneId: "Asia/Makassar",
      reducedMotion: "reduce",
    });
    const attempts = [],
      errors = [];
    await context.route("**/__preview/**", async (route) => {
      const url = new URL(route.request().url());
      if (url.pathname.endsWith("/cancel")) attempts.push(route.request().postDataJSON());
      await route.fulfill({
        response: await route.fetch({
          url: api.base + url.pathname.split("/__preview")[1] + url.search,
        }),
      });
    });
    const page = await context.newPage();
    page.setDefaultTimeout(10000);
    page.on("pageerror", (error) => errors.push(error.message));
    await page.clock.install({ time: new Date("2099-08-19T10:30:00+08:00") });
    await page.goto(origin + `/admin/booking?selected=${visit.id}`);
    await page.getByRole("button", { name: "Batalkan kunjungan", exact: true }).click();
    const reason = page.getByRole("textbox", { name: "Alasan pembatalan", exact: false });
    await reason.fill("     ");
    await page.getByRole("button", { name: "Batalkan kunjungan", exact: true }).click();
    await page.getByRole("alert").filter({ hasText: "Tuliskan alasan pembatalan" }).waitFor();
    assert.equal(attempts.length, 0);
    const text = "PRIVATE cancellation after start.";
    await reason.fill(text);
    const internalNote = page.getByRole("textbox", { name: "Catatan internal", exact: false });
    await internalNote.fill("INTERNAL confidential follow-up.");
    assert.equal(await internalNote.getAttribute("required"), null);
    visit = await post(`/bookings/${visit.id}/reschedule?role=PA`, {
      version: visit.version,
      date: "2099-08-19",
      time: "11:30",
      roomId: "room-1",
      duration: 60,
      reason: "Concurrent scheduling change.",
    });
    await page.getByRole("button", { name: "Batalkan kunjungan", exact: true }).click();
    await page.getByRole("alert").filter({ hasText: "Data telah berubah" }).waitFor();
    assert.equal(await reason.inputValue(), text);
    assert.equal(await internalNote.inputValue(), "INTERNAL confidential follow-up.");
    await page
      .getByRole("status")
      .filter({ hasText: "Kunjungan ini diperbarui oleh petugas lain" })
      .waitFor();
    await page.getByRole("button", { name: "Kembali", exact: true }).click();
    await page.getByRole("button", { name: "Batalkan kunjungan", exact: true }).click();
    assert.equal(await reason.inputValue(), text);
    const evidence = new URL("../.task-cache/cancellation-fields/", import.meta.url);
    await mkdir(evidence, { recursive: true });
    await internalNote.scrollIntoViewIfNeeded();
    await page.screenshot({
      path: fileURLToPath(new URL("form-390.png", evidence)),
      fullPage: true,
    });
    await page.getByRole("button", { name: "Batalkan kunjungan", exact: true }).click();
    await page
      .getByRole("status")
      .filter({ hasText: "Kunjungan dibatalkan oleh petugas." })
      .waitFor();
    assert.equal(attempts.length, 2);
    assert.equal(attempts[1].version, visit.version);
    assert.equal(attempts[1].reason, text);
    assert.equal(attempts[1].internalCancellationNote, "INTERNAL confidential follow-up.");
    await page.goto(origin + `/admin/booking?selected=${visit.id}`);
    await page
      .getByRole("region", { name: "Alasan pembatalan", exact: true })
      .getByText(text)
      .waitFor();
    await page
      .getByRole("region", { name: "Catatan internal pembatalan" })
      .getByText("INTERNAL confidential follow-up.")
      .waitFor();
    await page
      .getByRole("region", { name: "Catatan internal pembatalan" })
      .scrollIntoViewIfNeeded();
    assert.equal(
      await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
      true,
    );
    await page.screenshot({
      path: fileURLToPath(new URL("detail-390.png", evidence)),
      fullPage: true,
    });
    assert.deepEqual(errors, []);
    const publicVisit = await post("/lookup", { code: visit.code, email: visit.email });
    assert.equal(publicVisit.status, "cancelled");
    assert.equal(publicVisit.cancellationReason, text);
    assert.equal(JSON.stringify(publicVisit).includes("INTERNAL"), false);
    assert.equal(JSON.stringify(publicVisit.history).includes("PRIVATE"), false);
    await page.goto((process.env.VMS_VISITOR_URL || origin) + "/status");
    await page.locator("#status-code").fill(visit.code);
    await page.locator("#status-email").fill(visit.email);
    await page.getByRole("button", { name: /^(Lihat kunjungan|View visit)$/ }).click();
    await page
      .getByRole("region", { name: /^(Alasan pembatalan|Reason for cancellation)$/ })
      .getByText(text)
      .waitFor();
    assert.equal((await page.locator("body").innerText()).includes("INTERNAL confidential"), false);
    await page.screenshot({
      path: fileURLToPath(new URL("visitor-390.png", evidence)),
      fullPage: true,
    });
    await context.close();
  } finally {
    await browser.close();
    await api.close();
  }
});
