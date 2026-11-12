# Accessibility fixes after readiness recheck

3 November 2026, WITA. The user approved the recommended email accessibility and mobile navigation corrections after requesting the formatting fix separately. Scope: frontendx only. Existing design direction and visible labels are retained; no new asset, dependency or backend behavior was introduced.

## Changes

- **Finding 1 resolved:** the shared email username input uses its existing associated visible label instead of overriding it with a generic aria-label. Existing helper/error descriptions remain associated. The custom-domain action's accessible name begins with its visible Ubah/Change label. Explicit domain switches focus the newly rendered custom input or domain selector. Pasting an entire email address keeps focus in the username input and preserves the parsed domain. The pending focus flag and DOM references stay local to the shared component; there is no new global state, API request or dependency.
- **Finding 2 resolved:** mobile navigation uses Radix Dialog.Content's default div with the same sidebar classes and id. It retains its accessible title/description and opening/closing focus behavior. The desktop sidebar remains an aside with its resize rail.
- **Finding 3 resolved in the preceding formatting follow-up:** the admin and visitor stylesheets match the configured formatter and the full formatting check passes.

Application changes: packages/ui/src/index.tsx and apps/admin/src/main.tsx. Two regression scenarios were added to tests/production.browser.mjs so custom-domain and open-drawer states are included in future production checks.

## Verification

The two new scenarios pass against fresh production builds in headless Chrome. Email checks cover ID/EN associated labels, custom-domain entry, Tab/Enter activation, both switch directions, retained username, normal-domain selection, paste behavior and axe scans in the affected states. Drawer checks cover 320px and 390px, a div dialog, focus on the active navigation item, repeated forward/backward tabbing within the drawer, axe, Escape/trigger focus restoration, and desktop recovery to an aside.

TypeScript and both builds pass. Production screenshots for custom email and the 390px drawer were visually inspected; their established classes, layout and content remain in use. Checks used isolated preview servers because the ordinary staff port is occupied.

The first aggregate attempt exposed a temporary runner defect: Vite's in-process preview startup set NODE_ENV=development, which the runner passed to its child build. The resulting bundles compiled preview behavior and failed the production-only staff fixtures. The runner now removes that inherited setting from the child environment. This was corrected in the audit harness; application code was not changed to accommodate the fixture. The fresh, correctly built targeted scenarios had already passed before that aggregate attempt. The corrected aggregate run is recorded separately.

Evidence: evidence-2026-11-03/a11y-fix-targeted.log, a11y-fix-typecheck.log, a11y-fix-build.log, a11y-fix-full-check.log (invalid-environment attempt) and a11y-fix-full-check-final.log (corrected runner). The existing production suite writes the new email-custom-domain-*, email-custom-paste-* and mobile-navigation-* axe reports/screenshots to evidence-2026-11-02/production despite this run's 3 November date.

Final outcomes:

| Check | Result |
| --- | --- |
| Full formatting check, strict types, both builds | PASS |
| Logic/contract tests | 96 passed |
| API-fixture tests | 34 passed |
| Production-bundle suite, run independently after the corrected build | All 13 passed, including the two new accessibility regressions |
| Selected affected-state axe scans | Six scans, zero violations; see a11y-fixed-states.json |
| Fresh dependency audit | Zero reported vulnerabilities |
| Corrected aggregate preview browser run | 48 passed, 2 timed out; aggregate check exits 1 |
| Isolated recheck of those two preview tests | Inconclusive: Chrome launch timed out in both setup hooks |

The two corrected-aggregate timeouts were the existing main-screen accessibility loop (waiting for a visible h1) and the existing unsaved-staff-form refresh scenario (waiting for the refresh button). The earlier aggregate attempt passed all 50 preview tests, but its development-setting contamination invalidated the later production-bundle portion. The corrected attempt's two preview timeouts remain unresolved; do not describe the aggregate check as green or claim 193 tests passed in one clean run. The isolated retry failed during browser setup, so it neither confirms nor clears those two scenarios. No application changes were made to conceal those results. Logs: a11y-fix-production-final.log, a11y-fix-preview-rerun.log and a11y-fix-dependency-audit.json in evidence-2026-11-03.

## Scope and release limits

Finding 4's local mobile performance acceptance gap and the original audit's real-service, deployment, screen-reader, actual browser zoom, Safari and physical-device acceptance remain open. These fixes do not establish overall production readiness or full WCAG conformance.

## Scoped delivery gate

- Hard gate **PASS for the selected defects**: R-32/R-39 names, replacement-control focus and compatible mobile dialog semantics verified by the new browser scenarios and axe.
- Purpose gate **PASS for this change**: existing controls, content and classes perform the same tasks; no decorative or architectural additions.
- Liveliness **unchanged**: no new visual design, motion or imagery. Existing OIKN direction is retained.
- Craftsmanship **PASS for targeted behavior**: keyboard/paste/ID/EN and phone/desktop transitions have executable regression coverage. Aggregate verification **FAIL/incomplete** because of the two preview timeouts and the inconclusive browser-launch retry. No overall release approval is given.
