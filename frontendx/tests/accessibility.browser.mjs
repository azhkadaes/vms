import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import { chromium } from "playwright";
import AxeBuilder from "@axe-core/playwright";

const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3000";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);
const evidence = new URL("../anti-slop/evidence-2026-10-31/production/", import.meta.url);
let browser;
before(async () => {
  await mkdir(evidence, { recursive: true });
  browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
});
after(async () => browser?.close());
async function audit(page, name) {
  const results = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa", "best-practice"])
    .analyze();
  await writeFile(new URL(name + ".axe.json", evidence), JSON.stringify(results, null, 2));
  const failures = results.violations.map(({ id, nodes }) => ({
    id,
    nodes: nodes.map((node) => ({ html: node.html, failure: node.failureSummary })),
  }));
  assert.deepEqual(failures, []);
}

test("visitor and staff main screens have no automated WCAG A/AA failures at phone and desktop widths", async () => {
  for (const [path, width, role] of [
    ["/", 320],
    ["/?view=registration#registration", 320],
    ["/?view=status#booking-status", 390],
    ["/booking", 1280],
    ["/status", 390],
    ["/admin/dashboard", 390, "PA"],
    ["/admin/booking", 320, "PA"],
    ["/admin/check-in", 390, "Receptionist"],
    ["/admin/employees", 320, "Super Admin"],
    ["/admin/roles", 320, "Super Admin"],
    ["/admin/logs", 1280, "Super Admin"],
  ]) {
    const context = await browser.newContext({
      viewport: { width, height: 900 },
      reducedMotion: "reduce",
    });
    try {
      if (role)
        await context.addInitScript(
          (role) => sessionStorage.setItem("vms-preview-role", role),
          role,
        );
      const page = await context.newPage();
      await page.goto(origin + path);
      await page.getByRole("heading", { level: 1 }).waitFor();
      await page.locator(".skeleton").waitFor({ state: "hidden" });
      await audit(page, `${path.replaceAll(/[^a-z]/gi, "-")}-${width}`);
      assert.equal(
        await page.evaluate(() => document.documentElement.scrollWidth > innerWidth),
        false,
      );
    } finally {
      await context.close();
    }
  }
});

test("calendar and time popovers expose accessible controls and return keyboard focus", async () => {
  const context = await browser.newContext({
    viewport: { width: 390, height: 900 },
    reducedMotion: "reduce",
  });
  try {
    const page = await context.newPage();
    await page.goto(origin + "/booking");
    await page.locator("#officerId").selectOption("o2");
    await page.locator(".date-picker-trigger").click();
    await page.locator('[data-slot="calendar"]').waitFor();
    await audit(page, "calendar-390");
    await page.keyboard.press("Escape");
    await page.waitForFunction(() =>
      document.activeElement?.classList.contains("date-picker-trigger"),
    );
    await page.locator(".date-picker-trigger").click();
    await page.getByRole("combobox", { name: "Pilih tahun", exact: true }).selectOption("2099");
    await page.getByRole("combobox", { name: "Pilih bulan", exact: true }).selectOption("7");
    await page
      .locator('.shadcn-calendar-day[data-day="2099-08-19"]:not(:disabled)')
      .first()
      .click();
    await page.locator(".time-picker-trigger").click();
    await page.locator(".time-picker-popover").waitFor();
    await audit(page, "time-390");
    await page.keyboard.press("Escape");
    await page.waitForFunction(() =>
      document.activeElement?.classList.contains("time-picker-trigger"),
    );
  } finally {
    await context.close();
  }
});
