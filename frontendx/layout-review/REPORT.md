# Backoffice layout review — 2 November 2026

## Current disposition — 3 November 2026

The user chose option A for each item. All four approved fixes are implemented. RTL support is explicitly deferred for the current Indonesian/English scope; the mirror finding remains a future requirement, not a completed fix.

| Item | Decision and result | Verification |
| --- | --- | --- |
| Tablet tables and status tabs | Completed: reuse existing cards and the status selector when the booking panel is narrow; retain tables where they fit. | [96 layout combinations and control checks](./option-a/REVIEW.md) |
| Blank sort dropdown | Completed: visible “Urutkan” label and “Urutan bawaan” default text. | [36 layout combinations and existing sort test](./sort-label/REVIEW.md) |
| Narrow input typography | Completed: 16px text in narrow layouts, preserving compact desktop sizes. | [60 page/dialog combinations and calendar checks](./input-typography/REVIEW.md) |
| Role and integration card grouping | Completed: consistent panel insets and spacing between descriptions, permissions/status, and actions. | [12 layout combinations and dialog-opening checks](./card-spacing/REVIEW.md) |
| RTL adaptability | Deferred by the user; revisit before introducing a right-to-left language. | Original mirror findings retained below; no RTL fixes applied. |

TypeScript and the admin production build passed after the final source changes. No further tests were run for this documentation-only decision. Native 200% browser zoom, physical-device behavior, and production readiness remain unverified.

## Original audit snapshot — 2 November 2026

Reviewed `D:/oikn/VMS/frontendx`, including shared UI styles. Application source was unchanged at the time of the original audit. The findings, source line numbers, “After” recommendations, and verification limits below describe that original snapshot; use the follow-up reviews above for the implemented fixes and current source locations.

## Hint at hidden content / hold structure until it breaks

| Severity | Location | Before | After | Why |
| --- | --- | --- | --- | --- |
| HIGH | [Bookings.tsx:269](D:/oikn/VMS/frontendx/apps/admin/src/pages/Bookings.tsx:269), [Bookings.tsx:387](D:/oikn/VMS/frontendx/apps/admin/src/pages/Bookings.tsx:387), [styles.css:806](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:806), [styles.css:1574](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:1574) | On both Janji temu and Kunjungan hari ini, the expanded sidebar leaves 498px for the table at a 768px viewport. PA table content occupies 699px; Protocol table content occupies 650px. Status and detail actions sit past the visible edge. Status tabs also scroll: 864–869px inside 498px. The screenshots have no persistent scroll cue; the table's trailing actions are hidden for the full height of the rows. Keyboard scrolling is supported by the table region, but its availability is not visually apparent. | Switch to the existing cards when the content area becomes too narrow, or provide a persistent, visible horizontal-scroll cue and keep essential actions discoverable. Apply the same behavior to both list pages and their role variants. | Hidden content needs a cue. Viewport-only breakpoints keep the table expanded when the sidebar has already made its container too narrow. |

Evidence: [PA tablet table](./booking-PA-768.png), [Protocol tablet table](./booking-Protocol-Officer-768.png). The table is horizontally scrollable; this finding concerns discovery and access, not permanent data loss.

## Keep controls distinct from content

| Severity | Location | Before | After | Why |
| --- | --- | --- | --- | --- |
| MEDIUM | [Bookings.tsx:315](D:/oikn/VMS/frontendx/apps/admin/src/pages/Bookings.tsx:315) | The default sort option is empty. The bordered dropdown appears blank on desktop and phone, with its purpose exposed only through its accessible name. It has no visible label. | Give the default option visible text such as “Urutan bawaan” and show a consistent “Urutkan” label. Preserve existing sort behavior and URL state. | A control needs a visible purpose and current state. An empty dropdown can be mistaken for an unloaded or unselected field. |

Evidence: [Desktop booking controls](./booking-PA-1440.png), [Phone booking controls](./booking-PA-320.png).

## Typography: inputs at 16px on mobile

| Severity | Location | Before | After | Why |
| --- | --- | --- | --- | --- |
| MEDIUM | [Admin styles.css:841](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:841), [Admin styles.css:845](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:845), [Shared styles.css:24](D:/oikn/VMS/frontendx/packages/ui/src/styles.css:24), [Shared styles.css:477](D:/oikn/VMS/frontendx/packages/ui/src/styles.css:477) | At 320px, booking search computes to 13.12px and sort to 12.48px, while the status selector is 15px. Employee search, report filters, restricted-visit fields, integration fields, and reception code entry also compute to 15px. Standard appointment controls already use 16px. | Use a consistent mobile input size of at least 16px through the established styling system. Preserve the compact desktop density. | The same control roles have visibly different text scales. Sub-16px editable controls are also susceptible to focus zoom in iOS Safari; actual iPhone behavior was not tested. |

