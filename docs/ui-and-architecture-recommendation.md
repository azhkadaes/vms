# VMS UI and architecture recommendation

Prepared 20 October 2026, Asia/Makassar. Scope: source review and recommendations using the frontend-engineering skill, the two supplied diagrams, VMS-FE, and the current Go backend. This document proposes future work; it does not change application code, consolidate repositories, activate migrations, or certify a deployment.

**Recommended direction**

Keep one frontend repository with two independently built applications: visitor and staff. Turn the existing command wrapper into a proper dependency workspace. Share design tokens, selected UI primitives, and validated API contracts. Keep the existing Go backend in its own repository during the UI work, organized as one modular service.

This interprets “one FE and BE repo” as one repository for each. If the intended destination is one Git repository for the whole product, `frontend/` and `backend/` can live together later, with independent build and release jobs. That is useful when the same team owns both and frequently changes their contracts together. It does not determine URL routing or improve authorization by itself. Preserving backend history and existing uncommitted work requires a separate migration plan.

Use one local gateway at `http://localhost:3000`, with visitor at `/`, staff at `/admin/`, and the API at `/v1/`. Two applications can sit behind one address. For production, my preferred configuration for this identity-document workflow is a separate staff hostname with staff API access restricted there. The requested `/admin/` configuration remains a viable alternative when its shared browser origin is an accepted tradeoff.

**What the current source establishes**

| Finding | Evidence | Implication |
| --- | --- | --- |
| Both frontend applications already share a Git repository | [Workspace README](D:/oikn/VMS-FE/monorepo/README.md:3) | This is an architectural refinement and UI rebuild, rather than an initial merger of visitor and staff. |
| The root coordinates separate installs and commands, without a `workspaces` declaration | [Root package](D:/oikn/VMS-FE/monorepo/package.json:1) | Introduce one package manager, workspace configuration, lockfile, and consistent checks deliberately. |
| Visitor uses React 18 with Vite; admin uses React 19 with TanStack Start | [Visitor package](D:/oikn/VMS-FE/monorepo/apps/visitor/package.json:54), [Admin package](D:/oikn/VMS-FE/monorepo/apps/admin/package.json:47) | Align supported React and component dependency versions before sharing React components. Avoid forcing incompatible packages together. |
| The current launcher uses visitor port 3001 and admin port 3000; the visitor server proxies `/admin` | [Launcher](D:/oikn/VMS-FE/monorepo/scripts/dev.mjs:10), [Visitor configuration](D:/oikn/VMS-FE/monorepo/apps/visitor/vite.config.ts:45) | The desired port 3000 front door is a future routing change. Local development already demonstrates the path-based approach at port 3001. |
| Current production output is visitor `dist` plus an admin server output | [Build/deployment notes](D:/oikn/VMS-FE/monorepo/README.md:28) | The existing admin deployment requires its server runtime. Copying both outputs into one static folder would not implement the current architecture. |
| Visitor screens, translations, form behavior, and navigation occupy a 2,125-line `App.tsx` | [Visitor App](D:/oikn/VMS-FE/monorepo/apps/visitor/src/app/App.tsx:1) | Extract routes, form schemas, data hooks, and screen components as each flow is rebuilt. |
| Admin has reusable shell, table, dialog, loading, and empty-state components | [Admin shell](D:/oikn/VMS-FE/monorepo/apps/admin/src/components/shell/AppShell.tsx:1) | Reuse working interaction patterns and domain logic after reviewing them against the real API. |
| Admin feature repositories default to sample data | [Repositories provider](D:/oikn/VMS-FE/monorepo/apps/admin/src/app/RepositoriesContext.tsx:1) | Real staff authentication does not establish that booking and reporting screens use server records. |
| Staff sample models use `accepted` and “Personal Assistant”; backend uses `approved` and “PA” | [Preview models](D:/oikn/VMS-FE/monorepo/apps/admin/src/domain/types.ts:1), [Preview permissions](D:/oikn/VMS-FE/monorepo/apps/admin/src/domain/permissions.ts:101), [Backend scope](C:/Users/ASUS/go/src/github.com/thdoikn/VMS-BE/internal/usecase/staff_booking/staff_booking_usecase.go:45) | Validate and map contracts explicitly. Prefer backend-resolved permissions and available actions for connected screens. |
| Backend source includes staff sessions, permission/scope guards, visitor verification, booking decisions, QR, and reception routes | [Staff routes](C:/Users/ASUS/go/src/github.com/thdoikn/VMS-BE/internal/server/rest/router/staff_booking_router.go:1), [Visitor access routes](C:/Users/ASUS/go/src/github.com/thdoikn/VMS-BE/internal/server/rest/router/visitor_access_router.go:1) | Preserve this work and connect the UI in small slices. Route source is not evidence of current database activation or delivery reliability. |

