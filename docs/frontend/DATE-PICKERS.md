# Shared date and time controls

Updated 2 November 2026, WITA. Visitor scheduling, ordinary staff creation, Terbatas and staff rescheduling use packages/ui/src/date-picker.tsx and time-picker.tsx. These preserve YYYY-MM-DD and HH:mm through native FormData/react-hook-form and show WITA. Calendar values use Asia/Makassar independently of the browser timezone.

DatePicker lazily loads its control/calendar. Keyboard opening/selection, month/year controls, bounds, required errors, popover Escape and focus return are shared. Popovers inside a dialog remain within its active surface. Closing a picker must not close the parent form. Read-only restricted edit dates/times remain disabled; rescheduling is a separate authorized action.

TimePicker adapts the existing licensed ReUI control with labels, keyboard columns, immediate selection/clear behavior and minute precision. Each selection updates the field immediately; clicking outside or pressing Escape closes the picker and retains the selected time without a confirmation button. Current scheduling forms expose the validated 08:00–16:00 range. A selection does not reserve resources; conflicts are enforced on approval/rescheduling by the service.

The repaired tests/date-picker.browser.mjs uses isolated fixtures and a native-Date foreign-timezone case. A simulated clock's Date replacement does not preserve the timezone subclass correctly; that harness behavior must not be reported as a live timezone defect. This test distinction is explicit rather than skipping the foreign-timezone case.

Tests also cover leap day, required errors, FormData, discard behavior, read-only edits and 320/390/768px popover bounds. Axe audits cover open calendar/time controls. VERIFICATION.md records the latest actual pass status and remaining manual/device checks.

Retain upstream MIT/source notices and comments explaining behavior constraints. The 2 November comment-only vendor cleanup removed decorative banners while verifying parsed code equivalence. No replacement UI/motion framework was added.
