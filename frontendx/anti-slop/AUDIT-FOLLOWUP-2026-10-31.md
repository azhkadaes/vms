# Frontend audit follow-up and backend readiness

> Historical audit snapshot. Current implementation, disposition and evidence are in [the 2 November readiness review](PRODUCTION-READINESS-2026-11-02.md) and [verification](../../docs/frontend/VERIFICATION.md). Older reproductions are not current open findings where the new review records a fix.

Scope: only `D:/oikn/VMS/frontendx`. The future `D:/oikn/VMS/backend` was confirmed absent on 31 October. Other frontend/backend repositories supply no implementation or readiness evidence for this project.

## Verdict

The frontend has useful component, shared-control, schema, query and repository boundaries. The selected business gaps are now resolved in the synthetic preview or explicitly reconciled/deferred. It is not yet fully modular, performance-optimized or ready for a live backend. The remaining technical findings below still need work; a green build does not close them.

## User decisions carried through

| Disposition | Gaps and resulting behavior |
| --- | --- |
| App changed and preview-verified | G01 shared staff/reception freshness; G03 revised restricted fields/privacy; G04 reservation on approval; G05 arrival window/Missed; G06 complete public validation; G07 payload-bound submission/recovery; G08 authorized aggregate reporting; G10 responsive employee directory/search; G12 typed localized public history; G13 active heading/focus behavior; G15 strict visitor cancellation cutoff. |
| Requirements changed | G02 email plus code suffices for eligible public access; G09 one server-assigned active role per account; G11 current visitor choice-card/Back navigation and responsive language placement retained. |
| Deliberately deferred | G14 optional same-tab draft recovery remains planned for a later task. It is distinct from the implemented pending-submission-reference recovery. |

All 15 choices are recorded in [REQUIREMENT-GAPS](REQUIREMENT-GAPS-2026-10-30.md). Production authentication, storage, notification and database guarantees are not implemented by these preview fixes.

## Prioritized technical work still open

| Priority | Finding | Why it matters and concrete next work |
| --- | --- | --- |
| High | A2-05/A2-06: production session and HTTP boundary | Replace provisional first-array-role/default-role behavior with one validated canonical role from the new backend. Handle session expiry/denial, scoped-cache clearing and malformed responses consistently. The current production workflow repository remains explicitly unavailable. |
| High | A2-13: dependency advisory recorded in the audit | Execute the separately planned Zod upgrade after checking the current advisory/version, preserving schema/resolver/error behavior and rerunning validation/API/build checks. This follow-up did not change dependencies or perform a fresh advisory lookup. |
| High | A2-10 and remaining accessibility checks | Correct language/Brand/detail accessible names so they include visible wording. Recheck the shared date controls, other scrolling tables, actual browser zoom and screen-reader behavior. G10/G13 fix their selected reflow/heading/focus scope only. |
| High | A2-12: reproducible verification | New policy/API/browser coverage addresses many old gaps, but the older date-picker browser suite still has stale expectations and unresolved failures. Declare browser tooling and provide reproducible CI checks; do not treat the entire app as accepted from selected suites. |
| Medium | A2-14: startup cost | The phone choice screen still mounts hidden registration and loads its form/validation code early. Mount on first intent, then retain the instance for input continuity; improve discovery/responsive delivery of the existing hero and measure the motion runtime. No new field-speed or whole-app Lighthouse result is claimed here. |
| Medium | A2-01/A2-15: maintainability and API scale | Continue splitting data contracts/HTTP/session/query owners, shared UI exports and large booking/create form responsibilities. Replace fixture reference data with typed catalogs and define bounded server filters/pagination/detail queries. Avoid speculative abstraction, blanket memoization or workers for the current tiny fixtures. |
| Low | A2-16: comments and consistency | Finish the comment-only vendor-control cleanup, preserving source/license/constraints. Place the preview notice in an appropriate landmark and complete missing states in other lists. |

The [corrected audit](audit-002-2026-10-30.md) retains original evidence with follow-up dispositions. Its older reproductions are historical where marked superseded. These remaining issues are not silently classified as fixed by the user’s requirement choices.

## Backend readiness

The preview is a useful executable specification. Its production adapter is not connected, and no backend was created in this work. The implementation language, database schema and final API contracts remain to be selected for the new project.

Before integration, define authenticated single-role sessions and assignments; audience-specific public/staff/restricted projections; bounded catalogs/lists/details; aggregate reporting and independent read/export permissions; typed validation/conflict errors; durable payload-bound submission attempts and outcome resolution. The public history contract now carries action/time and explicitly public rescheduling text, with private actor/internal information kept separately.

Database transactions must enforce resource conflicts, version checks, arrival/cancellation deadlines and idempotency under real concurrency. Couple state changes with audit and notification intent, invalidate admission credentials on relevant changes, and specify authorized/audited late staff exceptions. An in-memory fixture cannot establish these guarantees. Real protected files, email delivery, rate limiting, credentials, identity/retention policy and operational/deployment evidence remain required before activation.

## Verification and delivery boundary

- Latest changes pass typecheck, 82 logic tests, 31 isolated synthetic API cases and both production builds. Seven new public-history/heading/cancellation browser cases plus 12 submission/restricted-arrival and 9 refresh/reservation regressions pass. The G10 phase passed 3 directory and 5 report browser cases. These are scoped checks across phases, not a full device/browser suite.
- Rendered checks include 320/390/768/1280px where applicable, long fields, localized public events, WITA timestamps, logical headings, keyboard focus and deadline/conflict behavior. Screenshots/accessibility trees are in `evidence-2026-10-31/`.
- Final staff/visitor entry sizes are 114.36/105.08kB gzip; Status is lazy-loaded at 5.95kB. No new dependency or visual asset. These sizes do not establish real-user latency or resolve A2-14.
- No real backend, database, email, upload, admission credential, physical camera/device, Safari, NVDA, actual 200% browser zoom or new whole-app axe/Lighthouse acceptance was verified.
- Scoped delivery gates are recorded in [directory/navigation](DIRECTORY-NAVIGATION-2026-10-30.md) and [public history/headings/cancellation](PUBLIC-HISTORY-HEADINGS-2026-10-31.md). Existing institutional identity and controls are preserved; open whole-app findings are explicit.
