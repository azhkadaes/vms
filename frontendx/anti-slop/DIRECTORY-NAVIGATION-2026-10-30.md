# Employee directory and retained visitor navigation

> Historical phase record. Current implementation/contracts and final evidence are consolidated in [the 2 November readiness review](PRODUCTION-READINESS-2026-11-02.md), [PLAN](../../docs/frontend/PLAN.md) and [VERIFICATION](../../docs/frontend/VERIFICATION.md). Earlier pending-work statements describe that phase.

30 October user decisions: G10 adjusts the app; G11 adjusts the requirement to retain the existing visitor choice-card/Back navigation. Scope is frontendx; the new backend remains unbuilt.

## Plan before implementation

- Goal: readable employee cards on phone/tablet and a desktop table, with long-name/email/role wrapping, stacked heading/action/search and explicit no-match/reset feedback. Preserve the preview-only add-record flow and single active role per account (G09).
- Data/state: the preview employee feature owns its local fictional records and add dialog. URL q owns search; filter the tiny loaded list by name/email/role. A presentational directory renders cards/table or explicit empty/no-match feedback. No employee API or production CRUD is invented.
- Components: extract employee directory from Settings so roles/integrations/reports do not own its search/add state. Reuse shared Button/Modal/Empty, return dialog focus and clear-search focus. Hide the unused responsive representation from the accessibility tree with display:none.
- States/accessibility: known local records, no match with reset, genuinely empty local list, add dialog/native required fields, semantic card headings/details and desktop table headers. Existing production API-required feedback remains. Future loading/server-error/denial states must be added with a real employee adapter; local fixtures must not be described as live records.
- Performance questions: no data fetch or new dependency is needed; derive matches from the small list and keep rendering in the directory subtree. No debounce/worker/virtualization for three preview rows. Backend filtering/pagination belongs to the future service when scale is specified.
- Tests: search by name/email/role, case/space handling, no-match reset and input focus, URL search persistence, long fields, phone/tablet cards vs desktop table, no document or inner-cell clipping, keyboard add/close focus, native field semantics, no booking-list fetch. Typecheck/build and affected regressions.
- G11: preserve the current photo/choice-card, Back/browser Back and responsive language-control placement. Direct contextual registration/status links are no longer required by this gap. Keep input continuity, accessible labels/focus and hidden/inert background controls. This is a requirement revision, not a visitor navigation redesign.

## Status

Completed on 31 October for the fictional local preview. EmployeeDirectory owns the records, URL search and add dialog; Settings routes to the feature without owning its state. Cards below 1001px and a native table above that width use the same derived matches. Long fields wrap; a polite count, no-match explanation and Reset provide feedback. Reset focuses search, and dialog save/Escape restore the trigger. Production still shows the API-required state.

Three directory browser cases pass: 320/390/768/1280px responsive/accessibility-tree checks; name/email/role search, trimming/case, reload and no-match/reset; keyboard add/close/native validation and unusually long fields. All five affected report browser cases, typecheck and both builds pass. Staff/visitor entry sizes remain 114.24/105.00kB gzip; production removes the local employee demo. No dependency, asset, polling or employee endpoint was added. Screenshots and accessibility trees are in `anti-slop/evidence-2026-10-31/employees/`. These checks do not establish real-browser 200% zoom, NVDA, physical-device/Safari or full-app WCAG acceptance.

G11 is reconciled in BR-UX-06, product/design/plan/readme and the decision sheet. Current visitor navigation is retained. G12/G13 were subsequently selected for app changes and are separate work.

## Scoped delivery gate

- Hard gate PASS: no changed-screen document/cell overflow at the four widths; long fields stay visible. Native labeled controls, result announcements, keyboard add/Escape/save/reset and focus return pass. The existing preview labels identify fictional data; no fabricated production claim or asset. Typecheck/build and rendered click-through pass. Other app audit findings remain open.
- Purpose gate PASS: existing opaque staff panels support readable employee information; equal cards represent equal records; search/plus icons identify their actions. No decorative effect or new motion is introduced.
- Liveliness PASS: ENERGY 1 / RHYTHM 1 / MOTION 1 follows the existing institutional staff direction. Page title/search establish hierarchy; the blue add action and current OIKN sidebar identity remain. Whitespace groups each record's details.
- Craftsmanship PASS: one feature owns the local state, filtered results are derived, both representations share records and inactive markup is hidden from the accessibility tree. Comments add no generic narration. Browser evidence supports this scope; future employee loading/error/authorization/pagination must be implemented with the new backend adapter.

## Backend handoff

No real employee API, account creation or authorization was implemented. Before activation, the new backend must define the employee directory contract, server-enforced people permission and single-role assignment, validation, pagination/filtering and provisioning behavior. Replace the local examples through the data layer, then verify loading/empty/error/denied/stale responses and real scale. Do not turn the demo add form into an account endpoint by assumption.
