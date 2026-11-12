# Frontend verification

Latest recheck: [4 November 2026 readiness and evidence](FRONTEND-READINESS-2026-11-04.md). Backend entry point: [4 November handoff](BACKEND-HANDOFF-2026-11-04.md). The results below are the historical 2 November baseline; do not treat their counts as the latest run.

## Historical verification — 2 November 2026

Scope: **D:/oikn/VMS/frontendx only**. The new D:/oikn/VMS/backend is absent. This verifies frontend source, synthetic policy behavior and the built client against a disposable API fixture. It does not verify a real backend, database, email, storage or admission service.

## Final check result

`npm.cmd run check` completed with **exit 0** after the final decision-focus/catalog feedback changes. It started/stopped its own preview servers and served production builds on a temporary loopback port. No application server was deployed or left running by that sequence.

| Check                               | Result                                                                                                                      |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Formatting                          | Prettier check passed for owned application/data/tooling/test source. Vendored controls are excluded from broad formatting. |
| Strict TypeScript                   | Passed.                                                                                                                     |
| Logic tests                         | **93 passed**, 0 failed/skipped.                                                                                            |
| Isolated synthetic API tests        | **34 passed**, 0 failed/skipped.                                                                                            |
| Preview browser/accessibility tests | **48 passed**, 0 failed/skipped, sequential Chrome.                                                                         |
| Built-production contract tests     | **10 passed**, 0 failed/skipped, Chrome.                                                                                    |
| Visitor and staff production builds | Both passed.                                                                                                                |
| Fresh dependency audit              | **0 reported vulnerabilities**.                                                                                             |
| Requirement mapping                 | **55 of 55 identifiers** mapped; local links in 14 current documents resolved.                                              |

There are **185 passing tests** across the four groups. Full local output: anti-slop/check-2026-11-02.log. The dependency scan reflects the lockfile/advisory data at this run. Upstream Zod comments produce nonfatal Rollup annotation warnings; both builds complete. Node's experimental type-stripping notices do not indicate test failures.

After the full check, the final phone review separated the document helper from its filename and shortened the staff cancellation summary. Formatting and strict TypeScript passed again; both production builds and all 10 built-production tests passed again. These repeat runs do not add to the 185 unique tests. Logs: anti-slop/format-2026-11-02.log, anti-slop/build-2026-11-02.log and anti-slop/production-2026-11-02.log.

## Browser coverage

- Public registration/review/receipt, complete field validation, payload-bound retry, uncertain-outcome recovery, reference-only reload recovery, localized public history, logical headings, cancellation deadline and keyboard focus.
- Staff/reception 30/15-second freshness, hidden pause, manual/return/reconnect refresh, denied/stale feedback, retained URL/input, approval conflicts and fixed rescheduling.
- Restricted four-field creation/projections, credential-gated reception identity, general export exclusion, arrival window/Missed and next-day checkout. Staff cancellation adds explicit permission/decision scope, required visitor-facing reason, optional internal note, stale-version review and resource release. Cancellation-field regression checks cover staff details, visitor status and exclusion of internal notes from public responses; notification delivery still requires the backend.
- Employee phone cards/desktop table, long fields, URL search/no-match/reset and focus. Production create/edit/deactivate/assign, administrator initial password, required password change, explicit conflict reload, role permissions, and integration secret keep/clear behavior.
- Actual built adapters reject stored preview-role authorization. Session outage/retry, login rate limiting, changed server role, bounded pages/direct detail and independent report aggregation/denial are exercised.
- Multipart bytes precede submission and the booking carries an opaque upload token. Protected download succeeds or shows denial. Production QR uses the server fixture payload and expires.
- Phone choices defer registration/calendar/validation code until intent and preserve the draft through Back. The hero loads on the landing route and is not fetched by standalone registration/status flow.
- Shared pickers cover required errors, ISO/FormData, leap day, keyboard confirm/clear/Escape, discard/read-only and popover bounds. The foreign-timezone case uses native Date; a simulated Date replacement is not mistaken for product behavior.

## Accessibility and visual limits

Automated axe scans report no selected WCAG A/AA or best-practice violations in **13 preview views/popovers and 5 production management/detail/cancellation states**. These 18 scoped scans are not full accessibility certification.

Checks use **320, 390, 768 and 1280px where applicable**, including long fields, modal/picker bounds and document overflow. Screenshots/accessibility snapshots were generated and selected production employee/editor/integration/detail/cancellation states visually inspected. The final 390px cancellation view keeps its required reason and actions within the dialog. Fixtures and clocks are fictional, often using 2099 for deterministic timing; screenshots are not live records.

Production evidence: anti-slop/evidence-2026-11-02/production. Existing phase scripts retain evidence-2026-10-30 and evidence-2026-10-31 directory names; this run's log records their current execution.

Actual 200% browser zoom, screen readers, Safari, physical scanners/cameras, deployed gateway/cookies/CSRF/security headers and real service/device performance remain unverified. A source fallback does not prove physical QR operation.

## Build measurements

| Asset                      | Gzip size                                      |
| -------------------------- | ---------------------------------------------- |
| Staff initial JavaScript   | **115.46 KB**                                  |
| Visitor initial JavaScript | **93.09 KB** after critical-request follow-up |
| Lazy booking detail        | **4.19 KB**                                    |
| Phone landing motion       | **0.71 KB**, previously **45.76 KB** with GSAP |

The original hero remains **516.92 KB** as a compatibility fallback. Responsive AVIF now selects **76.43 KB** on tested DPR 1 desktops and **132.91 KB** on higher-density phones. Existing photography/fonts/identity are reused. See [visitor performance results](VISITOR-PERFORMANCE-RESULTS.md) and the [critical-request follow-up](VISITOR-CRITICAL-REQUESTS.md) for the latest 98 desktop / 85 mobile local medians, exact profiles, responsive/ID/EN checks and the separate stylesheet formatting limit. These measurements do not establish field Core Web Vitals.

## Reproduce and accept

Use Node 22/npm from this directory, install dependencies and Chrome, then npm.cmd run check. Alternatively install Playwright Chromium and set PLAYWRIGHT_CHANNEL=chromium. With an existing preview server, set VMS_TEST_URL to its loopback visitor origin; otherwise the runner requires free ports 3000/3002 and owns their lifecycle. It never kills an unrelated listener.

CI is supplied for a repository rooted at frontendx and was not executed remotely. Real acceptance remains in [DEPLOYMENT](DEPLOYMENT.md) and [API-CONTRACT](API-CONTRACT.md): new backend authorization/transactions/recovery, actual accounts/passwords/integrations, private storage/content inspection, queued email, admission revocation/single use, approved privacy/retention and operational/device checks. G14 draft recovery stays deferred.
