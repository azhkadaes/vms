# Frontend audit and readiness

Scope: **D:/oikn/VMS/frontendx only**. Consolidated 2 November 2026, WITA. The proposed new **D:/oikn/VMS/backend is absent**. No other frontend/backend repository supplies readiness evidence. No deployment or real service/account/email/file mutation was performed.

## Verdict

The selected frontend work and all twenty app-versus-requirement decisions are implemented, reconciled or explicitly deferred. All 55 current business-requirement identifiers have a source/coverage mapping. G14 draft recovery remains deliberately deferred. The frontend is a candidate for integration with the new backend; it is **not an operationally accepted production system** without that service and deployment/device/policy acceptance.

Current references: [plan](../PLAN.md), [business requirements](../BUSINESS-REQUIREMENTS.md), [coverage matrix](../REQUIREMENTS-MATRIX.md), [service contract](../API-CONTRACT.md), [verification](../VERIFICATION.md), [deployment gates](../DEPLOYMENT.md). Superseded working documents were preserved under history/2026-11-02; dated audits remain historical evidence.

## Findings addressed

| Prior finding / gap | Result |
| --- | --- |
| A2-01/A2-05/A2-06 data/session boundary | Split contracts, HTTP/errors, session parsing/provider, queries/hooks, schemas and adapters. Canonical one-role session, explicit failure states, authenticated expiry and scoped-cache removal. No client-selected visit authorization role. |
| G16/G17 administration placeholders | Employee bounded search/pages/create/edit/deactivate/assign; administrator-set masked initial password and required-change gate; versioned role permissions and server-defined integration settings with write-only secrets. Shared discard/reload/busy behavior. Existing unavailable assignments are not silently cleared. |
| G18/G19 filename-only documents/unapproved capacity | Actual multipart frontend upload, opaque payload-bound upload token and authorized binary download, with initial 20-person/5 MB PDF/JPEG/PNG limits approved. Real protected storage/content inspection remains a backend obligation. |
| G20 missing staff cancellation | Explicit cancel permission plus existing decision scope, mandatory private reason, version protection and deliberate conflict review. Pending/Approved may be cancelled after start/end only before check-in. Public history receives the status change, not the staff reason. Synthetic reservation release and credential invalidation by status are checked. |
| A2-10 accessibility | Visible wording included in language/Brand/detail accessible names; logical headings/focus; named scrolling regions; preview landmark; corrected detail definition-list structure. Phone cards, long text, list feedback, dialog and calendar/time states checked. |
| A2-12 verification | Declared browser/axe/format tooling, repaired date-picker fixtures/expectations, self-starting sequential browser runner, built-production contract server/tests and CI workflow. Full check outcomes are in VERIFICATION.md. |
| A2-13 dependency advisory | Zod upgraded to 4.6.7 with resolver/validation behavior retained; fresh dependency audit is part of the check sequence. No live readiness is inferred from a dependency scan. |
| A2-14 startup work | Phone registration mounts on first intent and persists through Back. Native photo motion replaces GSAP: motion chunk 45.76 to 0.71 KB gzip. Hero preloading is limited to the landing route, so booking/status pages do not fetch the unused photo. |
| A2-15 scale/modularity | Production visit lists/reception use bounded pages; direct details are independent of list pages; dashboard totals come from server metrics. BookingDetail is a lazy feature. Shared freshness feedback covers operational and management views; URL, query and form owners stay separate. |
| A2-16 comment hygiene | Removed 23 decorative ReUI comment lines, retaining MIT/source and behavior constraints. Parsed code equivalence was verified for this comment-only cleanup. |

The earlier G01–G15 decisions remain: periodic/manual/focus/reconnect freshness; restricted projections; reserve on approval; server-time arrival/Missed; validation/public safe retry; independent summary reporting; one role; responsive directory/navigation; localized public history/headings; visitor cutoff. G02/G09/G11 revise requirements as chosen. No extra email-proof/OTP or workspace-switch flow has been reintroduced.

## Remaining acceptance gates

- **Service integration:** implement the proposed new backend, durable reference/outcome recovery, transactional version/resource checks, audit/notification intent, server role/permission/assignment enforcement and privacy-safe projections. Verify genuine concurrency and restart recovery.
- **Genuine credentials and services:** real session/password handling, email queue/retries, private upload inspection/storage/orphan cleanup, protected file access and admission issuance/replacement/revocation/single-use are unbuilt. A disposable fixture cannot prove them.
- **Organization/host policy:** publish the approved versioned ID/EN privacy notice and retention/deletion/rights-contact policy; verify actual HTTPS/cookies/CSRF, private caching/security headers, backups and monitoring. The production form blocks submission without a published notice.
- **Manual/device acceptance:** actual browser zoom, screen readers, Safari and physical scanner/camera behavior remain unverified. Axe covers tested states and is not a full WCAG certificate.
- **Field performance:** local bundle reductions are measured; the original photo remains approximately 517 KB. Real-device/network Core Web Vitals are unmeasured. No new asset or visual framework was introduced.

These gates do not mean that the frontend's API screens remain placeholders. They distinguish implemented client behavior from unbuilt services and operational evidence. Final exact commands/counts and screenshot limits are recorded in VERIFICATION.md.
