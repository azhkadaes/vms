import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import { chromium } from "playwright";
import AxeBuilder from "@axe-core/playwright";
import { startProduction } from "./helpers/production-server.mjs";

let browser, server;
const evidence = new URL("../anti-slop/evidence-2026-11-02/production/", import.meta.url);
before(async () => {
  await mkdir(evidence, { recursive: true });
  server = await startProduction();
  browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || "chrome",
    headless: true,
  });
});
after(async () => {
  await browser?.close();
  await server?.close();
});
const catalog = {
  officers: [{ id: "officer-server-42", name: "Pejabat dari layanan", vip: false, active: true }],
  rooms: ["Ruang dari layanan"],
  privacyNotice: {
    version: "fixture-policy-1",
    id: "Pemberitahuan privasi contoh untuk pengujian kontrak.",
    en: "Example privacy notice for contract testing.",
  },
};
async function open(path = "/admin/employees", options = {}) {
  const context = await browser.newContext({
    viewport: options.viewport || { width: 390, height: 900 },
    deviceScaleFactor: options.deviceScaleFactor || 1,
    reducedMotion: options.motion || "reduce",
    timezoneId: "Asia/Makassar",
  });
  await context.addInitScript(() => sessionStorage.setItem("vms-preview-role", "Protocol Officer"));
  const state = {
    sessionStatus: options.sessionStatus || 200,
    passwordRequired: !!options.passwordRequired,
    role: "Super Admin",
    conflict: false,
    employees: [
      {
        id: "employee-42",
        name: "Pegawai Layanan",
        email: "pegawai@example.test",
        role: "PA",
        active: true,
        assignedOfficerIds: ["officer-server-42"],
        version: 1,
      },
    ],
    definitions: [
      {
        id: "pa",
        name: "PA",
        description: "Pejabat yang ditugaskan.",
        permissions: ["bookings", "approve"],
        assignablePermissions: ["bookings", "approve", "reject"],
        editable: true,
        version: 1,
      },
    ],
    integrations: [
      {
        id: "email",
        name: "Email",
        description: "Layanan pemberitahuan.",
        enabled: false,
        health: "not_configured",
        version: 1,
        fields: [
          {
            key: "endpoint",
            label: "Alamat layanan",
            kind: "url",
            required: true,
            value: "https://email.example.test",
          },
          {
            key: "credential",
            label: "Kredensial layanan",
            kind: "secret",
            required: true,
            configured: true,
            value: "MUST-NEVER-BE-RENDERED",
          },
        ],
      },
    ],
    calls: [],
    errors: [],
    booking: null,
  };
  await context.route("**/v1/**", async (route) => {
    const req = route.request();
    const url = new URL(req.url());
    const path = url.pathname.slice(3);
    const method = req.method();
    const jsonBody = req.headers()["content-type"]?.includes("application/json")
      ? req.postDataJSON()
      : undefined;
    state.calls.push({
      path,
      query: url.search,
      method,
      body: jsonBody,
      raw: req.postData(),
      headers: req.headers(),
    });
    const send = (json, status = 200) => route.fulfill({ status, json });
    if (path === "/staff/session" && method === "GET")
      return send(
        {
          active: true,
          id: "admin-42",
          name: "Admin Layanan",
          role: state.role,
          permissions:
            state.role === "Super Admin"
              ? ["people", "roles", "integrations", "reports"]
              : ["bookings", "checkin", "checkout"],
          assignedOfficerIds: [],
          mustChangePassword: state.passwordRequired,
          passwordMinLength: 12,
        },
        state.sessionStatus,
      );
    if (path === "/staff/password") {
      state.passwordRequired = false;
      return route.fulfill({ status: 204 });
    }
    if (path === "/staff/sessions") {
      const status = state.loginStatus || 200;
      if (status === 200) state.sessionStatus = 200;
      return send(
        { message: status === 429 ? "Terlalu banyak permintaan. Coba lagi nanti." : "OK" },
        status,
      );
    }
    if (path.endsWith("/catalogs")) return send(catalog);
    if (path === "/staff/reports/summary")
      return state.summaryDenied
        ? send({ message: "Akses laporan tidak diizinkan." }, 403)
        : send({
            filter: Object.fromEntries(url.searchParams),
            scope: "organization",
            generatedAt: "2099-08-19T01:00:00Z",
            canExport: false,
            revision: "0".repeat(64),
            counts: {
              pending: 0,
              approved: 0,
              rejected: 0,
              cancelled: 0,
              checked_in: 0,
              checked_out: 0,
              missed: 0,
            },
            total: 0,
          });
    if (path === "/staff/employees" && method === "GET") {
      const q = url.searchParams.get("q")?.toLowerCase() || "";
      return send({
        items: state.employees.filter((person) => person.name.toLowerCase().includes(q)),
        total: state.employees.length,
        totalAll: state.employees.length,
        page: 1,
        pageSize: 25,
        passwordMinLength: 12,
      });
    }
    if (path === "/staff/employees" && method === "POST") {
      const { initialPassword, requirePasswordChange, ...person } = jsonBody;
      const record = { ...person, id: "employee-created", version: 1 };
      state.employees.push(record);
      return send(record, 201);
    }
    if (path.startsWith("/staff/employees/")) {
      const record = state.employees.find((person) => person.id === path.split("/").at(-1));
      if (method === "GET") return send(record);
      if (state.conflict) {
        state.conflict = false;
        record.version++;
        record.name = "Versi terbaru dari layanan";
        return send({ message: "Data telah berubah." }, 409);
      }
      Object.assign(record, jsonBody, { version: record.version + 1 });
      return send(record);
    }
    if (path === "/staff/roles") return send(state.definitions);
    if (path.startsWith("/staff/roles/")) {
      Object.assign(state.definitions[0], jsonBody, { version: 2 });
      return send(state.definitions[0]);
    }
    if (path === "/staff/integrations") return send(state.integrations);
    if (path.startsWith("/staff/integrations/")) {
      const record = state.integrations[0];
      record.enabled = jsonBody.enabled;
      record.version++;
      for (const field of record.fields) {
        if (field.kind === "secret")
          field.configured = jsonBody.clearSecrets.includes(field.key)
            ? false
            : field.configured || !!jsonBody.values[field.key];
        else field.value = jsonBody.values[field.key];
      }
      return send(record);
    }
    if (path === "/staff/bookings")
      return send({
        items: [],
        total: 0,
        totalScoped: 0,
        page: 1,
        pageSize: Number(url.searchParams.get("pageSize") || 7),
        metrics: { today: 0, pending: 0, arrivals: 0, inside: 0, completed: 0 },
      });
    if (path.endsWith("/attachments"))
      return send({
        token: "opaque-upload-token-42",
        expiresAt: "2099-08-19T12:00:00+08:00",
        serverTime: Date.parse("2099-08-19T09:00:00+08:00"),
      });
    if (path === "/public/bookings") {
      state.booking = {
        ...jsonBody,
        id: "visit-42",
        code: "88001001",
        host: catalog.officers[0].name,
        vip: false,
        duration: 60,
        room: "",
        status: "pending",
        version: 1,
        history: [{ at: "2099-08-19T01:00:00Z", action: "submitted" }],
        timing: {
          serverTime: Date.parse("2099-08-19T09:00:00+08:00"),
          opensAt: Date.parse("2099-08-19T09:30:00+08:00"),
          endsAt: Date.parse("2099-08-19T11:00:00+08:00"),
          displayStatus: "pending",
          canCancel: true,
          canCheckIn: false,
          canCheckOut: false,
        },
      };
      return send(state.booking, 201);
    }
    if (path === "/public/lookup") return send(state.booking);
    if (path.endsWith("/admission"))
      return send({
        payload: "OPAQUE-SERVER-ADMISSION-CREDENTIAL",
        expiresAt: "2099-08-19T09:00:02+08:00",
        serverTime: Date.parse("2099-08-19T09:00:00+08:00"),
      });
    if (path.endsWith("/cancel")) {
      state.booking.status = "cancelled";
      state.booking.version++;
      state.booking.timing.displayStatus = "cancelled";
      state.booking.timing.canCancel = false;
      return send(state.booking);
    }
    return send({ message: "Unexpected fixture route: " + path }, 500);
  });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  page.on("pageerror", (error) => state.errors.push(error.message));
  state.resources = [];
  page.on("request", (request) => state.resources.push(request.url()));
  if (!options.realClock) await page.clock.install({ time: new Date("2099-08-19T09:00:00+08:00") });
  await page.goto(server.origin + path);
  return { page, context, state, close: () => context.close() };
}
async function audit(page, name) {
  const result = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa", "best-practice"])
    .analyze();
  await writeFile(new URL(name + ".axe.json", evidence), JSON.stringify(result, null, 2));
  assert.deepEqual(
    result.violations.map((value) => ({
      id: value.id,
      nodes: value.nodes.map((node) => node.html),
    })),
    [],
  );
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
  await page.screenshot({
    path: new URL(name + ".png", evidence).pathname.replace(/^\/([A-Z]:)/, "$1"),
    fullPage: true,
  });
}

