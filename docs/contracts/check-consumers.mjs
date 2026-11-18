// Runs the real consumer schemas and inventories request ASTs with existing TypeScript.
import { createRequire } from 'node:module';
import { readFileSync, readdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import * as wire from '../../frontendx/packages/data/src/wire.ts';
import { parseSession } from '../../frontendx/packages/data/src/session-contract.ts';
import { createHttpClient } from '../../frontendx/packages/data/src/http.ts';
import { summaryResponseSchema } from '../../frontendx/packages/data/src/reports.ts';
import { employeeListSchema } from '../../frontendx/packages/data/src/management-contract.ts';

const root = fileURLToPath(new URL('../../', import.meta.url));
const require = createRequire(path.join(root, 'frontendx/package.json'));
const ts = require('typescript');
const examples = JSON.parse(readFileSync(new URL('examples.json', import.meta.url), 'utf8'));
const cases = [];
const gaps = [];
function check(name, accepted, expected = true) {
  if (accepted !== expected) throw new Error(`Consumer regression: ${name}`);
  cases.push(name);
}
function knownGap(name, mismatch) {
  if (mismatch) gaps.push(name);
}
const schemas = examples.schemas;
check('public booking accepted', wire.publicBookingResponse.safeParse(schemas.PublicBooking).success);
check('standard staff booking accepted', wire.staffBookingResponse.safeParse(schemas.StandardStaffBooking).success);
check('restricted reception projection accepted', wire.staffBookingResponse.safeParse(schemas.RestrictedReception).success);
check('restricted reception identity leak rejected', wire.staffBookingResponse.safeParse({...schemas.RestrictedReception, name:'Private visitor'}).success, false);
check('bounded read accepted', wire.staffReadModelResponse.safeParse(schemas.ReadModel).success);
check('summary accepted', summaryResponseSchema.safeParse(schemas.Summary).success);
check('Phase 1 current-user core accepted by existing parser', parseSession(schemas.CurrentUser).role === schemas.CurrentUser.role);
check('ordinary compatible additive response accepted', wire.staffBookingResponse.safeParse({...schemas.StandardStaffBooking, serverLabel:'compatible addition'}).success);
knownGap('restricted-unknown-private-field-stripped', wire.staffBookingResponse.safeParse({...schemas.RestrictedReception, passwordHash:'must never reach this projection'}).success);
const errorsRequest = createHttpClient('/v1', async () => new Response(JSON.stringify(examples.errors['422']), {status:422,headers:{'Content-Type':'application/json'}}));
let receivedErrors;
try { await errorsRequest('/staff/example'); } catch (failure) { receivedErrors = failure.fieldErrors; }
check('selected field-error arrays survive HTTP parsing', receivedErrors?.name?.[0] === examples.errors['422'].fieldErrors.name[0]);
knownGap('catalog-room-objects', !wire.catalogsResponse.safeParse(schemas.Catalogs).success);
const legacyCatalog = { ...schemas.Catalogs, rooms: ['Meeting room'] };
const legacyResult = wire.catalogsResponse.safeParse(legacyCatalog);
knownGap('attachments-capability-stripped', legacyResult.success && !('attachmentsEnabled' in legacyResult.data));
knownGap('phase1-employee-page-parser', !employeeListSchema.safeParse(schemas.EmployeePage).success);
const sessionText = readFileSync(path.join(root, 'frontendx/packages/data/src/session.tsx'), 'utf8');
const sessionSource = ts.createSourceFile('session.tsx', sessionText, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
const signIn = sessionSource.statements.find(node => ts.isFunctionDeclaration(node) && node.name?.text === 'signIn');
if (!signIn?.body) throw new Error('Review the changed signIn boundary before updating its gap.');
knownGap('phase1-auth-result-unconsumed', signIn.body.getText(sessionSource).includes('"/staff/sessions"') && !signIn.body.getText(sessionSource).includes('accessToken'));
const requests = [];
const request = createHttpClient('/v1', async (url, init) => {
  requests.push({url, headers:init.headers, method:init.method});
  return new Response(JSON.stringify(schemas.CurrentUser), {status:200, headers:{'Content-Type':'application/json'}});
});
await request('/staff/auth/me', {authenticated:true});
await request('/staff/auth/login', {body:{email:'staff@example.test',password:'synthetic-password'}});
knownGap('authenticated-request-missing-bearer', !Object.keys(requests[0].headers).some(k => k.toLowerCase()==='authorization'));

const calls = [];
const files = [];
function walk(directory) {
  for (const entry of readdirSync(directory, {withFileTypes:true})) {
    if (entry.isDirectory()) walk(path.join(directory,entry.name));
    else if (/\.(tsx?|mjs)$/.test(entry.name)) files.push(path.join(directory,entry.name));
  }
}
for (const dir of ['frontendx/packages/data/src','frontendx/apps/admin/src','frontendx/apps/visitor/src']) walk(path.join(root,dir));
for (const file of files.sort()) {
  const text = readFileSync(file,'utf8');
  const source = ts.createSourceFile(file,text,ts.ScriptTarget.Latest,true,file.endsWith('tsx')?ts.ScriptKind.TSX:ts.ScriptKind.TS);
  function visit(node) {
    if (ts.isCallExpression(node) && ['request','fetch','fetcher'].includes(node.expression.getText(source))) {
      calls.push({file:path.relative(root,file).replaceAll('\\','/'),call:node.getText(source).replace(/\s+/g,' ').trim()});
    }
    ts.forEachChild(node,visit);
  }
  visit(source);
}
process.stdout.write(JSON.stringify({cases, knownGaps:gaps.sort(), requestCalls:calls},null,2)+'\n');
