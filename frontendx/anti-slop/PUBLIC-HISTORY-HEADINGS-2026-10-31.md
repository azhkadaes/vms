# Public history and visitor headings

> Historical phase record. Current implementation/contracts and final evidence are consolidated in [the 2 November readiness review](PRODUCTION-READINESS-2026-11-02.md), [PLAN](../../docs/frontend/PLAN.md) and [VERIFICATION](../../docs/frontend/VERIFICATION.md). Earlier pending-work statements describe that phase.

User decisions on 31 October: G12 and G13 adjust the app. Scope is frontendx. G11 choice-card/Back navigation and responsive language placement stay as they are; the new backend remains unbuilt.

Subsequent decisions: G14 draft recovery stays planned for a later task; G15 adjusts the app to enforce visitor cancellation strictly before start.

## Plan before implementation

- G12: introduce a typed public action/time history that is separate from stored staff messages. Project an explicit allowlist for create, replay/recovery, lookup and cancellation. New synthetic events carry an action code; unknown/legacy events receive a generic localized update label rather than exposing stored text. Keep public staff-written reasons verbatim in designated public fields; staff attribution/internal notes remain in staff projections. WITA timestamps use the service clock and semantic time elements.
- State and components: server owns event codes/timestamps; LanguageProvider owns ID/EN; a small public-history component renders labels and times. Derive localized labels rather than storing translated copies. PublicBooking is separate from stored/staff history types. No new dependency, request or global state.
- G13: separate semantic heading level from embedded presentation. The desktop landing keeps its h1 with an inline h2 task; an expanded phone task or standalone page starts at h1. Form sections, roster, summary, recovery, rejection, history and ticket headings follow their parent. Preserve type sizes with classes/selectors rather than semantic tag defaults.
- Focus: initial successful lookup focuses a named result heading; refresh preserves keyboard position, checking another visit focuses the code input, and cancellation keeps the existing modal focus behavior. Preserve the same form instance and input continuity. Preparation content remains intentional supporting information after registration.
- G15: share a pure cancellation cutoff policy between the synthetic service and visitor UI. Enforce it on every cancellation request using service time, including an already-open confirmation. The UI uses the returned service clock and a local deadline timer without polling; after cutoff it explains the limit and disables cancellation. If its trigger becomes disabled while a dialog is open, return focus to Update. Do not invent staff-exception permissions or a staff cancellation endpoint; those require the future authenticated backend contract.
- Tests: public response projections across lifecycle/retry/recovery contain only action/time, unknown events cannot leak raw metadata, staff public reasons remain intact, and event order/times/localization are checked. Browser review covers ID/EN, phone expanded/standalone/desktop inline headings, lookup/check-another/dialog focus, no overflow and retained navigation/input. Run appropriate logic/API, submission/privacy and build regressions. No field performance, actual browser zoom, physical-device or screen-reader certification is implied.

## Status

Completed for G12/G13 and the visitor portion of G15 in the synthetic preview. G14 is explicitly deferred. The staff-exception path and all real backend guarantees remain unbuilt.

`PublicBooking` now has a separate `PublicVisitEvent` history, while stored/staff messages retain their existing type. The visitor projection emits an action and timestamp; only explicitly public rescheduling text is included as publicMessage. The public rejection reason remains in its dedicated field. Unknown/legacy action codes become a generic update instead of displaying raw messages. Public creation, replay, outcome recovery, lookup and cancellation use that projection. New preview events use the injected service clock. PublicVisitHistory owns ID/EN labels, semantic time elements and invalid-time/empty feedback. It makes no request and keeps no duplicate translation state.

Presentation no longer determines heading level. The landing h1 remains with inline h2 tasks on desktop; standalone and expanded phone tasks start at h1. Forms, roster, recovery, summary, ticket, rejection and history follow their parent. CSS keeps the existing sizes. Initial lookup focuses the result heading, refresh restores its button when temporary disabling dropped focus, and check-another focuses the retained code input. Back/input continuity is preserved.

`canVisitorCancel` is shared by service metadata and the UI. Each service cancellation request checks the pair, status/version and strict before-start cutoff before mutation. The local UI deadline uses the returned service clock plus elapsed time, with one bounded timer/focus update and no polling. It disables an already-open confirmation at cutoff. A server conflict refreshes the public record once, preserving the error; no cancellation is retried. The modal returns focus to Update if the old trigger is disabled/removed. Device wall-clock skew cannot bypass the service check. Response latency and suspended timers can delay visual feedback; service time remains authoritative.

## Verification

