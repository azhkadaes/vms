# Frontend production-readiness work

Scope: `D:/oikn/VMS/frontendx` only. User authorized reconciling plans/requirements and making the frontend production-ready. Existing After-mode findings A2-01/A2-05/A2-06/A2-10/A2-12/A2-13/A2-14/A2-15/A2-16 provide the reviewed backlog. Do not build or adapt a different backend. G02/G09/G11 remain the selected requirement revisions; G14 draft recovery remains deferred.

## Plan before edits

1. Reconcile a requirement-to-implementation matrix from the current BRD and confirmed decisions. Distinguish implemented frontend behavior, preview-only behavior, future backend obligations and deliberately deferred features. Remove contradictory current-plan wording; retain dated audit evidence as historical.
2. Upgrade the vulnerable Zod dependency using the current patched release and retain validation/resolver behavior. Declare browser/accessibility tooling and provide reproducible check scripts/CI rather than relying on the agent's external runtime. No UI/runtime dependency is added for testing.
3. Split data contracts, HTTP/error handling, repositories, session and hooks. Validate one canonical authenticated role, derive actions from that session and consistently handle transient service failures versus real authentication loss. Clear sensitive caches on session/identity/role changes. Preserve same-origin cookie authentication and document CSRF/server authorization requirements.
4. Replace production workflow stubs with a typed frontend API adapter and a documented proposed contract for the new backend. No legacy backend is assumed. Test the actual production build against a disposable local contract fixture. This verifies frontend requests/responses, not a real service. Keep unimplemented account provisioning, integrations, real files/email and admission credentials explicit rather than fabricating success.
5. Fix shared accessible names/landmarks and audit date/time controls, keyboard/reflow states and other list/table feedback. Retain current institutional identity/navigation, task/input continuity and reduced-motion behavior. Only approved visual assets are reused.
6. Avoid loading/mounting registration on the phone choice screen until first registration intent, then retain that instance. Improve discovery/delivery of the existing hero without inventing imagery. Keep code loading at feature boundaries; no worker, blanket memoization or virtualization for the tiny preview list.
7. Run meaningful logic/API, browser/accessibility and production-build contract checks; inspect rendered evidence. Record remaining integration/device/policy limitations precisely. A missing real backend prevents live operational acceptance regardless of frontend test results.

State ownership: URL owns filters/steps/task selection; forms own drafts; TanStack Query owns fetched data/session; local UI owns dialogs/preview directory; adapters own wire validation and normalized errors. Public/staff/restricted types remain separate. New backend routes/response schemas are a proposed frontend contract, not proof that the service exists.

Performance questions: avoid hidden registration work; defer schema/date/QR/scanner code to its feature; keep tiny derived filtering in its subtree; move real production filtering/aggregation/pagination to the backend contract. Library gate: Zod replaces an existing dependency; Playwright/axe are development-only verification tools. No new visual framework, motion runtime, worker or draft persistence.

## Status

G16 decision: build employee, role and integration management flows and their new backend contracts. G17 decision: administrators set initial passwords. Password limits come from backend metadata; stored passwords/secrets are never returned. Account setup supports a backend-required password change. Management pages use shared query/error/dialog primitives, URL search/page state, versioned edits, explicit deactivation and conflict feedback. The preview remains fictional; no real account, configuration, password or email is changed by this work.

Planning work closed 2 November 2026. G16–G20 management/password/document/capacity/staff-cancellation choices are implemented or reconciled. Current findings and acceptance gates are in [PRODUCTION-READINESS-2026-11-02.md](PRODUCTION-READINESS-2026-11-02.md); final passing checks are in [../VERIFICATION.md](../VERIFICATION.md). This file retains the plan-before-edits record. No production deployment was performed.
