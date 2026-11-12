import test, { before, after, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";
import { sessionFor } from "../packages/data/src/domain.ts";
import { startPreview } from "./helpers/preview-server.mjs";

let api, permissions, assignments;
before(async () => {
  api = await startPreview({
    time: "2099-08-19T09:00:00+08:00",
    session: (role) => ({
      ...sessionFor(role),
      ...(permissions ? { permissions } : {}),
      ...(assignments ? { assignedOfficerIds: assignments } : {}),
    }),
  });
});
beforeEach(() => {
  permissions = undefined;
  assignments = undefined;
});
after(async () => api?.close());
const filter = "from=2099-08-01&to=2099-08-31&status=all";
async function get(path) {
  const response = await fetch(api.base + path);
  return { status: response.status, data: await response.json() };
}
async function create(officerId = "o2", kind = "standard") {
  const input = {
    idempotencyKey: randomUUID(),
    visitKind: kind,
    officerId,
    date: "2099-08-19",
    time: "10:00",
    name: `PRIVATE ${randomUUID()}`,
  };
  const body =
    kind === "restricted"
      ? { ...input, serviceId: "0012345678" }
      : {
          ...input,
          organization: "PRIVATE ORG",
          email: "private@example.test",
          phone: "00123456789",
          purpose: "PRIVATE PURPOSE",
          visitors: [input.name],
          identityType: "KTP",
          ack: true,
        };
  const response = await fetch(
    api.base + (kind === "restricted" ? "/staff/bookings?role=Protocol%20Officer" : "/bookings"),
    { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) },
  );
  assert.equal(response.status, 201);
  return response.json();
}
test("summary/export permission is enforced before validation and never depends on booking-list access", async () => {
  await create();
  permissions = ["bookings"];
  assert.equal((await get("/reports/summary?role=PA&from=bad")).status, 403);
  assert.equal((await get(`/reports/export?role=PA&${filter}`)).status, 403);
  permissions = ["reports"];
  const summary = await get(`/reports/summary?role=Super%20Admin&${filter}`);
  assert.equal(summary.status, 200);
  assert.ok(summary.data.total > 0);
  assert.deepEqual((await get("/bookings?role=Super%20Admin")).data, []);
  assert.equal(
    (await get(`/reports/export?role=Super%20Admin&${filter}&revision=${summary.data.revision}`))
      .status,
    403,
  );
});
test("all authorized scopes exclude Terbatas before counts and contain no individual data", async () => {
  const ordinary = await create();
  const before = (await get(`/reports/summary?role=PA&${filter}`)).data.total;
  const restricted = await create("o2", "restricted");
  for (const role of ["PA", "Protocol Officer", "Super Admin"]) {
    const result = await get(`/reports/summary?role=${encodeURIComponent(role)}&${filter}`);
    assert.equal(result.status, 200);
    assert.equal(JSON.stringify(result.data).includes(ordinary.name), false);
    for (const secret of [
      restricted.name,
      restricted.code,
      "0012345678",
      "private@example.test",
      "PRIVATE ORG",
    ])
      assert.equal(JSON.stringify(result.data).includes(secret), false, secret);
  }
  assert.equal((await get(`/reports/summary?role=PA&${filter}`)).data.total, before);
  assignments = [];
  assert.equal((await get(`/reports/summary?role=PA&${filter}`)).data.total, 0);
});
test("CSV matches the displayed revision; changed counts, assignment or permissions invalidate export", async () => {
  await create();
  const summary = (await get(`/reports/summary?role=PA&${filter}`)).data;
  const exportPath = `/reports/export?role=PA&${filter}&revision=${summary.revision}`;
  const exported = await get(exportPath);
  assert.equal(exported.status, 200);
  assert.equal(exported.data.revision, summary.revision);
  assert.ok(
    exported.data.csv.includes('"Tanggal awal","Tanggal akhir","Cakupan","Status","Jumlah"'),
  );
  const total = exported.data.csv
    .split("\r\n")
    .slice(1)
    .reduce((sum, row) => sum + Number(row.split(",").at(-1).replaceAll('"', "")), 0);
  assert.equal(total, summary.total);
  await create();
  assert.equal((await get(exportPath)).status, 409);
  const latest = (await get(`/reports/summary?role=PA&${filter}`)).data;
  assignments = ["o3"];
  assert.equal(
    (await get(`/reports/export?role=PA&${filter}&revision=${latest.revision}`)).status,
    409,
  );
  permissions = ["reports"];
  assert.equal(
    (await get(`/reports/export?role=PA&${filter}&revision=${latest.revision}`)).status,
    403,
  );
});
test("invalid ranges/status fail with field feedback while a successful empty period returns explicit zero", async () => {
  for (const query of [
    "from=bad&to=2099-08-31",
    "from=2099-08-31&to=2099-08-01",
    "from=2099-01-01&to=2101-01-01",
    `${filter.replace("status=all", "status=unknown")}`,
  ]) {
    const result = await get(`/reports/summary?role=PA&${query}`);
    assert.equal(result.status, 422);
    assert.ok(result.data.fieldErrors);
    assert.equal("counts" in result.data, false);
  }
  const empty = await get("/reports/summary?role=PA&from=2099-07-01&to=2099-07-31&status=all");
  assert.equal(empty.status, 200);
  assert.equal(empty.data.total, 0);
  assert.ok(Object.values(empty.data.counts).every((count) => count === 0));
  assert.equal((await get(`/reports/summary?role=Receptionist&${filter}`)).status, 403);
});
