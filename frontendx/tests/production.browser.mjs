import test, { before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import { chromium } from "playwright";
import AxeBuilder from "@axe-core/playwright";
import qrcode from "qrcode";
import { startProduction } from "./helpers/production-server.mjs";

let browser, server;
const qrPayload = "OPAQUE-SERVER-ADMISSION";
const qrImage = await qrcode.toDataURL(qrPayload);
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
function staffToken(session, accessToken = "fixture-staff-token") {
  const serverTime = Date.now();
  return {
    accessToken,
    tokenType: "Bearer",
    accessExpiresAt: new Date(serverTime + 5 * 60_000).toISOString(),
    session: { ...session, timing: { serverTime } },
  };
}
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
    role: options.role || "Super Admin",
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
    const staffSession = () => ({
      active: true,
      id: "admin-42",
      name: "Admin Layanan",
      role: state.role,
      permissions:
        options.permissions ??
        (state.role === "Super Admin"
          ? ["people", "roles", "integrations", "reports"]
          : ["bookings", "checkin", "checkout"]),
      assignedOfficerIds: [],
      mustChangePassword: state.passwordRequired,
      passwordMinLength: 12,
    });
    if (path === "/staff/session" && method === "GET")
      return send(staffSession(), state.sessionStatus);
    if (path === "/staff/session/refresh")
      return send(staffToken(staffSession()), state.sessionStatus);
    if (path === "/staff/session/renew") return send(staffSession());
    if (path === "/staff/password") {
      state.passwordRequired = false;
      return send(staffToken(staffSession()));
    }
    if (path === "/staff/sessions") {
      const status = state.loginStatus || 200;
      if (status !== 200)
        return send({ message: "Terlalu banyak permintaan. Coba lagi nanti." }, status);
      state.sessionStatus = 200;
      return send(staffToken(staffSession()));
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
        allowedActions: ["approve", "reject", "cancel"],
      };
      return send(state.booking, 201);
    }
    if (path === "/public/lookup") return send(state.booking);
    if (path.endsWith("/admission"))
      return send({
        payload: qrPayload,
        imagePng: qrImage.split(",")[1],
        expiresAt: "2099-08-19T11:00:00+08:00",
        serverTime: state.booking.timing.serverTime,
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
      const control = page.getByRole("button", {
        name: language === "id" ? "ID · Bahasa Indonesia" : "EN · English",
        exact: true,
      });
      await control.click();
      assert.equal(await control.getAttribute("aria-pressed"), "true");
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
      assert.equal(await domain.evaluate((node) => node === document.activeElement), true);
      await audit(page, `email-custom-domain-${language}`);
      await domain.selectOption("gmail.com");
      assert.equal(await email.inputValue(), "visitor");
      assert.equal(await domain.evaluate((node) => node === document.activeElement), true);
      await email.fill("visitor@example.test");
      assert.equal(await email.evaluate((node) => node === document.activeElement), true);
      assert.equal(await page.locator("#status-email-custom-domain").inputValue(), "example.test");
      await audit(page, `email-custom-paste-${language}`);
      await domain.focus();
      await domain.selectOption("gmail.com");
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
    await env.page
      .getByRole("button", { name: "Lihat detail pegawai Pegawai Layanan", exact: true })
      .waitFor();
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
    await page.getByRole("combobox", { name: "Pejabat yang didampingi" }).click();
    const officer = page.getByRole("option", { name: "Pejabat dari layanan", exact: true });
    await officer.hover();
    assert.equal(await officer.getAttribute("data-highlighted"), "true");
    assert.equal(await officer.evaluate((node) => getComputedStyle(node).borderRadius), "6px");
    await officer.click();
    await audit(page, "employee-editor-production-390");
    await page.getByRole("button", { name: "Simpan pegawai", exact: true }).click();
    await page.getByRole("button", { name: "Lihat detail pegawai Pegawai Baru" }).waitFor();
    const create = state.calls.find(
      (call) => call.path === "/staff/employees" && call.method === "POST",
    );
    assert.equal(create.body.initialPassword, "Initial-password-42");
    assert.deepEqual(create.body.assignedOfficerIds, ["officer-server-42"]);
    assert.equal(create.body.requirePasswordChange, true);
    assert.equal(create.headers["x-vms-request"], "browser");
    await page.getByRole("button", { name: "Lihat detail pegawai Pegawai Layanan" }).click();
    await page.getByRole("button", { name: "Edit pegawai", exact: true }).click();
    await page.locator("#person-name").fill("Perubahan belum disimpan");
    state.conflict = true;
    await page.getByRole("button", { name: "Simpan pegawai", exact: true }).click();
    await page.getByText("Data telah berubah.", { exact: true }).waitFor();
    assert.equal(await page.locator("#person-name").inputValue(), "Perubahan belum disimpan");
    await page.getByRole("button", { name: "Muat versi terbaru" }).click();
    await page.getByRole("button", { name: "Buang perubahan dan muat" }).click();
    await page.getByRole("dialog", { name: "Detail pegawai", exact: true }).waitFor();
    await page.getByRole("button", { name: "Edit pegawai", exact: true }).click();
    await page.waitForFunction(
      () => document.querySelector("#person-name")?.value === "Versi terbaru dari layanan",
    );
    await page.getByLabel("Akun aktif", { exact: true }).uncheck();
    await page.getByRole("button", { name: "Simpan pegawai", exact: true }).click();
    await page
      .getByRole("button", { name: "Lihat detail pegawai Versi terbaru dari layanan" })
      .waitFor();
    assert.equal(state.employees[0].active, false);
    assert.deepEqual(state.errors, []);
  } finally {
    await env.close();
  }
});

