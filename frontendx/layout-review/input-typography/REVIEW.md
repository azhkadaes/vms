# Narrow-layout input typography — 3 November 2026

Implemented the approved option A: 16px input and dropdown text in narrow backoffice layouts, preserving compact desktop sizing.

| Severity | Location | Before | After | Why |
| --- | --- | --- | --- | --- |
| MEDIUM — resolved in inspected controls | [styles.css:661](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:661), [styles.css:1885](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:1885) | Narrow booking search used 13.12px and sorting used 12.48px; many other input controls used 15px. | At viewport widths up to 1000px, backoffice shell, sign-in, and modal text-entry controls and selects use 1rem (16px at the default root size). Booking controls also use 1rem whenever their panel is narrow enough for cards, even in a wider viewport. Calendar month/year dropdown text follows the same narrow-viewport size. | Consistent readable control text across narrow operational layouts. This also addresses the declared sub-16px input sizing risk; physical iOS behavior was not tested. |
| Adaptation required by the larger type | [styles.css:657](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:657) | A 150px sort control could crowd selected text after increasing the font size. | In card layouts, the sort field reserves up to a 12rem minimum, capped by its available width. At 320px it spans the toolbar and the VIP checkbox wraps below it. | Both “Urutan bawaan” and “Nama pengunjung” remain fully readable without shrinking their text. |

The 1000px viewport boundary follows the existing employee/report narrow-layout breakpoint. The booking container uses its previously approved 900px content boundary. Checkbox/radio/range/color controls are excluded from the input text rule. Shared visitor styles were not edited.

Validation passed: TypeScript and admin production build; 60 page/dialog/width browser combinations plus calendar dropdown text and containment. The checked surfaces were booking, today's visits, employee search, analytics/report filters, standard and restricted appointment dialogs, employee creation, integration configuration, and reception code entry. Widths were 1440, 1150, 1000, 768, 390 and 320px.

Measured narrow text-entry/select controls were at least 16px. Their rectangles stayed inside the viewport; documents and modal scroll containers had no horizontal overflow. Desktop booking search remains 13.12px and sorting 12.48px. Screenshots were inspected for the phone booking toolbar, the selected name sort option, and the calendar dropdowns. Checks used an isolated sample-data service.

[Phone preview](./booking-320.png) · [Selected sort text](./sort-selected-320.png) · [Calendar](./calendar-320.png) · [Measurements](./measurements.json)

Physical iOS/Android, native browser zoom, sign-in/password-change rendered flows, later appointment form steps, and production data remain **Not verified**. Sign-in/password-change input coverage is from the shared CSS selectors, not a browser run.

**Approve** for inspected narrow-control typography coverage. Management-card spacing and RTL alignment still await the user's choices.
