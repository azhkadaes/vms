import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
import { chromium } from 'playwright';
import { startPreview } from '../tests/helpers/preview-server.mjs';
const api=await startPreview({time:'2026-11-03T08:00:00+08:00'});
const browser=await chromium.launch({channel:'chrome',headless:true});
const out=new URL('./input-typography/',import.meta.url);await mkdir(out,{recursive:true});
const results=[];
const cases=[
 ['booking','Protocol Officer','booking'],['today','PA','today'],
 ['employees','Super Admin','employees'],['analytics','PA','analytics'],['reports','PA','logs'],
 ['appointment','PA','booking?new=1&kind=standard','dialog'],
 ['restricted','Protocol Officer','booking?new=1&kind=restricted','dialog'],
 ['employee-editor','Super Admin','employees','employee'],
 ['integration','Super Admin','integration','integration'],
 ['reception-code','Receptionist','check-in','code'],
];
try {
 for(const [name,role,path,action] of cases){
  const context=await browser.newContext({viewport:{width:1440,height:900},reducedMotion:'reduce'});
  await context.addInitScript(role=>sessionStorage.setItem('vms-preview-role',role),role);
  await context.route('**/__preview/**',async route=>route.fulfill({response:await route.fetch({url:route.request().url().replace('http://127.0.0.1:3002/__preview',api.base)})}));
  const page=await context.newPage();
  await page.goto('http://127.0.0.1:3002/admin/'+path);await page.locator('h1').waitFor();await page.waitForFunction(()=>!document.querySelector('.skeleton'));
  if(action==='employee')await page.getByRole('button',{name:'Tambah pegawai demo',exact:true}).click();
  if(action==='integration')await page.getByRole('button',{name:/Konfigurasi /}).first().click();
  if(action==='code')await page.getByRole('button',{name:/Kode kunjungan/}).click();
  if(action)await page.getByRole('dialog').waitFor();
  const scope=action?'.modal':'main';
  for(const width of [1440,1150,1000,768,390,320]){
   await page.setViewportSize({width,height:900});
   const data=await page.evaluate(scope=>{
    const controls=[...document.querySelectorAll(`${scope} input, ${scope} select, ${scope} textarea`)];
    const values=controls.filter(e=>e.getClientRects().length&&getComputedStyle(e).opacity!=='0'&&e.getBoundingClientRect().width>2&&!['hidden','checkbox','radio','range','color'].includes(e.type)).map(e=>({id:e.id,type:e.type,font:parseFloat(getComputedStyle(e).fontSize),left:e.getBoundingClientRect().left,right:e.getBoundingClientRect().right,container:e.closest('.booking-panel')?.clientWidth}));
    return {width:innerWidth,docWidth:document.documentElement.scrollWidth,controls:values,modals:[...document.querySelectorAll('.modal-scroll')].map(e=>({width:e.clientWidth,scroll:e.scrollWidth}))};
   },scope);
   assert.equal(data.docWidth,width,JSON.stringify({name,...data}));
   for(const field of data.controls){
    if(width<=1000||field.container<=900)assert.ok(field.font>=16,JSON.stringify({name,width,field}));
    assert.ok(field.left>=0&&field.right<=width+1,JSON.stringify({name,width,field}));
   }
   for(const m of data.modals)assert.ok(m.scroll<=m.width+1,JSON.stringify({name,width,m}));
   results.push({name,...data});
   if([1440,320].includes(width)&&['booking','employee-editor','restricted','integration','analytics'].includes(name))await page.screenshot({path:new URL(`${name}-${width}.png`,out).pathname.replace(/^\//,''),fullPage:false});
  }
  if(name==='booking'){
   const sort=page.getByRole('combobox',{name:'Urutan janji temu'});await sort.selectOption('name');
   await page.waitForFunction(()=>document.querySelector('#booking-sort').value==='name');
   await page.screenshot({path:new URL('sort-selected-320.png',out).pathname.replace(/^\//,''),fullPage:false});
  }
  if(name==='appointment'){
   await page.getByRole('button',{name:/Tanggal.*Pilih tanggal/}).click();await page.locator('.shadcn-calendar').waitFor();
   const caption=await page.locator('.rdp-caption_label').evaluateAll(nodes=>nodes.map(e=>({font:parseFloat(getComputedStyle(e).fontSize),w:e.clientWidth,scroll:e.scrollWidth})));
   for(const c of caption){assert.ok(c.font>=16);assert.ok(c.scroll<=c.w+1);}
   results.push({name:'calendar',width:320,captions:caption});
   await page.screenshot({path:new URL('calendar-320.png',out).pathname.replace(/^\//,''),fullPage:false});
  }
  await context.close();
 }
 await writeFile(new URL('measurements.json',out),JSON.stringify(results,null,2));
 console.log(`Passed ${results.length-1} page/dialog/width combinations plus calendar dropdown typography and containment.`);
 console.log('Desktop booking fonts: '+JSON.stringify(results.find(x=>x.name==='booking'&&x.width===1440).controls));
} finally {await browser.close();await api.close();}
