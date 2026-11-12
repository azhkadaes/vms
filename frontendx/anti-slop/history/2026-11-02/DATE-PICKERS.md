# Shared shadcn date and ReUI time pickers

Updated on 22 October 2026 in the current `frontendx` workspace at the user's request to use shadcn/ui dates, ReUI time selection and compact dropdowns across all scheduling forms.

## Components and coverage

All four scheduling flows use the shared controls: visitor booking, staff appointment creation, restricted-visit creation/read-only editing, and staff rescheduling. Dates use `@vms/ui/date-picker`; times use `@vms/ui/time-picker`. The earlier React Aria implementation and its two dependencies were removed.

The date control composes shadcn/ui's Radix Popover and React DayPicker Calendar. The time control uses the actual [ReUI time picker](https://reui.io/components/time-picker) with scrollable hour/minute columns, localized labels and the same shadcn Popover/Button primitives. It replaces the earlier native time input.

Source-owned components live in `packages/ui/src/components/ui/`: Button, Input, Popover and Calendar. They adapt shadcn's New York registry structure/data slots to the project's existing CSS tokens, without adding Tailwind/preflight to the rest of the app. The MIT notice is retained in `LICENSE.shadcn`. The component headers link to their registry sources.

ReUI's source lives in `packages/ui/src/components/reui/time-picker.tsx`, with its MIT notice in the adjacent `LICENSE`. It is adapted from the [official Radix source](https://github.com/keenthemes/reui/blob/main/registry-reui/bases/radix/reui/time-picker.tsx): local primitive imports, a Lucide clock icon and scoped project CSS. The unused typable-input variant is omitted. This update adds no runtime dependencies.

The calendar popup is 284px wide and the time popup 216px, both constrained to the viewport. Desktop rows are 32px tall; touch screens use 40px calendar days and 44px time rows. Subtle borders, muted labels and pale blue time selection keep the dropdowns compact and readable.

Dependencies: `@radix-ui/react-popover` 1.1.15, matching the existing Dialog's focus/dismissal dependencies; `react-day-picker` 10.0.2; `date-fns` 4.4.0. The lockfile records resolved versions.

## Behavior and contracts

- Dates display in Indonesian, with month/year dropdowns, keyboard navigation, selected/today states, unavailable past dates and a close control.
- Calendar math explicitly uses `Asia/Makassar` and date-only values are serialized as `YYYY-MM-DD`, without UTC date shifts.
- Time fields use `HH:mm`, all 60 minute choices and visible WITA/24-hour hints. Select an hour and minute, then choose Pilih to apply. Hapus clears the pending choice; Pilih applies the clearing. Escape or clicking outside discards pending changes. There is no browser-local Now shortcut. Existing future-start and business validation remains authoritative.
- Time columns support arrow keys, Home/End and numeric typeahead. Left/right moves between columns and Enter confirms.
- Both controls preserve labels, required marks, errors, focus targets, disabled/read-only states and React Hook Form values. Uncontrolled rescheduling still submits date and time using native FormData and constraint validation.
- Visually hidden form proxies supply validation/submission; visible date/time triggers receive validation focus.
- Restricted-visit detail editing preserves both read-only schedule values.

The modal separates its scrolling content from its overlay layer, allowing nested date/time popups to remain inside the dialog's focus boundary without being clipped. Escape closes the popup first and restores focus to its field. Existing dialog close/return-focus behavior remains.

Visitor step-one fields with errors revalidate as they are corrected. This prevents the Continue button from moving during a pointer click when a blur removes an old error message. Filled staff forms retain their existing discard confirmation.

## Loading

The lightweight date entry point loads its control implementation when a date field mounts. Calendar rendering/library code has a separate lazy boundary: it preloads on field hover/focus and mounts when opened, with a sized loading placeholder. The time picker does not import the calendar library. The current visitor landing embeds the booking form, so mounting that form can load the small date-control implementation; calendar loading still waits for intent/opening. No date/time API requests, animation library or external font service was added.

The production build keeps the deferred calendar in a separate chunk, about 22 KB gzip. ReUI is included with the scheduling form. This separation does not measure overall application speed.

## Verification

Type checking, 23 domain/validation tests and both production builds passed. Nine Chrome browser tests cover:

- shadcn data slots, required errors, future-date selection, leap-day/month/year navigation, keyboard Escape/focus, and progression after correcting visitor fields;
- staff nested-popup dismissal and dirty-form discard behavior;
- native rescheduling validity/FormData with ISO dates and minute-precision time;
- restricted-visit creation and read-only date/time editing;
- visitor/staff layouts at 320, 390 and 768 pixels, including horizontal and vertical date/time popup bounds;
- the same selected WITA date with an America/Los_Angeles browser timezone;
- ReUI draft cancellation, keyboard selection, confirmed clearing and required errors.

Desktop (1280px), tablet and phone renders were inspected. Screenshots are in `.impeccable/review/shadcn-date-time-*` and `compact-*-*`. Physical devices, Safari, Firefox and screen readers were not tested. These checks do not establish real backend or production readiness.

With a local synthetic preview running and Playwright available:

```powershell
$env:VMS_TEST_URL = 'http://127.0.0.1:3200'
# If Playwright is outside this workspace, set PLAYWRIGHT_MODULE to its index.mjs file URL.
node --test tests/date-picker.browser.mjs
```

The optional browser suite uses installed Chrome and rejects non-loopback origins. Verification used isolated preview ports 3200/3202 to avoid changing the existing preview processes. The original VMS-FE and Go backend remain outside this change. This workspace has no Git repository, so no commit was created.
