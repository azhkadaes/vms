# Current Kunjungan Terbatas contract

Consolidated 2 November 2026, WITA. Scope: frontendx only. The revised G03/G05/G09 rules below supersede the older baseline archived under anti-slop/history/2026-11-02.

Date and WITA start time are required. Only Nama, Instansi, No ID Dinas and Pejabat yang dituju are optional fields. Any active officer may be selected or the target left blank. No ID Dinas remains text with leading zeroes intact. No public registration, extra contacts, roster or attachment field is introduced.

| Actor/view | Read access | Actions |
| --- | --- | --- |
| Protocol | Full authorized restricted detail | Create/edit/approve/reject/fixed reschedule in permitted status. |
| Assigned PA | Supplied name/organization/target, schedule and last four ID characters, only while actively assigned to the target | Read only. |
| Super Admin | Full restricted detail | Read only through admin rights. |
| Reception general | Approved/Checked-in/Checked-out reference, schedule/status/timing | Attendance flow; no optional identity. |
| Reception after valid code/QR | Additionally supplied name, target and last four ID characters; no organization/full ID | Explicit identity verification followed by eligible check-in/out. |
| Public | No discovery/lookup/credential/file access | None. |

Scope and projection precede list/search/counts, direct detail and conflicts. Changed target/assignments revoke previous PA access. Empty target gives no PA read access. General visit/report/audit exports exclude restricted records and events; a separate restricted export is outside current scope.

Staff-created visits start Approved and reserve immediately. Creation/rescheduling checks same officer or occupied room against existing commitments. Public Pending requests do not reserve resources. Empty optional resources do not create fictitious reservations. Arrival opens start−30 minutes through end inclusive, derives Missed afterward if never checked in, and permits later checkout after check-in.

Source: RestrictedAppointment, BookingDetail and Reception features; domain.ts, validation.ts, scheduling.ts, attendance.ts and wire.ts. Synthetic policy tests verify projections/timing; production adapters validate the audience response and cannot grant authorization. The new backend is unbuilt and must enforce this contract transactionally. See API-CONTRACT.md, REQUIREMENTS-MATRIX.md and VERIFICATION.md.

G20 adds cancellation only for Protocol with explicit cancel permission in its restricted decision scope. A reason is mandatory; Pending/Approved visits may be cancelled after start/end only if never checked in. Replays/conflicts must not duplicate audit, and no internal reason becomes public.