test("production email controls preserve visible labels and keyboard focus across domain switches in ID and EN", async () => {
  const env = await open("/status");
  try {
    const { page } = env;
    for (const language of ["id", "en"]) {
      await page.locator(".language-select").selectOption(language);
      const email = page.getByRole("textbox", {
        name: language === "id" ? "Email pendaftaran" : "Registration email",
        exact: true,
      });
      await email.fill("visitor");
      const domain = page.locator("#status-email-domain");
      await domain.focus();
      await domain.selectOption("other");
      await page.waitForFunction(() => document.activeElement?.id === "status-email-custom-domain");
      await page.keyboard.type("example.test");
      await page.keyboard.press("Tab");
      const change = page.getByRole("button", {
        name:
          language === "id"
            ? "Ubah domain email: pilih dari daftar"
            : "Change email domain: choose from list",
        exact: true,
      });
      assert.equal(await change.evaluate((node) => node === document.activeElement), true);
      await audit(page, `email-custom-domain-${language}`);
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.activeElement?.id === "status-email-domain");
      assert.equal(await email.inputValue(), "visitor");
      await domain.selectOption("gmail.com");
      assert.equal(await domain.evaluate((node) => node === document.activeElement), true);
      await email.fill("visitor@example.test");
      assert.equal(await email.evaluate((node) => node === document.activeElement), true);
      assert.equal(await page.locator("#status-email-custom-domain").inputValue(), "example.test");
      await audit(page, `email-custom-paste-${language}`);
      await change.click();
      await page.waitForFunction(() => document.activeElement?.id === "status-email-domain");
    }
    assert.deepEqual(env.state.errors, []);
  } finally {
    await env.close();
  }
});

