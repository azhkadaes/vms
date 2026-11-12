# Visitor desktop and mobile performance changes

Implemented 2 November 2026, WITA, in frontendx. The existing visitor design, registration state, mobile Back behavior, API contracts and original image files are retained. No runtime dependency was added and nothing was deployed.

Subsequent [critical-request follow-up](VISITOR-CRITICAL-REQUESTS.md) adds direct font discovery and scroll-triggered motion loading. Its latest medians are **98 desktop / 85 mobile**, mobile FCP **2.55 s**, mobile LCP **3.75 s**, with zero measured CLS. The table below preserves the earlier image/shell comparison; current visitor entry JavaScript is **93.09 KB gzip**. The remaining mobile target and formatting limits still apply.

## What changed

- Responsive AVIF/WebP photo variants replace the full JPEG download in supporting browsers. Native picture selection retains the original JPEG fallback and the existing desktop/phone crop. Source sizes account for the full image behind the tall object-fit crop, including short landscape screens.
- A small build-generated script starts the matching AVIF preload after viewport metadata and before stylesheets. It runs only on `/`; direct `/booking` and `/status` do not download the hero. A production browser check verifies that preload and picture share exactly one request.
- Shared Brand uses 48/96/144px WebP logo candidates, retaining the PNG fallback. The 48px asset stays external rather than adding base64 bytes to initial JavaScript.
- Existing shell components moved unchanged into `packages/ui/src/shell.tsx`, with compatible re-exports. Visitor startup imports that leaf and the existing query provider directly, avoiding eager dialog/form dependencies. Form/cache ownership and lazy feature boundaries are unchanged.
- Added image MIME types to the production test server, responsive resource/ID/EN accessibility checks, an optional Lighthouse runner, and a reproducible image-conversion helper.

## Measured results

Three cold loads per profile, each with a new headless Chrome profile, no extensions, and the same local built-file server with synthetic catalog data. The server sends uncompressed files and no-store responses for both before and after. The runner does not touch personal browser storage.

| Metric | Desktop before | Desktop after | Mobile before | Mobile after |
| --- | --- | --- | --- | --- |
| Median Performance | 96 | 98 | 73 | 84 |
| Score range | 94-96 | 98-98 | 63-73 | 84-84 |
| Median FCP | 0.76 s | 0.75 s | 2.93 s | 2.78 s |
| Median LCP | 1.35 s | 1.06 s | 6.01 s | 3.76 s |
| Median TBT | 0 ms | 0 ms | 25.8 ms | 26 ms |
| Median CLS | 0 | 0 | 0.000326 | 0.000326 |
| Accessibility / Best Practices | 100 / 100 | 100 / 100 | 100 / 100 | 100 / 100 |

Mobile LCP is approximately 37% faster. The provisional mobile score >=90 and LCP <=2.5 s targets are **not met** on this uncompressed test host. Initial script transfer/render delay remains material. Enable and verify deployment-host Brotli/gzip before choosing further architecture changes; the test result does not predict the resulting score. Do not hide visible content or remove the existing photo/interaction simply to raise a lab score.

Tool: cached Lighthouse **12.8.2**, Headless Chrome **156.0.0.0**. Desktop uses Lighthouse's desktop preset (1350x940, DPR 1, simulated desktop throttling); mobile uses its Moto G Power default (412x823, DPR 1.75, simulated Slow 4G). Exact resolved settings are saved in each summary/report. The attached PDFs used Lighthouse 13.5.0, development resources and browser extensions; their 55/39 scores are not the comparison baseline. Desktop's original custom settings were not recoverable, so a declared standard preset replaces them.

Evidence:
- `anti-slop/evidence-2026-11-02/visitor-performance/before/`: three JSON/HTML reports and traces per profile, plus summary.json.
- `anti-slop/evidence-2026-11-02/visitor-performance/optimized/`: matching final reports/traces and summary.json. Intermediate `after` and `final` folders record the image-only and shell-import stages.
- `anti-slop/evidence-2026-11-02/production/visitor-responsive-images.json`: selected source, transferred bytes, link initiator and no duplicate hero across nine viewport/DPR cases.
- `anti-slop/evidence-2026-11-02/production/visitor-responsive-*.png` and `.axe.json`: phone/desktop ID/EN evidence. Selected phone and desktop images were visually inspected.

Source stylesheets received separate color-format edits during this work. Those edits are retained; the before/after captures describe the workspace at each measurement, rather than claiming a perfectly isolated experiment against every concurrent edit.

## Asset and bundle costs

| Resource | Before | After |
| --- | --- | --- |
| Original photo / full-size AVIF | 516,917 bytes | 132,908 bytes, approximately 74% smaller |
| Desktop DPR 1 selected AVIF | 516,917 bytes | 76,427 bytes |
| Full-size WebP fallback | 516,917 bytes | 219,742 bytes |
| Small / medium / large WebP logo | 16,760-byte PNG | 1,740 / 4,938 / 9,090 bytes |
| Visitor entry JavaScript, Vite gzip estimate | 105.30 KB | 93.06 KB, approximately 12% smaller |

