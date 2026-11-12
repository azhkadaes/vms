# Visitor and backoffice API contract review

Reviewed 4 November 2026 (WITA).

## Verdict and scope

The visitor and backoffice apps broadly implement their proposed contract in `frontendx/API-CONTRACT.md`. Two response-validation weaknesses remain, described below. The attendance-contract ambiguity has been clarified in the contract. This is frontend contract alignment, not verification against an implemented backend.

The sole baseline is `frontendx/API-CONTRACT.md`, together with the runtime schemas and tests it explicitly gives precedence to. The intended backend has not been built. Reference backend folders and other backend implementations are outside this review and are not compatibility targets.

Reviewed the production adapters, response schemas, session handling, visitor submission/status/admission flows, bounded staff queries, decisions, reception, documents, administration, and reports. This is not a full visual/accessibility audit. No application source was changed. Builds and browser checks regenerated local build/test evidence.

## Findings against the frontend contract

### 1. P2 — Standard booking validation accepts incomplete records

`packages/data/src/wire.ts:39,59,78` makes all identity/detail fields and history optional for staff records, and enforces projection restrictions only when `visitKind === "restricted"`.

Reproduced: a schedule-only ordinary booking parses successfully; a record with `restrictedView:"reception"`, omitted `visitKind`, and a name also parses successfully. This does not prove a server disclosure, but weakens the intended boundary against malformed successful responses and adapter projection mistakes. PA scope can then silently drop a record without officerId while server counts remain nonzero.

Recommendation: separate standard and restricted response variants; require ordinary fields appropriate to each endpoint/audience; reject restricted markers without the restricted discriminator. Keep server authorization independent.

### 2. P2 — Read responses are not correlated to the requested record/page

`packages/data/src/read-model.ts:40` checks schema, metrics presence, and item bounds, but does not verify that returned page/pageSize match the request. `readStaffBooking` checks schema but not that returned `id` matches the requested ID.

A structurally valid wrong-page or wrong-record response is accepted and presented under the requested query/detail. Recommendation: validate response identity and pagination metadata against request context, with an explicit rule if the service is allowed to clamp pages.

### Resolved documentation ambiguity — Identity verification is local only

`apps/admin/src/pages/Reception.tsx:300,323,330` requires an identity-confirmation checkbox, but sends only credential plus version. The revised frontend contract explicitly records this wire body and the UI confirmation step. No identityVerified field is currently required. A future requirement for a submitted attestation would need a coordinated contract and frontend change; it is not a demonstrated bypass of server admission checks.

## What already aligns with the frontend proposal

- Production uses same-origin `/v1`, cookie credentials, mutation marker headers, encoded record IDs, and no caller-selected production role.
- Public lookup credentials are sent in POST bodies; public 401 does not clear the staff session.
- Public submission retains a retry reference, retries temporary transport failure once, and requires resolution of uncertain outcomes before edited resubmission.
- Public forms consume service officer IDs, gate on privacy notice/consent, and normalize the representative-first roster.
- Booking actions include optimistic versions; staff list/detail/dashboard queries are invalidated after successful booking changes.
- Staff reads are bounded; active booking queries refresh at 30 seconds or 15 seconds for reception, pause when hidden, and refresh on return/reconnect.
- Restricted projections, public history, internal-note separation, protected document reads, and management secret stripping have dedicated handling/tests.
- Management edits use versions and preserve drafts on conflicts; reports use an independent aggregate source and revision-gated export.

## Verification performed

| Check | Result |
| --- | --- |
| TypeScript type check | Passed |
| Visitor production build | Passed |
| Backoffice production build | Passed |
| Unit/domain/contract tests | 96 passed, 0 failed |
| Fresh production-bundle browser tests | 13 passed, 0 failed |
| Targeted staff schema probes | Reproduced finding 1 |

The standard npm launcher fails because its configured npm-cli.js is missing. Checks were run directly using Node and the installed TypeScript/Vite entry points. Builds emitted third-party Zod comment-annotation warnings but completed successfully.

Production browser tests intercept `/v1/**` with fixtures matching the frontend proposal. They exercise the built frontends, including public upload/submission/admission, authentication failures, management, bounded lists/detail/download, permission changes, and reports. They establish behavior against those fixtures, not an implemented service. They do not cover every documented endpoint and failure combination.

## Recommended next step

Strengthen standard/restricted response discrimination and correlate record/page responses with request context. The API contract now documents field-level response/input requirements and screen-to-data traceability against frontendx, including the existing reception wire body. A documentation coverage check found all 107 top-level response-schema fields mentioned and all 24 full source paths valid; this is a coverage check, not proof of runtime interoperability. Keep this frontend-driven contract as the baseline for the future backend, then verify interoperability once that implementation exists. No changes to product workflows are indicated by this review.
