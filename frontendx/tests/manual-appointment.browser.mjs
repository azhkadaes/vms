import test from "node:test";
import assert from "node:assert/strict";
import { chromium } from "playwright";
import { startPreview } from "./helpers/preview-server.mjs";

const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3002";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);

test("manual visits show cancellation and rescheduling while public requests retain decisions", async () => {
  const api = await startPreview();
  let browser;
  try {
    const input = {
      idempotencyKey: crypto.randomUUID(),
      officerId: "o2",
      date: "2099-08-19",
      time: "10:00",
      name: "Manual Appointment Example",
      organization: "Example Organization",
      email: "manual@example.test",
      phone: "08123456789",
      purpose: "Manual appointment verification",
      visitors: ["Manual Appointment Example"],
      identityType: "KTP",
    };
    async function create(path, body) {
      const response = await fetch(api.base + path, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      assert.equal(response.status, 201);
      return response.json();
    }
    const manual = await create("/staff/bookings?role=PA", input);
    assert.equal(manual.status, "approved");
    assert.match(manual.history.at(-1).message, /otomatis disetujui/);
    const publicVisit = await create("/bookings", {
      ...input,
      idempotencyKey: crypto.randomUUID(),
      ack: true,
    });
    browser = await chromium.launch({ channel: "chrome", headless: true });
    const context = await browser.newContext();
    await context.addInitScript(() =>
      sessionStorage.setItem(
        "vms-preview-role",
        location.hash === "#protocol" ? "Protocol Officer" : "PA",
      ),
    );
    await context.route("**/__preview/**", async (route) => {
      const url = new URL(route.request().url());
      await route.fulfill({
        response: await route.fetch({
          url: api.base + url.pathname.split("/__preview")[1] + url.search,
        }),
      });
    });
    const page = await context.newPage();
    for (const width of [1280, 390]) {
      await page.setViewportSize({ width, height: 900 });
      await page.goto(origin + `/admin/booking?selected=${manual.id}`);
      const dialog = page.getByRole("dialog");
      await dialog.getByRole("button", { name: "Batalkan kunjungan", exact: true }).waitFor();
      await dialog.getByText(input.email, { exact: true }).waitFor();
      await dialog.getByText(input.phone, { exact: true }).waitFor();
      assert.equal(await dialog.getByRole("button", { name: "Setujui", exact: true }).count(), 0);
      assert.equal(await dialog.getByRole("button", { name: "Tolak", exact: true }).count(), 0);
      await dialog.getByRole("button", { name: "Edit kunjungan", exact: true }).click();
      const save = dialog.getByRole("button", { name: "Simpan perubahan", exact: true });
      assert.equal(await save.count(), 0);
      await dialog.locator("#reason").fill("Alasan perubahan jadwal.");
      assert.equal(await save.count(), 0);
      const duration = dialog.locator("#duration");
      const originalDuration = await duration.inputValue();
      await duration.fill(originalDuration === "45" ? "60" : "45");
      await save.waitFor();
      await duration.fill(originalDuration);
      assert.equal(await save.count(), 0);
    }
    await page.goto(origin + `/admin/booking?selected=${publicVisit.id}`);
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("button", { name: "Setujui", exact: true }).waitFor();
    await dialog.getByRole("button", { name: "Tolak", exact: true }).waitFor();
    await page.goto(origin + `/admin/booking?selected=${manual.id}#protocol`);
    await dialog.getByText(/^Hanya lihat\./).waitFor();
    await dialog.getByText(input.email, { exact: true }).waitFor();
    await dialog.getByText(input.phone, { exact: true }).waitFor();
  } finally {
    await browser?.close();
    await api.close();
  }
});
