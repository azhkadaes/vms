# Frontend implementation and release plan

Updated 4 November 2026, WITA. Scope: D:/oikn/VMS/frontendx. Intended backend directory D:/oikn/VMS/backend exists but is empty. This plan describes current source and remaining acceptance work; historical plans are archived in anti-slop/history/2026-11-02. See the [current readiness recheck](FRONTEND-READINESS-2026-11-04.md) and [backend handoff](BACKEND-HANDOFF-2026-11-04.md).

## Completed frontend work

| Area | Current implementation |
| --- | --- |
| Module boundaries | Separate HTTP/errors, contracts, sessions, query providers/hooks, wire schemas, visit/report/management/attachment adapters. Booking detail is a lazy feature, separate from its list. Management uses a shared conflict/discard dialog. |
| Session and access | One explicit server role; cookie session; distinct outage versus expiry; scoped keys and cache removal after identity/role/permission/assignment changes. Staff cannot self-select an authorization role or send one to visit APIs; authorized account role-assignment controls are separate. |
| Visitor lifecycle | Three steps; complete validation; payload-bound public references; one transport retry; uncertain-reference resolution; localized public history; server admission QR with expiry/replacement; cancellation before start. |
| Staff and reception | Scoped creation/decisions, reservation conflicts on approval, WITA attendance window/Missed and late checkout. Bounded production pages, direct details and server dashboard counts replace full-history UI downloads. PA visit tables hide the VIP-only filter and ignore it in page URLs. |
| Administration | Employee URL search/pages and create/edit/deactivate/assign; server password minimum and administrator-set initial password. Required password-change screen; role permission editor; typed integration editor with write-only secrets. |
| Documents | Optional multipart upload before creation; opaque upload token joins the idempotent payload. Authorized binary download uses server availability metadata. No filename-only success in production. |
| Reports | Separate date/status summary and revision-checked summary CSV; distinct read/export rights; Terbatas excluded. |
| Accessibility and responsiveness | Existing identity and phone task/Back continuity retained. Correct headings/names, named table regions, phone employee cards, keyboard focus, dialog discard/conflict recovery and explicit list states. |
| Startup and delivery | Phone registration mounts on first intent and then persists. Hero preloads; mobile photo motion uses a small native frame loop instead of GSAP. Features and calendar/QR/scanner remain lazy. |
| Verification | Declared Playwright/axe/formatter, pinned patched Zod, reproducible check runner and a CI workflow for a repository rooted at frontendx. |

Current evidence and limitations are recorded in [VERIFICATION](VERIFICATION.md). Implemented frontend behavior does not prove real server enforcement.

## Decisions

G01–G15 retain the recorded decisions: adjust app for G01/G03/G04/G05/G06/G07/G08/G10/G12/G13/G15; adjust requirements for G02 email/code credentials, G09 one role and G11 current visitor navigation; defer G14 draft recovery.

G16 builds administration flows. G17 uses administrator-set initial passwords. G18 builds protected uploads/access. G19 adopts 20 people and 5 MB PDF/JPEG/PNG for the initial release. G20 implements staff cancellation within existing decision authority plus explicit cancel permission, with a required audit reason, including after start before check-in. [Decision history](../../frontendx/anti-slop/REQUIREMENT-GAPS-2026-10-30.md).

## Remaining gates

The 4 November full check passed: 193 tests, formatting, strict types, both builds and a dependency audit reporting zero vulnerabilities. The matrix retains 55 unique requirement IDs. Evidence and limits are in [the readiness recheck](FRONTEND-READINESS-2026-11-04.md). Two P2 frontend issues remain: complete/coherent ordinary vs restricted staff-response validation, and request/response ID/page correlation. Close these before live integration acceptance. Backend implementation may proceed against the documented complete shapes. Remaining integration/operational gates:

1. Implement and review the new backend against [API-CONTRACT](API-CONTRACT.md), including durable idempotency/outcome resolution, transactional scheduling/version checks, permission projections, server-time deadlines, session invalidation, admission revocation and audit.
2. Connect actual protected storage, content inspection, orphan cleanup, email queue/retries and real account/password/configuration management. Test genuine failures and concurrency with the deployed frontend.
3. Publish the organization's approved privacy notice, retention/deletion rules and rights contact. The production form obtains versioned notice text from the service and refuses submission without it. A 30-minute draft plan does not set submitted-data retention.
4. Verify production host routes, TLS/cookies, CSRF/Origin checks, cache/security headers, backups, monitoring and recovery. Perform Safari, screen-reader, actual browser zoom and physical scanner/camera acceptance as applicable.
5. Measure real devices/networks after deployment. Responsive hero delivery and shell imports now reduce the selected photo to about 76-133 KB and visitor entry JS to about 93 KB gzip. After direct font preload and scroll-triggered motion, clean local medians are 98 desktop / 85 mobile; the provisional mobile target remains unmet on the uncompressed test host. See [visitor performance results](VISITOR-PERFORMANCE-RESULTS.md) and [critical-request follow-up](VISITOR-CRITICAL-REQUESTS.md) for settings, checks and current formatting limits. Field Core Web Vitals remain unmeasured.

## Later work

G14 optional same-tab draft recovery remains in DRAFT-RECOVERY.md, disabled. Detailed visitor exports, separate restricted exports, PDF/Excel reports, SSO/MFA and additional integrations require separate scope decisions. Do not silently activate them or treat sample assets as requirements.
