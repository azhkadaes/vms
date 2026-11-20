# 12: Reschedule and restricted update (ops 14, 16)

**What to build:** The owner can reschedule an approved visit, including a missed one up to seven days after its original end (inclusive). The new start must be in the future, between 08:00 and 16:00 WITA. Reschedule also serves as the general "Edit kunjungan": changing only notes or room does not mark the visit rescheduled or revoke its QR.

The Protocol Officer can update a restricted visit's identity fields, notes and room. Date and time stay read-only, per default D-03.

Context:
- [field rules](../../../docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md): rescheduling eligibility; restricted update reconciliation
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-12, A-07
- decision 29

**Blocked by:** 11.

**Status:** done (merge 83da6ee on vms/backend-api)

**Verification:** PostgreSQL reschedule racing approval, restricted date refusal and identity clearing, plus projection and HTTP checks.

- [ ] **Eligibility boundaries:**
  - [ ] eligible through `originalEnd + 7 × 24 h` inclusive
  - [ ] 1 ms later → 409 `reschedule_window_closed`
  - [ ] pending or any other non-approved stored state → 409 `invalid_state`
- [ ] **Reason:**
  - [ ] ordinary visits: at least 5 trimmed characters
  - [ ] restricted visits: optional, but at least 5 if supplied
  - [ ] at most 1000 characters
- [ ] **Room and conflicts:** `roomId` follows the omit / null / ID rule; conflicts are checked against the effective officer, time and room
- [ ] **Effects:**
  - [x] a change to date, time or duration writes a reschedule audit event (with the reason in `new_values`), which derives `rescheduled` and the visitor-visible message; it also revokes admission credentials
  - [ ] a notes- or room-only edit writes an ordinary update event instead, and does none of these
  - [x] the version always increments
- [ ] **Restricted update:**
  - [ ] requires Protocol Officer with the create permission, on a restricted visit that is pending or approved
  - [x] empty name, organization or `serviceId` clears the field
  - [ ] `officerId` cannot be cleared
  - [x] a changed date or time → 422
- [x] **Integration test:** a reschedule racing an approval of an overlapping visit → exactly one succeeds
