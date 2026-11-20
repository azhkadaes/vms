# 08: Static admission QR (op 07)

**What to build:** A visitor with an approved visit gets **one static QR** for that visit. Every request returns the same payload until the visit is cancelled, or rescheduled to a different date, time or duration.

The QR that checked a visit in can be presented again to check it out. A stable payload means a later server-side renderer (piglig/go-qr) produces the same image the browser draws today. Rendering itself is not part of this ticket.

- **Issue:** the first request for an approved, not-yet-ended visit creates the credential. Later requests return the identical payload and expiry, and write nothing.
- **Payload:** opaque and deterministic, derived from the credential row and a server secret. It is never stored; only its hash is.
- **Expiry:** the visit's scheduled end, which is the end of the check-in window. Checkout is not bound by it.
- **Refusal:** when the visit is not approved, or the scheduled end has passed.
- **No version change:** issuing does not change the visit's version, so the visitor's later actions still use the version from their last lookup.
- **Revocation:** public cancellation (ticket 07) now revokes the credential. Staff cancel (11) and schedule-changing reschedule (12) revoke it too.

This slice introduces `qr_kunjungan` with exactly the diagram's columns, and nothing else. The current visitor app works unchanged: it simply refreshes at `expiresAt`.

Context:
- [physical schema](../../../docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md): table 22, static credential lifecycle, decision 4
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): A-04, A-08
- earlier D96. The D21 rotation is superseded.

**Blocked by:** 07.

**Status:** done (merge 9e31e44 on vms/backend-api)

**Verification:** PostgreSQL concurrent issuance, replay, hash-only persistence, revocation and migration cycle.

- [x] **Migration:** one file for `qr_kunjungan`, with Up and Down
  - [x] columns: `id`, `id_kunjungan`, `token_hash`, `tanggal_dibuat`, `tanggal_expire`, `dicabut_pada` (TIMESTAMPTZ)
  - [x] partial unique index: one unrevoked credential per visit
  - [ ] grants per the physical schema
- [ ] **`POST /v1/public/bookings/{id}/admission` success:** `{code, email, version}` returns 200 `{payload, expiresAt, serverTime}`. `expiresAt` is the scheduled end in UTC `Z`.
- [ ] **Static payload:**
  - [x] repeated requests return a byte-identical payload and the same `expiresAt`, with no new row
  - [x] the payload is at most 4096 characters, fits a QR at medium error correction, and only its hash is stored
  - [ ] a payload cannot be forged for another visit without the server secret
- [ ] **Errors:**
  - [ ] proof mismatch → 404
  - [ ] stale version → 409 `version_conflict`
  - [ ] not approved, or `now ≥ endsAt` → 409 `invalid_state`
- [ ] **Behaviour:**
  - [x] the visit version is unchanged after issuing
  - [x] public cancel revokes the credential
  - [x] after revocation the next request issues a different payload
- [ ] **Rate limit:** admission shares the D29 credential budget with lookup and cancel
- [x] **Integration test:** concurrent first requests create exactly one credential, and all return the same payload