test("production mobile navigation has valid dialog semantics and retains keyboard focus", async () => {
  const env = await open();
  try {
    const { page } = env;
    for (const width of [320, 390]) {
      await page.setViewportSize({ width, height: 900 });
      const trigger = page.getByRole("button", { name: "Buka navigasi", exact: true });
      await trigger.focus();
      await page.keyboard.press("Enter");
      const dialog = page.getByRole("dialog", { name: "Navigasi backoffice", exact: true });
      await dialog.waitFor();
      assert.equal(await dialog.evaluate((node) => node.tagName), "DIV");
      await page.waitForFunction(() =>
        document.activeElement?.matches("#backoffice-sidebar nav a.active"),
      );
      for (const key of ["Tab", "Shift+Tab"]) {
        for (let count = 0; count < 16; count++) {
          await page.keyboard.press(key);
          assert.equal(
            await dialog.evaluate((node) => node.contains(document.activeElement)),
            true,
          );
        }
      }
      await audit(page, `mobile-navigation-${width}`);
      await page.keyboard.press("Escape");
      await dialog.waitFor({ state: "hidden" });
      assert.equal(await trigger.evaluate((node) => node === document.activeElement), true);
    }
    await page.setViewportSize({ width: 1280, height: 900 });
    assert.equal(
      await page.locator("#backoffice-sidebar").evaluate((node) => node.tagName),
      "ASIDE",
    );
    assert.equal(await page.getByRole("dialog").count(), 0);
    assert.deepEqual(env.state.errors, []);
  } finally {
    await env.close();
  }
});

