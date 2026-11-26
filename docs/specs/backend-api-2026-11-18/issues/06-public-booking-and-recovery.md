# 06: Public booking and submission recovery (ops 03, 04)

**What to build:** A visitor registers a visit in one request and gets back their PublicBooking with an 8-digit code. If the response is lost, they can resolve the submission by idempotency key plus email, without booking twice.

**Schema introduced:** one migration per table, exactly as in the physical schema:
- `detail_kunjungan`;
- `detail_pengunjung` (the agreed per-visit replacement for the diagram's Pengunjung);
- `kunjungan`, with the `tipe_kunjungan` and `status_penerimaan` ENUM types;
- `support.booking_requests`.

The creation audit event (visitor actor, `entity_type = kunjungan`) records the acknowledged privacy version. No extra column is added to `kunjungan` for it.

**Validation follows the field rules:**
- the officer is active;
- the start is in the future, between 08:00 and 16:00 WITA;
- purpose is 5–1000 characters;
- the roster has 1–20 names with the representative first;
- identity type is KTP, Paspor or SIM;
- the photo token is required and has the visitorPhoto purpose; the document token is optional;
- `ack` is true;
- the privacy version matches the published notice;
- `visitKind` is `standard`.

**New visits:**
- start pending at version 1 with a 60-minute duration;
- snapshot their VIP kind from the officer's position.

Context:
- [physical schema](../../../contracts/PHYSICAL-SCHEMA-2026-11-18.md): tables 19–21 and 30; derived values
- [spec](../spec.md): API decisions for ops 03/11
- [validation](../../../contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-06, V-07, A-01, A-06, A-20
- decisions 09, 11, 13, 22, 26, 27
- earlier D22, D30, D32

**Blocked by:** 05.

**Status:** done (merge a60761d on vms/backend-api)

**Verification:** PostgreSQL tests cover same-key concurrent creates, recovery racing a create, mismatched email, changed payload and rollback of attachment links. The route test covers the shared D32 budget.

- [ ] **Migrations:**
  - [ ] four files, one per table, each with Up and Down; Down also drops any ENUM type that table introduced
  - [ ] columns, types, nullability and checks exactly as in the physical schema
  - [ ] `jumlah_pengunjung` equals the roster length
  - [ ] the status/check-in/checkout consistency checks
  - [ ] readiness extended to the new tables
- [ ] **`POST /v1/public/bookings` responses:**
  - [ ] 201 with the PublicBooking
  - [x] same key and payload → 200 with the original booking
  - [x] same key with a changed payload → 409 `idempotency_conflict`
  - [ ] outdated privacy version → 409 `notice_outdated`
- [ ] **Validation errors:**
  - [ ] 422 with flat field keys (`visitors`, not `visitors.3`)
  - [ ] a wrong-purpose, expired, already-linked or foreign token gives a field error on its own token field
- [x] **Atomic create:** `kunjungan`, both detail rows, `form_documents` links (Nota Dinas on `detail_kunjungan`, photo on `detail_pengunjung`), the request record and the audit event (visitor actor) commit together, or nothing commits. Upload tokens are not consumed when a create fails.
- [ ] **Booking code:** the 8-digit code is unique, and a collision is retried
- [ ] **PublicBooking wire invariants:**
  - [ ] absent optional fields are omitted, never `null`
  - [ ] `room` is `""` when there is no room
  - [ ] timestamps are UTC `Z`; timing values are epoch milliseconds
  - [ ] `pejabat.nama` is present
- [ ] **`POST /v1/public/submissions/resolve`:**
  - [ ] a committed key returns `{outcome:"committed", booking}`
  - [ ] an unknown key returns `not_committed` and closes the key; a later create with it gets 409
  - [x] a committed key with a different email returns 404
- [x] **Rate limits:** D30 (30 new bookings per hour per IP; replays are free) and D32 (30 requests per minute) respond 429 with `Retry-After`
- [ ] **Integration tests:**
  - [x] parallel identical creates commit one booking
  - [x] a resolve racing a create never reports `not_committed` for a booking that committed
