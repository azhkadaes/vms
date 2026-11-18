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
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): A-18
- [conventions](../../../docs/BACKEND-GO-CONVENTIONS.md): §2, §4, and §7 items 1, 4, 5

**Blocked by:** 00 (it rewrites the auth repository queries this ticket touches).

**Status:** ready-for-agent

- [ ] Staff authentication and the workspace gate are available from the router dependencies, and existing session routes behave identically (existing tests pass)
- [ ] A group mounted with them returns:
  - [ ] 401 when unauthenticated
  - [ ] 403 when role selection is pending
  - [ ] 403 when a password change is pending
- [ ] A staff route registered before the auth router is still protected (regression test)
- [ ] `SessionValidator` is extracted, current-session reads use it, and its behaviour is unchanged
- [ ] The transaction-scoped authority lock locks the employee before any other row and returns the snapshot validated by `SessionValidator`
- [ ] The old authority helper is deleted, and no `pkg` package imports `internal`
- [ ] Policy uses role and permission constants; no new string literals for roles or permissions
- [ ] Pending-state allowances from A-18 are enforced and tested:
  - [ ] role selection required → only ops 33, 34, 35, 39, 40
  - [ ] password change required → only ops 33, 35, 36, 39, 40
