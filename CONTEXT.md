# VMS domain glossary

The language the code, tests and docs should use. When the code disagrees with this file, the code is the thing to fix.

## Visits

**Visit** (Kunjungan): one person's planned presence at the office, from submission until checkout or cancellation. It is the central aggregate. Go entities already use `Visit`; TypeScript types and URL paths still say `Booking` for historical reasons.
_Avoid_: booking, appointment, reservation (except in URL paths such as `/staff/bookings`, which stay for compatibility).

**Visit kind**: `standard` or `restricted`. Go stores the restricted kind as `terbatas`; on the wire it is `restricted`.

**Restricted visit** (Kunjungan Terbatas): a visit created by staff for a sensitive guest. Its fields are projected per role and per `restrictedView` (`full`, `arrival`, `reception`).

**Visit command**: a staff instruction that changes a visit: approve, reject, reschedule, update, attend (check-in or check-out) and cancel. Each command carries the visit `version` it was based on, and the server rejects stale versions with 409.

**Attendance**: recording that a visitor checked in or out. Sent on the wire as `POST …/attendance` with `action: "check_in" | "check_out"`.

**Officer** (Pejabat): the staff member a visitor comes to see. On the wire it is `pejabat.nama`, and the TypeScript projection calls it `host`.
_Avoid_: host (outside display code).

**Nota Dinas (ND)**: the official letter attached to a visit request (a PDF).

**Visit file**: a binary that belongs to a visit: the ND attachment or the visitor photo.

**Arrival / Reception**: the front-desk flow in which a receptionist resolves a visitor's credential (QR payload or code) to a visit and records attendance.

**Admission**: the visitor's QR entry pass for an approved visit (`AdmissionQR`, `…/admission`).
_Avoid_: using "admission" for the public rate limit; that is a **Submission quota**.

**Submission quota**: the per-IP budget that limits public, unauthenticated requests. The Go code still calls it `admissionusecase.AdmissionUsecase`; it is due to be renamed.

**Visit log** (Log Kunjungan): the filterable, exportable history of visits for reporting.

## Staff

**Staff session**: an authenticated staff member's access token, refresh cycle and current authority (role and permissions).

**Role**: PA, Protokol, Receptionist, Super Admin, among others; a role grants a set of permissions.

## Tooling

**Preview server**: `frontendx/tooling/preview.ts`, an in-memory fake of the production HTTP contract used for offline development and fast integration tests. It must accept exactly what the Go backend accepts (see ADR-0001).
