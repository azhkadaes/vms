# 14: Booking file reads (ops 20, 21, 37, 38)

**What to build:** Two ways to open a visit's Nota Dinas or visitor photo:
- **Authorized staff:** GET, for visits in their scope.
- **Receptionists:** POST with the visitor's code or QR. Every request rechecks the credential, scope and attendance eligibility, and grants no lasting access.

Restricted visits never expose files. An unavailable file gets the same non-disclosing response as a forbidden one.

Context:
- [field rules](../../../docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md): Reception and files
- decisions 02, 08, 21

**Blocked by:** 05, 13.

**Status:** ready-for-agent

- [ ] **GET** attachment and visitor-photo follow the staff projection's scope and permission. Receptionist using GET → 403.
- [ ] **POST** with `{credential}` requires:
  - [ ] the Receptionist role
  - [ ] a matching credential
  - [ ] either an approved, check-in-eligible visit, or a checked-in visit with the checkout permission
- [ ] **Recheck:** every POST is rechecked, and no grant is remembered
- [ ] **Response headers:**
  - [ ] `Content-Type` is exactly `application/pdf`, `image/jpeg` or `image/png`
  - [ ] `Cache-Control: no-store`
- [ ] **Non-disclosure:** a missing file, a restricted visit and a disallowed caller all return the same 404 `not_found` body
- [ ] **Credential privacy:** the credential never appears in the query string, logs or errors
