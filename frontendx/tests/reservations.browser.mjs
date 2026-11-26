import test from "node:test";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";

const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || "playwright");
const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3000";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);
const evidence = new URL("../anti-slop/evidence-2026-10-30/reservations/", import.meta.url);

test("manual visits start approved and rescheduling conflicts preserve input", async () => {
  await mkdir(evidence, { recursive: true });
  const browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
  let page;
  try {
    const context = await browser.newContext({
      viewport: { width: 390, height: 844 },
      reducedMotion: "reduce",
      timezoneId: "Asia/Makassar",
      hasTouch: true,
    });
    await context.addInitScript(() => sessionStorage.setItem("vms-preview-role", "PA"));
    const existing = await context.request
      .get(origin + "/__preview/bookings?role=PA")
      .then((response) => response.json());
    const date = Array.from(
      { length: 28 },
      (_, i) => `2097-07-${String(i + 1).padStart(2, "0")}`,
    ).find((date) => !existing.some((booking) => booking.date === date));
    assert.ok(date, "No unused synthetic test date");
    const fixture = await context.request.post(origin + "/__preview/staff/bookings?role=Protocol%20Officer", {
      data: {
        visitKind: "restricted",
        officerId: "o3",
        date,
        time: "09:00",
        name: "Browser Reservation Fixture",
        organization: "QA Example",
        idempotencyKey: crypto.randomUUID(),
      },
    });
    assert.equal(fixture.status(), 201);
    const committed = await fixture.json();
    const approval = await context.request.post(
      origin + `/__preview/bookings/${committed.id}/reschedule?role=Protocol%20Officer`,
      {
        data: {
          version: committed.version,
          date,
          time: "09:00",
          room: "Ruang Rapat Nusantara",
          duration: 60,
          reason: "Assign fixture meeting room.",
        },
      },
    );
    assert.equal(approval.status(), 200);

    page = await context.newPage();
    page.setDefaultTimeout(12000);
    const errors = [];
    let creations = 0;
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("request", (request) => {
      if (
        request.method() === "POST" &&
        new URL(request.url()).pathname.endsWith("/__preview/staff/bookings")
      )
        creations++;
    });
    await page.goto(origin + "/admin/booking?new=1&kind=standard");
    assert.equal(await page.locator("#appointment-officerId").inputValue(), "Direktorat Data dan Kecerdasan Buatan");
    assert.equal(await page.locator("#appointment-officerId").getAttribute("readonly"), "");
    await page.locator(".date-picker-trigger").click();
    await page.getByRole("combobox", { name: "Pilih tahun", exact: true }).selectOption("2097");
    await page.getByRole("combobox", { name: "Pilih bulan", exact: true }).selectOption("6");
    await page.locator(`.shadcn-calendar-day[data-day="${date}"]:not(:disabled)`).first().click();
    await page.locator(".time-picker-trigger").click();
    await page
      .getByRole("listbox", { name: "Jam", exact: true })
      .getByRole("option", { name: "09", exact: true })
      .click();
    await page
      .getByRole("listbox", { name: "Menit", exact: true })
      .getByRole("option", { name: "00", exact: true })
      .click();
    await page.locator(".date-picker > small").click();
    await page
      .locator("#appointment-purpose")
      .fill("Create an approved visit for a different officer.");
    await page.getByRole("button", { name: "Lanjutkan", exact: true }).click();
    await page.locator("#appointment-name").fill("Browser Approved Visit");
    await page.locator("#appointment-organization").fill("QA Example");
    await page.locator("#appointment-email").fill("reservation@example.test");
    await page.locator("#appointment-phone").fill("081234567890");
    await page.getByRole("button", { name: "Lanjutkan", exact: true }).click();
    assert.equal(
      await page.locator("#appointment-photo-error").innerText(),
      "Unggah foto wajah perwakilan.",
    );
    await page.locator("#appointment-photo").setInputFiles({
      name: "visitor.png",
      mimeType: "image/png",
      buffer: Buffer.from(
        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=",
        "base64",
      ),
    });
    await page.getByRole("button", { name: "Lanjutkan", exact: true }).click();
    await page.getByRole("heading", { name: "Periksa janji temu", exact: true }).waitFor();
    await page.waitForLoadState("networkidle");
    assert.equal(creations, 0, "Moving to review must not create the request");
    const [createdResponse] = await Promise.all([
      page.waitForResponse(
        (response) =>
          response.request().method() === "POST" &&
          new URL(response.url()).pathname.endsWith("/__preview/staff/bookings"),
      ),
      page.getByRole("button", { name: "Buat janji temu", exact: true }).click(),
    ]);
    assert.equal(createdResponse.status(), 201);
    assert.equal(creations, 1);
    const created = await createdResponse.json();
    assert.equal(created.status, "approved");

    await page.getByRole("button", { name: "Edit kunjungan", exact: true }).click();
    await page.locator("#room").selectOption("Ruang Rapat Nusantara");
    await page.locator("#duration").fill("45");
    await page.locator("#reason").fill("Assign a meeting room to this visit.");
    await page.getByRole("button", { name: "Simpan perubahan", exact: true }).focus();
    await page.keyboard.press("Enter");
    await page
      .getByRole("alert")
      .filter({ hasText: "Ruangan atau pejabat memiliki jadwal yang bertabrakan." })
      .waitFor();
    assert.equal(await page.locator("#room").inputValue(), "Ruang Rapat Nusantara");
    assert.equal(await page.locator("#duration").inputValue(), "45");
    assert.equal(await page.locator(".modal").count(), 1);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth), 390);
    await page.screenshot({
      path: new URL("approval-conflict-390.png", evidence).pathname.replace(/^\/(\w:)/, "$1"),
    });
    await page.locator("#room").selectOption("Ruang Rapat Meranti");
    const [result] = await Promise.all([
      page.waitForResponse((response) =>
        new URL(response.url()).pathname.endsWith(`/__preview/bookings/${created.id}/reschedule`),
      ),
      page.getByRole("button", { name: "Simpan perubahan", exact: true }).click(),
    ]);
    assert.equal(result.status(), 200);
    const approved = await result.json();
    assert.equal(approved.status, "approved");
    assert.equal(approved.room, "Ruang Rapat Meranti");
    assert.equal(approved.duration, 45);
    await page.locator(".modal").waitFor({ state: "hidden" });
    assert.deepEqual(errors, []);
  } catch (error) {
    if (page && !page.isClosed()) {
      await page.screenshot({
        path: new URL("failure.png", evidence).pathname.replace(/^\/(\w:)/, "$1"),
      });
      await writeFile(
        new URL("failure.aria.txt", evidence),
        await page.locator("body").ariaSnapshot(),
      );
      console.log(await page.locator(".modal").innerText());
    }
    throw error;
  } finally {
    await browser.close();
  }
});