test("production session outage offers retry and never accepts a stored preview role", async () => {
  const env = await open(undefined, { sessionStatus: 503 });
  try {
    await env.page.getByRole("heading", { name: "Layanan sesi belum tersedia." }).waitFor();
    assert.equal(await env.page.locator("#role").count(), 0);
    env.state.sessionStatus = 200;
    await env.page.getByRole("button", { name: "Coba lagi" }).click();
    await env.page.getByRole("heading", { name: "Pegawai", exact: true }).waitFor();
    await env.page.getByRole("heading", { name: "Pegawai Layanan", exact: true }).waitFor();
    assert.equal(
      env.state.calls.some((call) => /role=/.test(call.query)),
      false,
    );
    await audit(env.page, "employees-production-390");
  } finally {
    await env.close();
  }
});

test("production phone choices defer registration code until intent and retain its draft through Back", async () => {
  const env = await open("/", { motion: "no-preference" });
  try {
    const { page, state } = env;
    await page.locator("#visit-options").waitFor();
    await page.waitForFunction(() => {
      const image = document.querySelector(".hero-image");
      return image?.complete && image.naturalWidth > 0;
    });
    assert.equal(
      state.resources.some((url) => /\/nusantara-[^/]+\.avif/.test(url)),
      true,
    );
    assert.equal(await page.locator("#officerId").count(), 0);
    assert.equal(
      state.resources.some((url) => /\/landingMotion-/.test(url)),
      false,
    );
    const motionLoaded = page.waitForResponse((response) =>
      /\/landingMotion-/.test(response.url()),
    );
    await page.evaluate(() => window.dispatchEvent(new Event("scroll")));
    await motionLoaded;
    assert.equal(
      state.resources.some((url) => /\/(Booking-|validation-|calendar-)/.test(url)),
      false,
    );
    await page
      .locator("#visit-options")
      .getByRole("link", { name: "Daftar kunjungan", exact: true })
      .click();
    await page.locator("#purpose").fill("Retained production draft");
    await page.getByRole("link", { name: "Kembali", exact: true }).click();
    await page
      .locator("#visit-options")
      .getByRole("link", { name: "Daftar kunjungan", exact: true })
      .click();
    assert.equal(await page.locator("#purpose").inputValue(), "Retained production draft");
    assert.equal(
      await page.evaluate(() => document.documentElement.scrollWidth > innerWidth),
      false,
    );
    assert.deepEqual(state.errors, []);
  } finally {
    await env.close();
  }
});

test("responsive production hero uses one matching preload candidate across phone and desktop widths", async () => {
  const results = [];
  for (const [width, height, deviceScaleFactor] of [
    [320, 740, 1],
    [390, 844, 3],
    [430, 932, 2],
    [760, 900, 1],
    [761, 900, 1],
    [768, 900, 1],
    [1280, 900, 1],
    [1440, 1000, 2],
    [740, 390, 2],
  ]) {
    const env = await open("/", {
      viewport: { width, height },
      deviceScaleFactor,
      realClock: true,
    });
    try {
      const { page, state } = env;
      await page.waitForFunction(() => {
        const image = document.querySelector(".hero-image");
        return image?.complete && image.naturalWidth > 0;
      });
      await page.evaluate(() => document.fonts.ready);
      const result = await page.evaluate(() => {
        const image = document.querySelector(".hero-image");
        const preload = document.querySelector('link[rel="preload"][as="image"]');
        return {
          width: innerWidth,
          currentSrc: image.currentSrc,
          resources: performance
            .getEntriesByType("resource")
            .filter((entry) => /\/nusantara-/.test(entry.name))
            .map((entry) => ({
              name: entry.name,
              bytes: entry.encodedBodySize,
              initiator: entry.initiatorType,
            })),
          fontResources: performance
            .getEntriesByType("resource")
            .filter((entry) => /plus-jakarta-sans.*\.woff2/.test(entry.name))
            .map((entry) => ({ name: entry.name, initiator: entry.initiatorType })),
          sizes: image.parentElement.querySelector('source[type="image/avif"]').sizes,
          preloadSizes: preload?.imageSizes,
          overflow: document.documentElement.scrollWidth > innerWidth,
        };
      });
      assert.match(result.currentSrc, /\.avif$/);
      assert.equal(result.resources.length, 1, "Preload and image must share one request");
      assert.equal(result.resources[0].name, result.currentSrc);
      assert.equal(result.resources[0].initiator, "link");
      assert.equal(
        result.fontResources.length,
        1,
        "Font preload and stylesheet must share one request",
      );
      assert.equal(result.fontResources[0].initiator, "link");
      assert.ok(result.resources[0].bytes <= 150 * 1024);
      assert.equal(result.sizes, result.preloadSizes);
      assert.equal(result.overflow, false);
      assert.deepEqual(state.errors, []);
      results.push({ ...result, deviceScaleFactor });
      if ([320, 390, 1280].includes(width)) await audit(page, `visitor-responsive-${width}`);
      if ([390, 1280].includes(width)) {
        await page
          .locator(width < 761 ? ".hero-toolbar" : ".visitor-header")
          .getByRole("button", { name: "EN · English", exact: true })
          .click();
        await page.waitForFunction(() => document.documentElement.lang === "en");
        await audit(page, `visitor-responsive-${width}-en`);
      }
    } finally {
      await env.close();
    }
  }
  await writeFile(
    new URL("visitor-responsive-images.json", evidence),
    JSON.stringify(results, null, 2),
  );
});

