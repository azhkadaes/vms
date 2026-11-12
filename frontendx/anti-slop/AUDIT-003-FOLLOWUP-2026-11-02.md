# New appointment review: fixes 1-6

Implemented the user's approved findings 1-6 from audit-003-2026-11-02.md. Scope is the standard new-appointment dialog in frontendx. No backend, role rules, booking payload, approval behavior, visitor UI, or dependencies were changed.

## Changes

1. Review values now wrap, with zero default dd indentation and minmax(0, 1fr) tracks. Long emails and filenames remain selectable and readable at 320px.
2. Final review includes visit purpose and identity type.
3. Grouped visit details, representative/contact fields, counted ordered visitor list and document. Existing label/value typography and restrained spacing create hierarchy; two-column field groups collapse on phones.
4. Ubah actions navigate to the existing form steps and return directly to review after validating the edited section. Draft values and attachment selection remain mounted. Focus moves to the relevant input or review heading. The action footer stays available while scrolling, with scroll padding for focused controls.
5. Optional Modal className enables a scoped staff-appointment-modal font override using the existing Geist font. Visitor dialogs retain their original styles.
6. Indonesian long date format via Intl with Asia/Makassar, paired with the existing HH:mm WITA value. Submitted date/time contracts are unchanged.

Source files: apps/admin/src/features/CreateAppointment.tsx, apps/admin/src/styles.css and packages/ui/src/index.tsx. Evidence runner: anti-slop/evidence-2026-11-02/booking-review.mjs. Original before screenshots/results are retained; new artifacts use booking-review-after-* names.

## Verification

- PASS: npm.cmd run typecheck.
- PASS: npm.cmd run build --workspace @vms/admin. Vite reports dependency annotation warnings in Zod; build completes.
- PASS: node --experimental-strip-types anti-slop/evidence-2026-11-02/booking-review.mjs with a fresh Vite server on 127.0.0.1:3012 and an isolated preview API, headless Chrome, Asia/Makassar.
- PASS: forward wizard navigation, localized date, purpose and Paspor in summary.
- PASS: all four Ubah controls; direct return to review; retained name, purpose, companion, identity, email and document filename; focused destination/name/document controls and review heading.
- PASS: normal review at 1440, 390 and 320px; computed Geist typography and zero horizontal dialog overflow; zero axe violations in each tested review state.
- PASS: 20 visitors, long email and long PDF filename at 320px. Modal scrollWidth equals clientWidth: 288px, down from 1385px before the fix. Zero axe violations in this long-content state. Footer remains reachable and Back retains input.
- Visual QA: inspected desktop and 320px screenshots; native scrolling accommodates the full summary while the footer remains visible.

This test does not submit a booking. Production endpoints, saving/error/retry behavior, actual browser 200% zoom, screen reader speech and physical phones/Safari remain unverified. No claim of full WCAG certification or production readiness.

## Delivery gate

- Hard gate PASS within tested review states: R-03 reflow fixed and measured; no unnamed controls, automated contrast findings or page overflow; native buttons and labeled sections remain; no new assets, claims, theme, motion or data operations.
- Purpose gate PASS: grouping serves verification of entered fields; existing font and colors retain staff identity; border lines separate operational sections; no decorative effects or assets added.
- Liveliness PASS for this scoped operational refinement: existing OIKN palette and Geist identity retained; no unrelated visual redesign.
- Craftsmanship PASS for approved scope: all collected review fields are visible; section editing works and preserves values; full roster and document text wrap; existing source conventions and dependencies retained.
- Verification PASS for evidence above; untested environments and submission states are explicitly excluded.

Verdict: approved findings 1-6 resolved in the tested local preview.
