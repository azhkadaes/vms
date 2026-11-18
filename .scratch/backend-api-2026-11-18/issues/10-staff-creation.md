# 10: Staff booking creation (op 11)

**What to build:** Staff create visits on a guest's behalf. Staff-created visits start approved and record who created them, in which role. Who can create what:
- **PA:** standard visits for their assigned, active, non-VIP officers.
- **Protocol Officer:** standard visits for VIP officers, and restricted visits for any active officer.

This slice introduces the write lock order and the conflict engine that later decisions reuse:
- **Conflict rule:** same officer, or same non-null room, with overlapping `[start, end)` against approved or checked-in visits.
- **Lock order:** employee → officers → rooms, then the overlap query.

Context:
- [spec](../spec.md): architecture; transactions
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-06, A-06, A-07, A-14
- decisions 10, 15, 27, 28

**Blocked by:** 05, 09.

**Status:** ready-for-agent

- [ ] **Standard create:**
  - [ ] representative, contact, roster, identity type and photo token are required
  - [ ] the ND token is optional
  - [ ] duration 15–240, defaulting to 60
  - [ ] `roomId` is optional but must be a valid active room
  - [ ] notes are at most 2000 characters
  - [ ] the start is in the future, between 08:00 and 16:00 WITA
  - [ ] `visitKind` defaults to `standard`
- [ ] **Restricted create:**
  - [ ] `officerId`, date and time are required
  - [ ] name, organization, `serviceId` (≤100 characters, leading zeros kept), `roomId` and notes are optional; blank optional fields become absent
  - [ ] no contact, roster or photo fields are accepted
  - [ ] the visitor-detail row is still created
- [ ] **Errors:**
  - [ ] the creator isn't allowed this kind of visit or officer → 403
  - [ ] the officer is inactive or out of scope → 422 `officerId`
  - [ ] an overlap → 409 `schedule_conflict`
- [ ] **Idempotency:** same key and payload returns the original; a changed payload → 409
- [ ] **Created visit:** approved, version 1. The creation audit event records the staff actor and the active role in `new_values`; that is the source of `createdBy`. No creator column is added to `kunjungan`.
- [ ] **Integration test:** two concurrent creates for the same officer and overlapping time → one success, one 409
