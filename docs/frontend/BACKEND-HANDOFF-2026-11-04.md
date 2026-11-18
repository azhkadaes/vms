# Backend engineering handoff — 4 November 2026

## Readiness decision

**Ready to start backend engineering from the frontend contract, with tracked frontend follow-ups. Not cleared for live operations or direct connection to the separate Go backend.**

Scope is `D:/oikn/VMS/frontendx`, visitor and staff apps. This handoff preserves the agreed frontend workflows and does not select a backend implementation, database, or replacement visitor verification flow. `D:/oikn/VMS/backend` exists as an empty directory on this review date. The separate Go project is not the owning contract; its earlier mismatch review is [context for reuse decisions](BACKEND-COMPATIBILITY-2026-11-04.md), not fresh interoperability evidence from this check.

Use [API-CONTRACT.md](API-CONTRACT.md) as the request/response and business-rule baseline, [BUSINESS-REQUIREMENTS.md](BUSINESS-REQUIREMENTS.md) for requirements, and [REQUIREMENTS-MATRIX.md](REQUIREMENTS-MATRIX.md) for the route/source/test mapping. Read [current readiness evidence](FRONTEND-READINESS-2026-11-04.md) before treating passing fixture checks as integration evidence.

## Review plan and assumptions

- Goal/users: give backend engineers enough verified frontend behavior to implement visitor registration/status/admission, scoped PA/Protocol decisions, reception, administration and reporting.
- Scope: inspect production adapters, runtime/input schemas, session/cache ownership, bounded queries and existing quality checks; preserve behavior and record integration gates. This review changes documentation and adds reproducible evidence only.
- Work needed: reuse the existing checks, adapters, schemas and traceability rather than adding another data abstraction or dependency. Run build and contract checks now; defer deployment measurements and physical-device acceptance until a real environment exists. Persistence, authorization, aggregates and concurrency belong on the service; forms, navigation and temporary interaction state stay in the client.
- Data/state: server responses belong to TanStack Query, keyed by session identity/permissions/assignments; filters/sort/pages belong to URLs; drafts to forms; session to SessionProvider; preferences to browser storage. Public submission recovery stores a reference, not the submitted visitor payload.
- Fit: the existing shared data package is the integration boundary for both apps. Contract changes there recur across screens and must be coordinated. A broad frontend rewrite would increase integration risk without resolving a demonstrated requirement.
- Build order: session/errors/catalogs first, then one public creation/status slice, staff decisions, reception, restricted projections/documents, administration/reports. Complete failure/concurrency acceptance within each slice.
- Assumptions: retain G02 email/code public lookup, G09 one canonical staff role, G17 administrator-set initial password, G19 20-person/5 MB supporting-document limits and G20 staff cancellation rules. G14 optional draft recovery remains deferred. Do not adopt different behavior from another backend without a coordinated decision.

## Where integration belongs

| Concern | Owning frontend source under `packages/data/src/` |
| --- | --- |
| Preview selection and API prefix | `config.ts` |
| Cookies, mutation marker, abort/timeout and error envelope | `http.ts`, `errors.ts` |
| Staff sessions and cache identity | `session.tsx`, `session-contract.ts`, `query.tsx` |
| Public/staff creation, lookup, actions, admission, reporting requests | `repositories.ts`, `contracts.ts` |
| Bounded staff list/dashboard/reception and direct detail | `read-model.ts`, `hooks.ts` |
| Successful booking/catalog/admission responses | `wire.ts` |
| Normalization and field validation | `validation.ts`, `submission.ts`, `scheduling.ts`, `attendance.ts` |
| Directory/roles/integrations | `management.ts`, `management-contract.ts` |
| Upload and protected download | `attachments.ts` |
| Aggregate summary/export and public event allowlist | `reports.ts`, `history.ts` |

Components should continue consuming these adapters/hooks. Do not scatter new service URLs or backend-specific envelopes through pages. The contract's field-level and screen-to-data tables supply the complete response requirements; passing a permissive schema is insufficient.

## Running and switching to the service

Use Node 22 and npm from the frontend root:

```powershell
npm.cmd ci
npm.cmd run dev
npm.cmd run check
```

Visitor development origin is `http://127.0.0.1:3000/`; staff is `/admin/` through that origin or `http://127.0.0.1:3002/admin/`. Development on loopback automatically selects synthetic preview behavior. The configured development `/v1` proxy alone does **not** make the normal preview a live integration test. There is no `VITE_USE_MOCK_API` switch in this project.

Build with `npm.cmd run build`, then serve `apps/visitor/dist` at `/` and `apps/admin/dist` at `/admin/` behind one origin that routes `/v1` to the intended backend. Use SPA fallback for application routes only; API/file misses must retain real error responses. Production builds select the HTTP adapters. `VITE_API_BASE_PATH` is a build-time same-origin path prefix (default `/v1`), not a hostname or a payload/permission mapper. Rebuild both apps after changing it. All `VITE_*` values are public.

`tests/production.browser.mjs` serves the built apps and intercepts API requests with fixtures. Keep that suite for client regression coverage, and add separate tests against the real service without interception. Do not point destructive fixture scenarios at a production database. Use a disposable integration database and test accounts.

## Service implementation slices

