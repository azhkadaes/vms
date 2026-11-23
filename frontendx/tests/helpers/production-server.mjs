import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { resolve, extname } from "node:path";
import { fileURLToPath } from "node:url";

export async function startProduction() {
  const root = fileURLToPath(new URL("../..", import.meta.url));
  const types = {
    ".html": "text/html",
    ".js": "text/javascript",
    ".css": "text/css",
    ".jpg": "image/jpeg",
    ".png": "image/png",
    ".webp": "image/webp",
    ".avif": "image/avif",
    ".woff2": "font/woff2",
  };
  const server = createServer(async (req, res) => {
    try {
      const pathname = new URL(req.url, "http://127.0.0.1").pathname;
      const admin = pathname.startsWith("/admin/");
      const folder = resolve(root, "apps", admin ? "admin" : "visitor", "dist");
      const relative = admin ? pathname.slice(7) : pathname.slice(1);
      const file = resolve(folder, relative.startsWith("assets/") ? relative : "index.html");
      if (!file.startsWith(folder + "/") && !file.startsWith(folder + "\\")) {
        res.writeHead(403).end();
        return;
      }
      const data = await readFile(file);
      res.writeHead(200, {
        "Content-Type": types[extname(file)] || "application/octet-stream",
        ...(admin && !relative.startsWith("assets/")
          ? { "Set-Cookie": "__Host-vms-csrf=fixture-csrf; Secure; Path=/; SameSite=Lax" }
          : {}),
      });
      res.end(data);
    } catch {
      res.writeHead(404).end();
    }
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  return {
    origin: `http://localhost:${server.address().port}`,
    async close() {
      server.closeAllConnections();
      await new Promise((resolve) => server.close(resolve));
    },
  };
}
