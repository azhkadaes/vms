import { pathToFileURL } from "node:url";
import { createServer } from "node:http";
import { readFile, mkdir, writeFile } from "node:fs/promises";
import { resolve, extname } from "node:path";
const toolsRoot = process.env.LIGHTHOUSE_ROOT || ".task-cache/audit-tools/node_modules";
const loadTool = (path) => import(pathToFileURL(resolve(toolsRoot, path)).href);
const [{ default: lighthouse }, { default: desktop }, { launch }] = await Promise.all([
  loadTool("lighthouse/core/index.js"),
  loadTool("lighthouse/core/config/desktop-config.js"),
  loadTool("chrome-launcher/dist/index.js"),
]);
const phase = process.argv[2] || "measurement";
if (!/^[a-z0-9-]+$/.test(phase)) throw new Error("Use a simple name for the evidence folder.");
const output = resolve("anti-slop/evidence-2026-11-02/visitor-performance", phase);
await mkdir(output, { recursive: true });
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
const root = resolve("apps/visitor/dist");
const server = createServer(async (req, res) => {
  try {
    const path = new URL(req.url, "http://127.0.0.1").pathname;
    if (path === "/v1/public/catalogs") {
      res.writeHead(200, { "Content-Type": "application/json" }).end(
        JSON.stringify({
          officers: [{ id: "fixture-officer", name: "Fixture officer", vip: false, active: true }],
          rooms: ["Fixture room"],
          privacyNotice: {
            version: "fixture-1",
            id: "Pemberitahuan privasi contoh untuk pengujian.",
            en: "Example privacy notice for contract testing.",
          },
        }),
      );
      return;
    }
    if (path.startsWith("/v1/")) {
      res.writeHead(404, { "Content-Type": "application/json" }).end('{"message":"Not found"}');
      return;
    }
    const file = resolve(root, path.startsWith("/assets/") ? path.slice(1) : "index.html");
    if (!file.startsWith(root + "\\") && !file.startsWith(root + "/")) {
      res.writeHead(403).end();
      return;
    }
    res
      .writeHead(200, {
        "Content-Type": types[extname(file)] || "application/octet-stream",
        "Cache-Control": "no-store",
      })
      .end(await readFile(file));
  } catch {
    res.writeHead(404).end();
  }
});
await new Promise((r) => server.listen(0, "127.0.0.1", r));
const summaries = [];
try {
  for (const profile of ["desktop", "mobile"])
    for (let run = 1; run <= 3; run++) {
      const chrome = await launch({
        chromeFlags: ["--headless=new", "--disable-extensions", "--no-first-run"],
      });
      try {
        const result = await lighthouse(
          `http://127.0.0.1:${server.address().port}/`,
          {
            port: chrome.port,
            logLevel: "error",
            output: ["json", "html"],
            onlyCategories: ["performance", "accessibility", "best-practices"],
          },
          profile === "desktop" ? desktop : undefined,
        );
        const name = `${profile}-${run}`;
        await writeFile(resolve(output, name + ".json"), result.report[0]);
        await writeFile(resolve(output, name + ".html"), result.report[1]);
        await writeFile(
          resolve(output, name + ".trace.json"),
          JSON.stringify(result.artifacts.Trace),
        );
        const lhr = result.lhr;
        const summary = {
          profile,
          run,
          version: lhr.lighthouseVersion,
          browser: lhr.environment.hostUserAgent,
          settings: lhr.configSettings,
          score: 100 * lhr.categories.performance.score,
          accessibility: 100 * lhr.categories.accessibility.score,
          bestPractices: 100 * lhr.categories["best-practices"].score,
          metrics: Object.fromEntries(
            [
              "first-contentful-paint",
              "largest-contentful-paint",
              "total-blocking-time",
              "cumulative-layout-shift",
              "speed-index",
            ].map((k) => [k, lhr.audits[k].numericValue]),
          ),
          warnings: lhr.runWarnings,
        };
        summaries.push(summary);
        console.log(JSON.stringify({ phase, ...summary, settings: undefined }));
      } finally {
        await chrome.kill();
      }
    }
  await writeFile(resolve(output, "summary.json"), JSON.stringify(summaries, null, 2));
} finally {
  server.closeAllConnections();
  await new Promise((r) => server.close(r));
}
