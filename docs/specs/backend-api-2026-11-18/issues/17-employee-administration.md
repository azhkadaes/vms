# 17: Employee administration (ops 26–29)

**What to build:** Super Admin lists, searches, views, creates and edits employees:
- profile fields, multiple roles, active flag;
- a PA's single official assignment;
- an initial password and a forced change.

Edits are protected by version. Deactivation is a PATCH; there is no delete.

Changes that remove access end that employee's live sessions immediately. The system protects against self-lockout and against losing the last Super Admin.

Context:
- [field rules](../../../contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md): employee and role administration
- [validation](../../../contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-11, A-16
- decision 01

**Blocked by:** 02, 03, 04 (all depend on 00).

**Status:** done (merge 13b6210 on vms/backend-api)

**Verification:** PostgreSQL search, employee lifecycle, role and officer assignments, credential invalidation and audit. VIP assignment enforcement also checked in 72082af.

- [ ] **No migration:** the tables and grants come from ticket 00. Role assignments use `role_pegawai.id_karyawan`, and audit columns hold the acting employee ID as text.
- [ ] **List:** requires the people permission
  - [x] `q` searches name, email and NIP
  - [ ] `pageSize` 1–250
  - [ ] response includes `total` (filtered), `totalAll` (unfiltered) and `passwordMinLength`
  - [x] a page past the end → 200 with empty items
- [ ] **Detail, create and PATCH:**
  - [ ] carry jobTitle, deputy, directorate, phone, nip, `roles[]` (no singular role), active, assignedOfficerIds and version
  - [ ] passwords are never echoed
- [ ] **Create:**
  - [x] `initialPassword` follows the existing password policy
  - [x] the credential row and `requirePasswordChange` are stored
- [ ] **Errors:**
  - [x] duplicate email → 422 `email`
  - [x] PA assignment rules (PA only, at most one, non-VIP, active) → 422 `assignedOfficerIds`
  - [x] stale version → 409 `version_conflict`
- [ ] **Guards:**
  - [ ] you cannot change your own roles or active flag (403)
  - [ ] the last active Super Admin cannot be deactivated or lose that role
  - [ ] `""` clears optional profile fields
- [x] **Session effect:** deactivation, role removal or an assignment change bumps credential generation or role revision. The affected employee's next request then fails (test via the session read).
- [x] **Audit:** events record the staff actor and the employee target