Measured values and locations are recorded in [page measurements](./results.json) and [dialog measurements](./dialogs.json). Hidden validation inputs and checkbox font sizes are excluded from this finding.

## Group with space / align to shared edges

| Severity | Location | Before | After | Why |
| --- | --- | --- | --- | --- |
| LOW | [RoleManagement.tsx:152](D:/oikn/VMS/frontendx/apps/admin/src/features/RoleManagement.tsx:152), [IntegrationManagement.tsx:227](D:/oikn/VMS/frontendx/apps/admin/src/features/IntegrationManagement.tsx:227), [styles.css:1803](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:1803) | Role descriptions run directly into the permission paragraph, followed immediately by Edit akses. Integration descriptions, connection state, and configuration actions use the same unspaced sequence. Cards also touch the panel-heading border and preview notice, despite a 20px gap between cards. | Use the existing spacing tokens to separate description, access/status details, and actions. Give the record grid a consistent panel inset and space before the preview notice. | The current grouping makes distinct information read as one paragraph and crowds the action against the content. |

Evidence: [Role cards at 320px](./roles-Super-Admin-320.png), [Integration cards at 320px](./integration-Super-Admin-320.png).

## Logical properties for anything that mirrors

| Severity | Location | Before | After | Why |
| --- | --- | --- | --- | --- |
| MEDIUM — RTL stress only | [Admin styles.css:353](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:353), [Admin styles.css:1456](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:1456), [Shared styles.css:907](D:/oikn/VMS/frontendx/packages/ui/src/styles.css:907), [Shared styles.css:922](D:/oikn/VMS/frontendx/packages/ui/src/styles.css:922), [Shared styles.css:975](D:/oikn/VMS/frontendx/packages/ui/src/styles.css:975), [Shared styles.css:539](D:/oikn/VMS/frontendx/packages/ui/src/styles.css:539), [Shared styles.css:682](D:/oikn/VMS/frontendx/packages/ui/src/styles.css:682) | With `dir=rtl`, content and table columns reverse, but the sidebar remains on the left. Search icon/padding and explicit left text alignment do not mirror with surrounding content. | Use logical inline properties for the shell offset, search decoration, and text alignment, and review directional icon behavior. | A partial mirror gives controls and content conflicting leading edges. RTL support is not asserted by the current Indonesian product; this is an adaptability finding, separate from the current LTR blocker. |

Evidence: [Desktop RTL stress](./stress-rtl.png), [Narrow RTL stress](./reflow-rtl-720.png).

## Verification and limits

- Chromium using installed Chrome, headless. Populated data came from an isolated instance of the project's sample-data service. The existing local backoffice preview requests stayed pending, so those requests were redirected only inside the audit browser. No real accounts, integrations, bookings, email, or uploaded files were changed.
- 66 page/role/width combinations: dashboard, today's visits, appointments under PA and Protocol, check in, check out, employees, roles, integrations, analytics, and visit reports; widths 1440, 1280, 1024, 768, 390, and 320px. These checks found no document-level horizontal overflow. Local scroll-region overflow is reported separately above.
- 32 initial dialog/menu combinations at 1440, 768, 390, and 320px: standard appointment, restricted appointment, visit detail, employee creation, role editor, integration editor, reception code entry, and appointment-type menu. No measured horizontal overflow inside their modal containers.
- Calendar and time popovers were opened and visually checked at 320px after loading. Both fit horizontally. See [calendar](./calendar-320.png) and [time dropdown](./time-picker-320.png).
- Booking page checked at 720×450px as an effective reflow size, with an additional CSS `zoom:2` stress at 1440×900px. CSS zoom does not reproduce native browser zoom breakpoints; **native 200% browser zoom: Not verified**.
- German heading/action strings and expanded status labels were injected into the booking page at desktop width. The action fitted; expanded tabs reinforced the scroll-cue finding. Full pseudo-localization across every page and dialog: **Not verified**.
- RTL mirror checked on the booking page at desktop and narrow width. Other page-specific RTL states: **Not verified**.
- Physical iOS/Android, Safari/Firefox, widths outside the listed matrix, sidebar collapse/drag variants, scanner camera UI, login/password-change screens, every form step, validation/error/empty states, and production API data: **Not verified**. This is broad coverage of the requested operational surfaces, not approval of every possible state.
- Source review covered the page structure, shared form/table/modal controls, declared typography, responsive rules, and physical/logical alignment properties. Existing phone booking cards, employee cards, report filter stacking, and date/time popover containment should be retained.

The original audit added scripts, measurements, and screenshots under this directory. The subsequent approved application fixes are documented in the linked follow-up reviews.

The original **Block** for tablet table/tab discoverability is resolved within the tested option A coverage. RTL adaptability is deferred by explicit user decision.

**Approve** for the inspected and verified scope of the four implemented fixes. This does not approve untested states, native 200% browser zoom, physical devices, production readiness, or future RTL support.
