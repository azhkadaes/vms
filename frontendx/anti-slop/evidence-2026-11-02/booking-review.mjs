import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';
import { writeFile } from 'node:fs/promises';
import assert from 'node:assert/strict';
import { startPreview } from '../../tests/helpers/preview-server.mjs';
const api = await startPreview();
const browser = await chromium.launch({ channel: 'chrome', headless: true });
const out = new URL('./', import.meta.url);
try {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, timezoneId: 'Asia/Makassar' });
  const page = await context.newPage();
  await page.addInitScript(() => sessionStorage.setItem('vms-preview-role', 'PA'));
  await page.route('**/__preview/**', async route => {
    const u = new URL(route.request().url());
    await route.fulfill({ response: await route.fetch({ url: api.base + u.pathname.split('/__preview')[1] + u.search }) });
  });
  await page.goto('http://127.0.0.1:3012/admin/booking');
  await page.getByRole('button', { name: 'Buat janji temu', exact: true }).click();
  await page.locator('#appointment-officerId').selectOption('o2');
  await page.locator('.date-picker-trigger').click();
  await page.getByRole('combobox', { name: 'Pilih tahun', exact: true }).selectOption('2027');
  await page.getByRole('combobox', { name: 'Pilih bulan', exact: true }).selectOption('0');
  await page.locator('.shadcn-calendar-day[data-day="2027-01-12"]:not(:disabled)').first().click();
  await page.locator('.time-picker-trigger').click();
  await page.getByRole('listbox', { name: 'Jam', exact: true }).getByRole('option', { name: '10', exact: true }).click();
  await page.getByRole('listbox', { name: 'Menit', exact: true }).getByRole('option', { name: '00', exact: true }).click();
  await page.locator('.date-picker > small').click();
  await page.locator('#appointment-purpose').fill('Data uji: koordinasi layanan dan pemeriksaan rencana kunjungan.');
  await page.getByRole('button', { name: 'Lanjutkan', exact: true }).click();
  await page.locator('#appointment-name').fill('Perwakilan Contoh');
  await page.locator('#appointment-organization').fill('Instansi Contoh untuk Pengujian Tampilan');
  await page.locator('#appointment-email').fill('contoh.pengunjung');
  await page.locator('#appointment-email-domain').selectOption('gmail.com');
  await page.locator('#appointment-phone').fill('081234567890');
  await page.locator('#appointment-identityType').selectOption('Paspor');
  await page.getByRole('button', { name: 'Tambah pengunjung', exact: true }).click();
  await page.locator('#appointment-visitor-0').fill('Anggota Rombongan Contoh');
  await page.getByRole('button', { name: 'Lanjutkan', exact: true }).click();
  await page.getByRole('heading', { name: 'Periksa janji temu', exact: true }).waitFor();
  await page.waitForFunction(() => document.activeElement?.textContent === 'Periksa janji temu');
  assert.match(await page.locator('.appointment-review').innerText(), /12 Januari 2027/);
  assert.match(await page.locator('.appointment-review').innerText(), /Paspor/);
  assert.match(await page.locator('.appointment-review').innerText(), /Data uji: koordinasi/);
  await page.getByRole('button', {name:'Ubah detail kunjungan',exact:true}).click();
  await page.waitForFunction(() => document.activeElement?.id === 'appointment-officerId');
  assert.equal(await page.locator('#appointment-purpose').inputValue(),'Data uji: koordinasi layanan dan pemeriksaan rencana kunjungan.');
  await page.locator('#appointment-purpose').fill('Keperluan diperbarui melalui ringkasan.');
  await page.getByRole('button', {name:'Kembali ke periksa',exact:true}).click();
  await page.waitForFunction(() => document.activeElement?.textContent === 'Periksa janji temu');
  assert.match(await page.locator('.appointment-review').innerText(), /Keperluan diperbarui/);
  await page.getByRole('button', {name:'Ubah data perwakilan',exact:true}).click();
  await page.waitForFunction(() => document.activeElement?.id === 'appointment-name');
  assert.equal(await page.locator('#appointment-name').inputValue(),'Perwakilan Contoh');
  assert.equal(await page.locator('#appointment-visitor-0').inputValue(),'Anggota Rombongan Contoh');
  await page.getByRole('button', {name:'Kembali ke periksa',exact:true}).click();
  await page.getByRole('heading', {name:'Periksa janji temu',exact:true}).waitFor();
  console.log('REVIEW', await page.getByRole('dialog').ariaSnapshot());
  const results = [];
  for (const width of [1440, 390, 320]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.screenshot({ path: new URL(`booking-review-after-${width}.png`, out).pathname.replace(/^\//, ''), fullPage: true });
    const measure = await page.evaluate(() => ({
      pageWidth: document.documentElement.clientWidth, pageScroll: document.documentElement.scrollWidth,
      dialog: document.querySelector('.modal').getBoundingClientRect().toJSON(),
      review: document.querySelector('.appointment-review').getBoundingClientRect().toJSON(),
      scroll: { width: document.querySelector('.modal-scroll').clientWidth, scrollWidth: document.querySelector('.modal-scroll').scrollWidth },
      text: [...document.querySelectorAll('.appointment-review dt,.appointment-review dd')].map(e => ({tag:e.tagName, text:e.textContent, font:getComputedStyle(e).fontFamily, size:getComputedStyle(e).fontSize, weight:getComputedStyle(e).fontWeight, margin:getComputedStyle(e).margin, width:e.getBoundingClientRect().width})),
      focus: document.activeElement.textContent,
    }));
    const axe = await new AxeBuilder({ page }).include('[role="dialog"]').analyze();
    assert.equal(axe.violations.length,0);
    assert.ok(measure.scroll.scrollWidth <= measure.scroll.width + 1);
    assert.ok(measure.text.every(e=>e.font.includes('Geist')));
    results.push({width, measure, violations: axe.violations});
  }
  await writeFile(new URL('booking-review-after-results.json',out),JSON.stringify(results,null,2));
  await page.getByRole('button', {name:'Ubah rombongan pengunjung',exact:true}).click();
  for (let i=1;i<19;i++) {
    await page.getByRole('button',{name:'Tambah pengunjung',exact:true}).click();
    await page.locator(`#appointment-visitor-${i}`).fill(`Pengunjung Contoh ${i+2}`);
  }
  await page.getByRole('button',{name:'Kembali ke periksa',exact:true}).click();
  assert.equal(await page.locator('.appointment-review-roster li').count(),20);
  assert.match(await page.locator('#review-roster-title').innerText(),/20 orang/);
  await page.getByRole('button', {name:'Ubah dokumen pendukung',exact:true}).click();
  await page.waitForFunction(() => document.activeElement?.id === 'appointment-document');
  await page.locator('#appointment-email').fill('pengunjungcontoh'.repeat(4));
  await page.locator('#appointment-document').setInputFiles({name:'DokumenPermohonanKunjunganResmi'.repeat(5)+'.pdf',mimeType:'application/pdf',buffer:Buffer.from('%PDF-1.4 test')});
  await page.getByRole('button',{name:'Kembali ke periksa',exact:true}).click();
  const long = await page.evaluate(()=>({width:document.querySelector('.modal-scroll').clientWidth,scrollWidth:document.querySelector('.modal-scroll').scrollWidth, text:document.querySelector('.appointment-review').textContent}));
  assert.ok(long.scrollWidth <= long.width + 1);
  await page.getByRole('button',{name:'Buat janji temu',exact:true}).scrollIntoViewIfNeeded();
  assert.ok(await page.getByRole('button',{name:'Buat janji temu',exact:true}).isVisible());
  await page.locator('.modal-scroll').evaluate(e => { e.scrollTop = e.scrollHeight; });
  await page.screenshot({path:new URL('booking-review-after-320-long.png',out).pathname.replace(/^\//,''),fullPage:true});
  await writeFile(new URL('booking-review-after-long.json',out),JSON.stringify(long,null,2));
  const axeLong=await new AxeBuilder({page}).include('[role="dialog"]').analyze();
  assert.equal(axeLong.violations.length,0);
  await page.getByRole('button',{name:'Kembali',exact:true}).click();
  assert.equal(await page.locator('#appointment-email').inputValue(),'pengunjungcontoh'.repeat(4));
  assert.equal(await page.locator('#appointment-identityType').inputValue(),'Paspor');
  await page.getByRole('button',{name:'Lanjutkan',exact:true}).click();
  assert.equal(await page.locator('.appointment-review-roster li').count(),20);
  console.log('LONG',JSON.stringify(long));
  console.log('BASE', results.map(r=>({width:r.width,violations:r.violations.length,scroll:r.measure.scroll})));
} finally { await browser.close(); await api.close(); }
