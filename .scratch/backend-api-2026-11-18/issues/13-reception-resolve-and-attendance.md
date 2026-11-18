# 13: Reception resolve and attendance (ops 17, 18)

**What to build:** A receptionist resolves a typed booking code or a scanned QR into the arrival projection: roster, purpose, file availability, ND name, timing and version. After confirming the visitor's identity, they record check-in or check-out through one attendance command (`action`, `credential`, `version`).

Each successful attendance command writes exactly one transition, one server-time event and one audit event.

The admission QR is static (ticket 08). The same QR the visitor showed at check-in is scanned again at checkout. Checkout works for any checked-in visit, including overnight visits and after the QR's check-in expiry.

Context:
- [field rules](../../../docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md): Reception and files
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-03, V-14, A-05, A-10
- decisions 17, 18, 31
- earlier D96

**Blocked by:** 08, 09.

**Status:** ready-for-agent

- [ ] **Resolve output:** the arrival projection from V-03; restricted visits return only the arrival or reception view keys
- [ ] **Resolve rejections:** unknown, revoked or out-of-scope credentials → 404. The credential never appears in errors or logs.
- [ ] **`check_in`:**
  - [ ] requires the Receptionist role and the checkin permission
  - [ ] the visit must be approved and within `opensAt`..`endsAt` inclusive
  - [ ] the credential is the booking code or the visit's unrevoked QR, and the QR is within its expiry
  - [ ] sets checked_in with the server check-in time; the QR stays valid for checkout
- [ ] **`check_out`:**
  - [ ] the visit is checked_in, on any date
  - [ ] accepts the booking code, or the same unrevoked QR used at check-in even after its expiry; no consumption column exists
  - [ ] sets checked_out with the server checkout time; afterwards the QR no longer resolves for attendance
- [ ] **Errors:**
  - [ ] invalid or missing action → 422
  - [ ] stale version or a transition already done → 409
  - [ ] a check-in whose QR passed its expiry between resolve and confirm → 409 `credential_expired`
- [ ] **Integration tests:**
  - [ ] concurrent check-ins of one visit record exactly one transition
  - [ ] one QR checks a visit in, then checks it out on a later day
