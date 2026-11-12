import { startProduction } from '../../tests/helpers/production-server.mjs';
import { writeFile } from 'node:fs/promises';
import lighthouse from '../../.task-cache/audit-tools/node_modules/lighthouse/core/index.js';
import { chromium } from 'playwright';
const server = await startProduction();
let chrome;
try {
  chrome = await chromium.launch({channel:'chrome',headless:true,args:['--remote-debugging-port=9333']});
  const result = await lighthouse(server.origin+'/', {hostname:'127.0.0.1',port:9333,logLevel:'error',output:['json','html'],onlyCategories:['accessibility','performance','best-practices']});
  await writeFile(new URL('./lighthouse-mobile.json', import.meta.url), result.report[0]);
  await writeFile(new URL('./lighthouse-mobile.html', import.meta.url), result.report[1]);
  console.log(JSON.stringify({categories:Object.fromEntries(Object.entries(result.lhr.categories).map(([k,v])=>[k,v.score*100])),metrics:Object.fromEntries(['largest-contentful-paint','total-blocking-time','cumulative-layout-shift'].map(k=>[k,result.lhr.audits[k].numericValue])),failed:Object.values(result.lhr.audits).filter(a=>a.score===0).map(a=>({id:a.id,title:a.title,details:a.details})),warnings:result.lhr.runWarnings},null,2));
} finally {
  await chrome?.close();
  await server.close();
}
