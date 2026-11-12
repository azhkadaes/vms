# Approved audit implementation plan

31 October current disposition: G10 directory and G12/G13/G15 public-history/heading/focus/cancellation changes are verified in preview. G11 retains the current choice-card/Back navigation; G14 draft recovery stays planned for later. See DIRECTORY-NAVIGATION-2026-10-30.md and PUBLIC-HISTORY-HEADINGS-2026-10-31.md. These dispositions supersede older milestone wording below; technical findings and production integration remain open.

30 October G09 requirement revision: **one active role per account** replaces the multiple-assigned-role workspace-switch plan. The server assigns the active role and its permissions/officer scope; staff do not combine or self-switch production roles. An authorized administrative reassignment refreshes/invalidates the session and scoped caches, and actions retain role attribution. The development-only test-role selector remains a fixture tool. The new backend/session adapter is not implemented or certified by this requirement change; it must expose one canonical role rather than infer access from array order.

30 October scope update: `D:/oikn/VMS/frontendx` is the only audited application. Its backend will be a new project at `D:/oikn/VMS/backend`, not yet created. Older backend comparisons below are historical and do not define its implementation. Collect the user's app-versus-requirement direction for each gap in [the decision sheet](REQUIREMENT-GAPS-2026-10-30.md). G01 freshness is verified. G03 revised Terbatas and G05 arrival timing were subsequently selected for app changes and verified in preview; see RESTRICTED-ARRIVAL-2026-10-30.md. The user subsequently chose G04 app alignment (Pending overlap, resource checks on approval) and G02 requirement alignment (email/code sufficient for eligible QR/cancellation). These decisions supersede the older email-proof milestone below; other gaps remain open.

Decision date: 29 October 2026, Asia/Makassar. Source: the user's individual review of findings 1-16 in [audit 001](audit-001-2026-10-29.md), including the final approval of finding 16.

**Current status: G01/G03/G04/G05/G06/G07/G08 verified in preview; G02/G09 requirements revised on 30 October. Other application changes remain pending. G08 app summaries and G09 requirement revision to one active role per account were subsequently selected. G08 is implemented/verified in preview; G09 documentation is reconciled.** This document consolidates the agreed behavior and implementation order. It does not mark any finding fixed or establish production readiness.

## Objective and boundaries

Build the active frontend prototype toward a clear contract for the new, unbuilt D:/oikn/VMS/backend project. Preserve the agreed visitor/staff behavior and make the prototype's contracts and enforcement realistic. Backend activation, real email, real uploads and admission credentials belong to later integration milestones.

- Frontend: `D:/oikn/VMS/frontendx`, with visitor and staff apps plus shared UI/data packages.
- Future backend: new project at `D:/oikn/VMS/backend`, not yet created. Implementation language and contracts are not selected by this frontend audit.
- Legacy `D:/oikn/VMS-FE/monorepo` remains a reference, outside these frontend fixes.
- Retain the existing IKN identity, shared styling, mobile full-screen task surface, Back behavior and input continuity. No replacement UI framework, motion framework or visual assets are needed.
- The preview stays explicitly synthetic and loopback/development-only. A simulated verification, demo QR, filename or role fixture is not evidence of a real email, admission credential, stored document or staff authorization.
- First feature milestone remains the revised Terbatas behavior. The focused dependency upgrade is preparatory work rather than a different feature priority.

## Approved decisions and acceptance checks

