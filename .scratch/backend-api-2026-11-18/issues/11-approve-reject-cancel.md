# 11: Approve, reject and staff cancel (ops 12, 13, 15)

**What to build:** The visit's owner, holding the matching permission, can:
- **approve** a pending visit, with a conflict check;
- **reject** a pending visit, with a public reason and a separate internal note;
- **cancel** a pending or approved visit (including a missed one), with a reason and an internal note. Cancelling revokes admission credentials.

Every decision checks the expected version and is atomic.

Context:
- [field rules](../../../docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md): staff creation and commands
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-14, A-06, A-08, A-14, A-20
- decisions 13, 15, 25

**Blocked by:** 08, 10.

**Status:** ready-for-agent

- [ ] **Approve:**
  - [ ] pending → approved; date and time are unchanged
  - [ ] `roomId` follows the omit / null / ID rule
  - [ ] a blank duration becomes 60
  - [ ] notes follow the omit-keeps / empty-clears rule
  - [ ] an overlap → 409 `schedule_conflict`
- [ ] **Reject:**
  - [ ] pending → rejected
  - [ ] `rejectionReason` is 5–1000 characters
  - [ ] `internalRejectionNote` is at most 2000 characters
- [ ] **Cancel:**
  - [ ] pending or approved (including missed) → cancelled
  - [ ] `reason` is 5–1000 characters; `internalCancellationNote` is at most 2000
  - [ ] admission credentials are revoked
- [ ] **Errors:**
  - [ ] not the owner, or missing permission → 403
  - [ ] out of scope → 404
  - [ ] stale version → 409 `version_conflict`
  - [ ] wrong state → 409 `invalid_state`
  - [ ] no partial writes on any failure
- [ ] **On success:** the version increments and the audit event is written in the same transaction
- [ ] **Visibility:** visitors see public reasons; internal notes never appear in public or reception projections
- [ ] **Integration tests:**
  - [ ] concurrent approvals of two overlapping pending visits → exactly one succeeds
  - [ ] a double-submitted approve → one transition
