# Staff visit-data refresh

> Historical phase record. Current implementation/contracts and final evidence are consolidated in [the 2 November readiness review](PRODUCTION-READINESS-2026-11-02.md), [PLAN](../../docs/frontend/PLAN.md) and [VERIFICATION](../../docs/frontend/VERIFICATION.md). Earlier pending-work statements describe that phase.

User decision: 30 October 2026, Asia/Makassar. Adjust the app to BR-DAT-07: 30-second staff refresh, 15-second active Receptionist refresh, manual refresh and refresh on return/reconnection. Other audit gaps remain decision items.

## Plan before implementation

- Goal/users: staff see newly submitted visits and attendance changes from other tabs without reloading their operational screen.
- Scope: frontendx and its synthetic preview only. The future backend is `D:/oikn/VMS/backend`, which does not exist yet. No legacy backend contract is authoritative.
- Data/API: retain the current typed list repository and scoped booking query. No endpoint or dependency change. Refresh reuses the same query key and cancellation/deduplication.
- State ownership: query cache owns records, last successful update and request/error state; TanStack owns browser visibility/network state. URL owns filters/selected record. Form components retain drafts. No copied record state or per-widget timer.
- Component tree: one operational-route refresh component in the staff shell, using a polling option on the existing booking hook. Existing list/reception/dashboard components keep reading the shared query. QueryState keeps a previously successful result visible after a temporary refresh error; initial and access errors retain the error/retry UI.
- States: initial loading, updating, last success, offline, failed refresh with older data retained, successful recovery. A semantic time element uses WITA; a keyboard-operable refresh button and polite status text explain the state.
- Three performance questions: avoid separate widget requests by sharing one query/poll owner; defer polling outside operational routes and while hidden/offline; schedule requests in the query layer, leaving rendering and drafts in their existing owners.
- Pattern fit: existing TanStack Query already handles deduplication, retry, hidden-tab polling and reconnection. Add no polling framework, global store, server code, new visual asset or animation.
- Visual purpose: a compact wrapping row near the active page identifies data freshness and provides one explicit recovery action. Retain existing typography, colors and button states. ENERGY 1 / RHYTHM 1 / MOTION 1 for this operational control; no decorative motion.
- Build order: shared query options; one shell control; cached-error rendering; focused browser checks; typecheck/logic/integration/build; reconcile product/plan/design/readme/verification and corrected audit.
- Acceptance: 30s and 15s intervals; no duplicate list requests from extra observers; hidden-tab pause; immediate return/reconnect/manual refresh; retry without hiding prior data; preserved URL, scroll and unsaved form input; mobile/keyboard/accessibility coverage.

## Status

**Implemented and verified in the synthetic preview.** No new backend was created. Only the user-selected freshness behavior and the edit-version protection necessary for background updates were changed.

The shell mounts one refresh owner only on operational visit routes. It supplies the polling interval to the existing booking query, while page/widget/form observers share that query without starting intervals. TanStack's visibility/network managers pause hidden/offline polling and refetch on return/reconnection; the explicit manual action shares any in-flight request. A named section contains the WITA time, a 44px button and offline/stale feedback. A polite live region announces manual success/errors without repeatedly announcing routine successes.

QueryState retains a previously successful list after temporary errors, but hides it for access failures. An open decision/edit form retains the record/version it started from: background refresh cannot silently bless an outdated draft with a newer version. A concurrent-update notice appears on decision forms; unchanged server version checks remain authoritative.

## Verification

- PASS: type checking, 23 existing logic tests, 10 synthetic integration tests, both builds and 8 focused refresh browser tests.
- PASS: a second-client synthetic submission appeared at the 30-second poll; extra cache observers did not add a second polling request. Active Receptionist polling fired at 15 seconds.
- PASS: simulated hidden state paused requests; returning refreshed even recently fetched data. Manual keyboard activation, offline/reconnect recovery, failed-refresh retention and access-denied hiding behaved as expected.
- PASS: search/filter URL, an unsaved staff purpose and modal scroll survived a background refresh. A separate decision-draft test observed a newer record version, preserved its typed reason, submitted the original version and retained input after the simulated 409 conflict.
- PASS: scoped axe checks found no violations on the new refresh control in current/error states at 320px, 390px and 1280px. Document widths matched each viewport; the button measured 44px tall. Screenshots were inspected. Existing page-wide audit findings remain open.
- PASS: no new dependency, visual asset, timer per widget or copied server-record store. The staff entry is 113.00kB gzip; visitor entry remains 104.51kB gzip. These are payload measurements, not field speed claims.

Evidence: [accessibility/layout](evidence-2026-10-30/refresh/accessibility.json), [320px current](evidence-2026-10-30/refresh/staff-current-320.png), [320px failure](evidence-2026-10-30/refresh/stale-320.png), [browser tests](../tests/booking-refresh.browser.mjs).

Run the preview with `npm.cmd run dev`, then `npm.cmd run test:refresh` with Playwright available. This session used `PLAYWRIGHT_MODULE=file:///C:/Users/ASUS/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright/index.mjs`. Browser clock, visibility and network events were emulated; no physical-device, full screen-reader or new-backend acceptance is claimed. The older date-picker browser failures and other app-versus-requirement decisions remain separate.
