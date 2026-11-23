import { createServer, request } from "node:http";
import { readFile, realpath } from "node:fs/promises";
import { extname, resolve, sep } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { parseArgs } from "node:util";

const workspace = fileURLToPath(new URL("..", import.meta.url));
const types = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json",
  ".svg": "image/svg+xml",
  ".ico": "image/x-icon",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".png": "image/png",
  ".webp": "image/webp",
  ".avif": "image/avif",
  ".woff2": "font/woff2",
};

function forwardedHeaders(headers) {
  const excluded = new Set([
    "host",
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailer",
    "transfer-encoding",
    "upgrade",
    ...String(headers.connection || "")
      .toLowerCase()
      .split(",")
      .map((value) => value.trim()),
  ]);
  return Object.fromEntries(Object.entries(headers).filter(([key]) => !excluded.has(key)));
}

function unavailable(response) {
  if (response.headersSent) return response.destroy();
  response.writeHead(502, {
    "Content-Type": "application/json; charset=utf-8",
    "Cache-Control": "no-store",
  });
  response.end(JSON.stringify({ message: "Layanan VMS tidak dapat dihubungi. Coba lagi." }));
}

async function initializeStaff(api, req, path, gatewayKey) {
  return new Promise((accept, reject) => {
    const upstream = request(
      api,
      {
        path: "/internal/staff/initialize",
        headers: {
          Cookie: req.headers.cookie || "",
          "X-VMS-Gateway-Key": gatewayKey,
          "X-VMS-Original-Method": "GET",
          "X-VMS-Original-URI": path,
        },
      },
      (reply) => {
        reply.resume();
        if (reply.statusCode !== 204) return reject(new Error("Staff initialization failed."));
        accept(reply.headers["set-cookie"] || []);
      },
    );
    upstream.setTimeout(5_000, () => upstream.destroy());
    upstream.on("error", reject);
    upstream.end();
  });
}

const localSiteOrigin = "http://localhost:58101";

