# 14: Booking file reads (ops 20, 21, 37, 38)

**What to build:** Two ways to open a visit's Nota Dinas or visitor photo:
- **Authorized staff:** GET, for visits in their scope.
- **Receptionists:** POST with the visitor's code or QR. Every request rechecks the credential, scope and attendance eligibility, and grants no lasting access.

Restricted visits never expose files. An unavailable file gets the same non-disclosing response as a forbidden one.

Context:
- [field rules](../../../docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md): Reception and files
- decisions 02, 08, 21

**Blocked by:** 05, 13.

**Status:** done (merge bd6e8c1 on vms/backend-api)

**Verification:** PostgreSQL attachment metadata and credential scope, temporary private files, and binary response tests.

- [ ] **GET** attachment and visitor-photo follow the staff projection's scope and permission. Receptionist using GET → 403.
- [x] **POST** with `{credential}` requires:
  - [ ] the Receptionist role
  - [x] a matching credential
  - [ ] either an approved, check-in-eligible visit, or a checked-in visit with the checkout permission
- [ ] **Recheck:** every POST is rechecked, and no grant is remembered
- [ ] **Response headers:**
  - [x] `Content-Type` is exactly `application/pdf`, `image/jpeg` or `image/png`
  - [x] `Cache-Control: no-store`
- [x] **Non-disclosure:** a missing file, a restricted visit and a disallowed caller all return the same 404 `not_found` body
- [x] **Credential privacy:** the credential never appears in the query string, logs or errors