Some older project documents describe earlier ports and missing features that later source implements. Treat the diagrams as requirements references and reconcile them with current handlers, contracts, and database evidence. Their wording is not an instruction to run operations. Both active repositories contain pre-existing changes, which this review leaves intact.

**Repository and application boundaries**

Suggested frontend structure:

```text
vms-frontend/
  apps/
    visitor/src/{app,features,shared}/
    admin/src/{app,features,shared}/
  packages/
    tokens/       # colors, spacing, typography, semantic status tokens
    ui/           # reviewed shared inputs, buttons, dialogs, feedback
    contracts/    # validated wire models and API error definitions
  tooling/        # gateway, development commands, shared checks
  package.json
  package-lock.json

vms-backend/       # existing Go repository
  internal/       # booking, visitor access, staff auth, reception, etc.
  migration/
  docs/
```

Start with npm workspaces because the project already uses npm commands. Native workspaces link local packages and support commands across packages. Introduce a task-cache product only when build time justifies its additional configuration. [npm workspace documentation](https://docs.npmjs.com/cli/v11/using-npm/workspaces/).

Keep visitor booking workflows and staff operational workflows inside their respective apps. Share actual repeated primitives and wire contracts; keep feature state, navigation shells, authentication providers, and query caches separate. Shared TypeScript validation improves frontend consistency; Go still validates every request independently. Generate types from an accurate OpenAPI contract when practical, rather than assuming an existing specification reflects all current handlers.

For a fresh frontend, my default stack would be React, strict TypeScript, Vite, TanStack Router and Query, Tailwind with the existing Radix-based component approach, and React Hook Form with schema validation. These fit the current tools and the form/table workload. The two authenticated/task interfaces do not currently establish a need for a second application server solely for rendering. However, retain the current TanStack Start runtime during incremental migration; changing it needs a deliberate hosting and Lovable compatibility decision. A new framework migration is not a prerequisite for the first UI slice.

**URLs and deployment**

| Front door route | Destination |
| --- | --- |
| `/v1/*` | Go API, preserving the existing prefix and cookie paths |
| `/admin` | Redirect to `/admin/`, preserving relevant query parameters |
| `/admin/*` | Staff application, its assets, and its own route handling |
| Other application routes | Visitor application and its route handling |

The gateway owns port 3000 locally. Visitor and staff development servers use separate internal ports; the Go API can retain port 8080. Production uses HTTPS through an ingress or reverse proxy. Internal app/API ports are not the public entry points. API failures must remain API responses rather than falling through to visitor HTML.

Test direct loads and refreshes such as `/admin/bookings/123`, trailing slashes, missing assets, and route fallback order. Keep asset URLs scoped to the owning build. Vite documents the asset-base requirement for nested deployment and states that its preview server is not a production server. The current TanStack Start application also needs its framework-specific server routing. [Vite deployment documentation](https://vite.dev/guide/static-deploy.html).

Preferred production alternative: visitor at `visit.<organization-domain>` and staff at `staff.<organization-domain>`, optionally retaining the `/admin/` base on the staff host. Each host proxies only its required API paths to the same Go service. Deny staff API access through the visitor host and allow browser staff requests only from the staff origin. This preserves one backend while strengthening the browser boundary.

**Security implications**

`/` and `/admin/` on one hostname are the same browser origin. A path is not an isolation boundary; separate builds and cookie paths do not change that. This means a visitor-site script compromise could affect staff requests when staff use that same origin. A separate hostname supplies a different origin, provided API access, CORS, and cookie configuration preserve that separation. This is an architectural inference from the browser origin model. [MDN same-origin policy](https://developer.mozilla.org/en-US/docs/Web/Security/Defenses/Same-origin_policy).

Backend authorization must validate session, action permission, and access to the particular booking/document on every protected request. PA assignment, VIP handling, and reception document restrictions belong there. The current backend already has useful guards; the new UI must consume their results and handle 401/403 safely. Super Admin configuration access should not silently grant operational booking/document access. [OWASP authorization guidance](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html).

Keep separate opaque visitor and staff sessions in HttpOnly, Secure cookies with suitable SameSite behavior. Retain scoped cookie paths with the existing `/v1` routes. For separated hosts, use host-only cookies without a shared parent-domain attribute. HttpOnly prevents JavaScript from reading a token but does not prevent browser requests carrying it. Clear each app's sensitive query cache on logout or account changes. [MDN cookie attributes](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie).

Add a complete CSRF strategy for cookie-authenticated mutations, with CSRF tokens and strict origin validation appropriate to the deployment. The inspected handlers reject unlisted non-empty origins but allow missing origins; this is a hardening item requiring tests, not proof of an exploit. A staff-host deployment needs staff-specific trusted origins rather than the current combined visitor/staff list. SameSite alone does not address sibling-subdomain requests. [OWASP CSRF guidance](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html).

Other release work specific to this system: keep identity files private; enforce file type/size and document access on the server; agree retention and pending-booking expiry; protect integration credentials; remove demo adapters from production; configure restrictive security headers; and verify notification delivery, backups, restore, migration and monitoring behavior. Staff SSO/MFA is a sensible production addition, especially for privileged accounts. The current anonymous-request limiter is process-local; replicated deployments need a gateway/shared limiter and correct trusted-proxy configuration.

**UI direction: one identity, two working environments**

Use a calm Nusantara service identity: warm neutral page surfaces, white content areas, dark readable text, deep green primary actions, and restrained gold accents. Use one primary type family, a consistent spacing scale, 8–12px radii, subtle borders, and limited elevation. Reserve status colors for semantic meaning and pair them with labels/icons. Verify contrast before locking tokens; these are proposed design directions, not tested final colors.

The current visitor source uses scenic backgrounds and green overlays; admin uses warm orange primary tokens with DM Sans/Space Grotesk. Bring their typography, input behavior, status language, and feedback into one system. Use photography sparingly around the public welcome area, with opaque readable form surfaces. Keep motion short and purposeful, and respect reduced motion.

| Concern | Visitor | Staff/backoffice |
| --- | --- | --- |
| Main outcome | Request a visit and understand its next step | Process requests and arrivals accurately |
| Layout | Mobile-first guided form; wider desktop summary beside it | Stable sidebar, compact top bar, filters, tables, and detail views |
| Entry | “Buat kunjungan” and “Cek status” | First permitted operational or administrative task |
| Density | Spacious labels and plain explanations | More information per screen with clear hierarchy |
| Primary emphasis | Progress, confirmation, status and arrival instructions | Pending work, today's arrivals, exceptions and next actions |
| Language/time | Indonesian first, English preserved; explicit WITA | Consistent terminology and WITA across lists, details and exports |

**Visitor journey**

Proposed routes: `/`, `/booking`, `/booking/review`, `/booking/submitted`, `/status`, and authenticated `/visit/:bookingId`. Booking identifiers never grant access on their own. Keep verification codes, session tokens, and identity data out of URLs.

1. Welcome: two clear entry actions, concise requirements, opening/access guidance and help.
2. Booking: meeting details and representative/party details as short logical steps. Explain that party count includes the representative. Request the required identity upload with clear file rules and privacy context.
3. Review: editable summary, explicit WITA schedule, required confirmations and one protected submit action.
4. Submitted: clearly say the request is pending approval; show the booking code and how to return to status.
5. Verified status: booking code plus email verification leads to a clear status timeline, host/time/location, permitted cancellation or reschedule response, and admission QR only when eligible.

Room assignment remains a staff responsibility under the current product contract. Cancellation preserves history. The diagram's edit/delete actions need to be expressed through the approved lifecycle, including confirmed rescheduling, rather than assuming unrestricted changes after submission. An approved booking alone does not imply a QR has already been issued.

Use addressable routes for refresh and Back navigation. Store draft fields in form state, with an explicit decision about draft persistence; do not automatically persist identity images or identity numbers in browser storage. Preserve the existing idempotent submission behavior, including an uncertain-response state that prevents a retry from creating a second booking.

**Staff journey**

Organize navigation around work: Overview, Bookings, Reception, Reports, and permission-gated Settings. Existing check-in and check-out can become two modes within Reception while preserving direct links. People, roles, officer assignments, rooms, integrations and audit views belong to the relevant administrative area; exposing a future management screen depends on a real backend capability.

| Staff context | Recommended initial view |
| --- | --- |
| PA | Pending requests for assigned non-VIP officers, with date/host filters |
| Protocol | VIP requests and arrivals, plus other permitted read access |
| Reception | Today's arrivals, scan/manual-code resolution, current visitors and check-out |
| Super Admin | Employees, access, integrations and audit/report tasks allowed by grants |

Select the actual landing route from resolved permissions and scope, especially for people with multiple roles. Do not use a client role picker in production.

Make the overview prioritize actionable work before annual charts: awaiting decisions, expected arrivals, checked-in parties, and exceptions. Clearly label whether a count means bookings, expected people, or checked-in parties; these are distinct metrics. Scope all metrics on the server.

Bookings should have URL-based filters and pagination, accessible detail links, and a stable detail page or deep-linkable panel. The detail view brings together schedule, representative, party count, room, status history and permitted actions. Approval includes room selection; a version or room conflict preserves input and offers a safe refresh. Reception keeps scanning and manual entry equally discoverable and always revalidates eligibility on submission. A scan result resolves a record; it does not itself authorize admission.

**Engineering plan before implementation**

Goal and users: complete the visitor-to-reception flow and permission-scoped staff work described above. Scope: UI foundations, routing, typed adapters, lifecycle integration and state handling. Backend replacement, database activation and repository consolidation require their own work items.

Data contract: build from current `/v1/visitor/*` and `/v1/staff/*` handlers. Reconcile list pagination, session projections, status values, available actions, version checks, QR semantics and field-level errors. Start UI slices with a replaceable development adapter implementing the reconciled contract, then connect the HTTP adapter. Production must fail closed if a real adapter is unavailable.

| State | Owner |
| --- | --- |
| Filters, sort, page, selected booking | URL |
| Bookings, staff session, allowed actions, rooms | Query/data layer, with separate app caches |
| Booking/decision input and field errors | Form library/schema |
| Open dialog, upload progress, scanner interaction | Nearest local component/hook |
| Language and non-sensitive display preferences | Small app context/preference layer |
| Display totals derived from already fetched small data | Derived values; larger scoped totals come from API |

Component direction: tokens/UI primitives → reusable form and feedback components → feature components/hooks → app routes. Visitor routes compose a VisitorShell, BookingWizard, ReviewSummary and VisitStatus. Admin routes compose a StaffShell, BookingQueue, BookingDetail and ReceptionConsole. Share only components that really repeat.

Every async slice needs loading, empty, no-results, network/server failure, partial/stale, success, expired-session and forbidden states. Preserve user input through errors. Cover expired verification/QR, denied camera access, duplicate submission, revoked access, room/officer conflict and a booking changed by another staff member. Use inline errors tied to fields, an error summary with focus on submit, server validation as authority, keyboard navigation, visible focus, live status announcements and minimum 44px interaction targets. Validate at phone/tablet/desktop sizes and 200% zoom.

Performance decisions from the skill's three questions:

- Avoid unnecessary work: reuse query keys, avoid fetching entire booking history for an overview, and avoid a second global store for server data.
- Defer work: visitor loads no admin bundle; load charts, camera tools and exports only when needed. Provide matching skeletons and chunk-failure recovery.
- Put work in the right place: permissions, filtering, pagination, room conflicts and aggregate counts run in Go/database; immediate form interactions stay local. No worker or virtualization is justified until measured workload needs it.

Library decision: reuse the current ecosystem and choose one maintained compatible version of each needed package during implementation. No new state framework, microfrontend runtime, task orchestrator or chart library is justified by this review alone.

Pattern fit: feature modules match the repeated booking/reception/access concerns; shared primitives support both apps; contract adapters prevent known preview/backend differences from leaking into components. Enforce dependency direction, authorization and contract checks. Keep decorative choices easy to revise.

**Build order and acceptance**

1. Contract and design foundation: reconcile lifecycle/session shapes, define tokens, navigation and responsive wireframes. Acceptance: visitor and staff layouts express one identity and each role can find its main task.
2. First complete slice: create booking → verified visitor status → staff-scoped pending queue. Acceptance: the same synthetic booking is visible across both apps through the real API, with loading/error/forbidden behavior exercised.
3. Decisions and status: room approval/rejection → visitor outcome, then eligible QR and reschedule/cancellation actions. Acceptance: server rules, conflicts and repeat requests stay correct.
4. Reception: scan/manual resolution → check-in → check-out. Acceptance: expired/replaced credentials and competing attempts cannot create incorrect attendance.
5. Administrative features and production routing: connect each permitted management/report feature, verify nested routes and assets, apply security configuration, and review operational readiness.

Keep old routes available until replacement slices are accepted. Use synthetic fixtures for demonstrations. Validate pure contract/validation logic, backend permission/transition/concurrency behavior, and the critical browser journey; a successful build alone is insufficient. A release candidate needs real adapter coverage, correct cookie/CSRF behavior through the deployed gateway, reviewed migrations, email delivery, retention decisions, and an end-to-end rehearsal in its target environment.

**Review limits and remaining decisions**

This was a source-based review. No live UI, production configuration, database schema, email delivery, runtime dependency health, build, or test suite was verified in this turn. Existing documentation's earlier passes are historical evidence only. No new dependency is installed and no application source is changed.

Before implementation, resolve only the decisions that materially change the design: whether production must use the same hostname, whether Lovable/TanStack Start compatibility must continue, whether FE and BE should eventually share Git ownership, and the outstanding records-retention rules. Default recommendation: two frontend apps in one FE workspace; existing Go backend separate; one local address; separate staff origin for production; calm shared design foundations with visitor guidance and staff task queues.