| Slice | Required behavior | Integration exit evidence |
| --- | --- | --- |
| 1. Session/errors/catalogs | `/staff/sessions`, `/staff/session`, `/staff/password`; one canonical role plus permissions/assignments; published ID/EN privacy notice and authorized catalogs; top-level `{message,code?,fieldErrors?}` errors | Real cookie sign-in/out, required password change, 401/403 distinction, role/assignment revocation and cache clearing; 422 field mapping; 429 cooldown and 503 retry |
| 2. Public visit | `/public/attachments`, `/public/bookings`, `/public/submissions/resolve`, `/public/lookup`; ordinary named roster and consent version | Upload-link-create round trip; same-key replay; changed payload 409; committed/uncertain/not-committed recovery across restart; anti-enumeration and public projection |
| 3. Staff agenda/decisions | Bounded `/staff/bookings` list/dashboard/reception, direct `/:id`, staff creation and versioned action routes | Scope before search/count/page; full-scope metrics; double approval/resource race has one winner; staff-created visits auto-approve and reserve immediately; conflict preserves draft |
| 4. Admission/reception | Public admission, `/staff/resolve`, versioned `checkin`/`checkout` with `{credential,version}` | Real opaque issuance/replacement/expiry/revocation; replay prevention; concurrent scan behavior; server-time arrival boundary and prior-day checkout; UI identity confirmation remains local |
| 5. Privacy/documents | Restricted full/assigned-pa/reception/arrival variants; authorized file download | Cross-role direct-read/search/count/file/export denial; no restricted public/report data; no internal notes in public history/email; private storage/content validation and orphan cleanup |
| 6. Management/reporting | Employees, roles, integrations and independent summary/export endpoints | Version conflicts; inactive assignments; lockout/self-escalation protection; write-only passwords/secrets; correct aggregate scope and revision-checked summary CSV |

For all slices, persist the mutation, optimistic version, audit and logical notification intent consistently; enforce permission and scope on the service. Do not infer server authority from hidden buttons or preview roles. Use durable jobs/outbox handling for delivery and retries; a rendered receipt is not proof of email delivery.

## Rules that must survive implementation

- Dates/start times use Asia/Makassar (UTC+8); event timestamps are explicit ISO values and timing metadata is service-authoritative. Ordinary starts are 08:00–16:00. Validate future starts and live catalog authorization server-side.
- Pending public submissions do not reserve resources. Approved/checked-in commitments do; approval, staff creation and immediate fixed rescheduling check conflicts atomically. Mutations recheck `version` and report conflicts as 409.
- Public cancellation is Pending/Approved, before start and before check-in. Staff cancellation requires explicit `cancel` permission plus decision scope, a trimmed 5–1000 character visitor-facing reason, and optional private note up to 2000 characters; it may happen after start/end before check-in. Release resources, revoke admission and audit. Never send the private note publicly or by visitor email.
- Check-in is start minus 30 minutes through end inclusive. Never-checked-in visits become derived Missed after end; checked-in visits can check out later, including on another date.
- Public roster includes the representative first and at most 20 people. Phone/identity text preserves leading zeroes. Supporting ND files are PDF/JPEG/PNG up to 5 MB; they are not a required identity-image flow.
- Lists use page/pageSize, counts after authorization and filtering, and bounded samples (7 list, 25 reception, 50 dashboard; response maximum 100). Apply deterministic ordering and echo the requested page metadata, including empty pages; coordinate any clamping policy explicitly.
- Staff/reception active reads refresh at 30/15 seconds, pause when hidden and refresh on return/reconnect. Maintain data availability for these bounded models rather than requiring a full-history download.
- General reports exclude restricted visits before aggregation and use independent read/export permissions. Export is summary-only with revision checking.

## Open items and ownership

| Priority/owner | Item | Completion criterion |
| --- | --- | --- |
| P2 frontend, before live integration acceptance | Staff schema accepts incomplete ordinary records and restricted markers without the required discriminator | Explicit ordinary/restricted audience variants; reject inconsistent markers; regressions cover list/detail/resolve/mutations without breaking valid projections |
| P2 frontend, before live integration acceptance | Staff read responses are not correlated with requested record/page | Reject wrong detail ID and wrong page/pageSize; test valid/empty pages and any agreed clamping behavior |
| Backend + frontend lead, before reusing the separate Go implementation | Contract/workflow differences | Explicit choice for role/permission semantics, visitor verification, identity collection, immediate vs proposal rescheduling, and missing staff/restricted/report/admin routes; update contract and both sides together |
| Organization + backend, before genuine data collection | Privacy notice, retention/deletion and rights contact | Approved published policy version; enforce consent and storage/log/audit lifecycle; do not invent retention durations from optional draft plans |
| Deployment + QA, before launch | Environment/security/operational acceptance | Real TLS/cookies/CSRF/Origin, private cache/security headers, storage/email, restart recovery, backups/restore and monitoring evidence; device/browser/assistive technology checks |
| Frontend + deployment, follow-up | Visitor mobile performance target | Measure actual served compression/device/network; prior local Lighthouse mobile median was 85, with provisional mobile >=90/LCP <=2.5s unmet; no new Lighthouse score was measured here |

The two P2 frontend items are reproducible in `anti-slop/evidence-2026-11-04-handoff/response-probes.mjs`; no application code was changed by this review. Backend engineers should implement the documented complete shapes immediately rather than rely on those parser gaps. Detailed response expectations remain in API-CONTRACT.md.

## Handoff acceptance

Backend engineering can begin when this document, the API contract, requirements and open-item ownership are understood. Integration acceptance requires the real service to pass each slice above, plus the frontend response-validation follow-ups. Operational release additionally requires the [deployment gates](DEPLOYMENT.md). Record real service responses and failures, concurrency/restart evidence and manual device results separately from fixture output. Do not label this handoff as production approval.
