# Restricted visits and arrival timing

> Historical phase record. Current implementation/contracts and final evidence are consolidated in [the 2 November readiness review](PRODUCTION-READINESS-2026-11-02.md), [PLAN](../../docs/frontend/PLAN.md) and [VERIFICATION](../../docs/frontend/VERIFICATION.md). Earlier pending-work statements describe that phase.

User choices, 30 October 2026: adjust the app for G03 and G05. Scope is frontendx and its synthetic service. The future D:/oikn/VMS/backend remains unbuilt. Preserve all other gap decisions.

## Plan before implementation

- Goal/users: Protocol manages date/time plus four optional Terbatas fields; assigned PA and Super Admin read their permitted views; Receptionist receives redacted general records and credential-gated arrival detail. Public access and general reporting/export exclude Terbatas.
- Typed boundary: retain the full stored/ordinary visitor model; introduce an explicit staff view whose private fields may be absent, with restricted audience tags. Do not return admission codes in a general redacted reception view. Public visitor cancellation gets its own typed method instead of inheriting a staff response.
- State: query cache owns general scoped records and server-derived timing; reception owns the entered/scanned credential and temporary arrival view, clearing them on close. The form owns optional field drafts. Protocol actions preserve version checks. Permission/target changes recalculate the service projection before returning/searching/counting/exporting data.
- Components: replace the old restricted form with date/time plus optional Nama, Instansi, No ID Dinas and Pejabat yang dituju; retain shared field/date/time/dialog controls. Use a separate restricted identity presentation rather than rendering hidden ordinary-visit fields.
- API/service: no new real backend or dependency. Role projections are allowlists. A pure time policy takes an explicit current instant; the service owns that instant and returns display/attendance metadata. The stored six statuses remain unchanged. General exports use a shared filter that removes restricted records/events.
- Timing: check-in requires Approved and start minus 30 minutes <= now <= end. After end without check-in, display Missed and close admission/rescheduling; Checked-in visits can check out after end. Code and QR follow the same rule. Browser refresh updates displayed timing; service checks each mutation regardless of cached UI.
- UI states/a11y: absent optional values remain absent in payloads; UI can say not supplied without inventing identities/counts. Keep explicit physical-ID/available-detail verification and attendance confirmation, with no extra Protocol acknowledgment. Announce timing/denial/conflict feedback, retain input, return modal focus and verify narrow layouts.
- Three performance questions: avoid fetching full restricted records then hiding them; fetch arrival identity only after valid credentials; keep authoritative scope/time calculations in the synthetic service. Reuse existing shared query refresh instead of adding per-row clocks or request timers. No worker, global record copy or new library.
- Test strategy: deterministic pure clock boundary tests and isolated synthetic servers with injected clock/session fixtures; role/field/search/direct-ID/count/export tests; target/assignment changes and stale mutations; code/QR admission at boundaries; browser form, redacted reception, gated detail, read-only PA/admin, missed display and late checkout. Existing tests encoding old policies must be updated rather than retained as acceptance.
- Build order: pure timing/types/projections/schema; service enforcement; form/identity/reception/list/status/export integration; policy and API checks; rendered checks; typecheck/build/regressions; current docs and audit dispositions.
- Visual direction: preserve existing neutral IKN/Geist operational UI, ENERGY 1 / RHYTHM 1 / MOTION 1. No new visual asset or decorative animation. The service-ID maximum of 100 text characters is a preview validation limit, not a confirmed production identifier format; preserve leading zeroes.

## Status

**Implemented and verified in the synthetic preview, 30 October 2026.** G03 and G05 keep the requirements and adjust the app. No new backend project, dependency, asset, storage or delivery service was created.

## Changes and contract handoff

- `packages/data/src/attendance.ts` owns the pure WITA arrival policy. The service calls it with its own request time; list/detail/resolve/lookup responses carry `timing` metadata. UI renders that snapshot and existing staff refresh updates it. Visitor status updates on lookup/manual refresh. Each attendance mutation rechecks current service time regardless of a cached snapshot.
- `domain.ts` separates stored records, ordinary visitor responses and optional-field staff projections. Restricted projections are explicit allowlists. Assigned PA access follows the current target and assignment before list/detail/resolve/actions. The public projection refuses Terbatas. Production must repeat these checks before query filtering, pagination and counts.
- The strict restricted schema permits date/time and four optional fields only. Protocol creation/editing preserves leading zeroes and accepts blank optional values; editing explicitly clears them. Legacy contact, purpose, roster, identity-type and attachment fields are rejected for this kind. The preview uses a 100-character text limit for service ID; no production identifier format was selected.
- `RestrictedAppointment` owns the form, `RestrictedIdentity` renders the projected audience, and Reception owns only its current credential/arrival view. No identity is recovered through a general reference or record ID. Private details clear on close, denied/changed refreshed data, and rejected stale attendance confirmation. A newer code-resolution timing snapshot takes precedence over an older list snapshot.
- Staff status badges, filters, dashboard arrivals and visitor tickets consume the shared timing. Missed keeps raw Approved and its version unchanged, hides entry QR and disables rescheduling/check-in. Check-out lists include earlier dates for visitors still Checked in. A clock icon and text identify Missed without relying only on color.
- `generalReportRecords` excludes restricted records before preview analytics, log rendering and CSV creation. This closes G03 exclusion in existing surfaces; G08's independent summary contract, summary export and zero/error states remain open.

