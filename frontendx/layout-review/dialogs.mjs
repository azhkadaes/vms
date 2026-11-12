import { chromium } from 'playwright';
import { writeFile } from 'node:fs/promises';
import { startPreview } from '../tests/helpers/preview-server.mjs';
const preview=await startPreview({time:'2026-11-02T10:00:00+08:00'});
const browser=await chromium.launch({channel:'chrome',headless:true});
const results=[];
for(const [name,role,path,action] of [
 ['appointment','PA','booking?new=1&kind=standard'],
 ['restricted','Protocol Officer','booking?new=1&kind=restricted'],
 ['detail','PA','booking?selected=demo-2'],
 ['employee','Super Admin','employees','add'],
 ['role','Super Admin','roles','role'],
 ['integration','Super Admin','integration','integration'],
 ['code','Receptionist','check-in','code'],
 ['create-menu','Protocol Officer','booking','menu'],
]){
 const context=await browser.newContext({viewport:{width:1440,height:900},reducedMotion:'reduce'});
 await context.addInitScript(role=>sessionStorage.setItem('vms-preview-role',role),role);
 const page=await context.newPage();page.setDefaultTimeout(8000);
 await page.route('**/__preview/**',async route=>{const response=await route.fetch({url:route.request().url().replace('http://127.0.0.1:3002/__preview',preview.base)});await route.fulfill({response});});
 try{
 await page.goto('http://127.0.0.1:3002/admin/'+path);
 await page.locator('h1').first().waitFor(); await page.waitForTimeout(600);
 if(action==='add')await page.getByRole('button',{name:'Tambah pegawai demo',exact:true}).click();
 if(action==='role')await page.getByRole('button',{name:'Edit akses PA',exact:true}).click();
 if(action==='integration')await page.getByRole('button',{name:/Konfigurasi /}).first().click();
 if(action==='code')await page.getByRole('button',{name:/Kode kunjungan/}).click();
 if(action==='menu')await page.locator('summary').click();
 if(action!=='menu')await page.getByRole('dialog').waitFor();
 for(const width of [1440,768,390,320]){
  await page.setViewportSize({width,height:900});await page.waitForTimeout(100);
  results.push({name,width,...await page.evaluate(()=>({docWidth:document.documentElement.scrollWidth,
   boxes:[...document.querySelectorAll('.modal,.modal-scroll,.appointment-type-options')].filter(e=>e.getClientRects().length).map(e=>({cls:e.className,w:e.clientWidth,scroll:e.scrollWidth})),
   controls:[...document.querySelectorAll('.modal input:not([type=hidden]),.modal select')].filter(e=>e.getClientRects().length).map(e=>({id:e.id,type:e.type,font:getComputedStyle(e).fontSize,w:Math.round(e.getBoundingClientRect().width)}))}))});
  if(width===320||width===1440)await page.screenshot({path:new URL(`${name}-${width}.png`,import.meta.url).pathname.replace(/^\//,''),fullPage:true});
 }
 if(name==='appointment'){
  await page.setViewportSize({width:320,height:900}); await page.getByRole('button',{name:/Tanggal.*Pilih tanggal/}).click();
  await page.locator('.shadcn-calendar').waitFor();
  await page.screenshot({path:new URL('calendar-320.png',import.meta.url).pathname.replace(/^\//,''),fullPage:false});
 }
 }catch(error){results.push({name,error:error.message});}
 await context.close();
}
// Direction and an effective 200% reflow width; actual browser zoom is recorded separately.
for(const dir of ['ltr','rtl']){
 const context=await browser.newContext({viewport:{width:720,height:450}});const p=await context.newPage();
 await p.route('**/__preview/**',async route=>{await route.fulfill({response:await route.fetch({url:route.request().url().replace('http://127.0.0.1:3002/__preview',preview.base)})});});
 await p.goto('http://127.0.0.1:3002/admin/booking');await p.locator('h1').waitFor();await p.waitForTimeout(500);
 await p.evaluate(dir=>document.documentElement.dir=dir,dir);
 await p.screenshot({path:new URL(`reflow-${dir}-720.png`,import.meta.url).pathname.replace(/^\//,''),fullPage:true});
 results.push({name:'reflow-'+dir,...await p.evaluate(()=>({width:innerWidth,docWidth:document.documentElement.scrollWidth}))});
 await context.close();
}
await writeFile(new URL('dialogs.json',import.meta.url),JSON.stringify(results,null,2));console.log(JSON.stringify(results,null,2));
await browser.close();await preview.close();
