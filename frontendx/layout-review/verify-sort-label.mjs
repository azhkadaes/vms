import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
import { chromium } from 'playwright';
import { startPreview } from '../tests/helpers/preview-server.mjs';

const api=await startPreview({time:'2026-11-03T08:00:00+08:00'});
const browser=await chromium.launch({channel:'chrome',headless:true});
const out=new URL('./sort-label/',import.meta.url);await mkdir(out,{recursive:true});
const checks=[];
try {
 for(const role of ['PA','Protocol Officer','Receptionist']){
  const context=await browser.newContext({viewport:{width:1440,height:900},reducedMotion:'reduce'});
  await context.addInitScript(role=>sessionStorage.setItem('vms-preview-role',role),role);
  await context.route('**/__preview/**',async route=>route.fulfill({response:await route.fetch({url:route.request().url().replace('http://127.0.0.1:3002/__preview',api.base)})}));
  const page=await context.newPage();
  for(const route of ['booking','today']){
   await page.goto('http://127.0.0.1:3002/admin/'+route);
   await page.locator('h1').waitFor();await page.waitForFunction(()=>!document.querySelector('.skeleton'));
   const select=page.getByRole('combobox',{name:'Urutan janji temu'});
   assert.equal(await select.locator('option:checked').textContent(),'Urutkan');
   for(const width of [1440,1280,1024,768,390,320]){
    await page.setViewportSize({width,height:900});
    const check=await page.evaluate(()=>{
     const select=document.querySelector('#booking-sort'),label=document.querySelector('label[for="booking-sort"]'),search=document.querySelector('.search-field input');
     const s=select.getBoundingClientRect(),l=label.getBoundingClientRect(),q=search.getBoundingClientRect();
     const visible=e=>!!e.getClientRects().length;
     return {width:innerWidth,docWidth:document.documentElement.scrollWidth,label:label.textContent,labelAbove:l.bottom<=s.top,labelAligned:Math.abs(l.left-s.left)<1,sortX:s.x,sortRight:s.right,sortOverflow:select.scrollWidth>select.clientWidth+1,sameRow:Math.abs(q.top-s.top)<10,bottomDifference:Math.abs(q.bottom-s.bottom),
     overflowing:[...document.querySelectorAll('.booking-toolbar *')].filter(visible).filter(e=>{const r=e.getBoundingClientRect();return r.left<0||r.right>innerWidth+1}).length};
    });
    assert.equal(check.docWidth,width);assert.equal(check.label,'Urutkan');assert.ok(check.labelAbove);assert.ok(check.labelAligned);assert.equal(check.overflowing,0);assert.equal(check.sortOverflow,false);
    if(check.sameRow)assert.ok(check.bottomDifference<1);
    checks.push({role,route,...check});
    if(role==='Protocol Officer'&&route==='booking'&&[1440,768,320].includes(width))await page.screenshot({path:new URL(`booking-${width}.png`,out).pathname.replace(/^\//,''),fullPage:false});
   }
   await select.selectOption('name');
   await page.waitForFunction(()=>document.querySelector('#booking-sort').value==='name'&&new URL(location.href).searchParams.get('sort')==='name');
   assert.equal(await page.locator('label[for="booking-sort"]').innerText(),'Urutkan');
   assert.equal(new URL(page.url()).searchParams.get('sort'),'name');
   await select.selectOption('');
   await page.waitForFunction(()=>document.querySelector('#booking-sort').value===''&&!new URL(location.href).searchParams.has('sort')&&document.querySelector('#booking-sort').selectedOptions[0]?.textContent==='Urutkan');
  }
  await context.close();
 }
 await writeFile(new URL('measurements.json',out),JSON.stringify(checks,null,2));
 console.log(`Passed ${checks.length} role/page/width combinations: default text, visible label, alignment, overflow and sort/restore.`);
} finally {await browser.close();await api.close();}
