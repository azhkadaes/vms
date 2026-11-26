# Visitor Management System — frontendx

The public visitor portal and staff workspace share UI components and typed data adapters. The visitor app handles registration and admission. The staff app handles booking decisions, reception, employee administration and reporting.

The [Go backend](../backend/README.md) implements the new 40-operation contract. The [UI contract review](../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md#6-frontend-work-not-backend) lists the remaining adapter changes before the apps can use it end to end. Domain terms are in the [glossary](../CONTEXT.md) and architecture decisions in [docs/adr](../docs/adr).

## Run locally

For actual backend requests on Windows, use [Start VMS.cmd](../Start%20VMS.cmd). Docker serves both apps at `http://localhost:58101` and proxies requests to the Go API. The development command below continues to use demo data on loopback.

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

Build outputs: apps/visitor/dist at / and apps/admin/dist at /admin/. Both use the same-origin API prefix /v1. Copy .env.example to .env.local only when changing that public prefix; both Vite apps read the workspace environment directory. VITE_* values are public and must contain no secrets.

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