| Finding | Confirmed decision | Acceptance before closing the finding |
| --- | --- | --- |
| 1 | Frontend first, designed for real backend alignment over time. | Define explicit typed contracts and an alignment matrix. Components use replaceable adapters. Unsupported production operations remain explicit until their real integration passes. No claim that a prototype fix closes the production blocker. |
| 2 | Revised 30 October: email plus visit code is sufficient for visitor status, QR and cancellation when eligible. No additional email-link proof or OTP. | Match both credentials; reject missing/wrong pairs; retain status/version/privacy checks. Production QR remains server-issued and revocable. Public Terbatas access stays denied. Cancellation cutoff is the separate G15 decision. |
| 3 | Adopt the revised Terbatas form, role projections and export exclusions. | Test the full role matrix below, absent optional values, changed/blank targets, assignment changes, direct links, counts/search, gated arrival verification and general-export exclusion. |
| 4 | Poll lists/dashboards every 30 seconds and active reception screens every 15 seconds; refresh on focus/reconnection and offer manual refresh. | No polling in hidden tabs. Requests are shared/deduplicated. Preserve filters, scroll and unsaved input. Show last update and stale/error feedback; a failed refresh retains clearly marked older records. |
| 5 | Pending is a request; slots are reserved on approval. | Multiple Pending requests may overlap. Approval, rescheduling and resource/duration changes check current Approved/Checked-in commitments. Conflicts involving either the same pejabat or the same nonempty room are rejected. Adjacent intervals are allowed. Competing approvals cannot both reserve a conflicting slot. |
| 6 | Check-in opens 30 minutes early and remains available through the scheduled end, including arrivals after the start. | Compare authoritative time with `start - 30 minutes <= now <= end`, in WITA for display. After `end`, Approved visits without check-in show Missed. An already Checked-in visit can check out after its scheduled end. Code and QR follow the same rule. |
| 7 | Enforce current form requirements consistently in client, synthetic server and future backend. | Reject impossible/past schedules, invalid contact data, inactive/unknown destinations, incomplete rosters and out-of-limit supplied values. Optional fields remain optional. Safe normalization preserves identifiers/leading zeroes and never silently changes a schedule. Return field errors without losing input. |
| 8 | Use payload-bound submission references; retry once automatically for a temporary network failure. | Unchanged retries return one booking and one logical submission/notification event. Changed-payload reuse conflicts. No automatic retry for validation, authorization or business conflicts. Resolve an uncertain prior attempt before submitting edited details as a new attempt. Prevent repeated in-flight submissions. |
| 9 | Upgrade Zod in a focused change with regression checks. | Planned target is 4.6.7, checked during the decision review. Recheck advisory/package state at execution. Update only necessary compatibility code, retain agreed validation/messages, run type checks, logic/API/form checks, both builds and dependency audit. |
| 10 | Employee cards on phones; desktop table. | Name, email and role remain readable with long-text wrapping. Stack narrow-screen heading/actions and place search below the directory heading. No document overflow or sideways focus scrolling at 320px, 390px or 200% zoom. |
| 11 | Start with summary reports and summary export. | Reports have a separate contract/permission scope, date range and status counts. Super Admin can access authorized organizational summaries independently of ordinary record access. Other users require reporting permission and receive their authorized scope. Zero follows a successful zero result, not an error or missing permission. General reports/exports exclude Terbatas. Detailed visitor exports are deferred. |
| 12 | Revised 30 October (G09): one active role per account. | Use one canonical server-assigned role with its permissions/officer assignments. Do not offer a production multi-role switch or union of permissions. Administrative role changes refresh/invalidate session and private caches; staff mutations retain role attribution. Validate the role/permissions on every backend operation. |
| 13 | Use consistent loading, empty, no-match, error and access-denied states wherever list pages have gaps. | Empty directory offers an authorized next action; filtered zero results offer clear search/reset. Failures offer retry with filters/input preserved. Result changes are announced accessibly. Do not rewrite screens whose existing states already satisfy these checks. |
| 14 | G11 revised, 31 October: retain current choice-card/Back navigation. | Preserve the current responsive language placement, expanded-task toolbar, background isolation and input continuity. Direct registration/status cross-links are not required. Accessible control names remain a separate technical issue. |
| 15 | Visitor history contains localized actions and WITA timestamps only. | Typed public events render Indonesian/English system messages. Staff-written public reasons retain their original wording. Internal notes and detailed actor/role attribution stay in authorized staff audit views. New/future events cannot leak private metadata through a fallback string. |
| 16 | Correct the active page/task heading structure. | One appropriate main heading for each active task/page, logical section/subsection order, and styling independent of heading level. Verify mobile expanded, dedicated and desktop embedded routes without duplicate or skipped active heading levels. |

