import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';
import { writeFile } from 'node:fs/promises';
import { startProduction } from '../../tests/helpers/production-server.mjs';
const server = await startProduction();
const browser = await chromium.launch({channel:'chrome',headless:true});
const out = new URL('./', import.meta.url);
const results = [];
const catalog = {officers:[{id:'o2',name:'Pejabat Contoh',vip:false,active:true}],rooms:['Ruang Contoh'],privacyNotice:{version:'test-1',id:'Pemberitahuan privasi contoh.',en:'Example privacy notice.'}};
async function fixture(page) {
  await page.route('**/v1/**', route => {
    const path = new URL(route.request().url()).pathname;
    if(path.endsWith('/catalogs')) return route.fulfill({json:catalog});
    if(path==='/v1/staff/session') return route.fulfill({json:{active:true,id:'staff-test',name:'Pegawai Contoh',role:'PA',assignedOfficerIds:['o2'],permissions:['bookings','create','approve','reject','cancel']}});
    if(path==='/v1/staff/bookings') return route.fulfill({json:{items:[],total:0,totalScoped:0,page:1,pageSize:7,metrics:{today:0,pending:0,arrivals:0,inside:0,completed:0}}});
    return route.fulfill({status:404,json:{message:'No fixture for '+path}});
  });
}
async function capture(page,name) {
  await page.evaluate(()=>document.fonts.ready);
  await writeFile(new URL(name+'.tree.txt',out),await page.locator('body').ariaSnapshot());
  const axe = await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21a','wcag21aa','wcag22aa','best-practice']).analyze();
  await writeFile(new URL(name+'.axe.json',out),JSON.stringify(axe,null,2));
  const labels = await page.locator('input[id]').evaluateAll(inputs=>inputs.filter(e=>e.getAttribute('aria-label')&&e.labels?.length).map(e=>({id:e.id,visible:[...e.labels].map(l=>l.innerText).join(' '),name:e.getAttribute('aria-label')})));
  const bounds = await page.evaluate(()=>({width:innerWidth,scroll:document.documentElement.scrollWidth}));
  const labelAxe = await new AxeBuilder({page}).withRules(['label-content-name-mismatch']).analyze();
  await writeFile(new URL(name+'.label-name.json',out),JSON.stringify(labelAxe,null,2));
  await page.screenshot({path:new URL(name+'.png',out).pathname.replace(/^\/([A-Z]:)/,'$1'),fullPage:true});
  results.push({name,bounds,labels,violations:axe.violations.map(v=>({id:v.id,impact:v.impact,nodes:v.nodes.map(n=>({html:n.html,failure:n.failureSummary}))})),labelViolations:labelAxe.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.html)}))});
}
try {
  for(const width of [320,390,1280]) {
    const context = await browser.newContext({viewport:{width,height:900},reducedMotion:'reduce',timezoneId:'Asia/Makassar'}); const page = await context.newPage();
    await fixture(page);
    await page.goto(server.origin+'/status');
    await page.locator('#status-email').waitFor();
    await capture(page,'status-'+width);
    if(width===390) {
      await page.locator('#status-code').fill('12345678');
      await page.locator('#status-email').fill('test@example.test');
      await page.getByRole('button',{name:'Lihat kunjungan',exact:true}).click();
      await page.getByRole('alert').waitFor();
      await capture(page,'status-error');
      const change=page.getByRole('button',{name:'Pilih domain dari daftar',exact:true});
      await change.focus();
      await page.keyboard.press('Enter');
      results.push({name:'email-return-to-list-focus',active:await page.evaluate(()=>({tag:document.activeElement.tagName,id:document.activeElement.id})),domainVisible:await page.locator('#status-email-domain').isVisible()});
      await page.locator('#status-email').fill('test@example.test');
      await page.emulateMedia({forcedColors:'active'});
      await capture(page,'status-forced-colors');
      await page.emulateMedia({forcedColors:'none'});
    }
    const tabs=[];
    await page.goto(server.origin+'/status');
    await page.locator('#status-email').waitFor();
    for(let i=0;i<16;i++) {
      await page.keyboard.press('Tab');
      tabs.push(await page.evaluate(()=>{const e=document.activeElement,r=e.getBoundingClientRect(),s=getComputedStyle(e);return {tag:e.tagName,id:e.id,text:(e.textContent||'').trim().slice(0,80),name:e.getAttribute('aria-label'),outline:s.outline,boxShadow:s.boxShadow,rect:r.toJSON()}}));
    }
    await writeFile(new URL('status-'+width+'.keyboard.json',out),JSON.stringify(tabs,null,2));
    await page.close();
  }
  const context = await browser.newContext({viewport:{width:1280,height:900},reducedMotion:'reduce',timezoneId:'Asia/Makassar'}); const page = await context.newPage();
  await fixture(page);
  await page.goto(server.origin+'/admin/booking');
  await page.getByRole('button',{name:'Buat janji temu',exact:true}).waitFor();
  const rail=page.getByRole('separator',{name:'Ukuran navigasi'});
  await rail.focus();
  await page.keyboard.press('ArrowLeft');
  const collapsed=await rail.getAttribute('aria-valuenow');
  await page.keyboard.press('ArrowRight');
  const expanded=await rail.getAttribute('aria-valuenow');
  await rail.click();
  const clicked=await rail.getAttribute('aria-valuenow');
  results.push({name:'sidebar-alternatives',collapsed,expanded,clicked});
  await page.setViewportSize({width:390,height:900});
  await page.getByRole('button',{name:'Buka navigasi',exact:true}).focus();
  await page.keyboard.press('Enter');
  await capture(page,'staff-mobile-menu');
  await page.keyboard.press('Escape');
  results.push({name:'mobile-menu-escape',focus:await page.evaluate(()=>document.activeElement?.getAttribute('aria-label'))});
  await page.getByRole('button',{name:'Buat janji temu',exact:true}).focus();
  await page.keyboard.press('Enter');
  await page.getByRole('dialog').waitFor();
  await capture(page,'staff-create-visit');
  await page.getByRole('button',{name:'Lanjutkan',exact:true}).focus();
  await page.keyboard.press('Enter');
  results.push({name:'staff-create-validation',focus:await page.evaluate(()=>document.activeElement?.id)});
  await capture(page,'staff-create-errors');
  await page.keyboard.press('Escape');
  results.push({name:'staff-create-escape',dialogs:await page.getByRole('dialog').count(),focus:await page.evaluate(()=>document.activeElement?.textContent)});
  await page.close();
} finally {
  await writeFile(new URL('manual-results.json',out),JSON.stringify(results,null,2));
  console.log(JSON.stringify(results,null,2));
  await browser.close();
  await server.close();
}


