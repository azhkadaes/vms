# Local in-app notifications verification

Verified 23 November 2026 in the isolated checkout `D:/oikn/VMS/.scratch/form-release` (base `ebc271c`).

## Result

The notification feature is running at http://localhost:58101. The additive migration `202611230001_create_support_staff_notifications.sql` was the only pending migration and the VMS migrator reported exactly one applied migration to `vms_stage1`. No historical audit events were backfilled. Runtime readiness now requires `support.staff_notifications`.

Images `vms-backend:notifications` and `vms-frontend:notifications` were built from the verified local binaries/admin build, then tagged for local Compose. The migrator image was updated to the same embedded migration set. Only API and frontend were recreated. `vms-postgres` and `vms-uploads` remain attached; the database container was not recreated. Previous API/frontend images remain tagged; `vms-migrate:before-notifications` preserves the previous migrator image. The previous API/frontend can run with this additive schema; rollback was not exercised.

## Routing and database checks

- Migration SQL executed inside a rolled-back transaction before application: ordinary visits route to active PAs assigned to the destination officer; VIP/restricted visits route to active Protocol Officers.
- All six events were exercised: created, approved, updated, rescheduled, cancelled, checked_out. Check-in was excluded.
- After deployment, another rolled-back transaction added one active and one inactive fictional Protocol Officer, then exercised the same cases under `SET LOCAL ROLE vms_api`. All active Protocol Officers received each VIP/restricted event and inactive staff were excluded. This also verified runtime trigger permissions.
- Browser API verification confirmed ordinary PA versus Protocol isolation, restricted Protocol versus PA isolation, denied anonymous access, denied Receptionist/Super Admin access, rejected invalid/unknown query inputs, cross-role mark-read denial, and idempotent read timestamps.
- Retrying an identical ordinary creation produced one created notification. History pages contain at most 25 entries, ordered by event ID, with no overlap.

## Browser evidence

`notification-evidence/results.json` records the live flow. `notification-evidence/polling-results.json` records polling and recovery checks.

- Blue info toast displayed with matching VMS primary-blue icon and border. Dismiss worked; reload did not repeat a seen toast; a burst produced a summary toast.
- Foreground polling delivered a new event without navigation or reload.
- Empty history, unread history, mark-read, older/newer pagination and click-through to the selected visit worked.
- History reflowed at 320, 390, 768 and 1280 px without horizontal overflow. Keyboard Enter opened the dialog; Escape restored focus to the bell.
- A simulated notification-list 503 displayed a retry control, and retry recovered against the real API.
- No browser page errors were recorded. Screenshots: `notification-evidence/blue-toast.png`, `history-mobile.png`, `history-desktop.png`.

The polling harness saved all four successful checks and cancelled fixture #32, then lingered during browser shutdown. It was interrupted after those results were verified; this was not an assertion failure.

Fictional browser fixtures #27, #28 and #32 were cancelled after verification. Their audit/history records and the test photo upload remain intentionally retained; no real visit was modified. Rolled-back checks may consume identity sequence numbers.

## Build and test evidence

- Earlier completed logs inspected: `notifications-unit.log` (108 frontend tests passed), `notifications-build.log` (TypeScript and admin build passed).
- Earlier `notifications-go.log`: full Go suite had one readiness failure. After adding the new table to the readiness list, `go test ./pkg/helper/database ./migration ./internal/handler ./internal/repository/notification` passed. The full suite was not repeated after this one-line fix.
- Both Linux runtime binaries compiled successfully. Both Docker images built successfully. Local frontend readiness returned healthy.
- `git diff --check` passed; Git emitted line-ending notices for inherited files.

## API and behavior

Authenticated PA/Protocol sessions with bookings permission use `GET /v1/staff/notifications?before=<event-id>` and `POST /v1/staff/notifications/<event-id>/read`. List responses contain `items`, `unreadCount` and optional `nextBefore`; each item contains `id`, `visitId`, `event`, `at` and nullable `readAt`. Recipient identity comes from the current session. A PA must still hold the destination assignment when reading history. History is durable in PostgreSQL; toast deduplication is browser-local. Polling runs every 30 seconds while the page is active.

## Feature UI review

- Hard gate PASS: real API click-through, empty/error/retry states, keyboard behavior and mobile reflow verified above; notification content comes from real audit events.
- Purpose gate PASS: bell opens durable history; blue info icon/border distinguishes informational visit events; unread badge reports a measured count.
- Visual consistency PASS: inherited calm VMS layout, typography, spacing and blue action token retained. Notification list is bounded and uses the shared dialog.
- Functional quality PASS: toast, history, access denial and polling checks completed against local Docker. This is a feature review, not a renewed audit of every existing screen.

This establishes local readiness only. Production deployment, external delivery, cross-device toast deduplication and a database restore were not tested. No email was sent and no GitHub push was made.