Current prototype limits remain documented prototype values, not newly approved production retention or capacity policies. In particular, the ordinary roster limit of 20 and optional document validation at 5 MB remain the existing preview limits pending production confirmation.

## Terbatas role and data contract

The restricted form requires date and start time only. Its four optional fields are Nama, Instansi, No ID Dinas and Pejabat yang dituju. No ID Dinas is text; preserve leading zeroes. Omitted values stay absent. Any active pejabat can be selected; a blank target is valid.

| Workspace | Allowed view/actions |
| --- | --- |
| Protocol | Full authorized details; create, edit, approve, reject and reschedule. VIP creation remains limited to Kepala Otorita. |
| Assigned PA | Read a restricted visit only when its current target pejabat is in the PA's current assignment. May see name, organization, target and only the last four ID characters. No Terbatas editing or decisions. Blank target grants no PA access. |
| Super Admin | Full Terbatas read access, including the supplied ID, without edit/decision rights through the administrative workspace alone. This does not imply broad access to ordinary visit records. |
| Receptionist, general views | Only Approved, Checked in and Checked out. Reference, schedule, status and attendance controls; no name, organization, ID or target pejabat in the general list/detail projection. |
| Receptionist, credential-verified arrival | After valid code/QR resolution, may additionally see supplied name, last four ID characters and target pejabat for attendance verification. Organization and full ID remain hidden. Retain the existing explicit check-in/out verification step; add no separate Protocol acknowledgment. |
| Public/visitor | No Terbatas discovery, lookup, document, credential or action access. |

Apply projections before returning data, filtering search, computing counts or creating exports. General references must not themselves grant the credential-verified projection; do not return arrival credentials inside the general redacted payload. Each account receives only its single active role and authorized permissions. The combined-role/selected-workspace plan was superseded by G09.

General reports and general visit/audit exports receive no Terbatas records or events. Preserve restricted audit events internally for their permitted audience; a separate restricted-export feature is not part of this plan.

## Contract, state and component ownership

| Concern | Owner and required boundary |
| --- | --- |
| Staff identity and workspaces | Session contract exposes authenticated staff identity, one active role and its capabilities/officer assignments. The backend is authoritative; client preference cannot grant a role. Preview fixtures represent assignments explicitly. |
| Visits and summaries | Shared data layer owns typed queries, scoped query keys, cancellation, deduplication and mutation invalidation. Production supports bounded page/summary requests instead of fetching all history repeatedly. |
| Public versus restricted projections | Separate minimal response contracts for pair-authorized visitor status/actions, general staff views, restricted read views and credential-verified arrival detail. An optional field on a large shared object is not sufficient privacy isolation. |
| Form drafts | Existing form ownership remains local to the form. Preserve input on field/network errors and background refresh. Workspace changes prompt before abandoning an unsaved form. |
| Submission attempts | Request layer retains an attempt reference and normalized payload for safe replay. Server persists its fingerprint and result atomically with the logical submission. The reference is not an authentication credential. |
| URL state | Search, filters, sort, page and selected detail retain the existing URL ownership. Reset only the affected selections when scope changes. |
| Reporting | Separate reporting adapter/query contract and summary export model. An empty booking-list cache cannot stand in for a reporting result. |
| History | Public typed event kind + time + allowlisted public metadata; separate staff audit projection with actor and active workspace. Localized UI text is derived from the event kind. |
| Visual feedback | Reuse shared field errors, buttons, dialogs, QueryState/Empty and existing tokens. Add employee mobile cards within the existing directory feature. Shared fixes are limited to missing states and confirmed responsive/semantic problems. |

