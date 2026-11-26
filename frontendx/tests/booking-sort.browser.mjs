import test from "node:test";
import assert from "node:assert/strict";
import { chromium } from "playwright";

const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3002";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);

test("visit sort shows its default label and can restore the default order", async () => {
  const browser = await chromium.launch({ channel: "chrome", headless: true });
  try {
    for (const role of ["PA", "Receptionist"]) {
      const context = await browser.newContext({ timezoneId: "Asia/Makassar" });
      await context.addInitScript((role) => sessionStorage.setItem("vms-preview-role", role), role);
      const page = await context.newPage();
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
      };
      await page.route("**/__preview/bookings?*", (route) =>
        route.fulfill({
          json: [
            { ...base, id: "past", name: "Past", status: "checked_out", date: "2000-01-01" },
            { ...base, id: "upcoming" },
            ...(role === "PA"
              ? [{ ...base, id: "pending", name: "Pending", status: "pending" }]
              : []),
          ].map((booking) => ({
            ...booking,
            timing: {
              serverTime: Date.now(),
              opensAt: Date.parse(`${booking.date}T09:30:00+08:00`),
              endsAt: Date.parse(`${booking.date}T11:00:00+08:00`),
              displayStatus: booking.status,
              canCheckIn: false,
              canCheckOut: false,
              canCancel: false,
            },
          })),
        }),
      );
      await page.goto(`${origin}/admin/booking`);
      const sort = page.getByRole("combobox", { name: "Urutan kunjungan" });
      await sort.waitFor();
      await page.locator("tbody tr").first().waitFor();
      assert.equal(await sort.inputValue(), "");
      assert.equal(await sort.locator("option:checked").textContent(), "Urutkan");
      assert.equal(await page.locator('label[for="booking-sort"]').innerText(), "Urutkan");
      const nameSelector = role === "PA" ? "tbody .row-link" : "tbody .person-cell strong";
      const names = async () =>
        (await page.locator(nameSelector).allTextContents()).map((name) => name.trim());
      const expected = role === "PA" ? ["Pending", "Upcoming", "Past"] : ["Upcoming", "Past"];
      assert.deepEqual(await names(), expected);
      await page.getByRole("button", { name: "Jadwal · WITA", exact: true }).click();
      await page.getByRole("button", { name: "Jadwal · WITA", exact: true }).click();
      assert.match(page.url(), /dir=desc/);
      await sort.selectOption("");
      assert.equal(new URL(page.url()).searchParams.has("dir"), false);
      assert.equal(new URL(page.url()).searchParams.has("sort"), false);
      await page.waitForFunction(
        ({ selector, first }) => document.querySelector(selector)?.textContent.trim() === first,
        { selector: nameSelector, first: expected[0] },
      );
      assert.deepEqual(await names(), expected);
      await sort.selectOption("time");
      await page.waitForFunction(
        (selector) => document.querySelector(selector)?.textContent.trim() === "Past",
        nameSelector,
      );
      assert.equal((await names())[0], "Past");
      await page.reload();
      assert.equal(await sort.inputValue(), "time");
      await context.close();
    }
  } finally {
    await browser.close();
  }
});
