import { createServer } from 'vite';
import { spawn } from 'node:child_process';
const servers = [];
try {
  for (const [app, port, proxy] of [
    ['visitor', 3130, { '/admin': { target: 'http://127.0.0.1:3132', ws: true } }],
    ['admin', 3132, { '/__preview': { target: 'http://127.0.0.1:3130' } }],
  ]) {
    const server = await createServer({ configFile: `apps/${app}/vite.config.ts`, server: {host:'127.0.0.1', port, strictPort:true, proxy} });
    servers.push(server);
    await server.listen();
  }
  const code = await new Promise((resolve, reject) => {
    const args = process.argv[2] ? [process.argv[2], 'run', 'check'] : process.env.VMS_AUDIT_TEST_PATTERN
      ? ['--experimental-strip-types', '--test', '--test-concurrency=1', '--test-name-pattern='+process.env.VMS_AUDIT_TEST_PATTERN, 'tests/accessibility.browser.mjs', 'tests/booking-refresh.browser.mjs']
      : ['tooling/browser-check.mjs'];
    const env = {...process.env, VMS_TEST_URL:'http://127.0.0.1:3130'};
    delete env.NODE_ENV;
    const child = spawn(process.execPath, args, {env, stdio:'inherit', windowsHide:true});
    child.once('error', reject);
    child.once('exit', resolve);
  });
  process.exitCode = code;
} finally {
  for (const server of servers.reverse()) await server.close();
}