test("employee creation uses backend catalogs and an initial password; conflicts preserve edits until explicit reload", async () => {
  const env = await open();
  const { page, state } = env;
  try {
    await page.getByRole("button", { name: "Tambah pegawai", exact: true }).click();
    await page.locator("#person-name").fill("Pegawai Baru");
    await page.locator("#person-email").fill("baru@example.test");
    await page.locator("#person-password").fill("Initial-password-42");
    await page.getByLabel("Pejabat dari layanan", { exact: true }).check();
    await audit(page, "employee-editor-production-390");
    await page.getByRole("button", { name: "Simpan pegawai", exact: true }).click();
    await page.getByRole("heading", { name: "Pegawai Baru", exact: true }).waitFor();
    const create = state.calls.find(
      (call) => call.path === "/staff/employees" && call.method === "POST",
    );
    assert.equal(create.body.initialPassword, "Initial-password-42");
    assert.deepEqual(create.body.assignedOfficerIds, ["officer-server-42"]);
    assert.equal(create.body.requirePasswordChange, true);
    assert.equal(create.headers["x-vms-request"], "browser");
    await page.getByRole("button", { name: "Edit pegawai Pegawai Layanan", exact: true }).click();
    await page.locator("#person-name").fill("Perubahan belum disimpan");
    state.conflict = true;
    await page.getByRole("button", { name: "Simpan pegawai", exact: true }).click();
    await page.getByText("Data telah berubah.", { exact: true }).waitFor();
    assert.equal(await page.locator("#person-name").inputValue(), "Perubahan belum disimpan");
    await page.getByRole("button", { name: "Muat versi terbaru" }).click();
    await page.getByRole("button", { name: "Buang perubahan dan muat" }).click();
    await page.waitForFunction(
      () => document.querySelector("#person-name")?.value === "Versi terbaru dari layanan",
    );
    await page.getByLabel("Akun aktif", { exact: true }).uncheck();
    await page.getByRole("button", { name: "Simpan pegawai", exact: true }).click();
    await page.getByRole("heading", { name: "Versi terbaru dari layanan", exact: true }).waitFor();
    assert.equal(state.employees[0].active, false);
    assert.deepEqual(state.errors, []);
  } finally {
    await env.close();
  }
});

