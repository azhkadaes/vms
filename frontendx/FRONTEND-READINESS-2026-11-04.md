# Frontend readiness recheck — 4 November 2026

## Verdict and scope

**Ready for backend engineering handoff with explicit frontend follow-ups. Live integration and operational release remain unverified.**

Reviewed `D:/oikn/VMS/frontendx`. The contract baseline is the frontend's API-CONTRACT.md and field/screen requirements, not a separate backend's DTOs. No app source, workflow, dependency, backend, database or deployment was changed. Added documentation and reproducible diagnostic evidence; normal check tooling regenerated local bundles and browser evidence.

The full `npm.cmd run check` completed with exit 0 using Node 22.14.0 and npm 10.9.2 against the existing installed dependencies. No clean reinstall or remote CI run was performed. Prior results remain historical in VERIFICATION.md. Reproducible source probes confirmed both existing P2 findings from API-COMPATIBILITY-REVIEW-2026-11-04.md.

## Verified architectural findings

- Shared typed adapters isolate the API boundary; successful responses use runtime schemas. Cookie requests, mutation marker headers, abort/15-second timeout, HTTP/field-error handling and session-loss propagation are implemented in the shared data package.
- Query ownership and invalidation are centralized; staff reads are bounded in production, scoped query keys include session identity, permissions and assignments, and obsolete scoped queries are removed on session changes. UI filtering is supplementary; server scope remains mandatory.
- Public creation carries a payload-bound idempotency reference, consent version, normalized representative-first roster and opaque upload token. Unknown outcomes use a resolution path before edited resubmission. These client paths require durable service behavior.
- Production read/list/action, private-document, management and aggregate-report paths exist. Loopback development uses the synthetic preview, so a successful development click-through cannot establish real integration.
- No direct rendered HTML injection was found by the targeted `dangerouslySetInnerHTML` source search. Browser storage matches inspected non-token uses: language/sidebar preferences, preview role and submission reference. This is a targeted inspection, not a full security audit.

## Findings requiring follow-up

| ID | Severity | Source | Reproduced behavior and consequence | Required action |
| --- | --- | --- | --- | --- |
| FE-01 | P2 | `packages/data/src/wire.ts`, `staffBookingResponse` | Schedule-only ordinary records pass; a reception marker without restricted visitKind can also pass with identity data. Incomplete/wrong audience data can reach UI or disappear under supplemental scope filtering. This is not proof of a server disclosure. | Require complete ordinary records and coherent restricted audience variants; test valid and malformed projections. Service must authorize/provide complete data independently. |
| FE-02 | P2 | `packages/data/src/read-model.ts`, both reads | A valid detail for a different ID is accepted; a page 2/pageSize 25 response is accepted for a page 1/pageSize 7 request when item count fits. UI can show the wrong record/page under the requested selection. | Correlate parsed responses to request identity and page metadata; agree any page-clamping policy and test it. |

Diagnostic command: `node --experimental-strip-types anti-slop/evidence-2026-11-04-handoff/response-probes.mjs`. The probe asserts that these weaknesses currently reproduce; a successful probe is **not** evidence that they are fixed. It stubs transport in memory and does not contact any backend.

## Fresh automated verification

| Check | Fresh result |
| --- | --- |
| Formatting | Passed the configured Prettier scope |
| Strict TypeScript | Passed |
| Logic/domain/contract tests | 96 passed, 0 failed/skipped |
| Isolated synthetic API tests | 34 passed, 0 failed/skipped |
| Visitor and staff production builds | Both passed |
| Sequential preview browser/accessibility tests | 50 passed, 0 failed/skipped, headless Chrome |
| Built-production browser/contract tests | 13 passed, 0 failed/skipped, headless Chrome with API interception |
| Fresh npm dependency audit | 0 reported vulnerabilities |
| Requirement identifiers | 55 unique IDs in the existing matrix; completeness of business acceptance was not re-established by counting |

Total: **193 passing tests**, excluding the diagnostic probes. Full output: [check.log](anti-slop/evidence-2026-11-04-handoff/check.log). Browser tools started/stopped their own servers; no deployment was performed. The fixture suites verify client behavior against supplied/synthetic responses, not interoperability with a real backend. The current npm launcher worked; the launcher problem noted in an earlier same-day review did not reproduce in this run.

Evidence: [response probe results](anti-slop/evidence-2026-11-04-handoff/response-probes.log), [source SHA-256 manifest](anti-slop/evidence-2026-11-04-handoff/source-sha256.json), and [local document-link/requirement check](anti-slop/evidence-2026-11-04-handoff/document-check.json). The manifest covers 511 app/package/tooling/test and contract/configuration files, excluding installed dependencies and build output, to identify the reviewed snapshot in this non-Git frontend directory. Browser scripts retain their historical evidence directory names and regenerate those local screenshots/axe files; the dated full log above identifies this execution.

Non-blocking upstream Zod/Rollup annotation warnings and Node experimental type-stripping notices were emitted. No fresh Lighthouse run, actual-device check, full visual review, penetration test or backend test is included in the 193 count.

## Quality checklist disposition

| Area | Disposition |
| --- | --- |
| Requirements/conventions | Existing 55-identifier coverage map retained; no scope or acceptance-policy change. Not a fresh line-by-line business acceptance exercise. |
| Data/API/state | Inspected production adapters, schemas, session/cache and query behavior; FE-01/02 remain open. |
| States/forms/resilience | Existing logic, preview and built-contract suites exercise validation, errors, recovery/conflicts and retained input. Coverage is limited to named automated scenarios. |
| Accessibility/responsiveness | Existing browser runner includes scoped axe, phone/desktop/reflow, keyboard and focus checks. No new visual redesign or exhaustive interactive-element review. |
| Performance | Fresh production builds; existing lazy features, bounded requests and hero delivery retained. No new Lighthouse/field/mobile performance claim. |
| Security | Client boundary inspected; real authZ, cookies/gateway CSRF, secrets/storage/email and concurrency require actual backend/environment evidence. |
| Maintainability | Existing layer/state ownership retained; no new dependency, worker, cache abstraction or speculative refactor. |
| Deferred/not exercised | G14 draft recovery; live backend/database/email/storage, actual zoom/screen readers/Safari/physical devices/scanners, deployed headers, recovery and field performance. |

## Handoff

Start with [BACKEND-HANDOFF-2026-11-04.md](BACKEND-HANDOFF-2026-11-04.md), then API-CONTRACT.md and REQUIREMENTS-MATRIX.md. Close FE-01/02 before live integration acceptance. Preserve the frontend-driven contract or explicitly coordinate workflow changes; changing the API prefix alone cannot reconcile the separate Go implementation described in the existing compatibility review.
