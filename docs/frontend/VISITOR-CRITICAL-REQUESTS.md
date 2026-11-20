# Visitor critical-request follow-up

Implemented and checked 2 November 2026, WITA. This follows the visitor performance work and the user's Network dependency tree / LCP request discovery messages.

## Interpretation and change

“Request is discoverable in initial document” is a passing condition. Saved Lighthouse reports confirm `requestDiscoverable`, `priorityHinted` and `eagerlyLoaded` are true for the hero. No additional hero preload is needed; the existing route-aware preload and responsive picture already share one download.

The saved mobile network tree still included HTML -> stylesheet -> display font and HTML -> application JS -> optional landing motion. The display font is now preloaded directly in visitor HTML with the same URL and anonymous CORS mode used by CSS. Native viewport metadata remains before responsive image selection. Optional motion loads on the first scroll instead of during initial mounting. A CSS scale preserves the original eligible phone crop until the motion module loads; reduced-motion and short-landscape behavior remain unchanged.

Changed files: `apps/visitor/index.html`, `apps/visitor/src/main.tsx`, `apps/visitor/src/styles.css`, and `tests/production.browser.mjs`. No new library, API change, global state or extra hero request.

## Verification

- Strict TypeScript, scoped code formatting and both builds passed.
- All 11 production-flow tests passed, including ID/EN axe checks, nine viewport/DPR cases, one hero request, one HTML-initiated font request, no motion download before scrolling, motion download after scrolling, and retained mobile draft/Back behavior.
- Three new cold Lighthouse runs per profile retain Accessibility/Best Practices 100. Desktop scores are 98/98/98; mobile scores are 85/86/85 (median 85). Median mobile FCP is 2.55 s; median mobile LCP is 3.75 s. CLS is zero in all six runs. The uncompressed host still misses the provisional mobile 90 / 2.5 s LCP targets.
- Mobile network trees now contain only HTML -> main JS and HTML -> CSS, with no font or motion child chain. The observed local chain durations are 39/57/35 ms; these are local trace timings, not simulated mobile LCP or deployed network latency.
- Desktop still loads visible registration code and typed catalog data after application startup. This chain remains; the form was not hidden or disabled to remove a diagnostic.
- The cached Lighthouse 12.8.2 network-tree insight still reports score 0 despite the flattened mobile tree. No claim is made that this insight passes in that tool or in the user's Lighthouse 13.5.0. Hero request discovery continues to pass in all six saved reports.

Evidence: `anti-slop/evidence-2026-11-02/visitor-performance/critical-chain/` contains the reports, traces and summary. Latest responsive resource/ID/EN evidence is under `anti-slop/evidence-2026-11-02/production/`. Build and production logs are `anti-slop/visitor-critical-chain-build-2026-11-02.log` and `anti-slop/visitor-critical-chain-production-2026-11-02.log`.

No unrelated stylesheet color formatting was changed. The earlier aggregate formatting limitation remains. The existing preload CSP hash is unchanged because its script contents did not change. Real deployment compression, field metrics, physical-device/Safari/screen-reader checks and actual browser zoom remain separate acceptance work.

References checked: [Chrome: Network dependency tree](https://developer.chrome.com/docs/performance/insights/network-dependency-tree) and [Chrome: LCP request discovery](https://developer.chrome.com/docs/performance/insights/lcp-discovery). Chrome's current guidance treats early discovery, priority and eager loading as distinct requirements; an informative diagnostic's presence alone does not establish which requirement failed.
