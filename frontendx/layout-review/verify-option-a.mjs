import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
import { chromium } from 'playwright';
import { startPreview } from '../tests/helpers/preview-server.mjs';

const api=await startPreview({time:'2026-11-03T08:00:00+08:00'});
const browser=await chromium.launch({channel:'chrome',headless:true});
const evidence=new URL('./option-a/',import.meta.url);
await mkdir(evidence,{recursive:true});
const results=[];
try {
 for(const role of ['PA','Protocol Officer','Receptionist']) {
  for(const expanded of [true,false]) {
   const context=await browser.newContext({viewport:{width:1440,height:900},reducedMotion:'reduce'});
   await context.addInitScript(({role,expanded})=>{
    sessionStorage.setItem('vms-preview-role',role);
    localStorage.setItem('vms-backoffice-sidebar-expanded',String(expanded));
   },{role,expanded});
   await context.route('**/__preview/**',async route=>route.fulfill({response:await route.fetch({url:route.request().url().replace('http://127.0.0.1:3002/__preview',api.base)})}));
   const page=await context.newPage();
   for(const route of ['booking','today']) {
    await page.goto('http://127.0.0.1:3002/admin/'+route);
    await page.locator('h1').waitFor();
    await page.waitForFunction(()=>!document.querySelector('.skeleton'));
    for(const width of [1440,1280,1150,1024,900,768,390,320]) {
     await page.setViewportSize({width,height:900});
     const state=await page.evaluate(()=>{
      const panel=document.querySelector('.booking-panel');
      const visible=e=>!!e?.getClientRects().length;
      return {width:innerWidth,panelWidth:panel.clientWidth,docWidth:document.documentElement.scrollWidth,
       cards:visible(document.querySelector('.booking-mobile-list')),
       table:visible(document.querySelector('.booking-desktop-table')),
       tabs:visible(document.querySelector('.booking-tabs')),
       status:visible(document.querySelector('.mobile-status-filter')),
       scrollers:[...document.querySelectorAll('.booking-desktop-table,.booking-tabs')].filter(visible).map(e=>({w:e.clientWidth,scroll:e.scrollWidth})),
       clippedActions:[...document.querySelectorAll('.booking-mobile-item .button')].filter(visible).filter(e=>{const r=e.getBoundingClientRect();return r.left<0||r.right>innerWidth+1||e.scrollWidth>e.clientWidth+1}).length};
     });
     assert.equal(state.docWidth,state.width,`${role} ${route} document width`);
     assert.equal(state.cards,state.panelWidth-2<=900,JSON.stringify(state));
     assert.equal(state.table,!state.cards);
     assert.equal(state.tabs,!state.cards);
     assert.equal(state.status,state.cards);
     assert.equal(state.clippedActions,0);
     for(const s of state.scrollers) assert.ok(s.scroll<=s.w+1,JSON.stringify({role,route,...state}));
     results.push({role,route,expanded,...state});
     if(role==='PA'&&route==='booking'&&[1280,1024,768,320].includes(width))await page.screenshot({path:new URL(`booking-${width}-${expanded?'expanded':'collapsed'}.png`,evidence).pathname.replace(/^\//,''),fullPage:width!==320});
    }
   }
   // A sidebar change at the same viewport must change the list presentation.
   await page.setViewportSize({width:1150,height:900});
   await page.getByRole('separator',{name:'Ukuran navigasi'}).press('End');
   await page.locator('.booking-mobile-list').waitFor({state:'visible'});
   await page.getByRole('separator',{name:'Ukuran navigasi'}).press('Home');
   await page.locator('.booking-desktop-table').waitFor({state:'visible'});
   // Exercise controls and opening details through the newly visible cards.
   await page.setViewportSize({width:768,height:900});
   await page.goto('http://127.0.0.1:3002/admin/booking');
   await page.getByRole('combobox',{name:'Status janji temu'}).waitFor();
   await page.waitForFunction(()=>!document.querySelector('.skeleton'));
   const next=page.getByRole('button',{name:'Halaman berikutnya',exact:true});
   if(await next.isEnabled()) {
    await next.click();
    assert.equal(new URL(page.url()).searchParams.get('page'),'2');
    await page.getByRole('button',{name:'Halaman sebelumnya',exact:true}).click();
    assert.equal(new URL(page.url()).searchParams.get('page'),'1');
   }
   await page.getByRole('combobox',{name:'Status janji temu'}).selectOption('approved');
   assert.equal(new URL(page.url()).searchParams.get('status'),'approved');
   await page.getByRole('combobox',{name:'Urutan janji temu'}).selectOption('name');
   assert.equal(new URL(page.url()).searchParams.get('sort'),'name');
   await page.getByRole('textbox',{name:'Cari janji temu'}).fill('Salsabila');
   await page.waitForFunction(()=>document.querySelectorAll('.booking-mobile-item').length===1);
   await page.getByRole('button',{name:/Lihat detail/}).filter({visible:true}).first().click();
   await page.getByRole('dialog').waitFor();
   assert.ok(new URL(page.url()).searchParams.has('selected'));
   await context.close();
  }
 }
 await writeFile(new URL('measurements.json',evidence),JSON.stringify(results,null,2));
 console.log(`Passed ${results.length} role/page/width/sidebar combinations; search, status, sort and detail controls passed for each role/sidebar combination.`);
} finally {
 await browser.close();
 await api.close();
}
