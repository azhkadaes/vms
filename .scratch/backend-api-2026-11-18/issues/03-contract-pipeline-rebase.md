# 03: Contract pipeline rebase

**What to build:** The machine-readable contract tracks the 40-operation frontend-driven contract numbering. That contract is the manifest, the source baseline, the OpenAPI generation and the offline checker. Each later slice can then register its operations, and any drift between served and documented routes fails the check.

The implemented operations are marked with their real shapes, including the error body from ticket 01:
- 01 public catalog
- 32–36 sessions
- 39 refresh
- 40 renew

Context:
- [spec](../spec.md): housekeeping
- [conventions](../../../docs/BACKEND-GO-CONVENTIONS.md): §8 step 4
- the existing contract tooling in `docs/contracts`

**Blocked by:** 00 (the migration table list changes there), 01 (the error body schema changes there).

**Status:** ready-for-agent

- [ ] The manifest lists all 40 operations with current numbering, method, path and status (implemented / missing)
- [ ] The source baseline holds request and response schemas for the 8 implemented operations, matching what the server returns, including the new error body
- [ ] The contract check passes, and adding an unregistered route makes it fail (demonstrated, then reverted)
- [ ] The served OpenAPI is regenerated, and the path-count assertion is updated
- [ ] The documentation pipeline steps in the conventions doc still match reality