test("role edits and integration secrets use versioned writes and never reveal stored secret values", async () => {
  const env = await open("/admin/roles");
  const { page, state } = env;
  try {
    await page.getByRole("button", { name: "Edit akses PA", exact: true }).click();
    await page.getByRole("checkbox", { name: "Menolak permohonan", exact: true }).check();
    await page.getByRole("button", { name: "Simpan akses", exact: true }).click();
    await page.getByRole("status").filter({ hasText: "Akses peran disimpan." }).waitFor();
    assert.ok(state.definitions[0].permissions.includes("reject"));
    await page.goto(server.origin + "/admin/integration");
    await page.getByRole("button", { name: "Konfigurasi Email", exact: true }).click();
    assert.equal(await page.locator("#integration-credential").inputValue(), "");
    assert.equal((await page.locator("body").innerText()).includes("MUST-NEVER"), false);
    await audit(page, "integration-editor-production-390");
    await page.getByLabel("Integrasi aktif", { exact: true }).check();
    await page.getByRole("button", { name: "Simpan konfigurasi", exact: true }).click();
    await page.getByRole("status").filter({ hasText: "Konfigurasi disimpan." }).waitFor();
    const first = state.calls.find(
      (call) => call.method === "PATCH" && call.path.includes("integrations"),
    );
    assert.equal("credential" in first.body.values, false);
    assert.deepEqual(first.body.clearSecrets, []);
    await page.getByRole("button", { name: "Konfigurasi Email", exact: true }).click();
    await page.getByLabel("Hapus kredensial tersimpan", { exact: true }).check();
    assert.equal(
      await page.getByRole("button", { name: "Simpan konfigurasi", exact: true }).isDisabled(),
      true,
    );
    await page.getByLabel("Integrasi aktif", { exact: true }).uncheck();
    await page.getByRole("button", { name: "Simpan konfigurasi", exact: true }).click();
    await page.getByRole("status").filter({ hasText: "Konfigurasi disimpan." }).waitFor();
    assert.equal(state.integrations[0].fields[1].configured, false);
    assert.deepEqual(state.errors, []);
  } finally {
    await env.close();
  }
});

test("a backend-required initial password change gates the workspace", async () => {
  const env = await open(undefined, { passwordRequired: true });
  try {
    const { page, state } = env;
    await page.getByRole("heading", { name: "Ganti kata sandi awal" }).waitFor();
    assert.equal(
      state.calls.some((call) => call.path === "/staff/employees"),
      false,
    );
    await page.locator("#current-password").fill("Initial-password-42");
    await page.locator("#new-password").fill("Replacement-password-42");
    await page.getByRole("button", { name: "Simpan kata sandi" }).click();
    await page.getByRole("heading", { name: "Pegawai", exact: true }).waitFor();
    assert.equal(
      state.calls.find((call) => call.path === "/staff/password").body.newPassword,
      "Replacement-password-42",
    );
  } finally {
    await env.close();
  }
});

test("public production booking uploads bytes before submitting a token, then uses a server admission credential", async () => {
  const env = await open("/booking");
  const { page, state } = env;
  try {
    await page.locator("#officerId").selectOption("officer-server-42");
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
    await page.locator("#purpose").fill("Production contract verification");
    await page.locator('.booking-form .form-actions button[type="button"]').click();
    await page.locator("#name").fill("Visitor Contract");
    await page.locator("#organization").fill("Contract Organization");
    await page.locator("#email").fill("visitor@example.test");
    await page.locator("#phone").fill("00123456789");
    await page.locator("#nd").setInputFiles({
      name: "official.pdf",
      mimeType: "application/pdf",
      buffer: Buffer.from("%PDF-1.7\nSynthetic document bytes"),
    });
    await page.locator('.booking-form .form-actions button[type="button"]').click();
    await page.locator('.booking-form input[type="checkbox"]').check();
    await page.locator('.booking-form button[type="submit"]').click();
    await page.locator(".booking-code-display strong").waitFor();
    const upload = state.calls.find((call) => call.path === "/public/attachments");
    assert.match(upload.raw, /Synthetic document bytes/);
    assert.match(upload.headers["content-type"], /^multipart\/form-data; boundary=/);
    assert.equal(state.booking.attachmentToken, "opaque-upload-token-42");
    assert.equal(state.booking.officerId, "officer-server-42");
    assert.equal(
      state.resources.some((url) => /\/nusantara-[^/]+\.(jpg|webp|avif)/.test(url)),
      false,
    );
    state.booking.status = "approved";
    state.booking.version++;
    state.booking.timing.displayStatus = "approved";
    await page.goto(server.origin + "/status");
    await page.locator("#status-code").fill(state.booking.code);
    await page.locator("#status-email").fill(state.booking.email);
    await page.getByRole("button", { name: "Lihat kunjungan", exact: true }).click();
    await page.getByRole("img", { name: "QR masuk untuk Visitor Contract" }).waitFor();
    assert.ok(state.calls.find((call) => call.path.endsWith("/admission")));
    await page.clock.runFor(2100);
    await page.getByRole("status").filter({ hasText: "QR telah kedaluwarsa" }).waitFor();
    assert.equal(
      state.calls.some((call) => call.path.includes("__preview") || call.query.includes("role=")),
      false,
    );
    assert.deepEqual(state.errors, []);
  } finally {
    await env.close();
  }
});