The production boundary remains the repository/adapter layer. Prototype types describe agreed operations and projections; a reviewed wire contract must map actual server IDs, timestamps, statuses, permissions and error envelopes. Do not copy synthetic role parameters, fixed officer IDs or client verification flags into production authorization.

Required request outcomes include success, field validation, unauthenticated, forbidden, unavailable/not-found, stale-version conflict, resource conflict and temporary network failure. Keep error messages actionable and protect record existence where privacy requires generic unavailable responses.

## Ordered implementation milestones

### 0. Preserve the baseline and reconcile contracts

Findings: 1, with the accepted rules from 2-8 and 11-12.

Inspect the current checkout, listeners, dependency lock and existing local changes before editing. Capture targeted source/package versions for rollback. This frontend currently has no Git metadata; do not imply that a commit exists or initialize/move repositories silently.

Record the session/workspace, visit projections, validation, pair-authorized visitor actions, idempotency, scheduling, history and report contracts. Map which operations exist only in the preview and which contracts/implementation the new backend still needs. Keep the established `YYYY-MM-DD` / `HH:mm` UI contracts and WITA display; use explicit time zones at integration boundaries.

Exit evidence: approved decision coverage for all 16 findings, typed-contract change inventory, and baseline verification appropriate to the files about to change. Baseline evidence is not evidence of a fix.

### 1. Perform the isolated Zod upgrade

Finding: 9.

Update the dependency/lock and the minimum resolver/schema/type/error-mapping compatibility needed for the patched version. Preserve business rules. Existing form resolver source already supports Zod 4, but application compatibility still requires validation. This milestone does not add a new form library or change the visual design.

Exit evidence: dependency audit disposition, type check, existing logic and preview integration suites, both builds, and focused browser checks for standard/optional field validation, contact input, rejection messages and Terbatas defaults. Test meaningful boundary behavior rather than new tests for CSS or implementation details.

Rollback: restore the targeted dependency/compatibility snapshot if required, leaving the advisory explicitly open. A rollback is not permission to release an affected dependency.

### 2. Deliver restricted privacy with one active staff role

Findings: 3 and 12.

First implement the restricted data contracts and preview enforcement, then the field set and permitted read/arrival surfaces. Make PA and Super Admin reads explicit without granting decisions. The multi-workspace model was superseded by G09. Keep one active role per account; retain the visibly distinct development-only test-role selector.

Update obsolete tests asserting the older Protocol/Receptionist-only policy. Add positive and negative cases for every projection, field masking, inactive/blank/changed target, revoked assignment, direct ID, counts, general export, stale cache and role switching. Do not use client hiding as the only test of privacy.

Exit evidence: the full role matrix passes at the synthetic API and rendered UI, forbidden data is absent from response payloads, unsaved forms survive canceled switches, and restricted detail disappears when scope is lost. Real server enforcement remains an integration item.

### 3. Preserve pair-authorized visitor access

Finding 2 disposition, revised by the user on 30 October: adjust the requirement to the existing email-plus-visit-code model. This supersedes the earlier extra email-possession-proof, expired-link and resend-link milestone. Do not introduce a proof-link or OTP screen for this model.

Both credentials authorize only the permitted visitor projection and eligible actions on that visit. Retain status/version checks, generic wrong-credential feedback, private/internal/restricted exclusions, and the distinction between preview QR and server-issued revocable production QR. Notification emails remain required where the selected workflow calls for them; they are not an additional proof step.

Exit evidence: correct pairs access visitor status and eligible QR/cancellation; missing/wrong single credentials fail; ineligible/restricted visits do not gain actions. Existing synthetic tests cover pair lookup/cancellation. The new backend must implement real QR issuance/enforcement. Cancellation timing remains the separate G15 decision and is not changed here.
### 4. Bring submission, scheduling and attendance behavior into parity

Findings: 5-8.

Use matching validation building blocks with request-specific client/server schemas. Enforce the current requirements on public creation as well as staff creation. Keep field errors associated with their visible step; return the user to a step containing an error without clearing other input.

