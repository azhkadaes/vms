// Runs the real frontend parsers and HTTP client against the source baseline's implemented
// response shapes, and inventories every frontend request call site with TypeScript's parser.
// A mismatch the frontend adapter has not fixed yet is reported as a known gap, not hidden.
import { createRequire } from 'node:module';
import { summaryExportSchema, summaryResponseSchema } from '../../frontendx/packages/data/src/reports.ts';
import { visitLogResponseSchema, visitLogSnapshotSchema } from '../../frontendx/packages/data/src/visit-log.ts';
import { readFileSync, readdirSync } from 'node:fs';
import { fileURLToPath, pathToFileURL } from 'node:url';
import path from 'node:path';
import * as wire from '../../frontendx/packages/data/src/wire.ts';
import { parseSession } from '../../frontendx/packages/data/src/session-contract.ts';
import { roleDefinitionSchema } from '../../frontendx/packages/data/src/management-contract.ts';
import { createHttpClient, onAuthenticationLoss } from '../../frontendx/packages/data/src/http.ts';

const root = fileURLToPath(new URL('../../', import.meta.url));
const require = createRequire(path.join(root, 'frontendx/package.json'));
const ts = require('typescript');
// The baseline defaults to the one beside this script; check.py passes a draft copy's path.
const baseline = process.argv[2] ? pathToFileURL(path.resolve(process.argv[2])) : new URL('source-baseline.json', import.meta.url);
const spec = JSON.parse(readFileSync(baseline, 'utf8')).spec;
const cases = [];
const gaps = [];
function check(name, accepted, expected = true) {
  if (accepted !== expected) throw new Error(`Consumer regression: ${name}`);
  cases.push(name);
}
function knownGap(name, mismatch) {
  if (mismatch) gaps.push(name);
}
function success(route, method) {
  const responses = spec.paths[route][method].responses;
  const status = Object.keys(responses).find((code) => code.startsWith('2'));
  return responses[status].content['application/json'].example;
}
function failure(code) {
  const status = spec['x-error-catalogue'][code].status;
  return spec.components.responses[`Error${status}`].content['application/json'].examples[code].value;
}
function throws(fn) {
  try { fn(); return false; } catch { return true; }
}
function respond(status, body) {
  return createHttpClient('/v1', async () => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }));
}
async function rejection(request) {
  try { await request; } catch (error) { return error; }
  return null;
}

// Ops 01 and 08 share the frontend catalog parser; officers and the privacy notice parse; rooms are objects on the server (KD-14).
const catalogs = success('/public/catalogs', 'get');
const staffCatalogs = success('/staff/catalogs', 'get');
check('catalog officers and privacy notice accepted', wire.catalogsResponse.safeParse({ ...catalogs, rooms: [] }).success);
check('staff catalog officers with organization and inactive assignments accepted', wire.catalogsResponse.safeParse({ ...staffCatalogs, rooms: [] }).success);
knownGap('catalog-room-objects', !wire.catalogsResponse.safeParse(catalogs).success || !wire.catalogsResponse.safeParse(staffCatalogs).success);

// Ops 32-34, 36, 39, 40: Session and TokenResult.
const session = success('/staff/session', 'get');
check('current session accepted by the session parser', parseSession(session).role === session.role);
const token = success('/staff/sessions', 'post');
check('token result session accepted by the session parser', parseSession(token.session).id === token.session.id);
const renewed = success('/staff/session/renew', 'post');
check('renewed session accepted by the session parser', parseSession(renewed).id === renewed.id);
knownGap('role-token-result-parsed-as-session', throws(() => parseSession(success('/staff/session/role', 'post'))));
const sessionText = readFileSync(path.join(root, 'frontendx/packages/data/src/session.tsx'), 'utf8');
const sessionSource = ts.createSourceFile('session.tsx', sessionText, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
const signIn = sessionSource.statements.find((node) => ts.isFunctionDeclaration(node) && node.name?.text === 'signIn');
if (!signIn?.body) throw new Error('Review the changed signIn boundary before updating its gap.');
knownGap('login-token-result-unconsumed', !signIn.body.getText(sessionSource).includes('accessToken'));

check('role definitions accepted by the role parser', roleDefinitionSchema.array().safeParse(success('/staff/roles', 'get')).success);
check('updated role definition accepted by the role parser', roleDefinitionSchema.safeParse(success('/staff/roles/{id}', 'patch')).success);

// Error body: catalogue message and field errors reach the form; a wrong current
// password (422) keeps the session while an expired one (401) signs out.
const validation = failure('validation_failed');
const fieldFailure = await rejection(respond(422, validation)('/staff/password', { body: {}, authenticated: true }));
check('catalogue message and field errors reach the form', fieldFailure?.message === validation.message && fieldFailure?.fieldErrors?.newPassword?.[0] === validation.fieldErrors.newPassword[0]);
let signedOut = 0;
const stop = onAuthenticationLoss(() => { signedOut += 1; });
await rejection(respond(422, { ...validation, fieldErrors: { currentPassword: ['Kata sandi saat ini tidak sesuai.'] } })('/staff/password', { body: {}, authenticated: true }));
check('wrong current password keeps the session', signedOut === 0);
await rejection(respond(401, failure('unauthenticated'))('/staff/session', { authenticated: true }));
check('expired session signs out', signedOut === 1);
stop();

// Staff calls must carry Authorization: Bearer (decided 16 November 2026).
const sent = [];
const recording = createHttpClient('/v1', async (url, init) => {
  sent.push(init.headers);
  return new Response(JSON.stringify(session), { status: 200, headers: { 'Content-Type': 'application/json' } });
});
await recording('/staff/session', { authenticated: true });
knownGap('authenticated-request-missing-bearer', !Object.keys(sent[0]).some((key) => key.toLowerCase() === 'authorization'));

if (spec.paths['/staff/reports/summary']) {
  check('summary report passes strict parser', summaryResponseSchema.safeParse(success('/staff/reports/summary', 'get')).success);
  check('CSV report export passes strict parser', summaryExportSchema.safeParse(success('/staff/reports/export', 'get')).success);
  check('visit log passes strict parser', visitLogResponseSchema.safeParse(success('/staff/reports/visits', 'get')).success);
  check('visit log export passes strict parser', visitLogSnapshotSchema.safeParse(success('/staff/reports/visits/export', 'get')).success);
}

const calls = [];
const files = [];
function walk(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    if (entry.isDirectory()) walk(path.join(directory, entry.name));
    else if (/\.(tsx?|mjs)$/.test(entry.name)) files.push(path.join(directory, entry.name));
  }
}
for (const dir of ['frontendx/packages/data/src', 'frontendx/apps/admin/src', 'frontendx/apps/visitor/src']) walk(path.join(root, dir));
for (const file of files.sort()) {
  const text = readFileSync(file, 'utf8');
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, file.endsWith('tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS);
  function visit(node) {
    if (ts.isCallExpression(node) && ['request', 'fetch', 'fetcher'].includes(node.expression.getText(source))) {
      calls.push({ file: path.relative(root, file).replaceAll('\\', '/'), call: node.getText(source).replace(/\s+/g, ' ').trim() });
    }
    ts.forEachChild(node, visit);
  }
  visit(source);
}
process.stdout.write(JSON.stringify({ cases, knownGaps: gaps.sort(), requestCalls: calls }, null, 2) + '\n');