test("production login distinguishes rate limiting and keeps entered credentials for retry", async () => {
  const env = await open(undefined, { sessionStatus: 401 });
  try {
    const { page, state } = env;
    await page.waitForURL(/\/admin\/login$/);
    state.loginStatus = 429;
    await page.locator("#email").fill("admin@example.test");
    await page.locator("#password").fill("Example-password-42");
    await page.getByRole("button", { name: "Masuk ke backoffice", exact: true }).click();
    await page.getByRole("alert").filter({ hasText: "Terlalu banyak permintaan" }).waitFor();
    assert.equal(await page.locator("#email").inputValue(), "admin@example.test");
    assert.equal(await page.locator("#password").inputValue(), "Example-password-42");
    state.loginStatus = 200;
    await page.getByRole("button", { name: "Masuk ke backoffice", exact: true }).click();
    await page.waitForURL(/\/admin\/dashboard$/);
    assert.equal(await page.locator("#role").count(), 0);
  } finally {
    await env.close();
  }
});

test("production staff lists request bounded pages, load a direct detail, and gate document download at the API", async () => {
  const env = await open();
  const { page, context, state } = env;
  try {
    const rows = Array.from({ length: 12 }, (_, index) => ({
      id: `staff-visit-${index + 1}`,
      name: `Contract Visitor ${index + 1}`,
      organization: "Contract Organization",
      email: "visitor@example.test",
      phone: "00123456789",
      officerId: "officer-server-42",
      host: "Pejabat dari layanan",
      vip: false,
      date: "2099-08-19",
      time: "10:00",
      duration: 60,
      room: "Ruang dari layanan",
      purpose: "Contract verification",
      visitors: [`Contract Visitor ${index + 1}`],
      identityType: "KTP",
      code: "88001001",
      status: "approved",
      version: 2,
      ndName: "official.pdf",
      attachmentAvailable: true,
      history: [],
      timing: {
        serverTime: Date.parse("2099-08-19T09:00:00+08:00"),
        opensAt: Date.parse("2099-08-19T09:30:00+08:00"),
        endsAt: Date.parse("2099-08-19T11:00:00+08:00"),
        displayStatus: "approved",
        canCancel: true,
        canCheckIn: false,
        canCheckOut: false,
      },
    }));
    let permitCancellation = false;
    const cancellationCalls = [];
    await context.route("**/v1/staff/session", (route) =>
      route.fulfill({
        json: {
          active: true,
          id: "pa-42",
          name: "PA Layanan",
          role: "PA",
          permissions: [
            "bookings",
            "approve",
            "reschedule",
            ...(permitCancellation ? ["cancel"] : []),
          ],
          assignedOfficerIds: ["officer-server-42"],
        },
      }),
    );
    const listCalls = [];
    let downloadDenied = false;
    await context.route("**/v1/staff/bookings**", (route) => {
      const url = new URL(route.request().url());
      if (url.pathname.endsWith("/cancel")) {
        const body = route.request().postDataJSON();
        cancellationCalls.push({ body, headers: route.request().headers() });
        const row = rows.find((row) => row.id === url.pathname.split("/").at(-2));
        row.status = "cancelled";
        row.version++;
        row.timing.displayStatus = "cancelled";
        row.timing.canCancel = false;
        return route.fulfill({ json: row });
      }
      if (url.pathname.endsWith("/attachment"))
        return downloadDenied
          ? route.fulfill({ status: 403, json: { message: "Akses dokumen tidak diizinkan." } })
          : route.fulfill({
              contentType: "application/pdf",
              body: "%PDF-1.7\nSynthetic protected document",
            });
      if (url.pathname === "/v1/staff/bookings") {
        listCalls.push(url);
        const page = Number(url.searchParams.get("page"));
        return route.fulfill({
          json: {
            items: rows
              .filter((row) => row.status === url.searchParams.get("status"))
              .slice((page - 1) * 7, page * 7),
            total: rows.filter((row) => row.status === url.searchParams.get("status")).length,
            totalScoped: 12,
            page,
            pageSize: 7,
          },
        });
      }
      return route.fulfill({ json: rows.find((row) => row.id === url.pathname.split("/").at(-1)) });
    });
    await page.goto(server.origin + "/admin/booking?status=approved&page=2");
    await page
      .getByRole("button", { name: "Lihat detail Contract Visitor 8", exact: true })
      .waitFor();
    assert.equal(listCalls[0].searchParams.get("view"), "list");
    assert.equal(listCalls[0].searchParams.get("pageSize"), "7");
    assert.equal(listCalls[0].searchParams.get("page"), "2");
    assert.equal(listCalls[0].searchParams.has("role"), false);
    await page
      .getByRole("button", { name: "Lihat detail Contract Visitor 8", exact: true })
      .click();
    await page.getByRole("heading", { name: "Detail kunjungan", exact: true }).waitFor();
    const downloading = page.waitForEvent("download");
    await page.getByRole("button", { name: "Unduh dokumen pendukung" }).click();
    assert.equal((await downloading).suggestedFilename(), "official.pdf");
    downloadDenied = true;
    await page.getByRole("button", { name: "Unduh dokumen pendukung" }).click();
    await page.getByRole("alert").filter({ hasText: "Akses dokumen tidak diizinkan." }).waitFor();
    assert.deepEqual(state.errors, []);
    await audit(page, "staff-detail-production-390");
    assert.equal(
      await page.getByRole("button", { name: "Batalkan kunjungan", exact: true }).count(),
      0,
    );
    permitCancellation = true;
    await page.evaluate(() => window.dispatchEvent(new Event("visibilitychange")));
    await page.getByRole("button", { name: "Batalkan kunjungan", exact: true }).click();
    await page
      .getByRole("textbox", { name: "Alasan pembatalan", exact: false })
      .fill("Authorized staff cancellation reason.");
    await audit(page, "staff-cancellation-production-390");
    await page.getByRole("button", { name: "Batalkan kunjungan", exact: true }).click();
    await page
      .getByRole("status")
      .filter({ hasText: "Kunjungan dibatalkan oleh petugas." })
      .waitFor();
    assert.equal(cancellationCalls.length, 1);
    assert.deepEqual(cancellationCalls[0].body, {
      reason: "Authorized staff cancellation reason.",
      internalCancellationNote: "",
      version: 2,
    });
    assert.equal(cancellationCalls[0].headers["x-vms-request"], "browser");
  } finally {
    await env.close();
  }
});

