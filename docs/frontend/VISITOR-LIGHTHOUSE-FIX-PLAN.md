# Visitor desktop and mobile Lighthouse fix plan

Status: frontend recommendations implemented 2 November 2026, WITA. See [measured results and remaining acceptance limits](VISITOR-PERFORMANCE-RESULTS.md). The original plan below is retained for traceability; the mobile performance target remains unmet on the uncompressed test host.
Scope: `D:/oikn/VMS/frontendx`, visitor app and narrowly scoped shared assets/controls.
Traces to: user request for a desktop/mobile fix plan and the two supplied Lighthouse PDFs.
The PDFs are diagnostic evidence; their recommendations are not authorization to change the app.

## Goal and users

Visitors should see the landing page quickly, choose registration or status immediately, and complete either flow without delayed input, lost form values, or broken navigation. Retain Nusantara photography, existing typography/colors, desktop composition and the mobile expanding task surface.

## Evidence and interpretation

| Report | Profile | Performance | FCP | LCP | TBT | CLS |
| --- | --- | --- | --- | --- | --- | --- |
| `C:/Users/ASUS/Downloads/expanded lighthouse.pdf` | Desktop, custom throttling, 21:02 WITA | 55 | 2.2 s | 4.4 s | 250 ms | 0 |
| `C:/Users/ASUS/Downloads/expanded lighthouse2.pdf` | Moto G Power, Slow 4G, 21:00 WITA | 39 | 11.8 s | 24.2 s | 650 ms | 0 |

Both reports have Accessibility 100 and Best Practices 100. Both audit `http://127.0.0.1:3000/` with development module URLs, Vite client and React refresh. Both contain extension activity and warn about IndexedDB; the mobile report explicitly warns about extension interference. Scores establish symptoms, not a clean production baseline. Desktop and mobile throttling differ, so their timing difference cannot be attributed solely to responsive code.

Shared actionable evidence: hero transfer approximately 505 KiB; image-delivery savings estimated at 457 KiB desktop / 405 KiB mobile including the logo. Logo source is 320 square pixels for approximately 34/44 CSS-pixel display. Do not add overlapping savings estimates together or promise the full estimate.

Current source confirms:

- `packages/ui/src/assets/nusantara.jpg` is 516,917 bytes, and the logo is 16,760 bytes.
- `apps/visitor/src/main.tsx` gives the hero `fetchPriority="high"`, but uses width/height 1920/1080 while the report identifies the source as 1008/1080. Verify intrinsic dimensions and reserve the actual CSS crop before changing attributes.
- `apps/visitor/index.html` injects the landing-only hero preload through a module script. Improve discovery without loading a hero on standalone task routes or preloading the wrong responsive candidate.
- Mobile registration already mounts on first selection, then remains mounted to retain form state. Desktop registration mounts immediately because it is part of the page. Keep these distinct behaviors.
- Booking/Status are already lazy; calendar controls and phone motion already have deferred boundaries. Inspect built requests before changing those boundaries.
- `packages/ui/src/styles.css` already uses a local WOFF2 font with `font-display: swap`.
- `VERIFICATION.md` records visitor initial JS at 105.30 KB gzip and mobile motion at 0.71 KB gzip. These are prior documented measurements, not freshly reproduced results. A roughly 1 MiB development icon module is not proof of a similarly sized production payload.

## Scope and non-goals

Improve landing delivery, real initial work, responsive resilience, and visitor-flow regression coverage. No redesign, new backend, API migration, framework/library replacement, new visual asset generation, staff feature changes, activation or deployment. Preserve the original photo/logo when producing optimized derivatives. Backend readiness remains governed by the current API-CONTRACT.md and DEPLOYMENT.md; older workspace notes are superseded.

## Build order and acceptance gates