The 3x logo deliberately exceeds the provisional 5 KiB logo budget to preserve clarity; ordinary 1x/2x candidates satisfy it. The original JPEG/PNG remain in the source and build for compatibility but are not additionally fetched by the tested modern browser. More output files do not mean every image variant downloads.

## Validation

Validation logs are separate from older project acceptance evidence:
- `anti-slop/visitor-lighthouse-check-2026-11-02.log`: formatting/types, 96 logic tests, 34 isolated preview/API tests and both builds passed; the browser stage stopped at occupied port 3000.
- `anti-slop/visitor-lighthouse-browser-2026-11-02.log`: all 50 existing browser tests passed using a separate visitor server on 4310. The occupied process was not terminated. The existing staff server/proxy supplied staff preview coverage.
- `anti-slop/visitor-lighthouse-build-2026-11-02.log`: both final production builds passed. Existing upstream Zod annotation warnings are nonfatal.
- All **11 production-bundle tests passed**, including the new responsive resource/ID/EN case and existing submission/upload/admission flows. Dependency audit reports **0 vulnerabilities**. Results: `anti-slop/visitor-lighthouse-production-2026-11-02.log` and `anti-slop/visitor-lighthouse-audit-2026-11-02.log`.

Together the separate runs passed **191 tests** (96 logic + 34 isolated API + 50 browser + 11 production). Final strict TypeScript and scoped formatting also pass. This count does not turn the interrupted aggregate check into a passing aggregate run.

The complete check command was attempted again; it currently stops on formatting of separate OKLCH stylesheet edits in `apps/admin/src/styles.css` and `apps/visitor/src/styles.css`. Their values were preserved. Scoped formatting for this change's TypeScript/JavaScript/HTML files passes; see `anti-slop/visitor-lighthouse-check-final-2026-11-02.log` for the project-wide limitation.

Responsive coverage: 320/390/430/760/761/768/1280/1440px, selected DPR 1/2/3, and 740x390 landscape. The original mobile draft/Back regression remains covered. No new library, worker, virtualization, polling policy, global state, or API adapter was introduced. The existing contract fixture remains synthetic.

## Run and maintain

Changed source/configuration:
- Visitor: `apps/visitor/src/main.tsx`, `src/features/hero-image.ts`, `src/hero-image-settings.ts`, `src/styles.css`, `index.html`, `vite.config.ts`.
- Shared: `packages/ui/src/shell.tsx`, `src/index.tsx`, `src/styles.css`, `packages/ui/package.json`, `packages/data/package.json`, and the eleven optimized delivery image files under `packages/ui/src/assets`. Original photo/logo files are unchanged.
- Checks/tooling: `tests/production.browser.mjs`, `tests/helpers/production-server.mjs`, `tooling/visitor-lighthouse.mjs`, `tooling/optimize-visitor-images.py`, and the root formatting commands in `package.json`.
- Handoff: this report, VISITOR-LIGHTHOUSE-FIX-PLAN.md, PLAN.md, VERIFICATION.md and DEPLOYMENT.md. Unrelated concurrent color values were preserved.

Run `npm.cmd run dev` normally. The normal 3000/3002 ports must be available; this task's temporary visitor server used 4310 because 3000 was occupied. Run `npm.cmd run check` for the established suite, or set `VMS_TEST_URL` to a verified existing visitor preview to reuse it. No dummy/real API swap is needed for this change: existing same-origin `/v1` configuration and development fixtures remain in place.

For repeatable local Lighthouse measurements, build first, then run `node tooling/visitor-lighthouse.mjs measurement-name`. It uses the existing cached tool installation at `.task-cache/audit-tools/node_modules`; set `LIGHTHOUSE_ROOT` to another directory containing compatible lighthouse and chrome-launcher packages if needed. Tool installation is separate from the application lockfile. Reports disclose their actual tool version/settings. The runner serves temporary synthetic catalogs and closes its own server/Chrome processes.

To regenerate delivery assets, run `python tooling/optimize-visitor-images.py` in an environment with Pillow AVIF/WebP support. It reads the originals, writes only the named delivery variants, and is not run on every application build. AVIF quality 45/speed 6; WebP photo quality 50/method 6; logo quality 75/method 6. Rebuild, compare image quality, repeat request checks, and update any preload CSP hash afterward.

Deployment must serve AVIF/WebP MIME types, permit the exact generated preload script through a CSP hash, and enable appropriate HTML/JS/CSS compression and hashed-asset caching. See DEPLOYMENT.md. For this built preload, the current script hash is `sha256-PPQGW6Lwugz1xlQGuSKhdbQwItS0H9Uus3nYtoC5UPY=`; recompute from the exact built script whenever its contents change.

Field Core Web Vitals, a real deployed host/backend, warm-cache performance, physical phones, Safari, screen readers and actual 200% browser zoom remain unverified. Browser screenshots/axe results establish only the tested states. These frontend changes do not activate real submissions, storage, email or admission credentials.