test("role edits send the permissions and current version", async () => {
  const env = await open("/admin/roles");
  const { page, state } = env;
  try {
    await page.getByRole("button", { name: "Edit akses PA", exact: true }).click();
    await page.getByRole("checkbox", { name: "Menolak permohonan", exact: true }).check();
    await page.getByRole("button", { name: "Simpan akses", exact: true }).click();
    await page.getByRole("status").filter({ hasText: "Akses PA disimpan." }).waitFor();
    assert.ok(state.definitions[0].permissions.includes("reject"));
    const edit = state.calls.find(
      (call) => call.method === "PATCH" && call.path === "/staff/roles/pa",
    );
    assert.equal(edit.body.version, 1);
    assert.deepEqual(edit.body.permissions, ["bookings", "approve", "reject"]);
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
    await page.locator("#hasNotaDinas").check();
    await page.locator("#nd").setInputFiles({
      name: "official.pdf",
      mimeType: "application/pdf",
      buffer: Buffer.from("%PDF-1.7\nSynthetic document bytes"),
    });
    await page.locator('.booking-form .form-actions button[type="submit"]').click();
    await page.locator("#name").fill("Visitor Contract");
    await page.locator("#organization").fill("Contract Organization");
    await page.locator("#email").fill("visitor@example.test");
    await page.locator("#phone").fill("00123456789");
    await page.locator("#visitorPhoto").setInputFiles({
      name: "visitor.png",
      mimeType: "image/png",
      buffer: Buffer.from(
        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=",
        "base64",
      ),
    });
    await page.locator('.booking-form .form-actions button[type="submit"]').click();
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
    const qr = page.getByRole("img", { name: "QR kunjungan untuk Visitor Contract" });
    assert.equal(await qr.getAttribute("src"), qrImage);
    assert.equal(state.calls.filter((call) => call.path.endsWith("/admission")).length, 1);
    state.booking.status = "checked_in";
    state.booking.version++;
    Object.assign(state.booking.timing, {
      displayStatus: "checked_in",
      canCheckOut: true,
      canCancel: false,
      serverTime: Date.parse("2099-08-19T12:00:00+08:00"),
    });
    await page.getByRole("button", { name: "Perbarui", exact: true }).click();
    await page.getByText("Tunjukkan QR ini saat check out.").waitFor();
    assert.equal(await qr.getAttribute("src"), qrImage);
    assert.equal(state.calls.filter((call) => call.path.endsWith("/admission")).length, 2);
    assert.equal(await page.getByRole("button", { name: "Perbarui QR" }).count(), 0);
    for (const width of [320, 1280]) {
      await page.setViewportSize({ width, height: 900 });
      const box = await qr.boundingBox();
      assert.ok(box && box.width === box.height && box.x >= 0 && box.x + box.width <= width);
    }
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
      // The server decides allowed commands (ADR-0003); it grants cancel once the role may.
      get allowedActions() {
        return ["reschedule", ...(permitCancellation ? ["cancel"] : [])];
      },
    }));
    let permitCancellation = false;
    const cancellationCalls = [];
    const staffSession = () => ({
      active: true,
      id: "pa-42",
      name: "PA Layanan",
      role: "PA",
      permissions: ["bookings", "approve", "reschedule", ...(permitCancellation ? ["cancel"] : [])],
      assignedOfficerIds: ["officer-server-42"],
    });
    await context.route("**/v1/staff/session", (route) => route.fulfill({ json: staffSession() }));
    await context.route("**/v1/staff/session/refresh", (route) =>
      route.fulfill({ json: staffToken(staffSession()) }),
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
    await page.getByRole("button", { name: "Lihat Nota Dinas" }).click();
    const memoViewer = page.getByRole("dialog", { name: "Nota Dinas", exact: true });
    const downloading = page.waitForEvent("download");
    await memoViewer.getByRole("link", { name: "Unduh Nota Dinas" }).click();
    assert.equal((await downloading).suggestedFilename(), "official.pdf");
    await page.keyboard.press("Escape");
    await memoViewer.waitFor({ state: "hidden" });
    downloadDenied = true;
    await page.getByRole("button", { name: "Lihat Nota Dinas" }).click();
    await memoViewer
      .getByRole("alert")
      .filter({ hasText: "Akses dokumen tidak diizinkan." })
      .waitFor();
    await page.keyboard.press("Escape");
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
    await page
      .getByRole("button", { name: "Lihat detail pegawai Pegawai Layanan", exact: true })
      .waitFor();
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
  const env = await open("/admin/booking", {
    role: "PA",
    permissions: ["bookings", "reports", "reportExport"],
  });
  try {
    const { page, state } = env;
    await page.getByRole("button", { name: "Ekspor laporan kunjungan" }).click();
    await page.getByTestId("report-total").waitFor();
    assert.equal(await page.getByTestId("report-total").innerText(), "0");
    assert.equal(
      await page.getByRole("button", { name: "Unduh ringkasan CSV", exact: true }).isDisabled(),
      true,
    );
    assert.equal(
      state.calls.some((call) => call.path === "/staff/reports/summary"),
      true,
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

test("the visitor language control uses the same buttons across landing, task and standalone layouts", async () => {
  for (const width of [320, 768, 1280]) {
    const { page, close } = await open("/", { viewport: { width, height: 850 } });
    try {
      const mobile = width <= 760;
      const layouts = [
        ["/", mobile ? ".hero-toolbar" : ".visitor-header"],
        ...(mobile
          ? [
              ["/?view=registration#registration", ".landing-task-toolbar"],
              ["/?view=status#booking-status", ".landing-task-toolbar"],
            ]
          : []),
        ["/booking", ".visitor-header"],
        ["/status", ".visitor-header"],
      ];
      let language = "id";
      let buttonStyles;
      for (const [path, toolbar] of layouts) {
        if (path !== "/") await page.goto(server.origin + path);
        const control = page
          .locator(toolbar)
          .getByRole("group", { name: /^(Bahasa halaman|Page language)$/ });
        const id = control.getByRole("button", { name: "ID · Bahasa Indonesia", exact: true });
        const en = control.getByRole("button", { name: "EN · English", exact: true });
        const selected = language === "id" ? id : en;
        const other = language === "id" ? en : id;
        assert.equal(await selected.getAttribute("aria-pressed"), "true");
        assert.equal(await other.getAttribute("aria-pressed"), "false");

        const styles = await selected.evaluate((button) => {
          const { width, height, left, right } = button.getBoundingClientRect();
          const { backgroundColor, borderRadius, fontSize } = getComputedStyle(button);
          return { width, height, left, right, backgroundColor, borderRadius, fontSize };
        });
        const { left, right, ...appearance } = styles;
        assert.ok(left >= 0 && right <= width);
        assert.ok(appearance.width >= 44 && appearance.height >= 44);
        assert.equal(
          await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
          true,
        );
        buttonStyles ??= appearance;
        assert.deepEqual(appearance, buttonStyles);

        await other.press("Enter");
        language = language === "id" ? "en" : "id";
        assert.equal(await other.getAttribute("aria-pressed"), "true");
        assert.equal(await page.locator("html").getAttribute("lang"), language);
      }
    } finally {
      await close();
    }
  }
});

test("destination, purpose and picker fields share their pointer and keyboard focus ring", async () => {
  for (const width of [390, 1280]) {
    const { page, close } = await open("/booking", { viewport: { width, height: 900 } });
    try {
      const outline = (selector) =>
        page.locator(selector).evaluate((control) => {
          const { outlineStyle, outlineWidth, outlineOffset, outlineColor } =
            getComputedStyle(control);
          return { outlineStyle, outlineWidth, outlineOffset, outlineColor };
        });
      await page.locator("#purpose").click();
      const focusRing = await outline("#purpose");
      assert.equal(focusRing.outlineStyle, "solid");
      assert.equal(focusRing.outlineWidth, "3px");
      assert.equal(focusRing.outlineOffset, "4px");

      for (const selector of ["#officerId", "#date", ".time-picker-trigger"]) {
        await page.locator(selector).click();
        assert.deepEqual(await outline(selector), focusRing);
        if (selector === "#officerId") {
          await page.keyboard.press("ArrowDown");
          const option = page.locator("#officerId option:focus-visible");
          await option.waitFor();
          assert.equal((await outline("#officerId option:focus-visible")).outlineOffset, "-2px");
          const next = page.locator("#officerId option:enabled").last();
          await next.hover();
          assert.equal(await next.evaluate((node) => getComputedStyle(node).borderRadius), "6px");
          assert.notEqual(await next.evaluate((node) => getComputedStyle(node).backgroundColor), "rgba(0, 0, 0, 0)");
        }
        await page.keyboard.press("Escape");
      }
      await page.locator("#purpose").click();
      await page.keyboard.press("Shift+Tab");
      assert.deepEqual(await outline(".time-picker-trigger"), focusRing);

      await page.goto(server.origin + "/status");
      await page.locator("#status-code").click();
      assert.deepEqual(await outline(".input-otp"), focusRing);
      await page.locator("#status-email").click();
      assert.deepEqual(await outline(".email-address"), focusRing);
      assert.equal((await outline(".input-otp")).outlineStyle, "none");
    } finally {
      await close();
    }
  }
});

for (const role of ["PA", "Protocol Officer", "Receptionist", "Super Admin"]) {
  test(`staff avatar shows the current employee profile on hover for ${role} without directory permission`, async () => {
    const env = await open("/admin/login", { permissions: ["bookings"] });
    try {
      const { page, state } = env;
      state.role = role;
      state.employees.push({
        id: "admin-42",
        name: "Pegawai Profil Uji",
        email: "profile@example.test",
        roles: [role],
        active: true,
        version: 1,
        assignedOfficerIds: role === "PA" ? ["3"] : [],
        jobTitle: "Asisten",
        directorate: "Direktorat Uji",
        nip: "123456789",
        phoneNumbers: ["0812345678", "+62 542 123456"],
      });
      await page.goto(server.origin + "/admin/dashboard");
      const profileRequests = () => state.calls.filter((call) => call.path.startsWith("/staff/employees"));
      const avatar = page.getByRole("button", { name: "Buka profil saya" });
      await avatar.waitFor();
      assert.equal(profileRequests().length, 0);
      await avatar.hover();
      const dialog = page.getByRole("dialog", { name: "Profil saya", exact: true });
      await dialog.getByRole("heading", { name: "Pegawai Profil Uji" }).waitFor();
      await dialog.getByText(role, { exact: true }).waitFor();
      for (const value of [
        "profile@example.test", "Asisten", "Direktorat Uji", "123456789", "0812345678", "+62 542 123456",
      ]) await dialog.getByText(value, { exact: true }).waitFor();
      await dialog.hover();
      assert.equal(await dialog.isVisible(), true);
      assert.deepEqual(profileRequests().map(({ path, method, headers }) =>
        [path, method, headers.authorization]), [["/staff/employees/admin-42", "GET", "Bearer fixture-staff-token"]]);
      await page.mouse.move(0, 890);
      await dialog.waitFor({ state: "hidden" });
      await avatar.focus();
      await dialog.waitFor();
      await audit(page, `profile-hover-${role.replaceAll(" ", "-")}`);
      await page.keyboard.press("Escape");
      await dialog.waitFor({ state: "hidden" });
      await page.waitForFunction(
        () => document.activeElement?.getAttribute("aria-label") === "Buka profil saya",
      );
    } finally {
      await env.close();
    }
  });
}

test("login resumes an existing staff session instead of showing a new login form", async () => {
  const env = await open("/admin/login");
  try {
    await env.page.waitForURL("**/admin/dashboard");
    await env.page.locator(".admin-shell").waitFor();
    assert.equal(await env.page.locator("#password").count(), 0);
    assert.equal(env.state.calls.some((call) => call.path === "/staff/sessions"), false);
  } finally {
    await env.close();
  }
});

test("login renews an expired form once and preserves entered credentials", async () => {
  const env = await open("/admin/login", { sessionStatus: 401 });
  try {
    const { page, context } = env;
    let attempts = 0;
    let initialized = false;
    const bodies = [];
    page.on("request", (request) => {
      if (attempts === 1 && request.method() === "GET" && new URL(request.url()).pathname === "/admin/login")
        initialized = true;
    });
    await context.route("**/v1/staff/sessions", (route) => {
      bodies.push(route.request().postDataJSON());
      attempts++;
      return route.fulfill({
        status: attempts === 1 ? 403 : 401,
        json: attempts === 1
          ? { code: "csrf_expired", message: "Formulir sudah kedaluwarsa. Muat ulang halaman lalu coba lagi." }
          : { code: "invalid_credentials", message: "Email atau kata sandi tidak sesuai." },
      });
    });
    await page.locator("#email").fill("visitor@example.test");
    await page.locator("#password").fill("Fictional-password-42");
    await page.getByRole("button", { name: "Masuk ke backoffice", exact: true }).click();
    await page.getByText("Email atau kata sandi tidak sesuai.", { exact: true }).waitFor();
    assert.equal(initialized, true);
    assert.equal(attempts, 2);
    assert.deepEqual(bodies[0], bodies[1]);
    assert.equal(await page.locator("#password").inputValue(), "Fictional-password-42");
  } finally {
    await env.close();
  }
});

test("staff access expiry refreshes silently and only real activity extends the eight-hour session", async () => {
  const context = await browser.newContext();
  try {
    await context.addInitScript(() => {
      document.cookie = "__Host-vms-csrf=fixture-csrf; Secure; Path=/";
    });
    const page = await context.newPage();
    const minute = 60_000;
    const start = Date.parse("2026-11-20T00:00:00Z");
    await page.clock.install({ time: start });
    let idleExpiry = start + 8 * 60 * minute;
    let accessExpiry = start;
    let refreshes = 0;
    let renewals = 0;
    let rejectNextRead = false;
    let interruptNextRefresh = false;
    const refreshIds = [];
    await context.route("**/v1/**", async (route) => {
      const request = route.request();
      const path = new URL(request.url()).pathname;
      const now = await page.evaluate(() => Date.now());
      const session = () => ({
        active: true,
        id: "staff-42",
        name: "Staf Uji",
        role: "Super Admin",
        permissions: ["people"],
        assignedOfficerIds: [],
        timing: {
          serverTime: now,
          idleExpiresAt: new Date(idleExpiry).toISOString(),
          absoluteExpiresAt: new Date(start + 24 * 60 * minute).toISOString(),
        },
      });
      const unauthorized = () =>
        route.fulfill({ status: 401, json: { code: "unauthenticated" } });
      if (path === "/v1/staff/session/refresh") {
        assert.equal(request.headers()["x-csrf-token"], "fixture-csrf");
        if (now >= idleExpiry) return unauthorized();
        refreshIds.push(request.postDataJSON().refreshRequestId);
        if (interruptNextRefresh) {
          interruptNextRefresh = false;
          return route.fulfill({ status: 503, json: {} });
        }
        refreshes++;
        accessExpiry = Math.min(now + 5 * minute, idleExpiry);
        return route.fulfill({
          json: {
            accessToken: `test-access-${refreshes}`,
            tokenType: "Bearer",
            accessExpiresAt: new Date(accessExpiry).toISOString(),
            session: session(),
          },
        });
      }
      if (now >= accessExpiry || request.headers().authorization !== `Bearer test-access-${refreshes}`)
        return unauthorized();
      if (path === "/v1/staff/session/renew") {
        renewals++;
        idleExpiry = now + 8 * 60 * minute;
        return route.fulfill({ json: session() });
      }
      if (path === "/v1/staff/session") {
        if (rejectNextRead) {
          rejectNextRead = false;
          return unauthorized();
        }
        return route.fulfill({ json: session() });
      }
      if (path === "/v1/staff/employees")
        return route.fulfill({ json: { items: [], total: 0, totalAll: 0, page: 1, pageSize: 25 } });
      return route.fulfill({ status: 404, json: {} });
    });

    const visibilityCheck = () =>
      page.evaluate(() => window.dispatchEvent(new Event("visibilitychange")));
    const expectCurrent = async () => {
      const current = page.waitForResponse((result) => result.url().endsWith("/v1/staff/session"));
      await visibilityCheck();
      assert.equal((await current).status(), 200);
    };
    await page.goto(server.origin + "/admin/employees");
    const heading = page.getByRole("heading", { name: "Pegawai", exact: true });
    await heading.waitFor();
    await page.clock.fastForward(6 * minute);
    await expectCurrent();
    assert.equal(refreshes, 2);
    assert.equal(renewals, 0);

    rejectNextRead = true;
    interruptNextRefresh = true;
    const recovered = page.waitForResponse(
      (result) => result.url().endsWith("/v1/staff/session") && result.ok(),
    );
    await visibilityCheck();
    await recovered;
    assert.equal(refreshes, 3);
    assert.equal(refreshIds.at(-2), refreshIds.at(-1));

    const renewed = page.waitForResponse("**/staff/session/renew");
    await page.keyboard.press("Tab");
    assert.equal((await renewed).status(), 200);
    await page.keyboard.press("Tab");
    await page.clock.fastForward(8 * 60 * minute - 3 * minute);
    await expectCurrent();
    assert.equal(renewals, 1);
    await heading.waitFor();
    await page.clock.fastForward(3 * minute);
    await visibilityCheck();
    await page.waitForURL("**/admin/login");
  } finally {
    await context.close();
  }
});

test("PA approval submits the correct room field to the production API", async () => {
  const env = await open();
  const { page, context } = env;
  try {
    const visit = {
      id: "42",
      name: "Pengunjung Persetujuan",
      officerId: catalog.officers[0].id,
      host: catalog.officers[0].name,
      vip: false,
      date: "2099-08-19",
      time: "10:00",
      duration: 60,
      room: "",
      visitKind: "standard",
      status: "pending",
      version: 1,
      timing: {
        serverTime: Date.parse("2099-08-19T09:00:00+08:00"),
        opensAt: Date.parse("2099-08-19T09:30:00+08:00"),
        endsAt: Date.parse("2099-08-19T11:00:00+08:00"),
        displayStatus: "pending",
        canCheckIn: false,
        canCheckOut: false,
        canCancel: true,
      },
      allowedActions: ["approve", "reject", "cancel"],
    };
    const room = { id: "17", name: "Ruang Persetujuan", active: true };
    let expectedRoomId;
    const session = {
      id: "pa-42",
      name: "PA Layanan",
      active: true,
      role: "PA",
      permissions: ["bookings", "approve"],
      assignedOfficerIds: [visit.officerId],
      timing: {
        serverTime: visit.timing.serverTime,
        idleExpiresAt: "2099-08-19T17:00:00+08:00",
        absoluteExpiresAt: "2099-08-20T09:00:00+08:00",
      },
    };
    await context.addInitScript(() => {
      document.cookie = "__Host-vms-csrf=fixture-csrf; Secure; Path=/";
    });
    await context.route("**/v1/staff/session/refresh", (route) =>
      route.fulfill({
        json: {
          accessToken: "fixture-pa-token",
          tokenType: "Bearer",
          accessExpiresAt: "2099-08-19T09:05:00+08:00",
          session,
        },
      }),
    );
    await context.route("**/v1/staff/session", (route) =>
      route.fulfill({ json: session }),
    );
    await context.route("**/v1/staff/catalogs", (route) =>
      route.fulfill({ json: { ...catalog, rooms: [room] } }),
    );
    await context.route("**/v1/staff/bookings**", (route) => {
      const url = new URL(route.request().url());
      if (url.pathname.endsWith("/approve")) {
        const body = route.request().postDataJSON();
        if ("room" in body || body.roomId !== expectedRoomId) {
          return route.fulfill({
            status: 400,
            json: {
              code: "invalid_request",
              message: "Permintaan tidak valid. Periksa data lalu coba lagi.",
            },
          });
        }
        visit.status = "approved";
        visit.version++;
        if ("roomId" in body) {
          visit.room = body.roomId ? room.name : "";
          visit.roomId = body.roomId;
        }
        visit.timing.displayStatus = "approved";
      }
      if (url.pathname === "/v1/staff/bookings") {
        const items = visit.status === "pending" ? [visit] : [];
        return route.fulfill({
          json: { items, total: items.length, totalScoped: 1, page: 1, pageSize: 7 },
        });
      }
      return route.fulfill({ json: visit });
    });

    for (const scenario of [
      { initial: null, choose: room.name, roomId: room.id },
      { initial: room.id },
      { initial: room.id, choose: "Tanpa ruangan", roomId: null },
      { initial: null },
    ]) {
      expectedRoomId = scenario.roomId;
      visit.status = "pending";
      visit.version = 1;
      visit.roomId = scenario.initial;
      visit.room = scenario.initial ? room.name : "";
      visit.timing.displayStatus = "pending";
      await page.goto(server.origin + "/admin/booking?status=pending");
      await page.getByRole("button", { name: "Lihat detail Pengunjung Persetujuan" }).click();
      await page.getByRole("button", { name: "Setujui", exact: true }).click();
      const roomSelect = page.getByLabel(/Ruangan/);
      if (scenario.choose) await roomSelect.selectOption({ label: scenario.choose });
      else assert.equal(await roomSelect.inputValue(), scenario.initial ?? "");
      await page.getByLabel(/Instruksi pengunjung/).fill("Lapor ke resepsionis.");
      await page.getByLabel(/Catatan koordinasi internal/).fill("Hubungi petugas piket.");
      const submitted = page.waitForResponse("**/v1/staff/bookings/42/approve");
      await page.getByRole("button", { name: "Setujui kunjungan", exact: true }).click();
      const result = await submitted;
      assert.equal(result.status(), 200, await result.text());
      assert.deepEqual(result.request().postDataJSON(), {
        ...(scenario.roomId === undefined ? {} : { roomId: scenario.roomId }),
        duration: 60,
        visitorInstructions: "Lapor ke resepsionis.",
        coordinationNote: "Hubungi petugas piket.",
        version: 1,
      });
      await page.getByRole("status").filter({ hasText: "Kunjungan disetujui." }).waitFor();
    }
  } finally {
    await env.close();
  }
});

test("PA creates an approved visit with or without an optional room", async () => {
  const env = await open();
  const { page, context } = env;
  try {
    const officer = { ...catalog.officers[0], id: "2" };
    const room = { id: "17", name: "Ruang Kunjungan", active: true };
    const serverTime = Date.parse("2099-08-19T09:00:00+08:00");
    const session = {
      id: "pa-42",
      name: "PA Layanan",
      active: true,
      role: "PA",
      permissions: ["bookings", "create"],
      assignedOfficerIds: [officer.id],
      timing: { serverTime },
    };
    await context.addInitScript(() => {
      document.cookie = "__Host-vms-csrf=fixture-csrf; Secure; Path=/";
    });
    await context.route("**/v1/staff/session/refresh", (route) =>
      route.fulfill({
        json: {
          accessToken: "fixture-pa-token",
          tokenType: "Bearer",
          accessExpiresAt: "2099-08-19T09:05:00+08:00",
          session,
        },
      }),
    );
    await context.route("**/v1/staff/session", (route) => route.fulfill({ json: session }));
    await context.route("**/v1/staff/catalogs", (route) =>
      route.fulfill({ json: { ...catalog, officers: [officer], rooms: [room] } }),
    );
    const uploadToken = "vmsu1.synthetic-photo-token";
    await context.route("**/v1/staff/attachments", (route) => {
      const multipart = route.request().postDataBuffer().toString("latin1");
      assert.match(multipart, /name="purpose"\r\n\r\nvisitorPhoto\r\n/);
      return route.fulfill({
        status: 201,
        json: { token: uploadToken, expiresAt: "2099-08-19T10:00:00+08:00", serverTime },
      });
    });
    let expectedRoomId;
    let created;
    await context.route("**/v1/staff/bookings**", (route) => {
      const request = route.request();
      if (request.method() === "POST") {
        const body = request.postDataJSON();
        if ("room" in body || body.roomId !== expectedRoomId) {
          return route.fulfill({
            status: 400,
            json: {
              code: "invalid_request",
              message: "Permintaan tidak valid. Periksa data lalu coba lagi.",
            },
          });
        }
        created = {
          id: "53",
          name: body.name,
          officerId: body.officerId,
          pejabat: { nama: officer.name },
          visitKind: "standard",
          date: body.date,
          time: body.time,
          duration: body.duration,
          room: body.roomId ? room.name : "",
          roomId: body.roomId,
          status: "approved",
          version: 1,
          timing: {
            serverTime,
            opensAt: Date.parse("2099-08-19T09:30:00+08:00"),
            endsAt: Date.parse("2099-08-19T11:00:00+08:00"),
            displayStatus: "approved",
            canCheckIn: false,
            canCheckOut: false,
            canCancel: true,
          },
          allowedActions: ["reschedule", "cancel"],
        };
        return route.fulfill({ status: 201, json: created });
      }
      if (new URL(request.url()).pathname.endsWith("/53")) return route.fulfill({ json: created });
      return route.fulfill({ json: { items: [], total: 0, totalScoped: 0, page: 1, pageSize: 7 } });
    });

    for (const roomId of [room.id, undefined]) {
      expectedRoomId = roomId;
      await page.goto(server.origin + "/admin/booking?new=1&kind=standard");
      const dialog = page.getByRole("dialog", { name: "Buat janji temu" });
      await dialog.locator(".date-picker-trigger").click();
      await page.locator('.shadcn-calendar-day[data-day="2099-08-19"]:not(:disabled)').first().click();
      await dialog.locator(".time-picker-trigger").click();
      await page
        .getByRole("listbox", { name: "Jam", exact: true })
        .getByRole("option", { name: "10", exact: true })
        .click();
      await page
        .getByRole("listbox", { name: "Menit", exact: true })
        .getByRole("option", { name: "00", exact: true })
        .click();
      await dialog.locator(".date-picker > small").click();
      if (roomId) await dialog.locator("#appointment-room").selectOption({ label: room.name });
      await dialog.locator("#appointment-purpose").fill("Pertemuan pengujian kunjungan PA.");
      await dialog.getByRole("button", { name: "Lanjutkan", exact: true }).click();
      await dialog.locator("#appointment-name").fill("Pengunjung Uji PA");
      await dialog.locator("#appointment-organization").fill("Instansi Uji");
      await dialog.locator("#appointment-email").fill("visitor@example.test");
      await dialog.locator("#appointment-phone").fill("081234567890");
      await dialog.locator("#appointment-photo").setInputFiles({
        name: "visitor.png",
        mimeType: "image/png",
        buffer: Buffer.from(
          "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=",
          "base64",
        ),
      });
      await dialog.getByRole("button", { name: "Lanjutkan", exact: true }).click();
      await dialog.getByRole("heading", { name: "Periksa janji temu" }).waitFor();
      await dialog
        .locator("dd")
        .filter({ hasText: roomId ? room.name : "Belum ditetapkan" })
        .waitFor();
      const submitted = page.waitForResponse(
        (result) => result.url().endsWith("/v1/staff/bookings") && result.request().method() === "POST",
      );
      await dialog.getByRole("button", { name: "Buat janji temu", exact: true }).click();
      const result = await submitted;
      assert.equal(result.status(), 201, await result.text());
      const body = result.request().postDataJSON();
      assert.equal(body.officerId, officer.id);
      assert.equal(body.visitorPhotoToken, uploadToken);
      assert.deepEqual(body.visitors, ["Pengunjung Uji PA"]);
      assert.match(body.idempotencyKey, /^[0-9a-f-]{36}$/u);
      await page
        .getByRole("status")
        .filter({ hasText: "Janji temu berhasil dibuat dan langsung disetujui." })
        .waitFor();
    }
  } finally {
    await env.close();
  }
});

test("production restricted creation sends the selected officer and room IDs", async () => {
  const env = await open("/admin/login", { permissions: ["bookings", "create"] });
  try {
    const { page, context, state } = env;
    state.role = "Protocol Officer";
    const officer = { id: "3", name: "Pejabat Uji", vip: false, active: true };
    const room = { id: "7", name: "Ruangan Uji", active: true };
    await context.route("**/v1/staff/catalogs", (route) =>
      route.fulfill({ json: { ...catalog, officers: [officer], rooms: [room] } }),
    );
    await context.route("**/v1/staff/bookings**", (route) => {
      if (route.request().method() !== "POST")
        return route.fulfill({ json: { items: [], total: 0, totalScoped: 0, page: 1, pageSize: 7 } });
      const { idempotencyKey, ...body } = route.request().postDataJSON();
      assert.match(idempotencyKey, /^[0-9a-f-]{36}$/u);
      assert.deepEqual(body, {
        visitKind: "restricted",
        date: "2099-08-19",
        time: "10:00",
        officerId: officer.id,
        roomId: room.id,
        visitorInstructions: "",
        coordinationNote: "",
      });
      return route.fulfill({
        status: 201,
        json: {
          ...body,
          id: "53",
          reference: "TRB-53",
          restrictedView: "full",
          room: room.name,
          pejabat: { nama: officer.name },
          duration: 60,
          status: "approved",
          version: 1,
          timing: {
            serverTime: Date.parse("2099-08-19T09:00:00+08:00"),
            opensAt: Date.parse("2099-08-19T09:30:00+08:00"),
            endsAt: Date.parse("2099-08-19T11:00:00+08:00"),
            displayStatus: "approved",
            canCheckIn: false,
            canCheckOut: false,
            canCancel: true,
          },
          allowedActions: ["reschedule", "cancel", "update"],
        },
      });
    });
    await page.goto(server.origin + "/admin/today?new=1&kind=restricted");
    const dialog = page.getByRole("dialog", { name: "Buat kunjungan terbatas", exact: true });
    assert.equal(await dialog.locator("#restricted-host").getAttribute("required"), "");
    await dialog.locator(".date-picker-trigger").click();
    await page.locator('.shadcn-calendar-day[data-day="2099-08-19"]:not(:disabled)').first().click();
    await dialog.locator(".time-picker-trigger").click();
    await page.getByRole("listbox", { name: "Jam" }).getByRole("option", { name: "10", exact: true }).click();
    await page.getByRole("listbox", { name: "Menit" }).getByRole("option", { name: "00", exact: true }).click();
    await dialog.locator("#restricted-host").selectOption(officer.id);
    await dialog.locator("#restricted-room").selectOption(room.id);
    const saved = page.waitForResponse((result) =>
      result.url().endsWith("/v1/staff/bookings") && result.request().method() === "POST",
    );
    await dialog.getByRole("button", { name: "Buat kunjungan terbatas", exact: true }).click();
    assert.equal((await saved).status(), 201);
    await dialog.waitFor({ state: "hidden" });
  } finally {
    await env.close();
  }
});

test("large visitor rosters stay bounded and toggle in PA, protocol and reception previews", async () => {
  for (const role of ["PA", "Protocol Officer", "Receptionist"]) {
    for (const width of [390, 1280]) {
      const env = await open("/admin/login", {
        permissions: ["bookings", "approve", "checkin", "checkout"],
        viewport: { width, height: 900 },
      });
      const { page, context, state } = env;
      try {
        state.role = role;
        const session = {
          active: true,
          id: "staff-42",
          name: "Petugas Uji",
          role,
          permissions: ["bookings", "approve", "checkin", "checkout"],
          assignedOfficerIds: ["officer-server-42"],
        };
        await context.route("**/v1/staff/session", (route) =>
          route.fulfill({ json: session }),
        );
        await context.route("**/v1/staff/session/refresh", (route) =>
          route.fulfill({ json: staffToken(session) }),
        );
        const visit = {
          id: "roster-42",
          code: "88001001",
          name: "Pengunjung Uji",
          email: "visitor@example.test",
          phone: "08123456789",
          organization: "Instansi Contoh",
          host: "Pejabat Contoh",
          officerId: "officer-server-42",
          vip: role === "Protocol Officer",
          visitKind: "standard",
          date: "2099-08-19",
          time: "09:00",
          duration: 60,
          room: "Ruang Uji",
          status: "approved",
          version: 1,
          purpose: "Pengujian daftar rombongan",
          identityType: "KTP",
          visitors: Array.from(
            { length: 20 },
            (_, i) => `Anggota ${i + 1} ${"NamaPanjang".repeat(12)}`,
          ),
          timing: {
            serverTime: Date.parse("2099-08-19T09:00:00+08:00"),
            opensAt: Date.parse("2099-08-19T08:30:00+08:00"),
            endsAt: Date.parse("2099-08-19T10:00:00+08:00"),
            displayStatus: "approved",
            canCheckIn: true,
            canCheckOut: false,
            canCancel: false,
          },
          allowedActions: ["reschedule", "cancel"],
        };
        await context.route("**/v1/staff/resolve", (route) =>
          route.fulfill({ json: visit }),
        );
        await context.route("**/v1/staff/bookings**", (route) =>
          route.fulfill({
            json:
              new URL(route.request().url()).pathname === "/v1/staff/bookings"
                ? {
                    items: [visit],
                    total: 1,
                    totalScoped: 1,
                    page: 1,
                    pageSize: 7,
                  }
                : visit,
          }),
        );
        if (role === "Receptionist") {
          await page.goto(server.origin + "/admin/check-in");
          await page
            .getByRole("button", { name: /Kode kunjungan Masukkan/ })
            .click();
          await page.locator("#booking-code").fill(visit.code);
          await page.locator("#booking-code").press("Enter");
        } else {
          await page.goto(server.origin + "/admin/booking?status=approved");
          await page
            .getByRole("button", {
              name: "Lihat detail Pengunjung Uji",
              exact: true,
            })
            .click();
        }
        const dialog = page.getByRole("dialog", {
          name:
            role === "Receptionist"
              ? "Validasi pengunjung"
              : "Detail kunjungan",
          exact: true,
        });
        const roster = dialog.locator(".visitor-roster");
        const trigger = roster.locator("summary");
        const list = roster.getByRole("list", {
          name: "Nama anggota rombongan",
        });
        assert.equal(await roster.getAttribute("open"), null);
        assert.equal(await trigger.textContent(), "20 orang");
        assert.equal(await list.isVisible(), false);
        await trigger.click();
        assert.equal(await list.getByRole("listitem").count(), 20);
        assert.equal(await list.isVisible(), true);
        const bounds = await list.evaluate((n) => ({
          height: n.clientHeight,
          scroll: n.scrollHeight,
          width: n.clientWidth,
          scrollWidth: n.scrollWidth,
        }));
        assert.ok(bounds.height <= 193 && bounds.scroll > bounds.height);
        assert.ok(bounds.scrollWidth <= bounds.width);
        await list.focus();
        await list.press("End");
        await page.waitForFunction(
          () => document.querySelector(".visitor-roster ul")?.scrollTop > 0,
        );
        assert.equal(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth,
          ),
          true,
        );
        await audit(
          page,
          `visitor-roster-${role.replaceAll(" ", "-")}-${width}`,
        );
        await trigger.press("Enter");
        assert.equal(await list.isVisible(), false);
        await trigger.press("Space");
        assert.equal(await list.isVisible(), true);
        assert.deepEqual(state.errors, []);
      } finally {
        await env.close();
      }
    }
  }
});
