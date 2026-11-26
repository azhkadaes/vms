# 01: Error and transport prefactor

**What to build:** Every staff and visitor error becomes something the UI can act on:
- **Field errors** are keyed by the flat form key and carry a list of Indonesian messages.
- **Messages:** each error code has its own user-facing message.
- **409s:** each 409 cause has a distinct code.
- **Missing records** are reported as 404, not 500.
- **Operators** get a log line for every 5xx with the request ID and a redacted cause class.

This ticket also adds the shared request pieces every later slice needs:
- an allow-listed typed query parser;
- a tri-state optional field (omitted / null / value);
- a bounded multipart reader.

All API timestamps serialize as UTC with `Z`. A wrong current password on password change now returns a field error instead of signing the user out.

Context:
- [spec](../spec.md): transport and error contract
- [validation](../../../contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-08, V-10(a), V-13
- [conventions](../../../BACKEND-GO-CONVENTIONS.md): §4–5

**Blocked by:** None (can start immediately).

**Status:** done (commit 402e94f on vms/backend-api)

- [x] Error body is `{code, message, requestId, fieldErrors?}` with `fieldErrors: Record<string, string[]>`, and the existing single-field callers are migrated
- [x] Every code in the conventions catalogue has a user-facing Indonesian message, and responses never contain causes, SQL or secrets
- [x] A repository not-found sentinel maps to 404 `not_found`; 409 and 422 can no longer surface as 500
- [x] Each 5xx log line has request ID, status, code and a redacted cause class, with no bodies, tokens or credentials
- [x] Typed query parser tests:
  - [x] an unknown key gives 400 `invalid_query_parameter`
  - [x] repeated keys keep their order
  - [x] an empty value counts as unset
  - [x] integer bounds are enforced
- [x] Tri-state optional tests distinguish omitted, `null` and value under the strict JSON decoder
- [x] The multipart reader rejects oversized bodies before buffering (413) and reports missing parts as 422
- [x] Every API timestamp is UTC RFC 3339 with `Z`
- [x] Password change with a wrong current password:
  - [x] returns 422 with `fieldErrors.currentPassword`
  - [x] leaves the session intact
  - [x] has a test proving both
- [x] Existing auth HTTP tests and contract captures still pass
