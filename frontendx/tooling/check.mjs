import { spawn } from "node:child_process";

const npm = process.env.npm_execpath;
if (!npm) throw new Error("Run this check with npm run check.");
async function run(args) {
  await new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [npm, ...args], { stdio: "inherit", windowsHide: true });
    child.once("error", reject);
    child.once("exit", (code) =>
      code === 0 ? resolve() : reject(new Error(`Check failed: ${args.join(" ")}`)),
    );
  });
}
try {
  for (const script of [
    "format:check",
    "typecheck",
    "test",
    "test:preview",
    "test:backend-site",
    "build",
    "test:browser",
    "test:production",
  ])
    await run(["run", script]);
  await run(["audit", "--audit-level=high"]);
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