| Gate | Work | Desktop | Mobile | Acceptance evidence |
| --- | --- | --- | --- | --- |
| 1. Establish a reliable baseline | Build visitor; serve built files with existing contract-fixture support where needed. Use a fresh extension-free profile, disposable origin storage, identical browser/Lighthouse version and fixed settings. Run three cold loads per profile; save JSON/HTML, waterfalls and a trace. Record warm loads separately. | Record original custom settings if recoverable; otherwise declare the replacement preset. | Use the reported Moto G Power / Slow 4G profile. | Median plus range, image requests, initial compressed JS/CSS, LCP subparts, long-task attribution, no dev client/refresh scripts or extension requests. No personal browser storage is cleared. |
| 2. Optimize image delivery | Create responsive WebP and, if smaller at acceptable quality, AVIF derivatives plus fallback from the existing photo. Use picture/srcset/sizes aligned to real crop width and device pixel ratio. Produce smaller logo raster variants. | Maintain the current crop and sufficient detail at large widths/DPR. | Download a candidate appropriate to the phone crop/DPR. | Visual comparison at representative DPRs; provisional hero budgets <=150 KiB phone / <=220 KiB desktop; logo <=5 KiB at required DPR if quality permits. One selected hero request, high priority, no lazy loading, reserved geometry, no photo on standalone tasks. |
| 3. Shorten the actual critical path | Verify production tree shaking and module graph. Make hero discovery early and responsive preload selection match the rendered image. Remove unused imports first; split shared package entry points only if retained code is measured. | Keep visible registration available; defer heavy picker internals until their existing interaction boundary if a trace shows remaining cost. | Preserve no booking/calendar/validation load before registration intent on the unopened landing page. | Reduced or unchanged initial JS/CSS and shorter measured critical path; no duplicate image fetch, no extra serial chunk chain or delayed first-use control. Add clear chunk-load retry and correctly sized fallback where missing. |
| 4. Protect rendering and responsiveness | Profile scrolling, task opening, typing and font loading. Adjust only measured costly effects; retain reduced-motion behavior. Avoid broad CSS removal. | Check hero/form transition, sticky header, calendar/time popovers, keyboard and zoom. | Check task/Back draft retention, scroll restoration, body unlock, keyboard, orientation and safe areas. | No horizontal page scroll at 320px; 44px minimum intended tap targets; no hidden focused controls; no photo/text overlap or clipped picker/actions; no repeated layout thrash in the measured interaction trace. |
| 5. Verify and hand off | Rerun matching Lighthouse profiles after changes; run relevant existing checks and visitor regression flows. Document changed files, before/after evidence and remaining limits. | Landing, direct task links, full registration and status flow. | Landing choices, both tasks, Back/reopen, direct links and rotation. | Pass the matrix below; explain any missed budget. Review screenshots before accepting. Verify deployed compression/cache/HTTPS behavior separately when a host exists. |

Perform one bounded gate at a time, with its evidence ready for review before the next consequential change. Gate 1 requires no product redesign decision. Do not call a gate implemented merely because this plan describes it.

## Data/API contract and state ownership

No new endpoints, request shapes, credentials or permissions. Reuse existing typed repositories, response validation and same-origin `/v1` configuration from `packages/data`; existing disposable fixtures exercise failures. Public code/email lookup, submission idempotency/recovery, protected uploads, privacy notice and server admission rules remain authoritative contracts.

| State | Owner |
| --- | --- |
| Route, selected landing task and anchor | Existing React Router pathname/query/hash |
| Registration draft, step and field errors | Existing Booking form; retain its mounted lifetime |
| Reference records, privacy notice, status responses | Existing TanStack Query hooks/cache |
| Language | Existing language providers |
| Sheet geometry, animation handles and header visibility | Existing local state/refs/effects with cleanup |
| Responsive mode | Existing media-query subscription; no parallel breakpoint store |
| Image candidate and loading | Browser picture/srcset selection and native loading; no React image-loading store |

## Component boundaries

Retain `App -> landing shell -> LandingHero / LandingIntro / existing task sheet -> lazy Booking or Status`. Keep `TaskNavigation` and `VisitInformation`. Booking continues to own form steps and existing date/time controls; Status continues to compose history, VisitTicket and deferred AdmissionQR. Reuse `Brand`, RouteFocus, AppBoundary and shared fields. If responsive hero markup is reused between preload/render paths, share only its asset definitions; avoid a new universal media component.

## UI states, validation and edge cases

- Loading: stable hero frame and matching task/picker fallback; existing data remains visible during refresh.
- Empty/no result: retain reference-data and invalid/not-found lookup guidance with a useful next action.
- Error/partial: failed image keeps content readable; failed task chunk has retry; reference/notice/status errors are scoped and preserve form input. A required privacy notice failure must continue blocking submission.
- Offline/stale: show existing feedback; never invent submission success or admission permission. Keep uncertain-submission recovery and existing retry rules.
- Success/denied: preserve receipt/status details, cancellation eligibility, and server-controlled QR visibility. Maintain server validation/error mapping and no duplicate submission.
- Validation: retain current Zod/form rules, native required semantics, inline associated errors, first-invalid focus, WITA dates and YYYY-MM-DD/HH:mm values. Current documented limits remain 20 people and optional 5 MB PDF/JPEG/PNG. Any rule changes require separate scope.
- Exercise long Indonesian/English copy, slow/broken font or images, 320px reflow, 200% actual zoom, orientation, rapid task toggling, resize across 760px while a draft exists, on-screen keyboard, Back/Forward, direct links and lazy-chunk failure after a deployment.

