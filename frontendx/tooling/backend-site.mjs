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

export async function startBackendSite({ apiOrigin, port = 3000, root = workspace }) {
  const api = new URL(apiOrigin);
  if (
    api.protocol !== "http:" ||
    api.hostname !== "127.0.0.1" ||
    !api.port ||
    api.username ||
    api.password ||
    api.pathname !== "/" ||
    api.search ||
    api.hash
  )
    throw new Error("Select an HTTP API at 127.0.0.1 with an explicit port and no credentials.");

  const folders = {};
  for (const app of ["visitor", "admin"]) {
    folders[app] = await realpath(resolve(root, "apps", app, "dist"));
    await readFile(resolve(folders[app], "index.html"));
  }

  const server = createServer(async (req, res) => {
    const host = `127.0.0.1:${server.address().port}`;
    if (req.headers.host !== host) return res.writeHead(403).end();
    if (!req.url.startsWith("/") || req.url.startsWith("//")) return res.writeHead(400).end();

    const url = new URL(req.url, `http://${host}`);
    if (url.pathname === "/v1" || url.pathname.startsWith("/v1/")) {
      const upstream = request(
        api,
        { path: req.url, method: req.method, headers: forwardedHeaders(req.headers) },
        (reply) => {
          res.writeHead(reply.statusCode || 502, forwardedHeaders(reply.headers));
          reply.on("error", () => res.destroy());
          reply.pipe(res);
        },
      );
      upstream.setTimeout(20_000, () => upstream.destroy());
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
      res.writeHead(200, {
        "Content-Type": types[extname(file)] || "application/octet-stream",
        "Content-Length": data.length,
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      });
      res.end(req.method === "HEAD" ? undefined : data);
    } catch {
      res.writeHead(404).end();
    }
  });
  await new Promise((accept, reject) => {
    server.once("error", reject);
    server.listen(port, "127.0.0.1", accept);
  });
  return {
    origin: `http://127.0.0.1:${server.address().port}`,
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
    const port = Number(values.port);
    if (!values.api || !Number.isInteger(port) || port < 1024 || port > 65535)
      throw new Error("Use npm run serve:backend -- --api http://127.0.0.1:PORT [--port 3000].");
    const site = await startBackendSite({ apiOrigin: values.api, port });
    console.log(`Real backend site: ${site.origin}/ (staff: ${site.origin}/admin/)`);
    console.log(`API: ${new URL(values.api).origin}; configure its VMS_ORIGIN as ${site.origin}.`);
    console.log("Actions use the selected backend. This launcher does not start or migrate it.");
    let stopping = false;
    for (const signal of ["SIGINT", "SIGTERM"])
      process.on(signal, async () => {
        if (stopping) return;
        stopping = true;
        await site.close();
      });
  } catch {
    console.error(
      "Cannot start the local site. Build both apps, select a loopback API, and use a free port. See REAL-BACKEND-LOCAL.md.",
    );
    process.exitCode = 1;
  }
}