- Typecheck, all 82 logic tests, all 31 synthetic API cases and both production builds pass.
- Seven focused public-history/heading/cancellation browser cases pass in Chrome, covering expanded/standalone/inline headings, retained Back/input, ID/EN labels, WITA despite another browser timezone, public reasons, unknown/invalid event data, result/refresh/check-another focus, successful cancellation, a stale confirmation and an open-dialog cutoff despite an incorrect device clock.
- All 12 affected submission and restricted/arrival browser cases pass, plus all 9 freshness/reservation browser cases. The G10 phase also passed its 3 directory and 5 report cases; these are phase-specific checks, not a claim of a full browser/device suite.
- Screenshots and accessibility trees in `anti-slop/evidence-2026-10-31/public-history/` were inspected, including the scrolled history section at 320px. No document overflow or uncaught page errors were observed in the changed cases. Earlier failing captures are debugging history: refresh focus was fixed; an inert background button required the test to query hidden markup while its modal was open.
- Final staff entry: 114.36kB gzip; visitor entry: 105.08kB; lazy Status: 5.95kB; lazy Booking: 41.91kB. No dependency or asset was added. Existing eager hidden-registration/LCP concerns remain open under A2-14; these build sizes are not field-speed measurements.
- Actual 200% browser zoom, physical Safari/camera, NVDA/full screen-reader acceptance, production API/database/email/files/credentials and fresh whole-app axe/Lighthouse checks were not performed.

## Scoped delivery gate

| Check | Status and evidence for this change |
| --- | --- |
| R-02 copy | PASS: new action/deadline copy uses plain localized sentences; no new decorative punctuation. |
| R-03 layout | PASS: 320/390/768/1280px checks and scrolled phone history fit; long public text wraps. |
| R-17/R-18 statistics/testimonials | PASS: none added; test people/data remain explicitly synthetic. |
| R-23 assets | PASS: no visual asset or navigation structure added; existing identity retained. |
| R-24 navigation | PASS: Back/choice links work and retain form input; G11 is preserved. |
| R-25 contrast | PASS: changed text reuses existing dark/navy/muted tokens on opaque/pale surfaces; no new foreground/background pair. |
| R-26 controls | PASS: lookup, update, language, check-another, Back and cancellation/keep controls exercised; cutoff disables the action with an explanation. |
| R-27 states | PASS: loading/error/empty history/unknown-time/confirmation/cutoff/success feedback exercised; submitted data survives a conflict. |
| R-28 FAQ | PASS: no FAQ introduced. |
| R-32 keyboard | PASS: named result focus, update focus restoration, code-input return and modal return verified. |
| R-33 source | PASS: behavior is implemented directly in typed source modules, using existing architecture and controls. |
| R-34 themes | PASS: no theme behavior changed. |
| R-35 runtime | PASS: typecheck/build, 82 logic/31 API and focused rendered cases pass as listed. |
| R-36 claims | PASS: preview/backend and lab/device limits are explicit; no production claim. |
| R-37 direction | PASS: existing DESIGN.md institutional identity, typography and mobile task surface retained. |
| R-38 fabricated content | PASS: only labeled fictional test/preview records; no testimonial or live claim. |
| R-39 semantics | PASS for changed headings/time/controls: semantic level tests, keyboard focus and shared control states. Existing language/Brand accessible-name and preview-landmark findings remain open. |
| R-40 cost/media | PASS: no asset, dependency or polling; history stays in the existing lazy Status route. Build sizes above are measured, not speed claims. |
| Purpose gate | PASS: history hierarchy supports scanning; existing functional icons/status colors remain. No gradient, decorative motion, illustration or new glass. |
| Liveliness | PASS: ENERGY 1 / RHYTHM 1 / MOTION 1 follows the existing operational direction; the ticket/status, history and action hierarchy retain OIKN identity. |
| Craftsmanship | PASS: stored/public histories are separate; a shared cutoff policy replaces divergent eligibility; transient focus/time state stays local. New comments explain constraints without narration. This does not certify all existing app modules. |

## New-backend contract handoff

The future backend must persist typed event codes and timezone-aware occurrence timestamps separately from actor identity/internal notes. Public responses use audience allowlists on every read/replay/recovery/mutation, never translated or raw audit strings. Persist explicit public rescheduling reasons separately and retain public rejection copy without translation. Unknown events need safe forward-compatible behavior.

Cancellation must atomically recheck the current schedule, status, credential authority, version and server time before start; revoke active admission credentials, release reservations and persist audit/notification intent in that transaction. Later staff exceptions need explicit permissions, permitted transitions and audit attribution; no staff endpoint or permission is invented by this preview fix. Pair authorization/rate limiting, real notifications, database concurrency and delivery retry behavior remain backend work. `D:/oikn/VMS/backend` was confirmed absent on 31 October.
