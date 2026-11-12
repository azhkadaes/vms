import { spawn } from "node:child_process";
const children = ["visitor", "admin"].map((app) =>
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
    { cwd: process.cwd(), stdio: "inherit", windowsHide: true },
  ),
);
let stopping = false;
function stop() {
  if (stopping) return;
  stopping = true;
  for (const child of children) child.kill();
}
for (const child of children) {
  child.on("error", stop);
  child.on("exit", (code) => {
    if (!stopping) {
      process.exitCode = code || 1;
      stop();
    }
  });
}
process.on("SIGINT", stop);
process.on("SIGTERM", stop);
