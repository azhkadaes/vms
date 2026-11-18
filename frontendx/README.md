# Visitor Management System — frontendx

Current scope: **D:/oikn/VMS/frontendx only**, rechecked 4 November 2026, WITA. Visitor and staff applications share UI primitives, business rules and typed data adapters. **D:/oikn/VMS/backend is an empty directory** on this review date. This frontend's API contract remains the baseline; another repository is not automatically its implementation.

The production frontend has real API request paths and management/upload screens. A disposable contract fixture tests those paths; it is not a real backend. See [current readiness](../docs/frontend/PLAN.md), [requirements](../docs/frontend/BUSINESS-REQUIREMENTS.md), [coverage map](../docs/frontend/REQUIREMENTS-MATRIX.md), [API handoff](../docs/frontend/API-CONTRACT.md) and [verification](../docs/frontend/VERIFICATION.md).

Start backend work with the [4 November handoff](../docs/frontend/BACKEND-HANDOFF-2026-11-04.md) and [fresh readiness review](../docs/frontend/FRONTEND-READINESS-2026-11-04.md). Two frontend response-validation follow-ups remain before live integration acceptance. The [separate Go backend compatibility review](../docs/frontend/BACKEND-COMPATIBILITY-2026-11-04.md) records workflow and wire differences; a prefix change alone does not connect it.

## Run locally

For actual backend requests, use the [real-backend local launcher](../docs/frontend/REAL-BACKEND-LOCAL.md) after reviewing the compatible backend/database version. It serves both built apps at one local address and requires an explicit API target. The development command below continues to use demo data on loopback.

Use Node 22 and npm. From this directory:

```powershell
npm.cmd ci
npm.cmd run dev
```

Visitor: http://127.0.0.1:3000/. Staff: http://127.0.0.1:3002/admin/. The visitor server also proxies /admin. Development on loopback uses explicit fictional preview records, role fixtures and demo QR codes. No real account, password, email, file storage or admission credential is created by preview actions. Preview administration changes live in browser memory and reset on reload.

## Check and build

```powershell
npm.cmd run check
```

This runs formatting, strict types, logic tests, isolated preview API tests, both builds, sequential browser/accessibility checks, production-bundle contract checks and dependency audit. Install Chrome locally, or install Playwright Chromium and set PLAYWRIGHT_CHANNEL=chromium. Browser tooling is declared in devDependencies. On Windows, npm.cmd avoids PowerShell execution-policy shims.

The preview browser runner starts and stops its own two servers when the ports are free. To test a server you already started, set VMS_TEST_URL=http://127.0.0.1:3000. It never kills a process occupying a port. Production browser tests serve the latest dist files on a temporary loopback port; run npm.cmd run build first when invoking test:production alone.

Build outputs: apps/visitor/dist at / and apps/admin/dist at /admin/. Both use the same-origin API prefix /v1. Copy .env.example to .env.local only when changing that public prefix; both Vite apps read the workspace environment directory. VITE_* values are public and must contain no secrets. See [deployment gates](../docs/frontend/DEPLOYMENT.md).

## Current behavior

- Public registration, review and email/code status lookup; safe public submission retry/recovery; server-issued admission QR and replacement; visitor cancellation before start.
- Staff decisions, fixed rescheduling, scoped appointments and restricted visits; server-time attendance; 30-second staff / 15-second reception refresh with manual, return-to-tab and reconnect refresh.
- Separate aggregate reports and summary-only CSV with independent permissions.
- Employee search, bounded pages, create/edit/deactivate and PA assignments; administrator-set initial passwords; backend-required initial password change.
- Versioned role permissions and server-defined integration settings. Stored passwords and secrets are never populated in forms.
- Protected supporting-document upload and authorized download. Initial limits are 20 visitors and 5 MB PDF/JPEG/PNG, approved in G19.

G14 optional draft recovery remains deferred. G20 staff cancellation requires explicit permission, decision scope and a visitor-facing cancellation reason, with a separate optional internal note. Cancelled details show both to authorized staff and only the reason to visitors; already checked-in visits cannot be cancelled. Production authentication, authorization, transactions, storage, email, policy publication and operational acceptance require the new backend.

## Source ownership

apps/visitor owns visitor task/form state; apps/admin owns staff pages and management editors. packages/ui owns shared controls/tokens and accessibility behavior. packages/data separates contracts, HTTP, sessions, repositories, response validation and queries. tooling/preview.ts is development-only synthetic enforcement. URL state owns filters, sorting, pages and selection; fetched data belongs to TanStack Query; drafts stay in forms.

Historical audit notes remain under anti-slop. Superseded working documents were preserved under anti-slop/history/2026-11-02 before consolidation. Use the linked current documents for acceptance criteria.
