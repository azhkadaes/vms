import { chromium } from 'playwright';
import { writeFile } from 'node:fs/promises';
import { startPreview } from '../tests/helpers/preview-server.mjs';
const api=await startPreview();const browser=await chromium.launch({channel:'chrome',headless:true});const checks=[];
const context=await browser.newContext({viewport:{width:1440,height:900},reducedMotion:'reduce'});await context.addInitScript(()=>sessionStorage.setItem('vms-preview-role','PA'));
const page=await context.newPage();await page.route('**/__preview/**',async route=>route.fulfill({response:await route.fetch({url:route.request().url().replace('http://127.0.0.1:3002/__preview',api.base)})}));
for(const mode of ['zoom','growth','rtl']){
 await page.goto('http://127.0.0.1:3002/admin/booking');await page.locator('h1').waitFor();await page.waitForFunction(()=>!document.querySelector('.skeleton'));
 if(mode==='zoom')await page.evaluate(()=>document.documentElement.style.zoom='2');
 if(mode==='growth')await page.evaluate(()=>{
  document.querySelector('h1').textContent='Verwaltung der Besuchstermine';
  document.querySelector('.booking-intro-actions button').lastChild.textContent='Besprechungstermin vereinbaren';
  document.querySelectorAll('.booking-tabs button').forEach(e=>e.prepend('Besuchstermin '));
 });
 if(mode==='rtl')await page.evaluate(()=>document.documentElement.dir='rtl');
 checks.push({mode,...await page.evaluate(()=>({docWidth:document.documentElement.scrollWidth,width:innerWidth,body:document.querySelector('.admin-body').getBoundingClientRect().toJSON(),sidebar:document.querySelector('.sidebar')?.getBoundingClientRect().toJSON()}))});
 await page.screenshot({path:new URL(`stress-${mode}.png`,import.meta.url).pathname.replace(/^\//,''),fullPage:false});
}
await page.goto('http://127.0.0.1:3002/admin/booking?new=1&kind=standard');await page.getByRole('dialog').waitFor();await page.setViewportSize({width:320,height:900});
await page.getByRole('button',{name:/Waktu mulai.*Pilih waktu/}).click();await page.locator('.time-picker-popover').waitFor();
checks.push({mode:'time-picker',...await page.locator('.time-picker-popover').evaluate(e=>({w:e.clientWidth,scroll:e.scrollWidth,rect:e.getBoundingClientRect().toJSON()}))});
await page.screenshot({path:new URL('time-picker-320.png',import.meta.url).pathname.replace(/^\//,''),fullPage:false});
await writeFile(new URL('stress.json',import.meta.url),JSON.stringify(checks,null,2));console.log(JSON.stringify(checks));await browser.close();await api.close();
