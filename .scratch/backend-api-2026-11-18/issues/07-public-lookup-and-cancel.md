# 07: Public lookup and cancel (ops 05, 06)

**What to build:** A visitor looks up their visit by code and email (case-insensitive). They see:
- status, schedule, room and visitor instructions;
- public rejection or cancellation reasons;
- the rescheduled flag;
- timing;
- the visitor-visible history.

They can cancel a pending or approved visit strictly before it starts, sending the expected version.

This slice introduces the pure WITA timing rules in the policy, all computed from one server time per response: `opensAt`, `endsAt`, missed, `canCheckIn`, `canCheckOut`, `canCancel`.

`rescheduled` and the history come from the visit's `audit_log` events (`entity_type = kunjungan`), as the physical schema's derived values specify. The visitor-visible reschedule message is the allow-listed `reason` key only.

Context:
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): A-02, A-03, A-05, A-19
- decisions 12, 16, 25
- earlier D29

**Blocked by:** 06.

**Status:** ready-for-agent

- [ ] **Policy timing table tests:**
  - [ ] `opensAt` = start − 30 min
  - [ ] `endsAt` = start + duration
  - [ ] missed applies at `endsAt` + 1 ms
  - [ ] cancel is allowed strictly before start
- [ ] **Lookup scope:**
  - [ ] rejected, cancelled and checked-out visits are returned
  - [ ] no match, wrong email or a restricted visit → 404
- [ ] **Lookup never returns:**
  - [ ] internal notes or the coordination note
  - [ ] the service ID
  - [ ] any restricted record
- [ ] **History:** visitor-visible events only, oldest first; `publicMessage` only on reschedule events (A-03)
- [ ] **Cancel errors:**
  - [ ] the ID doesn't match the code and email pair → 404
  - [ ] stale version → 409 `version_conflict`
  - [ ] not pending or approved, or at or after start → 409 `invalid_state`
- [ ] **Cancel success:** sets cancelled, increments the version and writes an audit event with the visitor actor. Credential revocation is added in ticket 08.
- [ ] **Rate limit:** lookup and cancel share the D29 budget (30 per minute per IP) and return 429 with `Retry-After`