Bind public submission references to normalized payloads. Keep a stable reference for the single automatic temporary-network retry, prevent duplicate in-flight requests and handle uncertain outcomes. Do not automatically retry rejected validation, permission or resource/version conflicts.

Apply the resource overlap policy on approval and relevant updates. Model one winning conflicting approval even when two attempts are submitted. Pending records do not reserve a pejabat/room. Empty optional room/target values create no fabricated resource reservation; a supplied resource still participates in conflict checks.

Use the 30-minute-early admission rule, allow late arrival until the scheduled end, and expose authoritative eligibility/reason fields for the reception UI. After the end, show Missed only for an Approved visit that never checked in. Keep checkout available after the end for an already Checked-in visit. Missed remains a display condition within the six-status lifecycle.

Exit evidence: invalid calendar dates and past/invalid times fail; optional fields remain optional; changed-payload replay conflicts; network-loss replay creates one logical booking/event; same-resource overlap fails; different-resource and adjacent visits pass; exact early/start/end boundaries and after-end behavior pass for both code and QR paths.

The future Go implementation must use database concurrency controls for real approval/attendance/idempotency guarantees. A single-process fixture test does not prove database safety.

### 5. Add freshness and complete shared list feedback

Findings: 4 and 13.

Add visibility-aware 30-second list/dashboard and 15-second reception refresh, focus/reconnection refresh and a manual control. Share requests/cache rather than polling separately for every widget or row. Later production requests are paginated/filtered with dedicated summaries. Preserve current results on background failures, identify them as stale and retain a clear last-successful-update time.

Complete missing loading, empty, zero-result, retry and permission feedback across affected lists. Use a successful empty result to show an empty state; never treat an error or unknown permission state as an empty directory.

Exit evidence: a request submitted from a second tab appears without reload within the configured interval; hidden-tab polling pauses; manual/focus/reconnect refresh work; a failed refresh retains marked older data; filters, selected detail, scroll and form drafts are preserved or safely dismissed if authorization is lost.

### 6. Complete responsive navigation, cards, history and headings

Findings: 10 and 14-16; reuse the states from milestone 5.

G10 is implemented and verified for the local employee preview: cards/table share the same filtered records, narrow header/search stacks, long fields wrap, URL search has no-match/reset feedback and dialog/reset focus returns correctly. Browser checks cover 320/390/768/1280px. Actual browser zoom and production employee data/permissions remain future acceptance work.

G11 was revised on 31 October: retain existing choice-card/Back navigation and responsive language-control placement. Direct registration/status cross-links are no longer required. Preserve the expanded task toolbar, background isolation and input continuity; correct accessible control names under the separate technical finding.

Replace public history display strings with localized event rendering. Preserve public free text exactly as entered; staff actor/role details stay in the separate staff projection. Correct page/task heading levels without tying their visual size to the HTML level.

Exit evidence: browser review at 320px, 390px, 768px, desktop and short landscape; no page overflow before/after focus; keyboard navigation and visible focus; dialog focus return; Back/history and retained input; Indonesian/English event descriptions; WITA timestamps; correct active heading outline. Check reduced motion, contrast/transparency fallback and zoom. State separately which physical browsers/devices remain untested.

### 7. Deliver accurate summary reporting

Finding: 11.

Implement a dedicated summary repository with explicit permitted scope, date range, status counts and a defined summary export. Apply access and Terbatas exclusion before aggregation. Do not grant ordinary individual-record access merely to populate Super Admin charts. Detailed visitor exports are outside the initial reporting version.

Distinguish successful zero, loading, empty range, unavailable integration, access denied, error and stale summaries. Enable export only from an available authorized result, with its applied filters and freshness clear.

Exit evidence: known synthetic records produce known scoped totals; unrelated and restricted records do not affect general counts/export; unavailable data never appears as six zero totals; failed requests have retry; export matches the displayed authorized summary and contains no individual visitor rows.

