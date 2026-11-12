# Reservation policy and visitor-access decision

> Historical phase record. Current implementation/contracts and final evidence are consolidated in [the 2 November readiness review](PRODUCTION-READINESS-2026-11-02.md), [PLAN](../PLAN.md) and [VERIFICATION](../VERIFICATION.md). Earlier pending-work statements describe that phase.

User decisions, 30 October 2026, Asia/Makassar:

- G04: adjust the app. Pending requests may overlap. Reserve/check the same officer or same nonempty room when approving or changing an Approved/Checked-in commitment. Adjacent intervals and unrelated resources may coexist.
- G02: adjust the requirement. Email plus visit code is sufficient for visitor status, QR access and cancellation, subject to visit eligibility. No additional email-link proof or OTP screen is required. QR remains server-issued in the future real service and demo-only in this preview. Cancellation timing remains the separate pending G15 decision.

## Plan before implementation

Scope is frontendx and its synthetic service only; `D:/oikn/VMS/backend` remains unbuilt. Do not change other gap decisions.

Data/API: preserve the existing repository commands and version checks. A conflicting reservation returns 409 before changing any record. No new endpoint/dependency. The future backend needs an atomic resource check/write, but is not implemented here.

State/component ownership: remove reservation checking and its extra bookings observer from CreateAppointment. The form owns draft/validation; the preview owns resource reservation decisions; query cache/URL ownership stays unchanged. Reuse one pure scheduling policy for approval, rescheduling and approved restricted-host edits. Preserve original form input when decisions conflict.

Three performance questions: avoid the form's unnecessary booking query and duplicate overlap calculation; perform checks only when an action can reserve/change resources; execute the check at the synthetic service boundary rather than in a component. No timer, worker, virtualized list or generic scheduling framework is needed.

Validation/edge cases: same officer/different room, different officer/same room, unrelated resources, empty resource values, touching intervals, changed durations, self-exclusion, Pending/terminal records, Checked-in commitments, stale versions and competing approvals. Keep current form and scope validation unchanged.

UI: existing creation steps continue despite occupied requested times; approval/reschedule display the existing human conflict error and retain inputs. No visual redesign, asset, motion or new navigation. Retain the existing neutral operational style, ENERGY 1 / RHYTHM 1 / MOTION 1.

Build order: shared pure policy and tests; remove create-time reservation checks; service enforcement; integration/concurrent-request checks; focused browser creation/decision checks; typecheck/build/regression checks; reconcile requirement/plan/product/design/readme/verification and audit dispositions.

## Status

**G04 implemented and verified in the synthetic preview; G02 requirement/plan revised.** The new backend remains unbuilt. No arrival-window, restricted-projection, reporting, cancellation-cutoff or other pending gap policy was changed.

The full staff-form walkthrough also reproduced premature submission when the Next button changed into Submit during its click action. The correction prevents the navigation click's default submission and gates form submission to the review step, preserving BR-AJU-06. The browser check must assert zero create requests on entering review and exactly one on the separate confirmation action.

## Verification and delivery gate

- PASS: typecheck, 41 logic tests (including 18 resource-policy cases), 13 synthetic integration tests and both production builds.
- PASS: integration checks cover overlapping Pending requests, matching officer/different room, matching room/different officer, unrelated resources, adjacency, duration extension, unchanged record/version after conflicts, competing approvals, blank resources, approved restricted-host changes and Checked-in commitments. One competing approval succeeds and one returns 409 in this single-process preview; this does not establish a database transaction guarantee for a future backend.
- PASS: the focused 390px browser flow creates an overlapping request only after a separate review confirmation, receives a room-conflict error on approval, retains the chosen room/45-minute duration, then successfully approves after switching to a free room. Keyboard activation, no document overflow and no uncaught page errors were checked. [Conflict screenshot](evidence-2026-10-30/reservations/approval-conflict-390.png).
- PASS: all 8 existing refresh browser checks still pass, including unchanged draft versions during background updates.
- PASS: no new dependency, timer, backend project or asset. The staff creation chunk decreased from about 4.19kB to 4.03kB gzip after removing its unnecessary booking observer/calculation. This is a bundle-size observation, not a field-speed claim.
- PASS: current BRD/product/design/plan/readme, audit dispositions and the decision sheet distinguish G04 app alignment from G02 requirement alignment. The earlier extra email-link-proof milestone is explicitly superseded. The cancellation cutoff remains G15.

Run `npm.cmd run dev`, then `npm.cmd test`, `npm.cmd run test:preview` and `npm.cmd run test:reservations`; browser commands require Playwright (or PLAYWRIGHT_MODULE pointing to a local module). `npm.cmd run test:refresh` covers refresh regression. No actual email, production credential issuance, physical-device or new-backend acceptance is claimed. Audit-created fictional records disappear when the task-owned preview is stopped.
