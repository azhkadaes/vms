# Unresolved contract boundaries

These are explicit decisions still needed within the selected target. They are distinct from missing implementation and production configuration. The review does not supply invented defaults; strict conformance remains blocked while they are open.

## OPEN-READ-ORDER

Affected operations: staffBookings, staffEmployees.

Selected rules require stable sorting but do not freeze database text collation/normalization or ID tie comparison. Frontend preview uses Indonesian localeCompare and lexicographic ID ties; employee target says numeric ID ties. These are different with IDs 2 and 10 or accented names.

Decision needed: Confirm exact text comparison and tie rules for backend paging before freezing ordering fixtures. Query defaults and authorization scopes are already selected; do not silently inherit host database collation.

Blocks: Exact cross-client sort parity acceptance, not the operation inventory or query syntax.

Evidence: [frontendx/packages/data/src/domain.ts](../../frontendx/packages/data/src/domain.ts), [docs/BACKEND-API-CONTRACT-2026-11-11.md](../../docs/BACKEND-API-CONTRACT-2026-11-11.md).

## Deferred boundary (not a Phase 1 decision)

**OPEN-LOGOUT-PROOF**: Not applicable to local-only Phase 1 logout; deferred cookie/session flow still needs a future explicit decision.