### 8. Re-run the gate and prepare the real backend integration slices

Finding: 1 and all remaining production parts of 2-12.

Reconcile the current product/design/requirements/verification records after each implemented milestone. Mark findings as fixed in preview only when the corresponding checks pass. Keep the original audit as dated evidence, adding a follow-up disposition rather than rewriting its discoveries.

Design and implement the new backend at `D:/oikn/VMS/backend` in bounded slices after the applicable requirement decisions:

1. Staff session/workspaces, assignment checks, real officer/room catalogs and safe read projections.
2. Visitor submission, staff creation/decision, conflict/version/idempotency behavior and private status access under the agreed identity/roster/document policies.
3. Fixed staff rescheduling and notifications; remove the older proposal/visitor-acceptance mismatch from the new adapter path.
4. Pair-authorized visitor actions and server-issued revocable QR, with server timestamps and audit attribution. No extra proof-link/OTP step; decide cancellation timing separately.
5. Reception with the 30-minute early window, late arrival through end, duplicate/stale protection and permitted Checked-out history.
6. Independent role-scoped summary reporting, storage/notification integration and deployment/session configuration.

Each slice requires actual adapter/contract tests and disposable-database integration evidence before live activation. Review any required migration, data policy and operational configuration concretely at that stage. No live migration, account provisioning, external email, real upload, production deployment or camera activation is performed by this planning update.

## Performance and implementation discipline

- Do only the work a current view needs: bounded lists and summaries; no full-history polling, per-row requests or duplicated server state.
- Defer reports, scanners, heavy dialogs and optional motion until needed. Keep existing route splitting and shared dependencies.
- Authorization, projection, resource conflict, admission timing and idempotency decisions run on the server; client checks support usable feedback.
- Use the existing Query, form and UI ownership. No Worker or virtualization is justified by the small prototype fixtures; reassess real dataset/rendering cost after integration measurements.
- The only planned dependency change is the scoped Zod upgrade plus proven compatibility adjustments. Do not add an animation/table/state library for these fixes.
- Preserve unrelated local source/data and keep rollback snapshots scoped. No broad formatting/comment cleanup is included.

## Verification and completion record

For each milestone, record changed files, acceptance cases, commands, browser widths/actions and remaining limitations. Run focused tests for changed validation, authorization, serialization, timing, retries and concurrency behavior. Presentational fixes need rendered checks rather than tests that mirror CSS.

The 29 October audit baseline passed frontend type checking, 23 logic tests, 10 synthetic API tests and both builds; the separate Go run had 272 passing test/subtest results and seven database tests skipped. These are historical baseline results, not a pass for changes in this plan. See [audit evidence](audit-001-2026-10-29.md) and [verification](../VERIFICATION.md).

Final preview acceptance requires the selected findings' behavioral and visual checks, both builds, no new browser errors in exercised flows, and an updated Anti Slop gate. Real readiness additionally requires actual backend/session/database, notification, document, QR/admission and deployment evidence. Do not claim every device, screen reader, performance budget or operational path is verified when it was not exercised.

## Decisions intentionally left open

- Production retention for submitted visits, restricted IDs, documents, audits and sessions; the prior 30-minute unsent-draft decision is a different policy.
- Production capacity limits for party size/documents and the official summary report/export schema/period defaults. Summary-first scope is approved; detailed visitor export is deferred.
- Actual hostnames, email sender/provider configuration, private storage and real account/assignment activation.
- Production QR expiry/replacement and idempotency replay lifetimes, which need explicit service contracts before activation. Extra visitor email-proof links are no longer part of the requirement.
- The previously selected optional same-tab draft recovery remains a later task under DRAFT-RECOVERY.md; this audit did not authorize silently enabling personal-data storage.

These open integration details do not prevent implementing the approved prototype rules. Resolve them when their integration milestone needs them, without reopening the product decisions already confirmed here.