## Performance decisions: three questions

1. Does it need to happen? Avoid oversized images, duplicate preloads, unselected task code and unused production exports. Keep existing caching and meaningful lazy boundaries. Do not duplicate state or replace the stack based on development totals.
2. Must it happen now? Hero and visible headline do; unopened phone tasks, QR/calendar internals and optional motion do not. Load on intent or justified visibility, with keyboard/touch equivalents for hover. Do not defer visible desktop form content indiscriminately.
3. Where should it happen? Resize/encode images at build time; let browser source selection and CSS handle layout. Keep input urgent and effects local. No worker, virtualization, new state system or application-wide memoization: neither report demonstrates a need.

Library check: no new runtime dependency planned. Use existing React/Vite, CSS, native image markup and browser tooling. Asset conversion may use available local tooling; any added build tool must have an explicit measured benefit. Hosting compression/cache controls belong to the deployment layer.

Pattern fit: optimize existing shared assets because they recur; avoid a general rendering abstraction for one hero. Retain state/API boundaries because breaking them risks draft loss or contract drift. Asset and import changes are reversible; routing and form ownership remain stable. If a shared logo/UI file changes, include an admin build and targeted visual smoke check.

## Verification and completion criteria

| Area | Required coverage |
| --- | --- |
| Routes | `/`, both existing landing task query/hash links, standalone registration and `/status`; verify exact registration route from current router before scripting |
| Viewports | 320, 390, 430, 760/761, 768, 1280 and 1440px; selected DPR 1/2/3 image checks, landscape and actual 200% browser zoom |
| Accessibility | Existing axe checks plus keyboard/focus, labels/announcements, ID/EN, reduced motion and visible state contrast. Existing scores of 100 are regression baselines, not full conformance proof. |
| Functional | Complete registration/review/submission; status success/not-found/network error; draft retention across mobile Back/reopen; privacy failure, upload, uncertain submission and QR/cancellation regression paths |
| Performance | Three comparable cold runs/profile; provisional median score >=90, LCP <=2.5s, TBT <=200ms, CLS <=0.1 with zero preventable new shift. Report FCP and interaction traces too; no guarantee before a clean baseline. |
| Engineering | Existing formatting/types, relevant logic/visitor browser/production-fixture checks and both builds if shared assets change; finish with the established `npm.cmd run check` at the integration gate. Add only focused assertions for changed loading/state behavior. |

Field targets after deployment: p75 LCP <=2.5s, INP <=200ms, CLS <=0.1, segmented by desktop/mobile. Lab TBT is not INP, and local medians do not prove field performance. Physical Android/iPhone, Safari and screen-reader checks remain separate evidence if unavailable locally.

## Risks and unresolved measurements

Source has evolved beyond some findings in these reports. Gate 1 must distinguish remaining production issues from already-resolved development observations. Exact desktop custom-throttle values are not in the extracted footer. Photo crop/DPR can limit compression budgets. A smaller hero alone may not resolve render delay. Preload must remain route-aware and must match the chosen image source. More code splits can worsen first-interaction latency. Shared UI changes can affect staff. No real backend/host is present for operational acceptance.

## References and handoff

Current project references: README.md, PLAN.md, VERIFICATION.md, DESIGN.md, API-CONTRACT.md and DEPLOYMENT.md. Older memory was used only to locate the active workspace and prior review boundary; current source/documents supersede stale backend/repository assumptions.

External technical references checked for this plan:
- [Google: Optimize LCP](https://web.dev/articles/optimize-lcp): resource discovery, priority and render delay.
- [Google: Web Vitals](https://web.dev/articles/vitals): field thresholds and p75 segmentation.
- [Vite: Building for production](https://vite.dev/guide/build.html): serve a built bundle for the production baseline. Installed project Vite configuration remains the source for version-specific options.

Original planning review: findings map to evidence, priorities and acceptance criteria; existing identity/state/API boundaries retained; no new library or invented performance result. The subsequent implementation was explicitly authorized by the user. Current changes, tests, fresh Lighthouse results and unverified deployment/device work are recorded in VISITOR-PERFORMANCE-RESULTS.md.