export async function startBackendSite({ apiOrigin, port = 3000, root = workspace, dockerGateway, siteOrigin }) {
  const api = new URL(apiOrigin);
  if (
    api.protocol !== "http:" ||
    !(api.hostname === "127.0.0.1" || (dockerGateway && api.hostname === "api")) ||
    (!api.port && !dockerGateway) ||
    api.username ||
    api.password ||
    api.pathname !== "/" ||
    api.search ||
    api.hash
  )
    throw new Error("Select an HTTP API at 127.0.0.1 with an explicit port and no credentials.");

  const publicSite = dockerGateway ? new URL(siteOrigin ?? localSiteOrigin) : undefined;
  if (siteOrigin && (!publicSite || publicSite.protocol !== "https:" || publicSite.origin !== siteOrigin))
    throw new Error("VMS_SITE_ORIGIN requires an exact HTTPS origin in Docker gateway mode.");

  const folders = {};
  for (const app of ["visitor", "admin"]) {
    folders[app] = await realpath(resolve(root, "apps", app, "dist"));
    await readFile(resolve(folders[app], "index.html"));
  }

  const server = createServer(async (req, res) => {
    const host = publicSite?.host ?? `127.0.0.1:${server.address().port}`;
    if (req.headers.host !== host) return res.writeHead(403).end();
    if (!req.url.startsWith("/") || req.url.startsWith("//")) return res.writeHead(400).end();

    const url = new URL(req.url, `http://${host}`);
    const gatewayProbe = dockerGateway && (req.method === "GET" || req.method === "HEAD") && (
      url.pathname === "/health/live" ||
      url.pathname === "/health/ready"
    );
    const localSwagger = publicSite?.origin === localSiteOrigin && (
      url.pathname === "/swagger" || url.pathname.startsWith("/swagger/")
    );
    if (url.pathname === "/v1" || url.pathname.startsWith("/v1/") || gatewayProbe || localSwagger) {
      const headers = forwardedHeaders(req.headers);
      for (const name of Object.keys(headers)) {
        if (name === "forwarded" || name.startsWith("x-forwarded-") || name.startsWith("x-vms-gateway-"))
          delete headers[name];
      }
      const upstream = request(
        api,
        { path: req.url, method: req.method, headers },
        (reply) => {
          res.writeHead(reply.statusCode || 502, forwardedHeaders(reply.headers));
          reply.on("error", () => res.destroy());
          reply.pipe(res);
        },
      );
      upstream.setTimeout(80_000, () => upstream.destroy());
      upstream.on("error", () => unavailable(res));
      req.on("aborted", () => upstream.destroy());
      res.on("close", () => {
        if (!res.writableEnded) upstream.destroy();
      });
      req.pipe(upstream);
      return;
    }

    if (req.method !== "GET" && req.method !== "HEAD")
      return res.writeHead(405, { Allow: "GET, HEAD" }).end();
    if (url.pathname === "/admin")
      return res.writeHead(308, { Location: `/admin/${url.search}` }).end();

    try {
      const pathname = decodeURIComponent(url.pathname);
      if (
        /[\\:\0]/.test(pathname) ||
        pathname.split("/").some((part) => part.startsWith(".")) ||
        pathname === "/__preview" ||
        pathname.startsWith("/__preview/")
      )
        return res.writeHead(404).end();
      const admin = pathname.startsWith("/admin/");
      const folder = folders[admin ? "admin" : "visitor"];
      const relative = pathname.slice(admin ? 7 : 1);
      const route = !extname(relative) && !relative.startsWith("assets/");
      const file = await realpath(resolve(folder, route ? "index.html" : relative));
      if (!file.startsWith(folder + sep)) return res.writeHead(404).end();
      const data = await readFile(file);
      const cookies = dockerGateway && admin && route && req.method === "GET"
        ? await initializeStaff(api, req, pathname, dockerGateway).catch(() => null)
        : [];
      if (cookies === null) return unavailable(res);
      res.writeHead(200, {
        "Content-Type": types[extname(file)] || "application/octet-stream",
        "Content-Length": data.length,
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
        ...(cookies.length ? { "Set-Cookie": cookies } : {}),
      });
      res.end(req.method === "HEAD" ? undefined : data);
    } catch {
      res.writeHead(404).end();
    }
  });
  await new Promise((accept, reject) => {
    server.once("error", reject);
    server.listen(port, dockerGateway ? "0.0.0.0" : "127.0.0.1", accept);
  });
  return {
    origin: publicSite?.origin ?? `http://127.0.0.1:${server.address().port}`,
    port: server.address().port,
    async close() {
      server.closeAllConnections();
      await new Promise((accept, reject) =>
        server.close((error) => (error ? reject(error) : accept())),
      );
    },
  };
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  try {
    const { values } = parseArgs({
      options: { api: { type: "string" }, port: { type: "string", default: "3000" } },
    });
    const dockerGateway = process.env.VMS_DOCKER_SITE === "1"
      ? (await readFile("/run/secrets/gateway_init_key", "utf8")).trim()
      : undefined;
    if (process.env.VMS_DOCKER_SITE === "1" && !dockerGateway)
      throw new Error("The Docker staff gateway key is missing.");
    const port = dockerGateway ? 8180 : Number(values.port);
    const dockerApiHost = process.env.VMS_DOCKER_API_HOST || "127.0.0.1";
    const apiOrigin = dockerGateway ? `http://${dockerApiHost}:${process.env.APP_PORT || "8080"}` : values.api;
    if (!apiOrigin || !Number.isInteger(port) || port < 1024 || port > 65535)
      throw new Error("Use npm run serve:backend -- --api http://127.0.0.1:PORT [--port 3000].");
    const site = await startBackendSite({ apiOrigin, port, dockerGateway, siteOrigin: process.env.VMS_SITE_ORIGIN });
    console.log(`Real backend site: ${site.origin}/ (staff: ${site.origin}/admin/)`);
    console.log(`API: ${new URL(apiOrigin).origin}`);
    let stopping = false;
    for (const signal of ["SIGINT", "SIGTERM"])
      process.on(signal, async () => {
        if (stopping) return;
        stopping = true;
        await site.close();
      });
  } catch {
    console.error(
      "Cannot start VMS frontend. Check the built apps, gateway origin, upstream and listen port. See backend/docs/LOCAL-DEVELOPMENT.md.",
    );
    process.exitCode = 1;
  }
}
