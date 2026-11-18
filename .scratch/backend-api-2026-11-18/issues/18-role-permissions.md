# 18: Role permissions (ops 30, 31)

**What to build:** Super Admin views the four fixed roles with their permissions, assignable permissions, editability and version. Changes are limited:
- only a role's assignable permissions can be changed;
- the fixed subset never changes;
- unknown permissions are rejected.

A saved change takes effect for live sessions of that role immediately.

Context:
- [field rules](../../../docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md): employee and role administration
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): A-17

**Blocked by:** 02, 03.

**Status:** ready-for-agent

- [ ] **No migration:** the tables, seeds and grants come from ticket 00. A permission change edits `policy_permission` for the role's active policy, and updates `role.version` and `updated_at`/`updated_by` (employee ID as text).
- [ ] **List:** `GET /v1/staff/roles` requires the roles permission and returns `{id, name, description, permissions, assignablePermissions, editable, version}` for PA, Protocol Officer, Receptionist and Super Admin
- [ ] **`PATCH /v1/staff/roles/{id}`** takes `{permissions, version}` and responds:
  - [ ] unknown permission → 422
  - [ ] changed non-assignable subset → 422
  - [ ] stale version → 409 `version_conflict`
- [ ] **Session effect:** success bumps the role revision. Live sessions with that active role then get 409 `role_changed` until they re-read their session.
- [ ] **Audit:** an event is written with the staff actor and the role target
