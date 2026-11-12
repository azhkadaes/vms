import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";

const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || "playwright");
const origin = process.env.VMS_TEST_URL || "http://127.0.0.1:3002";
assert.match(origin, /^http:\/\/(127\.0\.0\.1|localhost):\d+$/);
const evidence = new URL("../anti-slop/evidence-2026-10-31/employees/", import.meta.url);
let browser;
before(async () => {
  await mkdir(evidence, { recursive: true });
  browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
});
after(async () => browser?.close());

async function open(width = 390, query = "") {
  const context = await browser.newContext({
    viewport: { width, height: 900 },
    reducedMotion: "reduce",
  });
  await context.addInitScript(() => sessionStorage.setItem("vms-preview-role", "Super Admin"));
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  const errors = [],
    requests = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => requests.push(new URL(request.url()).pathname));
  await page.goto(origin + "/admin/employees" + query);
  await page.getByRole("heading", { name: "Direktori pegawai", exact: true }).waitFor();
  return { page, context, errors, requests };
}

async function snapshot(env, name) {
  const { page } = env;
  assert.equal(
    await page.evaluate(() => document.documentElement.scrollWidth > innerWidth),
    false,
    "No horizontal document overflow",
  );
  assert.equal(
    await page.locator(".employee-card:visible, .employee-table:visible td").evaluateAll((nodes) =>
      nodes.every((node) => {
        const rect = node.getBoundingClientRect();
        return (
          rect.left >= 0 && rect.right <= innerWidth && node.scrollWidth <= node.clientWidth + 1
        );
      }),
    ),
    true,
    "Directory text stays within its cards/cells",
  );
  assert.deepEqual(env.errors, []);
  await page.screenshot({ path: fileURLToPath(new URL(name + ".png", evidence)), fullPage: true });
  await writeFile(new URL(name + ".aria.txt", evidence), await page.locator("body").ariaSnapshot());
}

test("phone/tablet cards and a desktop table expose one readable directory without booking requests", async () => {
  const env = await open(320);
  try {
    for (const width of [320, 390, 768, 1280]) {
      await env.page.setViewportSize({ width, height: 900 });
      const mobile = width <= 1000;
      assert.equal(await env.page.getByRole("table").count(), mobile ? 0 : 1);
      assert.equal(await env.page.getByRole("article").count(), mobile ? 3 : 0);
      assert.equal(
        await env.page
          .getByText("aulia@example.test", { exact: true })
          .filter({ visible: true })
          .count(),
        1,
      );
      assert.equal(
        await env.page
          .getByRole("status")
          .filter({ hasText: "Menampilkan 3 dari 3 pegawai." })
          .count(),
        1,
      );
      if (!mobile)
        assert.equal(
          await env.page.getByRole("columnheader", { name: "Peran aktif", exact: true }).count(),
          1,
        );
      await snapshot(env, `directory-${width}`);
    }
    assert.equal(env.requests.filter((path) => path === "/__preview/bookings").length, 0);
  } finally {
    await env.context.close();
  }
});

test("search covers name/email/role, survives reload, and no-match reset restores results and input focus", async () => {
  const env = await open();
  const { page } = env;
  const search = page.getByRole("searchbox", { name: "Cari nama, email, atau peran", exact: true });
  try {
    for (const value of ["NADIA", "  NADIA@EXAMPLE.TEST  ", "protocol"]) {
      await search.fill(value);
      await page.getByRole("status").filter({ hasText: "Menampilkan 1 dari 3 pegawai." }).waitFor();
      await page.getByRole("heading", { name: "Nadia Putri", exact: true }).waitFor();
      assert.equal(await page.getByRole("article").count(), 1);
      assert.equal(
        await page.getByRole("heading", { name: "Nadia Putri", exact: true }).count(),
        1,
      );
    }
    await page.reload();
    await page.getByRole("heading", { name: "Nadia Putri", exact: true }).waitFor();
    assert.equal(await search.inputValue(), "protocol");
    await search.fill("tidak-ada-pegawai");
    await page.getByText("Tidak ada pegawai yang sesuai", { exact: true }).waitFor();
    assert.equal(await page.getByRole("article").count(), 0);
    assert.equal(await page.getByRole("table").count(), 0);
    await snapshot(env, "no-match-390");
    await page.getByRole("button", { name: "Reset pencarian", exact: true }).click();
    assert.equal(await search.evaluate((node) => node === document.activeElement), true);
    assert.equal(await search.inputValue(), "");
    assert.equal(new URL(page.url()).searchParams.has("q"), false);
    await page.getByRole("heading", { name: "Aulia Rahma", exact: true }).waitFor();
    assert.equal(await page.getByRole("article").count(), 3);
    await snapshot(env, "reset-390");
  } finally {
    await env.context.close();
  }
});

test("keyboard add/close returns focus, rejects blank names and preserves long fields at narrow and desktop widths", async () => {
  const env = await open(320);
  const { page } = env;
  const add = page.getByRole("button", { name: "Tambah pegawai demo", exact: true });
  const longName = "NamaPegawaiTanpaSpasi".repeat(8);
  const longEmail = "pegawai".repeat(8) + "@" + "organisasi".repeat(5) + ".example.test";
  try {
    await add.focus();
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog", { name: "Tambah pegawai demo", exact: true });
    await dialog.waitFor();
    assert.equal(await dialog.locator("#person-name").evaluate((node) => node.required), true);
    assert.equal(await dialog.locator("#person-email").getAttribute("type"), "email");
    await page.keyboard.press("Escape");
    await dialog.waitFor({ state: "hidden" });
    await page.waitForFunction(
      () =>
        document.activeElement?.tagName === "BUTTON" &&
        document.activeElement.textContent.includes("Tambah pegawai demo"),
    );
    assert.equal(await add.evaluate((node) => node === document.activeElement), true);
    await page.keyboard.press("Enter");
    await dialog.locator("#person-name").fill("   ");
    await dialog.locator("#person-email").fill(longEmail);
    await dialog.getByRole("button", { name: "Simpan contoh", exact: true }).click();
    assert.equal(await dialog.isVisible(), true);
    assert.equal(
      await dialog
        .locator("#person-name")
        .evaluate((node) => node === document.activeElement && !node.validity.valid),
      true,
    );
    await dialog.locator("#person-name").fill(longName);
    await dialog.locator("#person-password").fill("Fictional-test-password-123");
    await dialog.locator("#person-role").selectOption("Super Admin");
    await dialog.getByRole("button", { name: "Simpan contoh", exact: true }).focus();
    await page.keyboard.press("Enter");
    await dialog.waitFor({ state: "hidden" });
    await page.waitForFunction(
      () =>
        document.activeElement?.tagName === "BUTTON" &&
        document.activeElement.textContent.includes("Tambah pegawai demo"),
    );
    assert.equal(await add.evaluate((node) => node === document.activeElement), true);
    assert.equal(await page.getByRole("article").count(), 4);
    for (const width of [320, 768, 1280]) {
      await page.setViewportSize({ width, height: 900 });
      assert.equal(
        await page.getByText(longName, { exact: true }).filter({ visible: true }).count(),
        1,
      );
      assert.equal(
        await page.getByText(longEmail, { exact: true }).filter({ visible: true }).count(),
        1,
      );
      await snapshot(env, `long-fields-${width}`);
    }
  } finally {
    await env.context.close();
  }
});
