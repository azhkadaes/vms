import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
import { chromium } from 'playwright';
const browser=await chromium.launch({channel:'chrome',headless:true});
const out=new URL('./card-spacing/',import.meta.url);await mkdir(out,{recursive:true});
const checks=[];
try {
 const context=await browser.newContext({viewport:{width:1440,height:900},reducedMotion:'reduce'});
 await context.addInitScript(()=>sessionStorage.setItem('vms-preview-role','Super Admin'));
 const page=await context.newPage();
 for(const route of ['roles','integration']){
  await page.goto('http://127.0.0.1:3002/admin/'+route);
  await page.locator('.management-record').first().waitFor();
  for(const width of [1440,1280,1024,768,390,320]){
   await page.setViewportSize({width,height:900});
   const data=await page.evaluate(()=>{
    const panel=document.querySelector('.management-panel'),grid=panel.querySelector('.management-records'),heading=panel.querySelector('.panel-heading'),hint=panel.querySelector(':scope > .hint');
    const g=grid.getBoundingClientRect(),h=heading.getBoundingClientRect(),n=hint.getBoundingClientRect();
    const inset=parseFloat(getComputedStyle(panel).getPropertyValue('--management-inset'));
    return {width:innerWidth,docWidth:document.documentElement.scrollWidth,inset,firstCardInset:grid.firstElementChild.getBoundingClientRect().left-g.left,headingToCard:grid.firstElementChild.getBoundingClientRect().top-h.bottom,
     hintInset:n.left-g.left,cardToHint:n.top-grid.lastElementChild.getBoundingClientRect().bottom,
     cards:[...grid.children].map(e=>{const paragraphs=[...e.querySelectorAll(':scope > p:not(.hint)')],button=e.querySelector('button'),last=paragraphs.at(-1);return {width:e.clientWidth,scroll:e.scrollWidth,paragraphGap:paragraphs[1].getBoundingClientRect().top-paragraphs[0].getBoundingClientRect().bottom,actionGap:button.getBoundingClientRect().top-last.getBoundingClientRect().bottom,buttonLeft:button.getBoundingClientRect().left,buttonRight:button.getBoundingClientRect().right};})};
   });
   assert.equal(data.docWidth,width);assert.equal(data.firstCardInset,data.inset);assert.equal(data.hintInset,data.inset);assert.equal(data.headingToCard,24);assert.ok(data.cardToHint>=24);
   for(const card of data.cards){assert.ok(card.scroll<=card.width+1);assert.ok(card.paragraphGap>=12);assert.ok(card.actionGap>=20);assert.ok(card.buttonLeft>=0&&card.buttonRight<=width);}
   checks.push({route,...data});
   if([1440,320].includes(width))await page.screenshot({path:new URL(`${route}-${width}.png`,out).pathname.replace(/^\//,''),fullPage:true});
  }
  const name=route==='roles'?'Edit akses PA':'Konfigurasi Email';
  await page.getByRole('button',{name,exact:true}).click();await page.getByRole('dialog').waitFor();
  assert.equal(await page.getByRole('dialog').isVisible(),true);
 }
 await writeFile(new URL('measurements.json',out),JSON.stringify(checks,null,2));
 console.log(`Passed ${checks.length} page/width checks; both edit dialogs opened.`);
 await context.close();
} finally {await browser.close();}
