import test from "node:test";
import assert from "node:assert/strict";
import { createServer, request } from "node:http";
import { mkdtemp, mkdir, writeFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, dirname, join, resolve } from "node:path";
import { startBackendSite } from "../tooling/backend-site.mjs";

test("local backend site preserves API requests, cookies, binary responses and failures", async (t) => {
  const root = await mkdtemp(join(tmpdir(), "vms-backend-site-"));
  t.after(() => {
    assert.equal(dirname(root), resolve(tmpdir()));
    assert.match(basename(root), /^vms-backend-site-/);
    return rm(root, { recursive: true, force: true });
  });
  for (const app of ["visitor", "admin"]) {
    await mkdir(join(root, "apps", app, "dist", "assets"), { recursive: true });
    await writeFile(join(root, "apps", app, "dist", "index.html"), `${app} built page`);
  }
  const binary = Buffer.from([0, 255, 128, 13, 10]);
  const upstream = createServer(async (req, res) => {
    if (req.url === "/v1/disconnected") return req.socket.destroy();
    if (req.url === "/v1/attachment") {
      res.writeHead(200, { "Content-Type": "application/pdf" });
      return res.end(binary);
    }
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    res.writeHead(422, {
      "Content-Type": "application/json",
      "Set-Cookie": ["session=synthetic; HttpOnly; SameSite=Lax", "other=synthetic"],
    });
    res.end(
      JSON.stringify({
        path: req.url,
        method: req.method,
        origin: req.headers.origin,
        cookie: req.headers.cookie,
        requestMarker: req.headers["x-vms-request"],
        body: Buffer.concat(chunks).toString("base64"),
      }),
    );
  });
  await new Promise((accept) => upstream.listen(0, "127.0.0.1", accept));
  t.after(() => {
    upstream.closeAllConnections();
    return new Promise((accept) => upstream.close(accept));
  });
  const apiOrigin = `http://127.0.0.1:${upstream.address().port}`;
  const site = await startBackendSite({ apiOrigin, port: 0, root });
  t.after(() => site.close());

  const result = await fetch(`${site.origin}/v1/upload?attempt=2`, {
    method: "POST",
    headers: {
      Origin: site.origin,
      Cookie: "session=synthetic",
      "X-VMS-Request": "browser",
      "Content-Type": "application/octet-stream",
    },
    body: binary,
  });
  assert.equal(result.status, 422);
  assert.equal(result.headers.getSetCookie().length, 2);
  assert.deepEqual(await result.json(), {
    path: "/v1/upload?attempt=2",
    method: "POST",
    origin: site.origin,
    cookie: "session=synthetic",
    requestMarker: "browser",
    body: binary.toString("base64"),
  });
  const file = await fetch(`${site.origin}/v1/attachment`);
  assert.equal(file.headers.get("content-type"), "application/pdf");
  assert.deepEqual(Buffer.from(await file.arrayBuffer()), binary);
  const failure = await fetch(`${site.origin}/v1/disconnected`);
  assert.equal(failure.status, 502);
  assert.match(failure.headers.get("content-type"), /application\/json/);

  assert.equal(await (await fetch(`${site.origin}/booking`)).text(), "visitor built page");
  assert.equal(await (await fetch(`${site.origin}/admin/integrations`)).text(), "admin built page");
  const redirect = await fetch(`${site.origin}/admin?lang=id`, { redirect: "manual" });
  assert.equal(redirect.status, 308);
  assert.equal(redirect.headers.get("location"), "/admin/?lang=id");
  for (const path of [
    "/assets/missing.js",
    "/admin/assets/missing",
    "/.env",
    "/__preview/catalogs",
    "/%5c.env",
  ])
    assert.equal((await fetch(site.origin + path)).status, 404, path);
  assert.equal((await fetch(`${site.origin}/booking`, { method: "POST" })).status, 405);
  const foreignHostStatus = await new Promise((accept, reject) => {
    request(site.origin, { headers: { Host: "untrusted.example" } }, (reply) => {
      reply.resume();
      accept(reply.statusCode);
    })
      .on("error", reject)
      .end();
  });
  assert.equal(foreignHostStatus, 403);
  await assert.rejects(
    startBackendSite({ apiOrigin: "http://example.com:8080", root }),
    /127\.0\.0\.1/,
  );
  await assert.rejects(
    startBackendSite({ apiOrigin: "http://user:secret@127.0.0.1:8080", root }),
    /no credentials/,
  );
  await assert.rejects(
    startBackendSite({ apiOrigin: "http://api:8080", root }),
    /127\.0\.0\.1/,
  );
  await assert.rejects(
    startBackendSite({ apiOrigin: "http://untrusted.example:8080", root, dockerGateway: "synthetic" }),
    /127\.0\.0\.1/,
  );
  const dockerSite = await startBackendSite({
    apiOrigin: "http://api:8080", root, dockerGateway: "synthetic", port: 0,
  });
  await dockerSite.close();
});
