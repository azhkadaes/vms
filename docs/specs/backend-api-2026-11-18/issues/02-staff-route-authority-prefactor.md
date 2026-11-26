# 02: Staff route and authority prefactor

**What to build:** Every future staff business route gets the same protection by construction:
- Bearer verification with live session authority and the per-employee limiter;
- the workspace gate, which blocks sessions that still need a role selected or a password changed.

Router dependencies provide both, and each staff group mounts them, so protection no longer depends on the order routes are registered.

Write transactions can recheck the caller's authority on their own transaction:
- The repository locks the employee row and reads the session snapshot.
- A `SessionValidator`, extracted from the current-session logic, validates it.
- The old helper, which made `pkg` import `internal`, is removed.

The policy gains named role and permission constants. Sessions in a pending state may only call the session operations listed in A-18.

Context:
- [spec](../spec.md): architecture
- [validation](../../../contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): A-18
- [conventions](../../../BACKEND-GO-CONVENTIONS.md): §2, §4, and §7 items 1, 4, 5

**Blocked by:** 00 (it rewrites the auth repository queries this ticket touches).

**Status:** done (merge ee11cf7 on vms/backend-api)

- [x] Staff authentication and the workspace gate are available from the router dependencies, and existing session routes behave identically (existing tests pass)
- [x] A group mounted with them returns:
  - [x] 401 when unauthenticated
  - [x] 403 when role selection is pending
  - [x] 403 when a password change is pending
- [x] A staff route registered before the auth router is still protected (regression test)
- [x] `SessionValidator` is extracted, current-session reads use it, and its behaviour is unchanged
- [x] The transaction-scoped authority lock locks the employee before any other row and returns the snapshot validated by `SessionValidator`
- [x] The old authority helper is deleted, and no `pkg` package imports `internal`
- [x] Policy uses role and permission constants; no new string literals for roles or permissions
- [x] Pending-state allowances from A-18 are enforced and tested:
  - [x] role selection required → only ops 33, 34, 35, 39, 40
  - [x] password change required → only ops 33, 35, 36, 39, 40
