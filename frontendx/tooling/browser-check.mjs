import { spawn } from "node:child_process";
import { readdir } from "node:fs/promises";
import { createServer } from "node:net";
import { setTimeout as delay } from "node:timers/promises";

const children = [];
function run(args) {
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, args, { stdio: "inherit", windowsHide: true });
    child.once("error", reject);
    child.once("exit", (code) =>
      code === 0 ? resolve() : reject(new Error(`Check exited with ${code}`)),
    );
  });
}
async function available(port) {
  const probe = createServer();
  await new Promise((resolve, reject) => {
    probe.once("error", reject);
    probe.listen(port, "127.0.0.1", resolve);
  });
  await new Promise((resolve) => probe.close(resolve));
}
try {
  if (!process.env.VMS_TEST_URL) {
    for (const port of [3000, 3002]) await available(port);
    for (const app of ["visitor", "admin"])
      children.push(
        spawn(
          process.execPath,
          [
            "node_modules/vite/bin/vite.js",
            "--config",
            `apps/${app}/vite.config.ts`,
            "--host",
            "127.0.0.1",
            "--port",
            app === "visitor" ? "3000" : "3002",
            "--strictPort",
          ],
          { stdio: "inherit", windowsHide: true },
        ),
      );
    let ready = false;
    for (let attempt = 0; attempt < 100; attempt++) {
      ready = await Promise.all(
        ["http://127.0.0.1:3000/", "http://127.0.0.1:3002/admin/"].map((url) =>
          fetch(url, { signal: AbortSignal.timeout(1000) })
            .then((response) => response.ok)
            .catch(() => false),
        ),
      ).then((values) => values.every(Boolean));
      if (ready) break;
      await delay(100);
    }
    if (!ready) throw new Error("Preview did not become ready.");
  }
  const files = (await readdir("tests"))
    .filter((file) => file.endsWith(".browser.mjs") && file !== "production.browser.mjs")
    .sort()
    .map((file) => `tests/${file}`);
  await run(["--experimental-strip-types", "--test", "--test-concurrency=1", ...files]);
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
} finally {
  for (const child of children) child.kill();
}
