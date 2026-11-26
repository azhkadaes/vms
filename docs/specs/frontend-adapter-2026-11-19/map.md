# Frontend adapter changes for the new backend contract

Status: tickets published 19 November 2026, approved by the user. Each ticket changes the frontend adapter, moves the preview mock backend to the same contract, and updates its tests. Each ticket also lists the backend tickets in [backend-api-2026-11-18](../backend-api-2026-11-18/issues/) that must be merged before it can be verified against the real backend.

Source: the agreed but deferred adapter decisions and the new defects in [UI ↔ contract validation §6](../../contracts/UI-CONTRACT-VALIDATION-2026-11-18.md#6-frontend-work-not-backend).

| # | Ticket | Blocked by | Backend needed |
| --- | --- | --- | --- |
| [01](issues/01-bearer-staff-session.md) | Bearer staff session: login, restore and logout | — | existing |
| [02](issues/02-token-adoption-and-activity.md) | Role switch, password change and activity renewal adopt server tokens | 01 | existing |
| [03](issues/03-error-codes-drive-messages.md) | Error codes drive conflict and validation messages | 01 | 01 |
| [04](issues/04-upload-purpose.md) | Uploads send their purpose | — | 05 |
| [05](issues/05-booking-response-model.md) | Booking responses use `pejabat.nama` and the new staff fields | — | 06, 09 |
| [06](issues/06-room-ids-and-requiredness.md) | Room IDs and officer/room requiredness in booking forms | 05 | 04, 10–12 |
| [07](issues/07-server-counts.md) | Booking tabs and dashboard counts come from the server | 05 | 09 |
| [08](issues/08-reception-attendance-and-files.md) | Reception uses the single attendance route and credential-bound file reads | 05 | 13, 14 |
| [09](issues/09-report-room-ids.md) | Report room filters use room IDs | — | 15, 16 |
| [10](issues/10-visitor-recovery.md) | Visitor app recovers from a stale QR and an outdated privacy notice | 05 | 06–08 |

The frontier starts at 01, 04, 05 and 09.