test("a changed server role removes the previous private management screen on return to the tab", async () => {
  const env = await open();
  try {
    const { page, state } = env;
    await page.getByRole("heading", { name: "Pegawai Layanan", exact: true }).waitFor();
    state.role = "Receptionist";
    await page.evaluate(() => window.dispatchEvent(new Event("visibilitychange")));
    await page.getByRole("heading", { name: "Akses terbatas", exact: true }).waitFor();
    assert.equal(await page.getByText("pegawai@example.test", { exact: true }).count(), 0);
    assert.equal(await page.locator("#role").count(), 0);
  } finally {
    await env.close();
  }
});

test("production reports use an independent aggregate route and hide counts after denied refresh", async () => {
  const env = await open("/admin/logs");
  try {
    const { page, state } = env;
    await page.getByTestId("report-total").waitFor();
    assert.equal(await page.getByTestId("report-total").innerText(), "0");
    assert.equal(
      await page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).isDisabled(),
      true,
    );
    assert.equal(
      state.calls.some((call) => call.path === "/staff/bookings"),
      false,
    );
    state.summaryDenied = true;
    await page.getByRole("button", { name: "Perbarui laporan", exact: true }).click();
    await page.getByRole("alert").filter({ hasText: "Akses laporan tidak diizinkan." }).waitFor();
    assert.equal(await page.getByTestId("report-total").count(), 0);
    assert.deepEqual(state.errors, []);
  } finally {
    await env.close();
  }
});
