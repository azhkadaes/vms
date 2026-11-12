import { chromium } from 'playwright';
import { mkdir, writeFile } from 'node:fs/promises';
import { startPreview } from '../tests/helpers/preview-server.mjs';
const preview=await startPreview({time:'2026-11-02T10:00:00+08:00'});
const out = new URL('./', import.meta.url);
await mkdir(out, {recursive:true});
const browser = await chromium.launch({channel:'chrome',headless:true});
const results=[];
const cases=[['PA','dashboard'],['PA','today'],['PA','booking'],['Protocol Officer','booking'],['Receptionist','check-in'],['Receptionist','check-out'],['Super Admin','employees'],['Super Admin','roles'],['Super Admin','integration'],['PA','analytics'],['PA','logs']];
for(const [role,route] of cases){
 const context=await browser.newContext({viewport:{width:1440,height:900},reducedMotion:'reduce'});
 await context.addInitScript(role=>sessionStorage.setItem('vms-preview-role',role),role);
 const page=await context.newPage();
 await page.route('**/__preview/**',async route=>{
  const response=await route.fetch({url:route.request().url().replace('http://127.0.0.1:3002/__preview',preview.base)});
  await route.fulfill({response});
 });
 await page.goto('http://127.0.0.1:3002/admin/'+route);
 await page.locator('h1').first().waitFor();
 await page.waitForFunction(()=>!document.querySelector('.skeleton'));
 await page.waitForTimeout(400);
 for(const width of [1440,1280,1024,768,390,320]){
  await page.setViewportSize({width,height:900});
  const data=await page.evaluate(()=>{
   const visible=e=>!!e.getClientRects().length&&getComputedStyle(e).visibility!=='hidden';
   return {width:innerWidth,docWidth:document.documentElement.scrollWidth,heading:document.querySelector('h1')?.innerText,
   overflow:[...document.querySelectorAll('main *,header *,footer *')].filter(visible).filter(e=>{const r=e.getBoundingClientRect();return r.right>innerWidth+1||r.left< -1}).slice(0,16).map(e=>({tag:e.tagName,cls:e.className,text:e.innerText?.slice(0,65),x:Math.round(e.getBoundingClientRect().x),w:Math.round(e.getBoundingClientRect().width)})),
   scrollers:[...document.querySelectorAll('.table-wrap,.booking-tabs')].filter(visible).map(e=>({cls:e.className,w:e.clientWidth,scroll:e.scrollWidth})),
   controls:[...document.querySelectorAll('main input:not([type=hidden]),main select,main .date-picker-trigger')].filter(visible).map(e=>({tag:e.tagName,id:e.id,type:e.type,font:getComputedStyle(e).fontSize,w:Math.round(e.getBoundingClientRect().width),value:e.value||e.innerText})),
   clipped:[...document.querySelectorAll('main *')].filter(visible).filter(e=>e.childElementCount===0&&e.scrollWidth>e.clientWidth+2&&['hidden','clip'].includes(getComputedStyle(e).overflowX)).map(e=>({tag:e.tagName,cls:e.className,text:e.innerText}))};
  });
  results.push({role,route,...data});
  if([1440,768,320].includes(width)) await page.screenshot({path:new URL(`${route}-${role.replaceAll(' ','-')}-${width}.png`,out).pathname.replace(/^\//,''),fullPage:true});
 }
 await context.close();
}
await writeFile(new URL('results.json',out),JSON.stringify(results,null,2));
console.log(JSON.stringify(results.filter(r=>r.docWidth>r.width||r.clipped.length).map(r=>({role:r.role,route:r.route,width:r.width,docWidth:r.docWidth,overflow:r.overflow,clipped:r.clipped})),null,2));
await browser.close();
await preview.close();