| Preview operation | Contract for the future backend |
| --- | --- |
| Staff create/update | Authenticate active staff and enforce Protocol authority. Restricted input: visitKind/date/time plus optional name/organization/serviceId/officerId. Create uses payload-bound idempotency; updates use version and the existing schedule. |
| Staff list/detail | Return the permitted projection, with no hidden fields or admission code in general reception responses. Recalculate target/assignment scope for every request. |
| Resolve arrival | Validate code/QR and lifecycle scope before returning name, target and last-four ID. No organization/full ID for reception. General reference/ID is insufficient. |
| Check-in/out | Require permission, preceding lifecycle, matching credential and version; compare authoritative time on every check-in. Prevent duplicate concurrent attendance atomically. Checkout remains available for Checked-in visits after end. |
| Public lookup/cancel | Email plus code under G02; never expose Terbatas. Cancellation cutoff remains pending G15 and was not changed. |
| Reports/export | Exclude restricted records/events in the actual report query/serializer. Separate summary/report authorization still needs G08 implementation. |

These are tested preview contracts, not a production HTTP-path/authentication design. The preview's query-string role fixtures and counter-based demo codes must not become production authentication or credential issuance. The HTTP repository still fails explicitly until the new backend is implemented and connected.

## Verification

- Typecheck and both production builds pass after the final source changes.
- All **49 logic tests** and **18 isolated synthetic API tests** pass. Exact cutoffs include one millisecond before opening, opening, scheduled end and one millisecond after end. Cases include optional clearing, leading zeroes, rejection of legacy fields, read-only PA/Admin, changed assignment, hidden direct IDs, code/QR projections, private-field omission, stale versions and next-day checkout.
- All **5 `test:restricted-arrival` browser tests** pass in local headless Chrome: 320px four-field create/edit/dirty-close/focus; 390px PA and 1280px Admin reads; CSV/log exclusion with an ordinary control record; redacted general reception then credential-gated verification; early denial, opening-time confirmation and next-day checkout; stale confirmation clearing; staff Missed filtering and visitor Missed ticket without QR.
- The existing reservation browser flow passes; the refresh suite passes **8/8 on its separate rerun**. One combined browser run observed an extra visibility-related request in the hidden-tab assertion (7/8 refresh checks); no polling implementation change was made to conceal it. Separate refresh execution passed all eight checks.
- Rendered accessibility-tree snapshots and screenshots were inspected for the changed views. The checks exercise keyboard submit/Escape, dirty-draft recovery, returned opener focus, disabled early confirmation and viewport overflow. They are not a full WCAG/screen-reader audit. No new automated axe/Lighthouse report was produced for this follow-up; the broader audit's existing accessibility/date-picker findings remain open.
- Staff entry is **113.74kB gzip** (previously 113.00); visitor entry **104.53kB gzip** (104.51). The restricted form is lazy-loaded at **2.04kB gzip**. No per-row timer, extra polling owner, worker or dependency was added.

Evidence: `anti-slop/evidence-2026-10-30/restricted-arrival/` contains form, PA/Admin, reception and Missed captures plus accessibility-tree snapshots. Test files are `tests/attendance.test.ts`, `tests/restricted.test.ts`, `tests/restricted.integration.mjs` and `tests/restricted-arrival.browser.mjs`. API tests start isolated loopback servers with injected clock/session fixtures; there is no test-clock HTTP endpoint.

Run `npm.cmd test`, `npm.cmd run test:preview`, `npm.cmd run typecheck` and `npm.cmd run build` from frontendx. For rendered checks, start `npm.cmd run dev` and run `npm.cmd run test:restricted-arrival` with Playwright available (or PLAYWRIGHT_MODULE pointing to the local module). Physical devices/Safari, real sessions, persistence, delivery, document storage, production credentials and the new backend remain unverified/unbuilt.
