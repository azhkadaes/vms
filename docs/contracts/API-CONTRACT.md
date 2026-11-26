# VMS API Contract Documentation — Phase 1

> **Visitor Management System**
> API Version: v1 | Phase 1 revision: 13 November 2026, WITA

> **Superseded, 18 November 2026.** The current contract is the 40-operation [operation reference](FRONTEND-DRIVEN-REST-API-2026-11-16.md) with its [field rules](FRONTEND-DRIVEN-DETAILS-2026-11-16.md) and [UI validation](UI-CONTRACT-VALIDATION-2026-11-18.md). Its machine-readable pipeline is [manifest.json](manifest.json), [source-baseline.json](source-baseline.json) and `check.py`; see [VALIDATION](VALIDATION.md). This document and its generated companions (ENDPOINTS, SCHEMAS, INTERFACES, GAPS, FRONTEND-TRACEABILITY, CONTRACT-ISSUES) are kept as the Phase 1 record, rendered by `render.py` from [the archived Phase 1 manifest](phase1-manifest-2026-11-13.json).

This was the **Phase 1 target**, following SIHP's contract structure. It adopts the user-approved simpler authentication flow: email/password login, a fixed-expiry Bearer access token, a current-user read, one manually assigned database role per account, and fixed backend permissions. Stored login sessions and their lifecycle are deferred.

The [canonical OpenAPI](../BACKEND-OPENAPI-2026-11-12.json), [Phase 1 manifest](phase1-manifest-2026-11-13.json) and [checked examples](examples.json) define this target together. The historical filename is retained; the contract version is `1.1.0-phase1-review-2026-11-13`. The [source baseline](source-baseline.json) separately records the unchanged existing binary, which still has seven superseded identity routes. It is source evidence, not a second target contract or an auth implementation.

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Authentication](#authentication)
4. [Base URLs](#base-urls)
5. [Response Format](#response-format)
6. [API Endpoints](#api-endpoints)
7. [Data Models & Enums](#data-models--enums)
8. [Error Codes](#error-codes)
9. [Implementation Status](#implementation-status)
10. [Deferred and Replaced Operations](#deferred-and-replaced-operations)
11. [Frontend Page-to-Endpoint Mapping](#frontend-page-to-endpoint-mapping)
12. [References actually inspected](#references-actually-inspected)
13. [Common wire and authority rules](#common-wire-and-authority-rules)
14. [Consolidation decisions](#consolidation-decisions)
15. [Enforcement workflow](#enforcement-workflow)

## Overview

Phase 1 covers visitor registration/status/tickets, staff booking and scheduling, reception attendance, a read-only employee directory, integration settings and scoped reporting. Each endpoint below names the actual frontend dropdown, form, card, table or action and the request/response fields it uses. Existing frontend code is evidence of its needs; it still requires coordinated changes for the new auth flow and deferred editors.

| Property | Phase 1 contract |
| --- | --- |
| API prefix | `/v1` on the configured audience origin |
| JSON | Direct camelCase DTOs, arrays or pages; no success/code/data envelope |
| Staff authentication | Signed Bearer access JWT plus current database account/single-role/PA-assignment checks |
| Login response | AuthResult: accessToken, tokenType, accessExpiresAt and user |
| Current-user response | CurrentUser: active, id, name, role, fixed permissions and assignedOfficerIds |
| Role administration | Controlled database setup; exactly one role per account, including Super Admin |
| Employee UI | Read-only directory/detail; account/role/assignment editors deferred |
| Logout | Local token/user/private-cache clearing; no HTTP endpoint |
| Token lifetime | Required positive finite server configuration; fixed per issue; accessExpiresAt is authoritative; no new numeric duration assumed |
| Token retention | Browser memory only; reload or expiry requires login |
| Deferred auth work | Stored sessions, refresh/rotation/recovery, idle/absolute renewal, cross-tab coordination, auth cookies/CSRF bootstrap, role selection/switching, forced-change/self-service password workflows |
| Scope | 29 target operations: one matching catalog route and 28 missing target routes; seven superseded identity routes still exist in source |
| Business data | Existing schedule/state/version/idempotency, ordinary/restricted privacy and public-pair rules retained |
| Implementation status | Contract/docs/checks revision only; no application auth, frontend behavior, database or migration changes |

### Phase 1 Precedence

This revision supersedes the earlier lifecycle, editable-role-policy and password-workflow decisions **for Phase 1**. Earlier D01-D100 booking, public proof, restricted projection, schedule, mutation-version, recovery and notification decisions remain unless this document explicitly changes their auth integration. Older Session/TokenResult/cookie diagrams and endpoint lists in the [historical contract](../BACKEND-API-CONTRACT-2026-11-11.md) and [decision register](../BACKEND-PLAN-2026-11-11.md) are deferred evidence, not requirements to reintroduce now.

| Topic | Phase 1 decision |
| --- | --- |
| Auth HTTP surface | `POST /v1/staff/auth/login` and `GET /v1/staff/auth/me` |
| Principal | Verified token subject identifies the employee; current database role/assignments decide access |
| Roles | One of PA, Protocol Officer, Receptionist, Super Admin; no union, client role or role picker |
| Permissions | Fixed backend role mapping; no editable database policy evaluation or permission-management API |
| Account setup | Create employee/credential and assign one role through controlled database setup; initial Super Admin follows the same path |
| Existing multi-role rows | Operator selects the intended single role before Phase 1 use; API never picks the first role silently |
| Session/logout state | No staff_sessions or refresh-lineage/recovery dependency; logout discards the browser's copy only |
| Earlier database structures | Retain existing migrations/history; deferral does not authorize dropping deployed session/policy tables |
| Employee administration | Read-only page/detail retained; create/edit/deactivate/role/assignment/password controls deferred |

## Quick Start

These are target protocol examples with synthetic values. The new login/me routes are **not implemented yet**. Use the deployment-configured API origin; no running host/port is assumed.

Public catalogs for registration dropdown/notice:

```http
GET /v1/public/catalogs
Accept: application/json
```

Staff login needs only a JSON email/password body; no initialization request or auth cookie is involved:

```http
POST /v1/staff/auth/login
Content-Type: application/json
Accept: application/json

{"email":"staff@example.test","password":"synthetic-passphrase"}
```

Read current user using the issued token:

```http
GET /v1/staff/auth/me
Authorization: Bearer <accessToken>
Accept: application/json
```

Login returns `AuthResult.user`; current-user reads return `CurrentUser` directly. Store token/expiry/user in memory. Logout clears them and private caches locally, then navigates to login. There is no refresh, renewal or logout API to call.

## Authentication

### Login and Token Validation

The backend normalizes email, verifies the existing encoded bcrypt password and checks the employee is active. It requires exactly one supported database role, derives the fixed permission list and current active PA assignments, then issues a signed access JWT. Passwords remain hashed; existing-password verification does not apply a new-password minimum.

JWT claims are `sub` (canonical employee ID), `iat`, `exp`, `iss` and `aud`. Verify the signature against a server-configured allowed algorithm/key, issuer, audience and issue/expiry times; never trust token-supplied algorithms/keys, browser role hints or role/permission claims as authority. The signing secret and finite token lifetime are required server configuration. The former five-minute access token combined with automatic refresh is deferred; no replacement numeric lifetime is selected in this revision.

Every protected request reads the current account active flag, exactly one database-assigned role and relevant active PA assignments. A missing/inactive account cannot authenticate; a missing/multiple/unsupported role assignment is denied rather than guessed. Database role/assignment changes take effect on the next protected request. Writes repeat relevant current authority/resource checks in their repository transaction. A database failure is a dependency error, not an anonymous/default-role success.

| Request class | Authentication / proof |
| --- | --- |
| Public catalogs and standard create | Anonymous; selected validation, consent, idempotency and rate rules still apply |
| Public lookup/cancel/admission | Matching standard-visit code + email in POST body; mutation version/path ID as specified |
| Submission recovery | Retained UUID reference + matching email, serialized with creation |
| Staff login | Email/password in JSON, basic login throttling; no stored session/auth cookie/CSRF initialization |
| Staff resources/current user | `Authorization: Bearer <accessToken>` plus current account, single role and assignment/resource/field checks |
| Sign out | Local action only; no request |

### Fixed Role Permissions

<!-- BEGIN GENERATED AUTHORITY -->

| Database role | Fixed permission keys |
| --- | --- |
| `PA` | `bookings`, `create`, `approve`, `reject`, `reschedule`, `cancel`, `reports`, `reportExport` |
| `Protocol Officer` | `bookings`, `create`, `approve`, `reject`, `reschedule`, `cancel`, `reports`, `reportExport` |
| `Receptionist` | `bookings`, `checkin`, `checkout` |
| `Super Admin` | `bookings`, `people`, `integrations`, `reports`, `reportExport` |

<!-- END GENERATED AUTHORITY -->

The fixed mapping uses the existing frontend role/permission vocabulary and removes deferred role/audit administration from Phase 1. `permissions` is a read-only server projection for navigation, not an editable policy document. It does not bypass the resource/field boundaries below.

| Database role | Resource boundary |
| --- | --- |
| `PA` | Assigned non-VIP ordinary visits and the selected assigned-pa restricted projection; ordinary decisions require current assignment |
| `Protocol Officer` | VIP ordinary decisions and restricted visit management under existing rules |
| `Receptionist` | Attendance plus selected operational projections; no private notes/full restricted detail or Phase 1 reports |
| `Super Admin` | Read-only employee directory, integrations, ordinary-only organization reports/export and full restricted reads; no automatic ordinary visit detail/document/decision rights |

### Manual Account, Role and Super Admin Setup

Use existing `vms.pegawai`, `support.employee_credentials`, `vms.role`, `vms.role_pegawai` and `vms.pegawai_pejabat` structures. This is an operator database/setup action; it creates no public signup or role-assignment HTTP endpoint.

1. Create the employee with a unique normalized email and active flag.
2. Generate the password hash using the existing bcrypt credential encoding; store the encoded hash in employee_credentials, never a plaintext password. Retain the selected 15-code-point/72-UTF-8-byte bound for newly provisioned passwords.
3. Assign exactly one existing role through role_pegawai. Initial Super Admin uses the exact role label `Super Admin` and logs in through the same login endpoint.
4. For PA, maintain authorized official assignments through pegawai_pejabat; the API validates current active non-VIP scope.
5. Keep the deferred `must_change_password` workflow out of Phase 1; provision a usable operator-managed password. The existing column/policy/session tables can remain without being used for Phase 1 authorization.

The current schema permits multiple role rows, so the Phase 1 authority reader must enforce exactly one; no migration or data cleanup was executed here. Seed credentials/setup values are deployment data and are not invented or committed by this contract revision.

### Expiry and Logout Behavior

`accessExpiresAt` equals the token's signed expiry and is the client deadline. There is no sliding idle/absolute timer or automatic refresh. Losing the in-memory token on reload requires login. Polling does not extend expiry.

Logout clears the in-memory token, current user and private caches and prevents stale in-flight responses from restoring cleared state. It does **not** revoke another copy of the token. A copy can authenticate until expiry while its employee remains active; manual password changes likewise do not revoke existing tokens in Phase 1. Account deactivation denies subsequent requests. These are explicit limitations of the selected simpler flow.

### Baseline Controls Retained

Keep credential hashing, token validation, input validation, basic login throttling, backend role/resource checks, public code/email proof and ordinary/restricted field projections. Retain HTTPS for non-loopback deployments and explicit browser-origin configuration. Phase 1 resource authentication does not accept cookies, so the previous refresh/anonymous/CSRF cookies and staff-HTML initialization are unnecessary. If cookie authentication is introduced later, its protections must be designed with that change.

## Base URLs

| Audience / environment | Base URL |
| --- | --- |
| Canonical relative prefix | `/v1` |
| Visitor frontend | Configured visitor API origin with public audience routes |
| Staff frontend | Configured staff API origin with Bearer-protected staff routes; no auth initialization subrequest |
| Local development | Actual configured host/port; existing [setup guide](../MANUAL-SETUP-GUIDE.md) describes current-source setup and still needs adaptation with the Phase 1 implementation |
| Staging / production | Deployment-provided origins; no hostname selected by this contract |

Health/private Swagger remain operational surfaces outside the business inventory. `/internal/staff/initialize` is a superseded source surface, not a Phase 1 browser/gateway requirement; stop exposing it when replacing the auth slice.

## Response Format

### Direct Authentication Result

```json
{
  "accessToken": "synthetic-access-token-not-a-credential",
  "tokenType": "Bearer",
  "accessExpiresAt": "2030-01-02T01:00:00Z",
  "user": {
    "active": true,
    "id": "1",
    "name": "Staff Example",
    "role": "PA",
    "permissions": ["bookings", "create", "approve", "reject", "reschedule", "cancel", "reports", "reportExport"],
    "assignedOfficerIds": ["1"]
  }
}
```

Current-user success is the `user` object directly. No Session, refresh secret, availableRoles, roleRevision, mustChangePassword or idle/absolute timing is part of either Phase 1 DTO. Login does not set authentication cookies. All JSON remains direct camelCase without a success/code/data envelope.

### Paginated Response

```json
{
  "items": [],
  "total": 0,
  "totalAll": 0,
  "page": 1,
  "pageSize": 25
}
```

This is an empty read-only employee page. Booking pages use `totalScoped`; dashboard reads additionally require `metrics`. Password-policy metadata is absent from the Phase 1 directory because account/password editors are deferred. Optional fields are omitted rather than null.

### Error Response

```json
{
  "code": "validation_failed",
  "message": "Periksa isian yang ditandai.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "fieldErrors": {"email": ["Periksa alamat email."]}
}
```

HTTP status is authoritative. `fieldErrors` maps form keys to string arrays. Private/authenticated/pair-authorized responses and failures remain no-store with request correlation and nosniff; `429` includes integer-seconds Retry-After. Each endpoint lists its applicable errors.

### Business and File Results

New committed booking/upload returns `201`; accepted same-reference replay returns `200` without another logical mutation/notification. Booking commands return the complete authorized booking. Verified private files return PDF/JPEG/PNG bytes with safe Content-Disposition; `disposition=inline` views and `attachment` downloads. Report export remains JSON `filename/csv/revision`, converted to a CSV download locally.

## API Endpoints

<!-- BEGIN GENERATED ENDPOINTS -->

All 29 Phase 1 target business operations are listed once by feature below. Each group starts with **Daftar Endpoint**, followed by the full specification. Responses are VMS direct records, pages or bytes. A source-registered route can still have the implementation gaps shown in its section.

Jump to a feature group:

- [Authentication](#authentication)
- [Visitor registration](#visitor-registration)
- [Visitor status and ticket](#visitor-status-and-ticket)
- [Staff catalogs](#staff-catalogs)
- [Staff bookings and dashboard](#staff-bookings-and-dashboard)
- [Reception and attendance](#reception-and-attendance)
- [Private attachments](#private-attachments)
- [Employee directory (read-only)](#employee-directory-read-only)
- [Integrations](#integrations)
- [Reports](#reports)

### Authentication

**Daftar Endpoint**

| Endpoint | Method | Frontend feature / control | Success response | Error statuses |
| --- | --- | --- | --- | --- |
| [`/v1/staff/auth/login`](#stafflogin) | `POST` | Staff email/password login form | `200` [AuthResult](SCHEMAS.md#authresult) | 400, 401, 403, 413, 415, 422, 429, 500, 503 |
| [`/v1/staff/auth/me`](#staffme) | `GET` | Current user, navigation and fixed-role access checks | `200` [CurrentUser](SCHEMAS.md#currentuser) | 400, 401, 403, 413, 429, 500, 503 |

---

#### staffLogin

**POST /v1/staff/auth/login — Sign in with email and password**

**Auth Required:** No existing login. Email/password in JSON; no authentication cookies, CSRF token or initialization call.

**Authorization / resource scope:** Anonymous JSON login; no staff token, authentication cookie, signed CSRF token or HTML initialization. Generic invalid/inactive-account credentials; basic selected login limits.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Phase 1 Bearer-only authentication; supersedes the stored-session login/current-session contract.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F11](FRONTEND-TRACEABILITY.md#f11); /admin/login | `Login` — Staff login email/password form and sign-in button | POST email/password to /staff/auth/login; keep AuthResult.accessToken/accessExpiresAt/user in memory. No auth cookies, CSRF initialization or session/role/password gate. | [frontendx/apps/admin/src/main.tsx](../../frontendx/apps/admin/src/main.tsx); Existing signIn still calls /staff/sessions and discards its response; new login/AuthResult consumer and Bearer transport are missing. |
| [F46](FRONTEND-TRACEABILITY.md#f46); all API consumers | `createHttpClient` — Shared Bearer-only HTTP transport and field-error/rate-limit handling | POST JSON login with no auth-cookie/CSRF dependency; parse AuthResult and retain token/expiry/user in memory. Current signIn still calls the superseded session route and discards its result. | [frontendx/packages/data/src/http.ts](../../frontendx/packages/data/src/http.ts); Existing shared client sends no Bearer and still includes cookies; Phase 1 must not rely on them. Its field-error parser expects string[]; current backend string-valued errors are filtered out (BE-ERRORS). |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

No staff authentication header/cookie is required. Public pair/reference proof, when applicable, belongs in the request body.

**Request Body**

Media type: `application/json`; body required: **yes**; model: [Login](SCHEMAS.md#login).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `password` | [ExistingPassword](SCHEMAS.md#existingpassword) | Yes | No wire default | Verify existing credential; new-password minimum is not imposed on current-password/login verification.; Min length: `1`; Max length: `72`; Normalization: `"none"`; Max UTF-8 bytes: `72` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "email": "visitor@example.test",
  "password": "synthetic-passphrase"
}
```

**Business Logic and Side Effects**

Normalize email and verify the existing encoded bcrypt hash; never require the new-password minimum on existing-password verification. Reject missing/multiple/unsupported role assignments rather than picking or unioning roles. Derive fixed role permissions and current PA assignments. Issue signed JWT with sub/iat/exp/iss/aud and configured finite lifetime; return AuthResult without Set-Cookie or a session row. Keep 5 failed attempts/account/15m and 100 requests/IP/15m admission; invalid credential errors are generic.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [AuthResult](SCHEMAS.md#authresult) | Common response headers | Complete Phase 1 authorized result |

**Response Fields**

[AuthResult](SCHEMAS.md#authresult)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `accessToken` | `string` | Yes | No wire default | Min length: `1` |
| `tokenType` | `"Bearer"` | Yes | No wire default | Must equal `"Bearer"` |
| `accessExpiresAt` | `string` | Yes | No wire default | Format: `"date-time"` |
| `user` | [CurrentUser](SCHEMAS.md#currentuser) | Yes | No wire default | Additional keys: `false` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "accessToken": "synthetic-access-token-not-a-credential",
  "tokenType": "Bearer",
  "accessExpiresAt": "2030-01-02T01:00:00Z",
  "user": {
    "active": true,
    "id": "1",
    "name": "Staff Example",
    "role": "PA",
    "permissions": [
      "bookings",
      "create",
      "approve",
      "reject",
      "reschedule",
      "cancel",
      "reports",
      "reportExport"
    ],
    "assignedOfficerIds": [
      "1"
    ]
  }
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/auth_handler.go:Login` → usecase `AuthUsecase.Login` → repository operations `AuthRepository.FindCredentialByEmail`, `AuthRepository.GetCurrentAuthority`. Serializer: `AuthSerializer.ToAuthResult`. DTOs: `Login` → `AuthResult`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-PHASE1-AUTH, BE-AUTH-TESTS, FE-AUTH, BE-ERRORS. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffMe

**GET /v1/staff/auth/me — Read current database-authorized user**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer JWT plus current active employee, exactly one database-assigned role and active PA assignments. No stored session, role selection or password gate.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Phase 1 Bearer-only authentication; supersedes the stored-session login/current-session contract.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F12](FRONTEND-TRACEABILITY.md#f12); /admin/* | `SessionProvider` — Workspace current-user load, fixed token expiry and local Sign out action | GET /staff/auth/me with Bearer; CurrentUser drives authenticated state. Local logout clears token/user/private caches without HTTP. Reload/expiry requires login; no refresh or idle renewal. | [frontendx/packages/data/src/session.tsx](../../frontendx/packages/data/src/session.tsx); Existing provider uses GET/DELETE /staff/session with cookie assumptions; those calls are superseded by Phase 1 me plus local logout. |
| [F44](FRONTEND-TRACEABILITY.md#f44); /admin/* | `Shell / Guard` — Staff name/role display, fixed-permission navigation and route guards | Use CurrentUser.name/role/permissions/assignedOfficerIds; there are no availableRoles, roleRevision, role-selection or required-password gates. Backend independently checks current authority. | [frontendx/apps/admin/src/main.tsx](../../frontendx/apps/admin/src/main.tsx); UI guard is presentation; backend must independently enforce every route |
| [F46](FRONTEND-TRACEABILITY.md#f46); all API consumers | `createHttpClient` — Shared Bearer-only HTTP transport and field-error/rate-limit handling | Attach the in-memory Bearer token; parse CurrentUser, clear auth/private state on expiry or unauthorized response. Current client still uses the old session endpoint/shape. | [frontendx/packages/data/src/http.ts](../../frontendx/packages/data/src/http.ts); Existing shared client sends no Bearer and still includes cookies; Phase 1 must not rely on them. Its field-error parser expects string[]; current backend string-valued errors are filtered out (BE-ERRORS). |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

Validate JWT signature, configured algorithm/key/issuer/audience, iat/exp and canonical employee subject; load current active account and exactly one database role. Derive fixed permissions and current PA assignments; return CurrentUser. Database role/deactivation changes take effect on the next protected request. No token refresh, renewal, role revision, cookie or expiry extension; dependency failure is not an empty/successful user.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [CurrentUser](SCHEMAS.md#currentuser) | Common response headers | Complete Phase 1 authorized result |

**Response Fields**

[CurrentUser](SCHEMAS.md#currentuser)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `active` | `true` | Yes | No wire default | Must equal `true` |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `role` | [Role](SCHEMAS.md#role) | Yes | No wire default | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `permissions` | array of [Permission](SCHEMAS.md#permission) | Yes | No wire default | Min items: `0`; Max items: `12`; Unique items: `true` |
| `assignedOfficerIds` | array of [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | At most one current official assignment per employee; historical inactive rows do not add scope.; Min items: `0`; Max items: `1`; Unique items: `true` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "active": true,
  "id": "1",
  "name": "Staff Example",
  "role": "PA",
  "permissions": [
    "bookings",
    "create",
    "approve",
    "reject",
    "reschedule",
    "cancel",
    "reports",
    "reportExport"
  ],
  "assignedOfficerIds": [
    "1"
  ]
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/auth_handler.go:Me` → usecase `AuthUsecase.GetCurrentUser` → repository operations `AuthRepository.GetCurrentAuthority`. Serializer: `AuthSerializer.ToCurrentUser`. DTOs: `(no request DTO)` → `CurrentUser`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-PHASE1-AUTH, BE-AUTH-TESTS, FE-AUTH, BE-ERRORS. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

### Visitor registration

**Daftar Endpoint**

| Endpoint | Method | Frontend feature / control | Success response | Error statuses |
| --- | --- | --- | --- | --- |
| [`/v1/public/catalogs`](#publiccatalogs) | `GET` | Registration destination dropdown and bilingual privacy dialog | `200` [Catalogs](SCHEMAS.md#catalogs) | 400, 413, 429, 500, 503 |
| [`/v1/public/bookings`](#publiccreatebooking) | `POST` | Public three-step submission and receipt card | `200` [PublicBooking](SCHEMAS.md#publicbooking); `201` [PublicBooking](SCHEMAS.md#publicbooking) | 400, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/public/submissions/resolve`](#resolvesubmission) | `POST` | Uncertain-submission recovery panel | `200` [Resolution](SCHEMAS.md#resolution) | 400, 409, 413, 415, 422, 429, 500, 503 |

---

#### publicCatalogs

**GET /v1/public/catalogs — Read public catalogs**

**Auth Required:** No staff login; endpoint-specific public proof still applies where stated.

**Authorization / resource scope:** Anonymous; active visitor-safe officials/rooms and published notice only; overall public120/min/trusted IP.

**Implementation Status:** Source registered; the gaps below still apply.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F01](FRONTEND-TRACEABILITY.md#f01); /; /?view=registration; /booking | `BookingForm` — Registration destination/official dropdown in the first booking step | officers[].id/name populate officerId options; officers[].vip selects VIP presentation/requirements; privacyNotice supplies consent content. | [frontendx/apps/visitor/src/pages/Booking.tsx](../../frontendx/apps/visitor/src/pages/Booking.tsx); useCatalogs -> repository.catalogs; preview uses domain fixtures |
| [F03](FRONTEND-TRACEABILITY.md#f03); /; /booking; /status | `PrivacyNotice` — Bilingual privacy notice dialog, consent text and retry button | privacyNotice.version/id/en supply the notice and its language; the form uses the published version/ack for consent. Missing notice blocks genuine collection. | [frontendx/apps/visitor/src/features/PrivacyNotice.tsx](../../frontendx/apps/visitor/src/features/PrivacyNotice.tsx); Shares cached useCatalogs; real form blocks without published notice |
| [F48](FRONTEND-TRACEABILITY.md#f48); /; /booking; /status | `App` — Visitor landing shell with embedded registration/status views; hero/info content is static | Registration child consumes the shared catalogs snapshot for dropdown/notice; no landing statistics API. | [frontendx/apps/visitor/src/main.tsx](../../frontendx/apps/visitor/src/main.tsx); Static hero/info; embedded Booking/Status supply dynamic data |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

No staff authentication header/cookie is required. Public pair/reference proof, when applicable, belongs in the request body.

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

One active officials+rooms+notice snapshot; absent notice omitted. Attachments false until verified capability enabled. No staff permission/assignment fields.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [Catalogs](SCHEMAS.md#catalogs) | Common response headers | Complete authorized result |

**Response Fields**

[Catalogs](SCHEMAS.md#catalogs)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `officers` | array of [Officer](SCHEMAS.md#officer) | Yes | No wire default | See linked model and business rules. |
| `rooms` | array of [Room](SCHEMAS.md#room) | Yes | No wire default | See linked model and business rules. |
| `attachmentsEnabled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `privacyNotice` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `privacyNotice.version` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `120` |
| `privacyNotice.id` | `string` | Yes, when parent present | - | Min length: `1` |
| `privacyNotice.en` | `string` | Yes, when parent present | - | Min length: `1` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "officers": [
    {
      "id": "1",
      "name": "Official Example",
      "vip": false,
      "active": true
    }
  ],
  "rooms": [
    {
      "id": "1",
      "name": "Meeting room",
      "active": true
    }
  ],
  "attachmentsEnabled": false,
  "privacyNotice": {
    "version": "2026-11-12",
    "id": "Pemberitahuan organisasi (contoh).",
    "en": "Organization notice (example)."
  }
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/catalog_handler.go:PublicCatalogs` → usecase `CatalogUsecase.GetPublicCatalogs` → repository operations `CatalogRepository.GetPublicCatalogs`. Serializer: `CatalogSerializer.ToPublicCatalog`. DTOs: `(no request DTO)` → `PublicCatalog`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: FE-CATALOG, BE-ERRORS. Requirement and source evidence: Phase 1 target; source registered with reviewed gaps.

Current handler: [backend/internal/handler/catalog_handler.go](../../backend/internal/handler/catalog_handler.go). Route registration does not establish complete behavior.

---

#### publicCreateBooking

**POST /v1/public/bookings — Submit standard booking**

**Auth Required:** No staff login; endpoint-specific public proof still applies where stated.

**Authorization / resource scope:** Anonymous standard-only creation; active official and published notice association checked at commit; shared request30/min and new committed30/hour/trusted IP plus public ceiling.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Create Pending; retained idempotent outcome. Resolve committed reference before mutable privacy/version validation. Request limit30/min/IP; new commits30/hour/IP.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F02](FRONTEND-TRACEABILITY.md#f02); /; /booking | `BookingForm` — Three-step booking form, companion-name rows, optional supporting-document picker and submit/receipt panel | Send normalized representative/contact/schedule/officerId/purpose/visitors/identityType/ack, UUID idempotencyKey and optional attachmentToken/ndName/privacyVersion; PublicBooking.code/status/schedule/history populate the receipt. | [frontendx/apps/visitor/src/pages/Booking.tsx](../../frontendx/apps/visitor/src/pages/Booking.tsx); BookingSubmission; actual create adapter exists; no backend booking route |
| [F08](FRONTEND-TRACEABILITY.md#f08); /status; booking receipt | `VisitTicket` — Booking receipt / printable VisitTicket card | Use the successful create response directly for the receipt and ticket. | [frontendx/apps/visitor/src/features/VisitTicket.tsx](../../frontendx/apps/visitor/src/features/VisitTicket.tsx); Local rendering/print; shares existing booking; no ticket endpoint |
| [F46](FRONTEND-TRACEABILITY.md#f46); all API consumers | `createHttpClient` — Shared Bearer-only HTTP transport and field-error/rate-limit handling | Parse PublicBooking or ApiError.fieldErrors arrays; retain uncertainty/recovery behavior and Retry-After handling. | [frontendx/packages/data/src/http.ts](../../frontendx/packages/data/src/http.ts); Existing shared client sends no Bearer and still includes cookies; Phase 1 must not rely on them. Its field-error parser expects string[]; current backend string-valued errors are filtered out (BE-ERRORS). |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

No staff authentication header/cookie is required. Public pair/reference proof, when applicable, belongs in the request body.

**Request Body**

Media type: `application/json`; body required: **yes**; model: [PublicCreate](SCHEMAS.md#publiccreate).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `idempotencyKey` | [Reference](SCHEMAS.md#reference) | Yes | No wire default | UUID request/submission/upload reference, distinct from a business ID.; Format: `"uuid"`; Min length: `36`; Max length: `36` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | [StartTime](SCHEMAS.md#starttime) | Yes | No wire default | Start only, Asia/Makassar. Must be in future on creation/schedule edits; end may exceed 16:00.; Min length: `5`; Max length: `5`; Pattern: `"^(?:(?:08\|09\|1[0-5]):[0-5][0-9]\|16:00)$"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `attachmentToken` | `string` | No; omit when absent | No wire default | Min length: `16`; Max length: `4096` |
| `ack` | `true` | Yes | No wire default | Must equal `true` |
| `privacyVersion` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `120` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "visitKind": "standard",
  "idempotencyKey": "550e8400-e29b-41d4-a716-446655440000",
  "date": "2030-01-02",
  "time": "09:00",
  "ack": true
}
```

**Business Logic and Side Effects**

Require representative=visitors[0],1-20 names, ack=true. Resolve committed replay before current mutable privacy validation. Pending, duration60, no room/reservation. Persist visitor/detail/visit/roster, audit, submission outcome and logical notification together;201 new/200 replay. Upload optional.

- roster[0] equals representative name after normalization
- Pending; duration=60; no room input/reservation
- D90 omitted privacyVersion resolves current server version; D91 supplied stale/unknown fails before new write; committed retry preserves original association.
- Attachment token and display name refer to same verified owned upload.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [PublicBooking](SCHEMAS.md#publicbooking) | Common response headers | Same-reference committed replay; no additional write/notification |
| `201` | application/json: [PublicBooking](SCHEMAS.md#publicbooking) | Common response headers | Complete authorized result |

**Response Fields**

[PublicBooking](SCHEMAS.md#publicbooking)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `history` | array of [PublicEvent](SCHEMAS.md#publicevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "pending",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "pending",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

HTTP `201` returns the same body shape/example above; status meaning is defined in the success table.

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:PublicCreate` → usecase `VisitUsecase.CreatePublicBooking` → repository operations `VisitRepository.Transaction`, `VisitTransaction.ResolveSubmission`, `VisitTransaction.CreateVisit`, `VisitTransaction.AppendAudit`, `VisitTransaction.EnqueueNotifications`. Serializer: `VisitSerializer.ToPublicBooking`. DTOs: `PublicCreate` → `PublicBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-VISITS, FE-AUTH, FE-BOUNDS, FE-UPLOAD. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### resolveSubmission

**POST /v1/public/submissions/resolve — Resolve and close submission reference**

**Auth Required:** No staff login; endpoint-specific public proof still applies where stated.

**Authorization / resource scope:** Anonymous UUID+normalized email bound to the original audience/reference. Reference alone grants no read authority; shared create/resolve30/min plus public ceiling.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Serialize with create. not_committed closes key against delayed create; uncertainty/503 is never not_committed.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F04](FRONTEND-TRACEABILITY.md#f04); /; /booking | `SubmissionRecovery` — Uncertain-submission recovery panel and email recovery action | Send retained idempotencyKey plus email; outcome=committed restores booking/receipt, not_committed allows a new submission after durable closure. | [frontendx/apps/visitor/src/features/SubmissionRecovery.tsx](../../frontendx/apps/visitor/src/features/SubmissionRecovery.tsx); BookingForm -> BookingSubmission.recover; only reference in sessionStorage |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

No staff authentication header/cookie is required. Public pair/reference proof, when applicable, belongs in the request body.

**Request Body**

Media type: `application/json`; body required: **yes**; model: [SubmissionResolve](SCHEMAS.md#submissionresolve).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `idempotencyKey` | [Reference](SCHEMAS.md#reference) | Yes | No wire default | UUID request/submission/upload reference, distinct from a business ID.; Format: `"uuid"`; Min length: `36`; Max length: `36` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "idempotencyKey": "550e8400-e29b-41d4-a716-446655440000",
  "email": "visitor@example.test"
}
```

**Business Logic and Side Effects**

Serialize against same scoped create reference. Return committed/current public projection or not_committed only after durable closure that prevents delayed creates.503/timeout remains uncertain.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [Resolution](SCHEMAS.md#resolution) | Common response headers | Complete authorized result |

**Response Fields**

[Resolution](SCHEMAS.md#resolution)

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: `object`**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `outcome` | `"committed"` | Yes | No wire default | Must equal `"committed"` |
| `booking` | [PublicBooking](SCHEMAS.md#publicbooking) | Yes | No wire default | Additional keys: `false` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: `object`**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `outcome` | `"not_committed"` | Yes | No wire default | Must equal `"not_committed"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "outcome": "committed",
  "booking": {
    "id": "1",
    "visitKind": "standard",
    "date": "2030-01-02",
    "time": "09:00",
    "duration": 60,
    "room": "",
    "status": "pending",
    "version": 1,
    "timing": {
      "serverTime": 1893542400000,
      "opensAt": 1893544200000,
      "endsAt": 1893549600000,
      "displayStatus": "pending",
      "canCheckIn": false,
      "canCheckOut": false,
      "canCancel": false
    },
    "code": "01234567",
    "name": "Visitor Example",
    "organization": "Example Office",
    "email": "visitor@example.test",
    "phone": "081234567890",
    "officerId": "1",
    "host": "Official Example",
    "vip": false,
    "purpose": "Discuss visitor services",
    "visitors": [
      "Visitor Example"
    ],
    "identityType": "KTP",
    "history": []
  }
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:ResolveSubmission` → usecase `VisitUsecase.ResolveSubmission` → repository operations `VisitRepository.Transaction`, `VisitTransaction.ResolveSubmission`. Serializer: `VisitSerializer.ToResolution`. DTOs: `SubmissionResolve` → `Resolution`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-VISITS. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

### Visitor status and ticket

**Daftar Endpoint**

| Endpoint | Method | Frontend feature / control | Success response | Error statuses |
| --- | --- | --- | --- | --- |
| [`/v1/public/lookup`](#publiclookup) | `POST` | Code/email status form, visit card, ticket and history | `200` [PublicBooking](SCHEMAS.md#publicbooking) | 400, 404, 413, 415, 422, 429, 500, 503 |
| [`/v1/public/bookings/{id}/cancel`](#publiccancel) | `POST` | Status-card cancellation confirmation | `200` [PublicBooking](SCHEMAS.md#publicbooking) | 400, 404, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/public/bookings/{id}/admission`](#publicadmission) | `POST` | Admission QR image, countdown and renewal | `200` [AdmissionResult](SCHEMAS.md#admissionresult) | 400, 404, 409, 413, 415, 422, 429, 500, 503 |

---

#### publicLookup

**POST /v1/public/lookup — Look up standard booking**

**Auth Required:** No staff login; endpoint-specific public proof still applies where stated.

**Authorization / resource scope:** Matching eight-digit code+normalized email for standard visit only. Generic unavailable response for wrong pair/restricted/absent. Shared pair30/min/IP plus public ceiling.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Code+email pair, standard only; generic failure. Shared30/min/IP lookup/cancel/admission.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F05](FRONTEND-TRACEABILITY.md#f05); /status; /?view=status | `Status` — Status lookup form: eight-digit code input, email input, View visit and Update buttons | Send code/email in POST body; returned status/timing, schedule, host, roster, instructions and public history populate the visit status card. | [frontendx/apps/visitor/src/pages/Status.tsx](../../frontendx/apps/visitor/src/pages/Status.tsx); repository.lookup; invalid pair errors preserve staff auth |
| [F06](FRONTEND-TRACEABILITY.md#f06); /status; /?view=status | `Status` — Cancel visit button and confirmation dialog on the status card | After a version/state conflict, repeat code/email lookup and replace the displayed booking with current authority. | [frontendx/apps/visitor/src/pages/Status.tsx](../../frontendx/apps/visitor/src/pages/Status.tsx); repository.cancel; conflict attempts a fresh lookup |
| [F08](FRONTEND-TRACEABILITY.md#f08); /status; booking receipt | `VisitTicket` — Booking receipt / printable VisitTicket card | Use existing PublicBooking.code/date/time/host/room/visitors; no separate ticket fetch. | [frontendx/apps/visitor/src/features/VisitTicket.tsx](../../frontendx/apps/visitor/src/features/VisitTicket.tsx); Local rendering/print; shares existing booking; no ticket endpoint |
| [F09](FRONTEND-TRACEABILITY.md#f09); /status | `PublicVisitHistory` — Public visit-history timeline within the status view | history[] and public reasons/instructions populate the timeline; no private actors/notes. | [frontendx/apps/visitor/src/features/PublicVisitHistory.tsx](../../frontendx/apps/visitor/src/features/PublicVisitHistory.tsx); Renders history already in booking; no history endpoint |
| [F10](FRONTEND-TRACEABILITY.md#f10); /status | `useCancellationWindow` — Time-based enabled/disabled state of the public cancellation button | timing.serverTime/canCancel and schedule establish the local conservative cancellation countdown. | [frontendx/apps/visitor/src/features/useCancellationWindow.ts](../../frontendx/apps/visitor/src/features/useCancellationWindow.ts); Local conservative boundary timer; mutation still authoritative |
| [F47](FRONTEND-TRACEABILITY.md#f47); all booking consumers | `createRepository` — Shared production repository used by booking/status components; preview adapter is explicit | Use pair-authorized PublicBooking for status/history; preview role hints never enter production authority. | [frontendx/packages/data/src/repositories.ts](../../frontendx/packages/data/src/repositories.ts); Production list() legacy array branch exists but useBookings uses readStaffBookings instead |
| [F48](FRONTEND-TRACEABILITY.md#f48); /; /booking; /status | `App` — Visitor landing shell with embedded registration/status views; hero/info content is static | Status child consumes pair-authorized booking; the shell needs no extra landing endpoint. | [frontendx/apps/visitor/src/main.tsx](../../frontendx/apps/visitor/src/main.tsx); Static hero/info; embedded Booking/Status supply dynamic data |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

No staff authentication header/cookie is required. Public pair/reference proof, when applicable, belongs in the request body.

**Request Body**

Media type: `application/json`; body required: **yes**; model: [Pair](SCHEMAS.md#pair).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "code": "01234567",
  "email": "visitor@example.test"
}
```

**Business Logic and Side Effects**

Return full safe PublicBooking with allowlisted public history/timing; omit internal notes/actors/service IDs. Read-only, no state or expiry renewal.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [PublicBooking](SCHEMAS.md#publicbooking) | Common response headers | Complete authorized result |

**Response Fields**

[PublicBooking](SCHEMAS.md#publicbooking)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `history` | array of [PublicEvent](SCHEMAS.md#publicevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "pending",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "pending",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:PublicLookup` → usecase `VisitUsecase.GetPublicBooking` → repository operations `VisitRepository.GetPublicBooking`. Serializer: `VisitSerializer.ToPublicBooking`. DTOs: `Pair` → `PublicBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-READS, BE-VISITS, FE-CATALOG. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### publicCancel

**POST /v1/public/bookings/{id}/cancel — Cancel before start**

**Auth Required:** No staff login; endpoint-specific public proof still applies where stated.

**Authorization / resource scope:** Path ID must match standard visit authorized by code+email pair; version and strict before-start/no-checkin checks. Same pair-access budget.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Pending/Approved, strictly before start and no prior check-in.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F06](FRONTEND-TRACEABILITY.md#f06); /status; /?view=status | `Status` — Cancel visit button and confirmation dialog on the status card | Send selected id plus code/email/version; returned cancelled PublicBooking replaces the card/history. | [frontendx/apps/visitor/src/pages/Status.tsx](../../frontendx/apps/visitor/src/pages/Status.tsx); repository.cancel; conflict attempts a fresh lookup |
| [F09](FRONTEND-TRACEABILITY.md#f09); /status | `PublicVisitHistory` — Public visit-history timeline within the status view | Use the returned booking.history and cancellation state to refresh the same timeline. | [frontendx/apps/visitor/src/features/PublicVisitHistory.tsx](../../frontendx/apps/visitor/src/features/PublicVisitHistory.tsx); Renders history already in booking; no history endpoint |
| [F10](FRONTEND-TRACEABILITY.md#f10); /status | `useCancellationWindow` — Time-based enabled/disabled state of the public cancellation button | Send the current version and pair; the server rechecks the time boundary even if the button was enabled. | [frontendx/apps/visitor/src/features/useCancellationWindow.ts](../../frontendx/apps/visitor/src/features/useCancellationWindow.ts); Local conservative boundary timer; mutation still authoritative |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

No staff authentication header/cookie is required. Public pair/reference proof, when applicable, belongs in the request body.

**Request Body**

Media type: `application/json`; body required: **yes**; model: [PairVersion](SCHEMAS.md#pairversion).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "code": "01234567",
  "email": "visitor@example.test",
  "version": 1
}
```

**Business Logic and Side Effects**

Pending/Approved and now<start; no prior checkin. Atomically version/save/release commitments/revoke admission/audit/outbox; code/email remain private.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [PublicBooking](SCHEMAS.md#publicbooking) | Common response headers | Complete authorized result |

**Response Fields**

[PublicBooking](SCHEMAS.md#publicbooking)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `history` | array of [PublicEvent](SCHEMAS.md#publicevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "cancelled",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "cancelled",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:PublicCancel` → usecase `VisitUsecase.CancelPublicBooking` → repository operations `VisitRepository.Transaction`, `VisitTransaction.LockVisit`, `VisitTransaction.SaveVisit`, `VisitTransaction.RevokeAdmission`, `VisitTransaction.AppendAudit`, `VisitTransaction.EnqueueNotifications`. Serializer: `VisitSerializer.ToPublicBooking`. DTOs: `PairVersion` → `PublicBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-VISITS. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### publicAdmission

**POST /v1/public/bookings/{id}/admission — Issue replaceable admission QR**

**Auth Required:** No staff login; endpoint-specific public proof still applies where stated.

**Authorization / resource scope:** Matching standard visit ID/code/email/version and eligible Approved check-in window; same pair budget. No public authorization from code alone.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Eligible Approved standard visit. Expiry=min(now+5m,checkin end); revoke predecessor, never issue expired credential.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F07](FRONTEND-TRACEABILITY.md#f07); /status; booking receipt | `AdmissionQR` — Admission QR image, expiry/countdown and Renew QR code/Try again buttons | Send id plus code/email/version; encode payload locally as QR, use expiresAt/serverTime for countdown and replacement. | [frontendx/apps/visitor/src/features/AdmissionQR.tsx](../../frontendx/apps/visitor/src/features/AdmissionQR.tsx); repository.admission; preview constructs VMS-DEMO credential locally |
| [F08](FRONTEND-TRACEABILITY.md#f08); /status; booking receipt | `VisitTicket` — Booking receipt / printable VisitTicket card | Render the returned admission payload locally as QR; printing is a local browser action. | [frontendx/apps/visitor/src/features/VisitTicket.tsx](../../frontendx/apps/visitor/src/features/VisitTicket.tsx); Local rendering/print; shares existing booking; no ticket endpoint |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

No staff authentication header/cookie is required. Public pair/reference proof, when applicable, belongs in the request body.

**Request Body**

Media type: `application/json`; body required: **yes**; model: [PairVersion](SCHEMAS.md#pairversion).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "code": "01234567",
  "email": "visitor@example.test",
  "version": 1
}
```

**Business Logic and Side Effects**

Replace/revoke predecessor; expiry=min(issue+5min,end), never issue expired payload. Opaque credential has no person data; QR issuance is not attendance.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [AdmissionResult](SCHEMAS.md#admissionresult) | Common response headers | Complete authorized result |

**Response Fields**

[AdmissionResult](SCHEMAS.md#admissionresult)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `payload` | `string` | Yes | No wire default | Min length: `1`; Max length: `4096` |
| `expiresAt` | `string` | Yes | No wire default | Format: `"date-time"` |
| `serverTime` | `integer` | Yes | No wire default | Min: `0`; Max: `9007199254740991` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "payload": "vmsq1.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "expiresAt": "2030-01-02T01:00:00Z",
  "serverTime": 1893542400000
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/reception_handler.go:PublicAdmission` → usecase `ReceptionUsecase.IssueAdmission` → repository operations `VisitRepository.Transaction`, `VisitTransaction.LockVisit`, `VisitTransaction.ReplaceAdmission`. Serializer: `ReceptionSerializer.ToAdmissionResult`. DTOs: `PairVersion` → `AdmissionResult`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-ATTENDANCE, BE-VISITS. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

### Staff catalogs

**Daftar Endpoint**

| Endpoint | Method | Frontend feature / control | Success response | Error statuses |
| --- | --- | --- | --- | --- |
| [`/v1/staff/catalogs`](#staffcatalogs) | `GET` | Staff official/room selectors and PA assignment checkboxes | `200` [Catalogs](SCHEMAS.md#catalogs) | 400, 401, 403, 409, 413, 429, 500, 503 |

---

#### staffCatalogs

**GET /v1/staff/catalogs — Read authorized staff catalogs**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Current Bearer identity and fixed role scope; safe official/room catalog only. Employee role/assignment editors are deferred; no credentials or private employee directory in catalogs.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F21](FRONTEND-TRACEABILITY.md#f21); /admin/booking?new=1; /admin/today?new=1 | `CreateAppointment` — Staff standard-appointment dialog: official dropdown, schedule, representative/roster, note fields, document picker and submit | officers[].id/name/vip populate and scope the official selector; attachmentsEnabled gates the optional file picker. | [frontendx/apps/admin/src/features/CreateAppointment.tsx](../../frontendx/apps/admin/src/features/CreateAppointment.tsx); useCreateAppointment; UUID retained while payload unchanged; room/duration not collected |
| [F22](FRONTEND-TRACEABILITY.md#f22); /admin/booking?new=1&kind=restricted | `RestrictedAppointment` — Kunjungan Terbatas creation dialog: date/time, optional identity/target fields, notes and create button | officers[].id/name populate optional officerId selector; no public contact/roster/file requirement is added. | [frontendx/apps/admin/src/features/RestrictedAppointment.tsx](../../frontendx/apps/admin/src/features/RestrictedAppointment.tsx); Protocol create gate; no contact/identity number/roster/file required |
| [F23](FRONTEND-TRACEABILITY.md#f23); /admin/booking?selected={id} | `RestrictedAppointment` — Restricted visit edit dialog and Simpan perubahan button | Use authorized official IDs/names and retained choices for the optional target selector. | [frontendx/apps/admin/src/features/RestrictedAppointment.tsx](../../frontendx/apps/admin/src/features/RestrictedAppointment.tsx); useBookingAction(update); date/time inputs readOnly on existing record |
| [F24](FRONTEND-TRACEABILITY.md#f24); /admin/booking; /admin/today | `BookingDetail` — Approve dialog: room dropdown, duration input, note fields and confirmation | rooms[].id/name/active supply the approval room selector; selected value must become stable roomId. | [frontendx/apps/admin/src/features/BookingDetail.tsx](../../frontendx/apps/admin/src/features/BookingDetail.tsx); Form sends room label and duration; target requires roomId |
| [F26](FRONTEND-TRACEABILITY.md#f26); /admin/booking; /admin/today | `BookingDetail` — Reschedule dialog: new date/time, room dropdown, duration, reason and notes | rooms[].id/name/active populate room options; current form still sends a room label. | [frontendx/apps/admin/src/features/BookingDetail.tsx](../../frontendx/apps/admin/src/features/BookingDetail.tsx); room label sent; domain.can permits Approved only while D100 also selects Pending |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

Reuse one Catalogs request for staff official and room selectors. Preserve stable IDs and authorized retained selections needed by existing visits; no new role-policy or assignment-editor request. Consistent directory snapshot; no invented truncation.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [Catalogs](SCHEMAS.md#catalogs) | Common response headers | Complete authorized result |

**Response Fields**

[Catalogs](SCHEMAS.md#catalogs)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `officers` | array of [Officer](SCHEMAS.md#officer) | Yes | No wire default | See linked model and business rules. |
| `rooms` | array of [Room](SCHEMAS.md#room) | Yes | No wire default | See linked model and business rules. |
| `attachmentsEnabled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `privacyNotice` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `privacyNotice.version` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `120` |
| `privacyNotice.id` | `string` | Yes, when parent present | - | Min length: `1` |
| `privacyNotice.en` | `string` | Yes, when parent present | - | Min length: `1` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "officers": [
    {
      "id": "1",
      "name": "Official Example",
      "vip": false,
      "active": true
    }
  ],
  "rooms": [
    {
      "id": "1",
      "name": "Meeting room",
      "active": true
    }
  ],
  "attachmentsEnabled": false,
  "privacyNotice": {
    "version": "2026-11-12",
    "id": "Pemberitahuan organisasi (contoh).",
    "en": "Organization notice (example)."
  }
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/catalog_handler.go:StaffCatalogs` → usecase `CatalogUsecase.GetStaffCatalogs` → repository operations `CatalogRepository.GetStaffCatalogs`. Serializer: `CatalogSerializer.ToStaffCatalog`. DTOs: `(no request DTO)` → `PublicCatalog`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-ADMIN, BE-VISITS, FE-PENDING, FE-ROOM, FE-UPLOAD, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

### Staff bookings and dashboard

**Daftar Endpoint**

| Endpoint | Method | Frontend feature / control | Success response | Error statuses |
| --- | --- | --- | --- | --- |
| [`/v1/staff/bookings`](#staffbookings) | `GET` | Booking search/table/cards, dashboard cards/agenda and reception table | `200` [ReadModel](SCHEMAS.md#readmodel) | 400, 401, 403, 409, 413, 422, 429, 500, 503 |
| [`/v1/staff/bookings/{id}`](#staffbooking) | `GET` | Selected booking-detail dialog | `200` [StaffBooking](SCHEMAS.md#staffbooking) | 400, 401, 403, 404, 409, 413, 429, 500, 503 |
| [`/v1/staff/bookings`](#staffcreatebooking) | `POST` | Standard and restricted staff create dialogs | `200` [StaffBooking](SCHEMAS.md#staffbooking); `201` [StaffBooking](SCHEMAS.md#staffbooking) | 400, 401, 403, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/staff/bookings/{id}/approve`](#staffapprove) | `POST` | Approval room/duration/notes form | `200` [StaffBooking](SCHEMAS.md#staffbooking) | 400, 401, 403, 404, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/staff/bookings/{id}/reject`](#staffreject) | `POST` | Public rejection reason and internal note form | `200` [StaffBooking](SCHEMAS.md#staffbooking) | 400, 401, 403, 404, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/staff/bookings/{id}/reschedule`](#staffreschedule) | `POST` | New schedule/room/duration/reason form | `200` [StaffBooking](SCHEMAS.md#staffbooking) | 400, 401, 403, 404, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/staff/bookings/{id}/update`](#staffrestrictedupdate) | `POST` | Restricted identity/target/notes edit dialog | `200` [RestrictedFull](SCHEMAS.md#restrictedfull) | 400, 401, 403, 404, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/staff/bookings/{id}/cancel`](#staffcancel) | `POST` | Staff cancellation reason/internal-note dialog | `200` [StaffBooking](SCHEMAS.md#staffbooking) | 400, 401, 403, 404, 409, 413, 415, 422, 429, 500, 503 |

---

#### staffBookings

**GET /v1/staff/bookings — Read typed authorized booking view**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer + bookings + role scope before filters/counts: PA assigned non-VIP ordinary and assigned restricted projection; Protocol all authorized records; Reception approved/checked-in/checked-out with minimal restricted projection; Super Admin restricted full reads only.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Reject unknown/inapplicable query keys. Defaults page1; list7/reception25/dashboard50 items. vip0=no VIP filter, vip1=ordinary VIP only. Scope before search/counts. Checkout list includes earlier-day still-inside visits.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F16](FRONTEND-TRACEABILITY.md#f16); /admin/booking; /admin/today | `Bookings` — Janji temu / Kunjungan hari ini search box, status tabs, VIP filter, sort selector, cards/table and pagination | Send view=list plus q/status/sort/dir/date/vip/page/pageSize=7; items populate cards/table, total/totalScoped/page/pageSize control counts and pagination. | [frontendx/apps/admin/src/pages/Bookings.tsx](../../frontendx/apps/admin/src/pages/Bookings.tsx); Production readStaffBookings; preview full list then local selectBookings |
| [F18](FRONTEND-TRACEABILITY.md#f18); /admin/dashboard | `Dashboard` — Dashboard metric cards, Agenda hari ini / Menunggu kedatangan table and Perlu perhatian / Sedang berkunjung list | Send view=dashboard/date/page=1/pageSize=50; metrics.today/pending/arrivals/inside/completed supply cards, items supply the bounded agenda/attention union. Super Admin currently skips operational reads. | [frontendx/apps/admin/src/pages/Dashboard.tsx](../../frontendx/apps/admin/src/pages/Dashboard.tsx); Shares booking read; Super Admin skips operational query |
| [F19](FRONTEND-TRACEABILITY.md#f19); /admin/dashboard; /admin/today; /admin/booking; /admin/check-in; /admin/check-out | `ProductionRefresh` — Visible-page refresh timer, focus refresh and manual refresh for lists/details/reception | Reissue the active typed list/dashboard/reception query; replace rows/counts/metrics together. | [frontendx/apps/admin/src/components/BookingRefresh.tsx](../../frontendx/apps/admin/src/components/BookingRefresh.tsx); Refetches active queries; no new endpoint; avoid auth renewal on polling |
| [F20](FRONTEND-TRACEABILITY.md#f20); /admin/* | `DataRefresh` — DataRefresh freshness/offline/error notice and retry controls | Refetch existing booking view; use query fetch/error/update state for the freshness control. | [frontendx/apps/admin/src/components/DataRefresh.tsx](../../frontendx/apps/admin/src/components/DataRefresh.tsx); Pure query control; no persisted business data |
| [F30](FRONTEND-TRACEABILITY.md#f30); /admin/check-in; /admin/check-out | `Reception` — Reception scheduled-arrivals / still-inside table and previous/next page controls | Send view=reception/mode/page/pageSize=25; check-in sends today date, checkout spans all dates. items and total/page/pageSize populate the table and pagination. | [frontendx/apps/admin/src/pages/Reception.tsx](../../frontendx/apps/admin/src/pages/Reception.tsx); useBookings typed view; one endpoint reused |
| [F47](FRONTEND-TRACEABILITY.md#f47); all booking consumers | `createRepository` — Shared production repository used by booking/status components; preview adapter is explicit | Use typed paged ReadModel for staff hooks; legacy array list() is not the target wire shape. | [frontendx/packages/data/src/repositories.ts](../../frontendx/packages/data/src/repositories.ts); Production list() legacy array branch exists but useBookings uses readStaffBookings instead |

**Query Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `view` | `string` | Yes | None declared | Allowed: `"list"`, `"reception"`, `"dashboard"` |
| `q` | `string` | No | None declared | Min length: `0`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `status` | `string` | No | None declared | Allowed: `"all"`, `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"`, `"missed"` |
| `sort` | `string` | No | None declared | Allowed: `"priority"`, `"name"`, `"organization"`, `"host"`, `"time"`, `"status"` |
| `dir` | `string` | No | None declared | Allowed: `"asc"`, `"desc"` |
| `date` | [Date](SCHEMAS.md#date) | No | None declared | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `vip` | `string` | No | None declared | Allowed: `"0"`, `"1"` |
| `page` | `integer` | No | `1` | Min: `1`; Max: `2147483647` |
| `pageSize` | `integer` | No | None declared | Min: `1`; Max: `100` |
| `mode` | `string` | No | None declared | Allowed: `"checkin"`, `"checkout"` |

The following view variants are mutually exclusive. Their required keys/defaults override the broad parameter inventory above; other-view keys are rejected.

**`view=list` query variant**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `view` | `"list"` | Yes | No wire default | Must equal `"list"` |
| `q` | `string` | No; omit when absent | `""` | Min length: `0`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `status` | `string` | No; omit when absent | `"all"` | Allowed: `"all"`, `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"`, `"missed"` |
| `sort` | `string` | No; omit when absent | `"priority"` | Allowed: `"priority"`, `"name"`, `"organization"`, `"host"`, `"time"`, `"status"` |
| `dir` | `string` | No; omit when absent | `"asc"` | Allowed: `"asc"`, `"desc"` |
| `date` | [Date](SCHEMAS.md#date) | No; omit when absent | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `vip` | `string` | No; omit when absent | `"0"` | Allowed: `"0"`, `"1"` |
| `page` | `integer` | No; omit when absent | `1` | Min: `1`; Max: `2147483647` |
| `pageSize` | `integer` | No; omit when absent | `7` | Min: `1`; Max: `100` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**`view=reception` query variant**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `view` | `"reception"` | Yes | No wire default | Must equal `"reception"` |
| `mode` | `string` | Yes | No wire default | Allowed: `"checkin"`, `"checkout"` |
| `date` | [Date](SCHEMAS.md#date) | No; omit when absent | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `page` | `integer` | No; omit when absent | `1` | Min: `1`; Max: `2147483647` |
| `pageSize` | `integer` | No; omit when absent | `25` | Min: `1`; Max: `100` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**`view=dashboard` query variant**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `view` | `"dashboard"` | Yes | No wire default | Must equal `"dashboard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `page` | `1` | No; omit when absent | `1` | Must equal `1` |
| `pageSize` | `integer` | No; omit when absent | `50` | Min: `1`; Max: `100` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

Reception `mode=checkout` rejects `date` and includes still-inside visits from prior dates. Dashboard requires `date` and `page=1`. Check-in date behavior and exact sorting boundaries remain as specified by the selected rules and OPEN-READ-ORDER.

```http
GET /v1/staff/bookings?view=list
```

```http
GET /v1/staff/bookings?view=list&page=2147483647&pageSize=100
```

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

Three mutually exclusive query variants; max100; list default7, reception25, dashboard50; one-based page echoed even beyond end. q trimmed max255. vip0=no filter,1=ordinary VIP. approved excludes Missed. Scope/projection before search/sort/count; snapshot for rows/counts/metrics. Dashboard deduplicates first5 agenda+first4 attention. Reception checkout spans earlier dates.

Unresolved boundary **OPEN-READ-ORDER**: Selected rules require stable sorting but do not freeze database text collation/normalization or ID tie comparison. Frontend preview uses Indonesian localeCompare and lexicographic ID ties; employee target says numeric ID ties. These are different with IDs 2 and 10 or accented names. See [CONTRACT-ISSUES](CONTRACT-ISSUES.md).

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [ReadModel](SCHEMAS.md#readmodel) | Common response headers | Complete authorized result |

**Response Fields**

[ReadModel](SCHEMAS.md#readmodel)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `items` | array of [StaffBooking](SCHEMAS.md#staffbooking) | Yes | No wire default | Min items: `0`; Max items: `100` |
| `total` | `integer` | Yes | No wire default | Min: `0`; Max: `9007199254740991` |
| `totalScoped` | `integer` | Yes | No wire default | Min: `0`; Max: `9007199254740991` |
| `page` | `integer` | Yes | No wire default | Min: `1`; Max: `2147483647` |
| `pageSize` | `integer` | Yes | No wire default | Min: `1`; Max: `100` |
| `metrics` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `metrics.today` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `metrics.pending` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `metrics.arrivals` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `metrics.inside` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `metrics.completed` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "items": [
    {
      "id": "1",
      "visitKind": "standard",
      "date": "2030-01-02",
      "time": "09:00",
      "duration": 60,
      "room": "",
      "status": "pending",
      "version": 1,
      "timing": {
        "serverTime": 1893542400000,
        "opensAt": 1893544200000,
        "endsAt": 1893549600000,
        "displayStatus": "pending",
        "canCheckIn": false,
        "canCheckOut": false,
        "canCancel": false
      },
      "code": "01234567",
      "name": "Visitor Example",
      "organization": "Example Office",
      "email": "visitor@example.test",
      "phone": "081234567890",
      "officerId": "1",
      "host": "Official Example",
      "vip": false,
      "purpose": "Discuss visitor services",
      "visitors": [
        "Visitor Example"
      ],
      "identityType": "KTP",
      "history": []
    }
  ],
  "total": 1,
  "totalScoped": 1,
  "page": 1,
  "pageSize": 7
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:List` → usecase `VisitUsecase.GetBookings` → repository operations `VisitRepository.GetBookings`. Serializer: `VisitSerializer.ToReadModel`. DTOs: `BookingQuery` → `ReadModel`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-READS, FE-CORRELATION, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffBooking

**GET /v1/staff/bookings/{id} — Read one authorized booking**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Same resource and field scope as staffBookings; independently resolve ID beyond current list page. Restricted full read does not automatically authorize private notes.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F17](FRONTEND-TRACEABILITY.md#f17); /admin/booking?selected={id}; /admin/today?selected={id} | `Bookings / BookingDetail` — Selected booking-detail dialog opened from a row/card or selected URL parameter | Send path id; returned authorized StaffBooking variant populates the dialog even when the booking is outside the current list page. | [frontendx/apps/admin/src/pages/Bookings.tsx](../../frontendx/apps/admin/src/pages/Bookings.tsx); useBookingDetail -> readStaffBooking; does not verify returned ID |
| [F19](FRONTEND-TRACEABILITY.md#f19); /admin/dashboard; /admin/today; /admin/booking; /admin/check-in; /admin/check-out | `ProductionRefresh` — Visible-page refresh timer, focus refresh and manual refresh for lists/details/reception | Reissue the selected-id read; replace only with a currently authorized matching detail. | [frontendx/apps/admin/src/components/BookingRefresh.tsx](../../frontendx/apps/admin/src/components/BookingRefresh.tsx); Refetches active queries; no new endpoint; avoid auth renewal on polling |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

ID must equal requested ID; current field projection and timing; no dependency on current page. Private history included only with field authority.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [StaffBooking](SCHEMAS.md#staffbooking) | Common response headers | Complete authorized result |

**Response Fields**

[StaffBooking](SCHEMAS.md#staffbooking)

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: [StandardStaffBooking](SCHEMAS.md#standardstaffbooking)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `attachmentAvailable` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: [RestrictedReception](SCHEMAS.md#restrictedreception)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"reception"` | Yes | No wire default | Must equal `"reception"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 3: [RestrictedArrival](SCHEMAS.md#restrictedarrival)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"arrival"` | Yes | No wire default | Must equal `"arrival"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 4: [RestrictedAssignedPA](SCHEMAS.md#restrictedassignedpa)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"assigned-pa"` | Yes | No wire default | Must equal `"assigned-pa"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 5: [RestrictedFull](SCHEMAS.md#restrictedfull)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"full"` | Yes | No wire default | Must equal `"full"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | No; omit when absent | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `serviceId` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `100`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | No; omit when absent | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "pending",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "pending",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:Detail` → usecase `VisitUsecase.GetBooking` → repository operations `VisitRepository.GetBooking`. Serializer: `VisitSerializer.ToStaffBooking`. DTOs: `(no request DTO)` → `StaffBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-READS, FE-CORRELATION, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffCreateBooking

**POST /v1/staff/bookings — Create approved staff booking**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer + create. Standard: PA assigned non-VIP official, Protocol VIP official. Restricted: Protocol, optional valid active official. No Super Admin/Reception creation.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F21](FRONTEND-TRACEABILITY.md#f21); /admin/booking?new=1; /admin/today?new=1 | `CreateAppointment` — Staff standard-appointment dialog: official dropdown, schedule, representative/roster, note fields, document picker and submit | Send visitKind=standard and ordinary fields, UUID idempotencyKey and optional notes/upload token; returned approved booking is the save result. | [frontendx/apps/admin/src/features/CreateAppointment.tsx](../../frontendx/apps/admin/src/features/CreateAppointment.tsx); useCreateAppointment; UUID retained while payload unchanged; room/duration not collected |
| [F22](FRONTEND-TRACEABILITY.md#f22); /admin/booking?new=1&kind=restricted | `RestrictedAppointment` — Kunjungan Terbatas creation dialog: date/time, optional identity/target fields, notes and create button | Send visitKind=restricted, idempotencyKey/date/time and optional name/organization/serviceId/officerId/notes; render returned authorized restricted booking. | [frontendx/apps/admin/src/features/RestrictedAppointment.tsx](../../frontendx/apps/admin/src/features/RestrictedAppointment.tsx); Protocol create gate; no contact/identity number/roster/file required |
| [F28](FRONTEND-TRACEABILITY.md#f28); staff create / approve / reschedule / restricted edit dialogs | `VisitNotesFields` — Shared Instruksi pengunjung and Catatan koordinasi internal textareas | Include visitorInstructions/coordinationNote in create; returned fields render only for the permitted projection. | [frontendx/apps/admin/src/features/VisitNotesFields.tsx](../../frontendx/apps/admin/src/features/VisitNotesFields.tsx); Shared form fields; notes travel with existing command |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `application/json`; body required: **yes**; model: [StaffCreate](SCHEMAS.md#staffcreate).

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: [StandardStaffCreate](SCHEMAS.md#standardstaffcreate)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `idempotencyKey` | [Reference](SCHEMAS.md#reference) | Yes | No wire default | UUID request/submission/upload reference, distinct from a business ID.; Format: `"uuid"`; Min length: `36`; Max length: `36` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | [StartTime](SCHEMAS.md#starttime) | Yes | No wire default | Start only, Asia/Makassar. Must be in future on creation/schedule edits; end may exceed 16:00.; Min length: `5`; Max length: `5`; Pattern: `"^(?:(?:08\|09\|1[0-5]):[0-5][0-9]\|16:00)$"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `attachmentToken` | `string` | No; omit when absent | No wire default | Min length: `16`; Max length: `4096` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: [RestrictedCreate](SCHEMAS.md#restrictedcreate)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `idempotencyKey` | [Reference](SCHEMAS.md#reference) | Yes | No wire default | UUID request/submission/upload reference, distinct from a business ID.; Format: `"uuid"`; Min length: `36`; Max length: `36` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | [StartTime](SCHEMAS.md#starttime) | Yes | No wire default | Start only, Asia/Makassar. Must be in future on creation/schedule edits; end may exceed 16:00.; Min length: `5`; Max length: `5`; Pattern: `"^(?:(?:08\|09\|1[0-5]):[0-5][0-9]\|16:00)$"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceId` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `100`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

Example: [StandardStaffCreate](SCHEMAS.md#standardstaffcreate)

```json
{
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "visitKind": "standard",
  "idempotencyKey": "550e8400-e29b-41d4-a716-446655440000",
  "date": "2030-01-02",
  "time": "09:00"
}
```

Example: [RestrictedCreate](SCHEMAS.md#restrictedcreate)

```json
{
  "visitKind": "restricted",
  "idempotencyKey": "550e8400-e29b-41d4-a716-446655440000",
  "date": "2030-01-02",
  "time": "09:00"
}
```

**Business Logic and Side Effects**

Discriminated standard/restricted body, one route. Approved, duration60, optional room absent; lock assigned resources/check complete interval. Same scoped-key replay200, new201. Create/autoapprove audit and one logical notification atomically.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [StaffBooking](SCHEMAS.md#staffbooking) | Common response headers | Same-reference committed replay; no additional write/notification |
| `201` | application/json: [StaffBooking](SCHEMAS.md#staffbooking) | Common response headers | Complete authorized result |

**Response Fields**

[StaffBooking](SCHEMAS.md#staffbooking)

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: [StandardStaffBooking](SCHEMAS.md#standardstaffbooking)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `attachmentAvailable` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: [RestrictedReception](SCHEMAS.md#restrictedreception)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"reception"` | Yes | No wire default | Must equal `"reception"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 3: [RestrictedArrival](SCHEMAS.md#restrictedarrival)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"arrival"` | Yes | No wire default | Must equal `"arrival"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 4: [RestrictedAssignedPA](SCHEMAS.md#restrictedassignedpa)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"assigned-pa"` | Yes | No wire default | Must equal `"assigned-pa"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 5: [RestrictedFull](SCHEMAS.md#restrictedfull)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"full"` | Yes | No wire default | Must equal `"full"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | No; omit when absent | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `serviceId` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `100`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | No; omit when absent | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "approved",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "approved",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

HTTP `201` returns the same body shape/example above; status meaning is defined in the success table.

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:StaffCreate` → usecase `VisitUsecase.CreateStaffBooking` → repository operations `VisitRepository.Transaction`, `VisitTransaction.ResolveSubmission`, `VisitTransaction.LockResources`, `VisitTransaction.CheckOverlap`, `VisitTransaction.CreateVisit`, `VisitTransaction.AppendAudit`, `VisitTransaction.EnqueueNotifications`. Serializer: `VisitSerializer.ToStaffBooking`. DTOs: `StaffCreate` → `StaffBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-VISITS, FE-BOUNDS, FE-UPLOAD, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffApprove

**POST /v1/staff/bookings/{id}/approve — Approve pending visit**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer + approve + decision owner: PA assigned ordinary non-VIP, Protocol VIP ordinary or restricted. Current assignments/classification and Pending state rechecked.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F24](FRONTEND-TRACEABILITY.md#f24); /admin/booking; /admin/today | `BookingDetail` — Approve dialog: room dropdown, duration input, note fields and confirmation | Send version/roomId/duration and optional notes; replace detail with returned approved StaffBooking. Current form sends room label and requires coordinated conversion. | [frontendx/apps/admin/src/features/BookingDetail.tsx](../../frontendx/apps/admin/src/features/BookingDetail.tsx); Form sends room label and duration; target requires roomId |
| [F28](FRONTEND-TRACEABILITY.md#f28); staff create / approve / reschedule / restricted edit dialogs | `VisitNotesFields` — Shared Instruksi pengunjung and Catatan koordinasi internal textareas | Include optional note fields in approve; omission preserves, empty clears. | [frontendx/apps/admin/src/features/VisitNotesFields.tsx](../../frontendx/apps/admin/src/features/VisitNotesFields.tsx); Shared form fields; notes travel with existing command |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `application/json`; body required: **yes**; model: [Approve](SCHEMAS.md#approve).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `duration` | `integer` | No; omit when absent | No wire default | Min: `15`; Max: `240` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "version": 1
}
```

**Business Logic and Side Effects**

Pending->Approved; standard public request requires active roomId and duration15-240; absent required selection422. Notes omitted preserve/empty clears. Lock visit/resources then overlap check, version/audit/outbox atomically.

- Pending only. Ordinary public approval requires valid room/duration; restricted may omit.
- Omitted room preserves existing; omitted notes preserve, empty clears.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [StaffBooking](SCHEMAS.md#staffbooking) | Common response headers | Complete authorized result |

**Response Fields**

[StaffBooking](SCHEMAS.md#staffbooking)

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: [StandardStaffBooking](SCHEMAS.md#standardstaffbooking)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `attachmentAvailable` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: [RestrictedReception](SCHEMAS.md#restrictedreception)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"reception"` | Yes | No wire default | Must equal `"reception"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 3: [RestrictedArrival](SCHEMAS.md#restrictedarrival)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"arrival"` | Yes | No wire default | Must equal `"arrival"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 4: [RestrictedAssignedPA](SCHEMAS.md#restrictedassignedpa)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"assigned-pa"` | Yes | No wire default | Must equal `"assigned-pa"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 5: [RestrictedFull](SCHEMAS.md#restrictedfull)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"full"` | Yes | No wire default | Must equal `"full"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | No; omit when absent | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `serviceId` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `100`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | No; omit when absent | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "approved",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "approved",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:Approve` → usecase `VisitUsecase.Approve` → repository operations `VisitRepository.Transaction`, `VisitTransaction.LockVisit`, `VisitTransaction.LockResources`, `VisitTransaction.CheckOverlap`, `VisitTransaction.SaveVisit`, `VisitTransaction.AppendAudit`, `VisitTransaction.EnqueueNotifications`. Serializer: `VisitSerializer.ToStaffBooking`. DTOs: `Approve` → `StaffBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-VISITS, FE-BOUNDS, FE-ROOM, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffReject

**POST /v1/staff/bookings/{id}/reject — Reject pending visit**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer + reject and same PA/Protocol decision-owner rules; Pending only. Public reason separate from private note.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F25](FRONTEND-TRACEABILITY.md#f25); /admin/booking; /admin/today | `BookingDetail` — Reject dialog: public rejection-reason textarea, private note textarea and reject button | Send version/rejectionReason/optional internalRejectionNote; returned rejected booking refreshes visible public/private sections according to authority. | [frontendx/apps/admin/src/features/BookingDetail.tsx](../../frontendx/apps/admin/src/features/BookingDetail.tsx); rejectionSchema -> useBookingAction; real backend absent |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `application/json`; body required: **yes**; model: [Reject](SCHEMAS.md#reject).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rejectionReason` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "version": 1,
  "rejectionReason": "Requested appointment unavailable"
}
```

**Business Logic and Side Effects**

Pending->Rejected; trimmed reason5-1000, private note<=2000. Version/audit/outbox atomic; never publish private note.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [StaffBooking](SCHEMAS.md#staffbooking) | Common response headers | Complete authorized result |

**Response Fields**

[StaffBooking](SCHEMAS.md#staffbooking)

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: [StandardStaffBooking](SCHEMAS.md#standardstaffbooking)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `attachmentAvailable` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: [RestrictedReception](SCHEMAS.md#restrictedreception)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"reception"` | Yes | No wire default | Must equal `"reception"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 3: [RestrictedArrival](SCHEMAS.md#restrictedarrival)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"arrival"` | Yes | No wire default | Must equal `"arrival"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 4: [RestrictedAssignedPA](SCHEMAS.md#restrictedassignedpa)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"assigned-pa"` | Yes | No wire default | Must equal `"assigned-pa"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 5: [RestrictedFull](SCHEMAS.md#restrictedfull)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"full"` | Yes | No wire default | Must equal `"full"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | No; omit when absent | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `serviceId` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `100`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | No; omit when absent | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "rejected",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "rejected",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:Reject` → usecase `VisitUsecase.Reject` → repository operations `VisitRepository.Transaction`, `VisitTransaction.LockVisit`, `VisitTransaction.SaveVisit`, `VisitTransaction.AppendAudit`, `VisitTransaction.EnqueueNotifications`. Serializer: `VisitSerializer.ToStaffBooking`. DTOs: `Reject` → `StaffBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-VISITS, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffReschedule

**POST /v1/staff/bookings/{id}/reschedule — Reschedule Pending or Approved visit, excluding Missed**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer + reschedule and same PA/Protocol decision-owner rules. Pending or Approved excluding derived Missed; restricted remains Protocol.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F26](FRONTEND-TRACEABILITY.md#f26); /admin/booking; /admin/today | `BookingDetail` — Reschedule dialog: new date/time, room dropdown, duration, reason and notes | Send version/date/time/duration/optional roomId/reason/notes; returned schedule/version/timing refresh detail. Missing roomId clears the room under full schedule replacement. | [frontendx/apps/admin/src/features/BookingDetail.tsx](../../frontendx/apps/admin/src/features/BookingDetail.tsx); room label sent; domain.can permits Approved only while D100 also selects Pending |
| [F28](FRONTEND-TRACEABILITY.md#f28); staff create / approve / reschedule / restricted edit dialogs | `VisitNotesFields` — Shared Instruksi pengunjung and Catatan koordinasi internal textareas | Include optional note fields in reschedule; public/private projection remains distinct. | [frontendx/apps/admin/src/features/VisitNotesFields.tsx](../../frontendx/apps/admin/src/features/VisitNotesFields.tsx); Shared form fields; notes travel with existing command |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `application/json`; body required: **yes**; model: [Reschedule](SCHEMAS.md#reschedule).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | [StartTime](SCHEMAS.md#starttime) | Yes | No wire default | Start only, Asia/Makassar. Must be in future on creation/schedule edits; end may exceed 16:00.; Min length: `5`; Max length: `5`; Pattern: `"^(?:(?:08\|09\|1[0-5]):[0-5][0-9]\|16:00)$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `reason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "version": 1,
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60
}
```

**Business Logic and Side Effects**

Preserve lifecycle state; full replacement future WITA schedule/duration15-240. Standard room/reason required; restricted optional, supplied reason5-1000. Omitted roomId clears on reschedule; omitted notes preserve. Lock old/new resources, overlap check, revoke prior admission, audit/outbox atomically.

- D100 Pending or Approved excluding derived Missed; new start future. Coordinate frontend can()/action visibility, which currently only allows Approved/non-Missed.
- Ordinary requires room/reason, restricted may omit. Omitted room clears; omitted notes preserve and empty clears.
- Preserve lifecycle; Pending does not reserve; Approved uses full interval conflict locks. Revoke incompatible admission credentials, audit and enqueue notification atomically.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [StaffBooking](SCHEMAS.md#staffbooking) | Common response headers | Complete authorized result |

**Response Fields**

[StaffBooking](SCHEMAS.md#staffbooking)

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: [StandardStaffBooking](SCHEMAS.md#standardstaffbooking)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `attachmentAvailable` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: [RestrictedReception](SCHEMAS.md#restrictedreception)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"reception"` | Yes | No wire default | Must equal `"reception"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 3: [RestrictedArrival](SCHEMAS.md#restrictedarrival)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"arrival"` | Yes | No wire default | Must equal `"arrival"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 4: [RestrictedAssignedPA](SCHEMAS.md#restrictedassignedpa)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"assigned-pa"` | Yes | No wire default | Must equal `"assigned-pa"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 5: [RestrictedFull](SCHEMAS.md#restrictedfull)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"full"` | Yes | No wire default | Must equal `"full"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | No; omit when absent | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `serviceId` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `100`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | No; omit when absent | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "pending",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "pending",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:Reschedule` → usecase `VisitUsecase.Reschedule` → repository operations `VisitRepository.Transaction`, `VisitTransaction.LockVisit`, `VisitTransaction.LockResources`, `VisitTransaction.CheckOverlap`, `VisitTransaction.SaveVisit`, `VisitTransaction.RevokeAdmission`, `VisitTransaction.AppendAudit`, `VisitTransaction.EnqueueNotifications`. Serializer: `VisitSerializer.ToStaffBooking`. DTOs: `Reschedule` → `StaffBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-VISITS, FE-BOUNDS, FE-PENDING, FE-ROOM, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffRestrictedUpdate

**POST /v1/staff/bookings/{id}/update — Update restricted visit**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer Protocol + create + restricted full editable record, Pending/Approved. No invented update permission. Other role full read grants no edit.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F23](FRONTEND-TRACEABILITY.md#f23); /admin/booking?selected={id} | `RestrictedAppointment` — Restricted visit edit dialog and Simpan perubahan button | Send version/visitKind/date/time with optional identity/target/notes; omission preserves and empty values clear. Current schedule controls are read-only; schedule changes use reschedule. | [frontendx/apps/admin/src/features/RestrictedAppointment.tsx](../../frontendx/apps/admin/src/features/RestrictedAppointment.tsx); useBookingAction(update); date/time inputs readOnly on existing record |
| [F28](FRONTEND-TRACEABILITY.md#f28); staff create / approve / reschedule / restricted edit dialogs | `VisitNotesFields` — Shared Instruksi pengunjung and Catatan koordinasi internal textareas | Include optional note fields in restricted update; full read alone does not grant coordination-note authority. | [frontendx/apps/admin/src/features/VisitNotesFields.tsx](../../frontendx/apps/admin/src/features/VisitNotesFields.tsx); Shared form fields; notes travel with existing command |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `application/json`; body required: **yes**; model: [RestrictedUpdate](SCHEMAS.md#restrictedupdate).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | [StartTime](SCHEMAS.md#starttime) | Yes | No wire default | Start only, Asia/Makassar. Must be in future on creation/schedule edits; end may exceed 16:00.; Min length: `5`; Max length: `5`; Pattern: `"^(?:(?:08\|09\|1[0-5]):[0-5][0-9]\|16:00)$"` |
| `name` | one of `string`, `""` | No; omit when absent | No wire default | See linked model and business rules. |
| `organization` | one of `string`, `""` | No; omit when absent | No wire default | See linked model and business rules. |
| `serviceId` | one of `string`, `""` | No; omit when absent | No wire default | See linked model and business rules. |
| `officerId` | one of [BusinessId](SCHEMAS.md#businessid), `""` | No; omit when absent | No wire default | See linked model and business rules. |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "version": 1,
  "visitKind": "restricted",
  "date": "2030-01-02",
  "time": "09:00"
}
```

**Business Logic and Side Effects**

Date/time retained by current editor; changed schedules use reschedule. Omitted optional identity/target fields preserve, empty clears; notes omitted preserve/empty clears. Recompute scope and resource conflicts under locks; retain historical snapshot/audit without hard-deleting identity.

- Protocol create authority; restricted Pending/Approved; recheck future submitted date/time, scope and conflicts. This remains separate from D100 Reschedule availability.
- Omitted identity/target/notes preserve; empty clears. No ordinary personal data.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [RestrictedFull](SCHEMAS.md#restrictedfull) | Common response headers | Complete authorized result |

**Response Fields**

[RestrictedFull](SCHEMAS.md#restrictedfull)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"full"` | Yes | No wire default | Must equal `"full"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | No; omit when absent | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `serviceId` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `100`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | No; omit when absent | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "restricted",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "pending",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "pending",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "reference": "TRB-1",
  "restrictedView": "full"
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:UpdateRestricted` → usecase `VisitUsecase.UpdateRestricted` → repository operations `VisitRepository.Transaction`, `VisitTransaction.LockVisit`, `VisitTransaction.LockResources`, `VisitTransaction.CheckOverlap`, `VisitTransaction.SaveVisit`, `VisitTransaction.AppendAudit`. Serializer: `VisitSerializer.ToRestrictedFull`. DTOs: `RestrictedUpdate` → `RestrictedFull`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-VISITS, FE-BOUNDS, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffCancel

**POST /v1/staff/bookings/{id}/cancel — Cancel visit before check-in**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer + cancel + same PA/Protocol decision scope; Pending/Approved and never checked in. May be after schedule start/end.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Pending/Approved before any check-in; authorized PA/Protocol decision scope plus cancel; may be after start/end.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F27](FRONTEND-TRACEABILITY.md#f27); /admin/booking; /admin/today | `BookingDetail` — Staff cancellation dialog: public reason, internal cancellation note and confirm button | Send version/reason/optional internalCancellationNote; use returned cancelled booking and only authorized private-note fields. | [frontendx/apps/admin/src/features/BookingDetail.tsx](../../frontendx/apps/admin/src/features/BookingDetail.tsx); useBookingAction(cancel); separate from public pair authorization |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `application/json`; body required: **yes**; model: [StaffCancel](SCHEMAS.md#staffcancel).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `reason` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "version": 1,
  "reason": "Schedule changed"
}
```

**Business Logic and Side Effects**

Pending/Approved->Cancelled, reason5-1000/private note<=2000. Version/commitment release/revoke QR/audit/outbox atomically. No implicit authority from elapsed time.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [StaffBooking](SCHEMAS.md#staffbooking) | Common response headers | Complete authorized result |

**Response Fields**

[StaffBooking](SCHEMAS.md#staffbooking)

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: [StandardStaffBooking](SCHEMAS.md#standardstaffbooking)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `attachmentAvailable` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: [RestrictedReception](SCHEMAS.md#restrictedreception)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"reception"` | Yes | No wire default | Must equal `"reception"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 3: [RestrictedArrival](SCHEMAS.md#restrictedarrival)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"arrival"` | Yes | No wire default | Must equal `"arrival"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 4: [RestrictedAssignedPA](SCHEMAS.md#restrictedassignedpa)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"assigned-pa"` | Yes | No wire default | Must equal `"assigned-pa"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 5: [RestrictedFull](SCHEMAS.md#restrictedfull)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"full"` | Yes | No wire default | Must equal `"full"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |
| `organization` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | No; omit when absent | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `serviceId` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `100`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | No; omit when absent | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "cancelled",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "cancelled",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/visit_handler.go:StaffCancel` → usecase `VisitUsecase.CancelStaffBooking` → repository operations `VisitRepository.Transaction`, `VisitTransaction.LockVisit`, `VisitTransaction.SaveVisit`, `VisitTransaction.RevokeAdmission`, `VisitTransaction.AppendAudit`, `VisitTransaction.EnqueueNotifications`. Serializer: `VisitSerializer.ToStaffBooking`. DTOs: `StaffCancel` → `StaffBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-VISITS, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

### Reception and attendance

**Daftar Endpoint**

| Endpoint | Method | Frontend feature / control | Success response | Error statuses |
| --- | --- | --- | --- | --- |
| [`/v1/staff/resolve`](#staffresolve) | `POST` | Manual code and camera-QR confirmation dialog | `200` [ArrivalBooking](SCHEMAS.md#arrivalbooking) | 400, 401, 403, 404, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/staff/bookings/{id}/checkin`](#staffcheckin) | `POST` | Identity-confirmed check-in button | `200` [ArrivalBooking](SCHEMAS.md#arrivalbooking) | 400, 401, 403, 404, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/staff/bookings/{id}/checkout`](#staffcheckout) | `POST` | Identity-confirmed check-out button | `200` [ArrivalBooking](SCHEMAS.md#arrivalbooking) | 400, 401, 403, 404, 409, 413, 415, 422, 429, 500, 503 |

---

#### staffResolve

**POST /v1/staff/resolve — Resolve arrival or checkout credential**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer Receptionist with attendance permission appropriate to current record eligibility; credential resolution returns only arrival projection; no attendance transition.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** No attendance side effect. Status-specific QR validation: D96 consumed/nonrevoked QR resolves its checked-in visit after admission expiry. Restricted arrival projection never enters ordinary reception cache.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F31](FRONTEND-TRACEABILITY.md#f31); /admin/check-in; /admin/check-out | `Reception` — Manual visit-code form or QR resolve result before attendance confirmation | Send credential; authorized ArrivalBooking fields populate the identity/visit confirmation dialog. Resolve never changes attendance. | [frontendx/apps/admin/src/pages/Reception.tsx](../../frontendx/apps/admin/src/pages/Reception.tsx); repository.resolve; no attendance side effect; restricted result kept local |
| [F32](FRONTEND-TRACEABILITY.md#f32); /admin/check-in; /admin/check-out | `Scanner` — Camera scanner preview and decoded-QR handler | Send only BarcodeDetector raw string as credential to resolve; returned booking opens confirmation. Camera frames stay local. | [frontendx/apps/admin/src/features/Scanner.tsx](../../frontendx/apps/admin/src/features/Scanner.tsx); Camera local only; no image upload/storage/API |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `application/json`; body required: **yes**; model: [Resolve](SCHEMAS.md#resolve).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `credential` | [Credential](SCHEMAS.md#credential) | Yes | No wire default | Manual eight-digit code or opaque QR. Admission and checkout have different eligibility; D96 permits the QR consumed on successful check-in for checkout after admission expiry, unless revoked.; Min length: `1`; Max length: `4096` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "credential": "01234567"
}
```

**Business Logic and Side Effects**

Read current credential/visit eligibility and restricted arrival projection; no consume/checkin/checkout. Do not put expanded restricted arrival data in ordinary reception cache.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [ArrivalBooking](SCHEMAS.md#arrivalbooking) | Common response headers | Complete authorized result |

**Response Fields**

[ArrivalBooking](SCHEMAS.md#arrivalbooking)

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: [StandardStaffBooking](SCHEMAS.md#standardstaffbooking)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `attachmentAvailable` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: [RestrictedArrival](SCHEMAS.md#restrictedarrival)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"arrival"` | Yes | No wire default | Must equal `"arrival"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "pending",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "pending",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/reception_handler.go:Resolve` → usecase `ReceptionUsecase.Resolve` → repository operations `ReceptionRepository.ResolveCredential`. Serializer: `ReceptionSerializer.ToArrivalBooking`. DTOs: `Resolve` → `ArrivalBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-ATTENDANCE, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffCheckin

**POST /v1/staff/bookings/{id}/checkin — Confirm check-in**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer Receptionist + checkin + matching manual/QR credential and record/version; Approved in server-time admission window; explicit identity confirmation in UI.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Approved only; [start-30m,end] inclusive. Validate matching credential/version/current authority under locks; atomically consume admission QR when used. UI confirms identity.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F33](FRONTEND-TRACEABILITY.md#f33); /admin/check-in | `Reception` — Reception identity-confirmation checkbox and check-in confirmation button | Send resolved id, matching credential and version; returned checked_in ArrivalBooking refreshes attendance. Checkbox stays local; no identityVerified wire field. | [frontendx/apps/admin/src/pages/Reception.tsx](../../frontendx/apps/admin/src/pages/Reception.tsx); useBookingAction(checkin); checkbox local; no identityVerified wire field |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `application/json`; body required: **yes**; model: [Attendance](SCHEMAS.md#attendance).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `credential` | [Credential](SCHEMAS.md#credential) | Yes | No wire default | Manual eight-digit code or opaque QR. Admission and checkout have different eligibility; D96 permits the QR consumed on successful check-in for checkout after admission expiry, unless revoked.; Min length: `1`; Max length: `4096` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "credential": "01234567",
  "version": 1
}
```

**Business Logic and Side Effects**

Approved->checked_in once in [start-30min,end] inclusive. Lock visit/credential, validate current version, consume QR if used, append attendance/audit atomically. Manual code alternative.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [ArrivalBooking](SCHEMAS.md#arrivalbooking) | Common response headers | Complete authorized result |

**Response Fields**

[ArrivalBooking](SCHEMAS.md#arrivalbooking)

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: [StandardStaffBooking](SCHEMAS.md#standardstaffbooking)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `attachmentAvailable` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: [RestrictedArrival](SCHEMAS.md#restrictedarrival)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"arrival"` | Yes | No wire default | Must equal `"arrival"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "checked_in",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "checked_in",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/reception_handler.go:Checkin` → usecase `ReceptionUsecase.Checkin` → repository operations `VisitRepository.Transaction`, `VisitTransaction.LockVisit`, `VisitTransaction.ConfirmAttendance`, `VisitTransaction.AppendAudit`. Serializer: `ReceptionSerializer.ToArrivalBooking`. DTOs: `Attendance` → `ArrivalBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-ATTENDANCE, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffCheckout

**POST /v1/staff/bookings/{id}/checkout — Confirm checkout**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer Receptionist + checkout + matching credential/version; Checked-in even on later dates. Consumed nonrevoked admission QR remains usable only for that checkout.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Checked-in only, late checkout supported. D96 allows successful-checkin consumed QR after admission expiry, unless revoked; manual code alternative. Atomically append one checkout.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F34](FRONTEND-TRACEABILITY.md#f34); /admin/check-out | `Reception` — Reception identity-confirmation checkbox and check-out confirmation button | Send resolved id/credential/version; returned checked_out ArrivalBooking removes the visit from still-inside results; late/prior-date checkout is supported. | [frontendx/apps/admin/src/pages/Reception.tsx](../../frontendx/apps/admin/src/pages/Reception.tsx); useBookingAction(checkout); consumed nonrevoked QR usable per D96 |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `application/json`; body required: **yes**; model: [Attendance](SCHEMAS.md#attendance).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `credential` | [Credential](SCHEMAS.md#credential) | Yes | No wire default | Manual eight-digit code or opaque QR. Admission and checkout have different eligibility; D96 permits the QR consumed on successful check-in for checkout after admission expiry, unless revoked.; Min length: `1`; Max length: `4096` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "credential": "01234567",
  "version": 1
}
```

**Business Logic and Side Effects**

checked_in->checked_out once, including late/next-day checkout. Matching consumed nonrevoked QR or manual code; no duplicate attendance. Append event/audit and release commitment atomically.


**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [ArrivalBooking](SCHEMAS.md#arrivalbooking) | Common response headers | Complete authorized result |

**Response Fields**

[ArrivalBooking](SCHEMAS.md#arrivalbooking)

Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.

**Variant 1: [StandardStaffBooking](SCHEMAS.md#standardstaffbooking)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"standard"` | Yes | No wire default | Must equal `"standard"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `code` | [BookingCode](SCHEMAS.md#bookingcode) | Yes | No wire default | Min length: `8`; Max length: `8`; Pattern: `"^[0-9]{8}$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `organization` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `phone` | `string` | Yes | No wire default | Min length: `8`; Max length: `20`; Pattern: `"^[0-9]{8,20}$"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `vip` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `purpose` | `string` | Yes | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitors` | array of `string` | Yes | No wire default | Min items: `1`; Max items: `20` |
| `identityType` | `string` | Yes | No wire default | Allowed: `"KTP"`, `"Paspor"`, `"SIM"` |
| `ndName` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `visitorInstructions` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `rejectionReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `cancellationReason` | `string` | No; omit when absent | No wire default | Min length: `5`; Max length: `1000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `coordinationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalRejectionNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `internalCancellationNote` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `2000`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy` | `object` | No; omit when absent | No wire default | Additional keys: `false` |
| `createdBy.name` | `string` | Yes, when parent present | - | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `createdBy.role` | [Role](SCHEMAS.md#role) | Yes, when parent present | - | Allowed: `"PA"`, `"Protocol Officer"`, `"Receptionist"`, `"Super Admin"` |
| `attachmentAvailable` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `history` | array of [StaffEvent](SCHEMAS.md#staffevent) | Yes | No wire default | See linked model and business rules. |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Variant 2: [RestrictedArrival](SCHEMAS.md#restrictedarrival)**

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `visitKind` | `"restricted"` | Yes | No wire default | Must equal `"restricted"` |
| `date` | [Date](SCHEMAS.md#date) | Yes | No wire default | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `time` | `string` | Yes | No wire default | Min length: `5`; Max length: `5`; Pattern: `"^(?:[01][0-9]\|2[0-3]):[0-5][0-9]$"` |
| `duration` | `integer` | Yes | No wire default | Min: `15`; Max: `240` |
| `roomId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `room` | `string` | Yes | No wire default | Min length: `0`; Max length: `255` |
| `status` | [Status](SCHEMAS.md#status) | Yes | No wire default | Allowed: `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `rescheduled` | `boolean` | No; omit when absent | No wire default | See linked model and business rules. |
| `timing` | [VisitTiming](SCHEMAS.md#visittiming) | Yes | No wire default | Additional keys: `false` |
| `reference` | `string` | Yes | No wire default | Min length: `1` |
| `restrictedView` | `"arrival"` | Yes | No wire default | Must equal `"arrival"` |
| `name` | `string` | No; omit when absent | No wire default | Min length: `2`; Max length: `120`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `officerId` | [BusinessId](SCHEMAS.md#businessid) | No; omit when absent | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `host` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `serviceIdLast4` | `string` | No; omit when absent | No wire default | Min length: `1`; Max length: `4` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "visitKind": "standard",
  "date": "2030-01-02",
  "time": "09:00",
  "duration": 60,
  "room": "",
  "status": "checked_out",
  "version": 1,
  "timing": {
    "serverTime": 1893542400000,
    "opensAt": 1893544200000,
    "endsAt": 1893549600000,
    "displayStatus": "checked_out",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "code": "01234567",
  "name": "Visitor Example",
  "organization": "Example Office",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "1",
  "host": "Official Example",
  "vip": false,
  "purpose": "Discuss visitor services",
  "visitors": [
    "Visitor Example"
  ],
  "identityType": "KTP",
  "history": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/reception_handler.go:Checkout` → usecase `ReceptionUsecase.Checkout` → repository operations `VisitRepository.Transaction`, `VisitTransaction.LockVisit`, `VisitTransaction.ConfirmAttendance`, `VisitTransaction.AppendAudit`. Serializer: `ReceptionSerializer.ToArrivalBooking`. DTOs: `Attendance` → `ArrivalBooking`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-ATTENDANCE, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

### Private attachments

**Daftar Endpoint**

| Endpoint | Method | Frontend feature / control | Success response | Error statuses |
| --- | --- | --- | --- | --- |
| [`/v1/public/attachments`](#publicupload) | `POST` | Public optional supporting-document picker | `200` [UploadResult](SCHEMAS.md#uploadresult); `201` [UploadResult](SCHEMAS.md#uploadresult) | 400, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/staff/attachments`](#staffupload) | `POST` | Staff standard-appointment document picker | `200` [UploadResult](SCHEMAS.md#uploadresult); `201` [UploadResult](SCHEMAS.md#uploadresult) | 400, 401, 403, 409, 413, 415, 422, 429, 500, 503 |
| [`/v1/staff/bookings/{id}/attachment`](#staffattachment) | `GET` | Private document view/download in booking detail | `200` `string`, `string`, `string` | 400, 401, 403, 404, 409, 413, 422, 429, 500, 503 |

---

#### publicUpload

**POST /v1/public/attachments — Upload optional private attachment**

**Auth Required:** No staff login; endpoint-specific public proof still applies where stated.

**Authorization / resource scope:** Anonymous audience-owned upload reference/content, only when capability enabled; new files60/hour/trusted IP shared with staff plus overall public admission.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Draft capability only; safe reference/content replay; one file. Token1h; unlinked cleanup24h. New upload allowance60/hour/IP shared across audiences; stage/cleanup bounded.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F02](FRONTEND-TRACEABILITY.md#f02); /; /booking | `BookingForm` — Three-step booking form, companion-name rows, optional supporting-document picker and submit/receipt panel | Send the selected file and UUID reference; retain UploadResult.token as attachmentToken and check expiresAt/serverTime. New uploads require attachmentsEnabled=true. | [frontendx/apps/visitor/src/pages/Booking.tsx](../../frontendx/apps/visitor/src/pages/Booking.tsx); BookingSubmission; actual create adapter exists; no backend booking route |
| [F45](FRONTEND-TRACEABILITY.md#f45); public/staff submission forms | `uploadAttachment` — Shared file-picker upload/retry transport for public and staff forms | Send one file and stable per-File/audience UUID reference; token/expiresAt/serverTime feed the public create attachmentToken field. | [frontendx/packages/data/src/visit-files.ts](../../frontendx/packages/data/src/visit-files.ts); WeakMap reuses reference for same File/audience; optional upload still attempted without capability check |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

No staff authentication header/cookie is required. Public pair/reference proof, when applicable, belongs in the request body.

**Request Body**

Media type: `multipart/form-data`; body required: **yes**; model: [Upload](SCHEMAS.md#upload).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `file` | `string` | Yes | No wire default | One nonempty verified PDF/JPEG/PNG <=5242880 bytes.; Format: `"binary"`; Max bytes: `5242880` |
| `reference` | [Reference](SCHEMAS.md#reference) | Yes | No wire default | UUID request/submission/upload reference, distinct from a business ID.; Format: `"uuid"`; Min length: `36`; Max length: `36` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

Send actual file bytes in the `file` part and a UUID string in `reference`. The file is not a JSON/base64 string. One file only; file limit 5,242,880 bytes; multipart limit 6,291,456 bytes; verified PDF/JPEG/PNG.

**Business Logic and Side Effects**

Multipart exactly file+reference, one nonempty PDF/JPEG/PNG <=5242880 bytes within6291456 total. Verify signature/content/ownership; no public URL. Token1h, unlinked cleanup24h; reference+same content eligible replay200, changed content409.

- Same reference plus same content/audience/owner replays only an eligible accepted upload. Changed content/reference binding conflicts.
- One nonempty verified PDF/JPEG/PNG; unsupported extra multipart fields/parts rejected.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [UploadResult](SCHEMAS.md#uploadresult) | Common response headers | Same-reference committed replay; no additional write/notification |
| `201` | application/json: [UploadResult](SCHEMAS.md#uploadresult) | Common response headers | Complete authorized result |

**Response Fields**

[UploadResult](SCHEMAS.md#uploadresult)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `token` | `string` | Yes | No wire default | Min length: `16`; Max length: `4096` |
| `expiresAt` | `string` | Yes | No wire default | Format: `"date-time"` |
| `serverTime` | `integer` | Yes | No wire default | Min: `0`; Max: `9007199254740991` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "token": "vmsu1.bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "expiresAt": "2030-01-02T01:00:00Z",
  "serverTime": 1893542400000
}
```

HTTP `201` returns the same body shape/example above; status meaning is defined in the success table.

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/document_handler.go:PublicUpload` → usecase `DocumentUsecase.UploadPublic` → repository operations `DocumentRepository.StageUpload`. Serializer: `DocumentSerializer.ToUploadResult`. DTOs: `Upload` → `UploadResult`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-DOCUMENTS, BE-VISITS, FE-BOUNDS, FE-UPLOAD. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffUpload

**POST /v1/staff/attachments — Upload authorized staff attachment**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer plus fixed create permission and current standard-visit creation scope. Upload ownership uses employee ID/audience, never a browser role or stored login session.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F21](FRONTEND-TRACEABILITY.md#f21); /admin/booking?new=1; /admin/today?new=1 | `CreateAppointment` — Staff standard-appointment dialog: official dropdown, schedule, representative/roster, note fields, document picker and submit | Upload file/reference and use token as attachmentToken on the create command; keep private bytes outside JSON. | [frontendx/apps/admin/src/features/CreateAppointment.tsx](../../frontendx/apps/admin/src/features/CreateAppointment.tsx); useCreateAppointment; UUID retained while payload unchanged; room/duration not collected |
| [F45](FRONTEND-TRACEABILITY.md#f45); public/staff submission forms | `uploadAttachment` — Shared file-picker upload/retry transport for public and staff forms | Use the same multipart/result shape with Bearer/current create authority; token feeds staff standard creation. | [frontendx/packages/data/src/visit-files.ts](../../frontendx/packages/data/src/visit-files.ts); WeakMap reuses reference for same File/audience; optional upload still attempted without capability check |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `multipart/form-data`; body required: **yes**; model: [Upload](SCHEMAS.md#upload).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `file` | `string` | Yes | No wire default | One nonempty verified PDF/JPEG/PNG <=5242880 bytes.; Format: `"binary"`; Max bytes: `5242880` |
| `reference` | [Reference](SCHEMAS.md#reference) | Yes | No wire default | UUID request/submission/upload reference, distinct from a business ID.; Format: `"uuid"`; Min length: `36`; Max length: `36` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

Send actual file bytes in the `file` part and a UUID string in `reference`. The file is not a JSON/base64 string. One file only; file limit 5,242,880 bytes; multipart limit 6,291,456 bytes; verified PDF/JPEG/PNG.

**Business Logic and Side Effects**

Same verified file/replay/expiry rules as public upload, distinct owner/audience scope. Disabled feature fails safely and optional booking continues without document.

- Same reference plus same content/audience/owner replays only an eligible accepted upload. Changed content/reference binding conflicts.
- One nonempty verified PDF/JPEG/PNG; unsupported extra multipart fields/parts rejected.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [UploadResult](SCHEMAS.md#uploadresult) | Common response headers | Same-reference committed replay; no additional write/notification |
| `201` | application/json: [UploadResult](SCHEMAS.md#uploadresult) | Common response headers | Complete authorized result |

**Response Fields**

[UploadResult](SCHEMAS.md#uploadresult)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `token` | `string` | Yes | No wire default | Min length: `16`; Max length: `4096` |
| `expiresAt` | `string` | Yes | No wire default | Format: `"date-time"` |
| `serverTime` | `integer` | Yes | No wire default | Min: `0`; Max: `9007199254740991` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "token": "vmsu1.bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "expiresAt": "2030-01-02T01:00:00Z",
  "serverTime": 1893542400000
}
```

HTTP `201` returns the same body shape/example above; status meaning is defined in the success table.

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/document_handler.go:StaffUpload` → usecase `DocumentUsecase.UploadStaff` → repository operations `DocumentRepository.StageUpload`. Serializer: `DocumentSerializer.ToUploadResult`. DTOs: `Upload` → `UploadResult`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-DOCUMENTS, BE-VISITS, FE-UPLOAD, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffAttachment

**GET /v1/staff/bookings/{id}/attachment — View or download private attachment**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer + bookings permission + ordinary visit and attachment field scope. Authorized assigned PA or eligible Protocol; no public/restricted/reception or incidental Super Admin document access. View/download same authority.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Default inline. Same private field authority for view/download; audit access. No public path/token. Verified MIME, safe filename, no-store and nosniff.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F29](FRONTEND-TRACEABILITY.md#f29); /admin/booking?selected={id} | `AttachmentAccess` — Private supporting-document action in BookingDetail; existing action downloads only | Send path id and disposition=inline for view or attachment for download; MIME/Content-Disposition name the returned bytes. Current download helper omits explicit disposition=attachment. | [frontendx/apps/admin/src/features/AttachmentAccess.tsx](../../frontendx/apps/admin/src/features/AttachmentAccess.tsx); downloadAttachment only; request lacks explicit disposition=attachment |

**Query Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `disposition` | `string` | No | `"inline"` | Allowed: `"inline"`, `"attachment"` |

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

Protected binary PDF/JPEG/PNG; disposition defaults inline, explicit attachment download. MIME verified; safe filename/no-store/nosniff; audit access. No public file/object path.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/pdf: `string`; image/jpeg: `string`; image/png: `string` | Content-Disposition, Cache-Control, X-Content-Type-Options | Authorized verified private bytes |

**Response Fields**

No JSON success fields. Private file success returns verified bytes with Content-Disposition, Cache-Control: no-store and X-Content-Type-Options: nosniff.

**Response (200) — synthetic example**

Binary bytes of the verified MIME type; not a JSON object.

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/document_handler.go:StaffAttachment` → usecase `DocumentUsecase.GetAttachment` → repository operations `DocumentRepository.GetAttachment`, `DocumentRepository.AuditAccess`. Serializer: `(no JSON projection)`. DTOs: `AttachmentQuery` → `PrivateFile`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-DOCUMENTS, FE-FILEVIEW, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

### Employee directory (read-only)

**Daftar Endpoint**

| Endpoint | Method | Frontend feature / control | Success response | Error statuses |
| --- | --- | --- | --- | --- |
| [`/v1/staff/employees`](#staffemployees) | `GET` | Employee search, cards/table, counts and pagination | `200` [EmployeePage](SCHEMAS.md#employeepage) | 400, 401, 403, 409, 413, 422, 429, 500, 503 |
| [`/v1/staff/employees/{id}`](#staffemployee) | `GET` | Employee edit-conflict reload | `200` [Employee](SCHEMAS.md#employee) | 400, 401, 403, 404, 409, 413, 429, 500, 503 |

---

#### staffEmployees

**GET /v1/staff/employees — Read employee page**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer plus current Super Admin role and fixed people permission; read-only authorized directory. No role/password/workspace session gates.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** People permission; default page1/pageSize25; metadata present on empty pages.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F20](FRONTEND-TRACEABILITY.md#f20); /admin/* | `DataRefresh` — DataRefresh freshness/offline/error notice and retry controls | Refetch employee page; no separate freshness endpoint. | [frontendx/apps/admin/src/components/DataRefresh.tsx](../../frontendx/apps/admin/src/components/DataRefresh.tsx); Pure query control; no persisted business data |
| [F35](FRONTEND-TRACEABILITY.md#f35); /admin/employees | `EmployeeDirectory` — Read-only employee directory search, cards/table, pagination and detail; editing controls deferred | Send q/page/pageSize=25; EmployeePage.items/total/totalAll/page/pageSize populate the read-only directory. Password policy metadata and create/edit actions are deferred. | [frontendx/apps/admin/src/features/EmployeeDirectory.tsx](../../frontendx/apps/admin/src/features/EmployeeDirectory.tsx); management.employees; local preview array; no backend admin routes |
| [F43](FRONTEND-TRACEABILITY.md#f43); /admin/analytics; /admin/logs; /admin/employees; /admin/roles; /admin/integration | `Settings` — Settings shell mounting read-only employees, integrations and reports; roles/editor routes deferred | Delegate to employee directory; no additional shell response. | [frontendx/apps/admin/src/pages/Settings.tsx](../../frontendx/apps/admin/src/pages/Settings.tsx); Lazy router; guard permissions in main.tsx |

**Query Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `q` | `string` | No | None declared | Min length: `0`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `page` | `integer` | No | `1` | Min: `1`; Max: `2147483647` |
| `pageSize` | `integer` | No | `25` | Min: `1`; Max: `250` |

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

Read-only directory: q trimmed max255, case-insensitive name/email/assigned role-label match; page1/pageSize25 max250; stable name then numericID tie subject to OPEN-READ-ORDER. total filtered; totalAll before search. No password metadata or create/edit/role/assignment controls in Phase 1.

Unresolved boundary **OPEN-READ-ORDER**: Selected rules require stable sorting but do not freeze database text collation/normalization or ID tie comparison. Frontend preview uses Indonesian localeCompare and lexicographic ID ties; employee target says numeric ID ties. These are different with IDs 2 and 10 or accented names. See [CONTRACT-ISSUES](CONTRACT-ISSUES.md).

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [EmployeePage](SCHEMAS.md#employeepage) | Common response headers | Complete authorized result |

**Response Fields**

[EmployeePage](SCHEMAS.md#employeepage)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `items` | array of [Employee](SCHEMAS.md#employee) | Yes | No wire default | Min items: `0`; Max items: `250` |
| `total` | `integer` | Yes | No wire default | Min: `0`; Max: `9007199254740991` |
| `totalAll` | `integer` | Yes | No wire default | Min: `0`; Max: `9007199254740991` |
| `page` | `integer` | Yes | No wire default | Min: `1`; Max: `2147483647` |
| `pageSize` | `integer` | Yes | No wire default | Min: `1`; Max: `250` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "items": [],
  "total": 0,
  "totalAll": 0,
  "page": 1,
  "pageSize": 25
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/management_handler.go:Employees` → usecase `ManagementUsecase.GetEmployees` → repository operations `ManagementRepository.GetEmployees`. Serializer: `ManagementSerializer.ToEmployeePage`. DTOs: `EmployeeQuery` → `EmployeePage`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-ADMIN, BE-READS, BE-PHASE1-AUTH, FE-DIRECTORY. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### staffEmployee

**GET /v1/staff/employees/{id} — Read employee**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer Super Admin + people; current resource authority; no credential/hash response.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F35](FRONTEND-TRACEABILITY.md#f35); /admin/employees | `EmployeeDirectory` — Read-only employee directory search, cards/table, pagination and detail; editing controls deferred | Read the employee id/profile/single-role array/assignments/version for an authorized read-only detail. Existing edit-conflict reload is a deferred use; no account writes in Phase 1. | [frontendx/apps/admin/src/features/EmployeeDirectory.tsx](../../frontendx/apps/admin/src/features/EmployeeDirectory.tsx); management.employees; local preview array; no backend admin routes |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

Return one authorized read-only Employee with exactly one roles entry, current profile/active flag/PA assignments and version. No password/hash or editable policy graph; no write operation is enabled by this read.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [Employee](SCHEMAS.md#employee) | Common response headers | Complete authorized result |

**Response Fields**

[Employee](SCHEMAS.md#employee)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `name` | `string` | Yes | No wire default | Min length: `2`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `email` | [Email](SCHEMAS.md#email) | Yes | No wire default | One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.; Format: `"email"`; Min length: `3`; Max length: `254`; Pattern: `"^[\\x21-\\x7e]+$"`; Max UTF-8 bytes: `254` |
| `roles` | array of [Role](SCHEMAS.md#role) | Yes | No wire default | Min items: `1`; Max items: `1`; Unique items: `true` |
| `active` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `assignedOfficerIds` | array of [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | At most one current official assignment per employee; historical inactive rows do not add scope.; Min items: `0`; Max items: `1`; Unique items: `true` |
| `jobTitle` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `deputy` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `directorate` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `phone` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `nip` | `string` | No; omit when absent | No wire default | Min length: `0`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "name": "Visitor Example",
  "email": "visitor@example.test",
  "roles": [
    "PA"
  ],
  "active": true,
  "assignedOfficerIds": [],
  "version": 1
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/management_handler.go:Employee` → usecase `ManagementUsecase.GetEmployee` → repository operations `ManagementRepository.GetEmployee`. Serializer: `ManagementSerializer.ToEmployee`. DTOs: `(no request DTO)` → `Employee`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-ADMIN, BE-PHASE1-AUTH, FE-DIRECTORY. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

### Integrations

**Daftar Endpoint**

| Endpoint | Method | Frontend feature / control | Success response | Error statuses |
| --- | --- | --- | --- | --- |
| [`/v1/staff/integrations`](#staffintegrations) | `GET` | Integration cards and safe dynamic-field metadata | `200` array of [Integration](SCHEMAS.md#integration) | 400, 401, 403, 409, 413, 429, 500, 503 |
| [`/v1/staff/integrations/{id}`](#patchintegration) | `PATCH` | Provider settings and secret keep/clear save form | `200` [Integration](SCHEMAS.md#integration) | 400, 401, 403, 404, 409, 413, 415, 422, 429, 500, 503 |

---

#### staffIntegrations

**GET /v1/staff/integrations — Read safe integration metadata**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer Super Admin + integrations; safe metadata/configured flags only, never secret values or private paths.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F20](FRONTEND-TRACEABILITY.md#f20); /admin/* | `DataRefresh` — DataRefresh freshness/offline/error notice and retry controls | Refetch safe integration metadata; no separate freshness endpoint. | [frontendx/apps/admin/src/components/DataRefresh.tsx](../../frontendx/apps/admin/src/components/DataRefresh.tsx); Pure query control; no persisted business data |
| [F40](FRONTEND-TRACEABILITY.md#f40); /admin/integration | `IntegrationManagement / IntegrationEditor` — Integration service cards, Konfigurasi dialog, enabled checkbox, dynamic settings fields and secret keep/clear controls | Integration.name/description/health/enabled/fields populate cards and editor; secret fields expose configured flags, never values. | [frontendx/apps/admin/src/features/IntegrationManagement.tsx](../../frontendx/apps/admin/src/features/IntegrationManagement.tsx); Generic live form uses server metadata; preview endpoint/bucket/credential examples are not selected providers |
| [F43](FRONTEND-TRACEABILITY.md#f43); /admin/analytics; /admin/logs; /admin/employees; /admin/roles; /admin/integration | `Settings` — Settings shell mounting read-only employees, integrations and reports; roles/editor routes deferred | Delegate to integration cards/editor; no additional shell response. | [frontendx/apps/admin/src/pages/Settings.tsx](../../frontendx/apps/admin/src/pages/Settings.tsx); Lazy router; guard permissions in main.tsx |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

Bounded SMTP metadata fields; secret configured boolean only. health from stored worker/service state, not synchronous network probe each GET; deferred storage disabled.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: array of [Integration](SCHEMAS.md#integration) | Common response headers | Complete authorized result |

**Response Fields**

array of [Integration](SCHEMAS.md#integration)

Array items: [Integration](SCHEMAS.md#integration). See linked model and business rules.

**Response (200) — synthetic example**

```json
[]
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/management_handler.go:Integrations` → usecase `ManagementUsecase.GetIntegrations` → repository operations `ManagementRepository.GetIntegrations`. Serializer: `ManagementSerializer.ToIntegration`. DTOs: `(no request DTO)` → `[]Integration`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-ADMIN, BE-INTEGRATIONS, BE-READS, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### patchIntegration

**PATCH /v1/staff/integrations/{id} — Update allowlisted provider settings**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer Super Admin + integrations; versioned allowlisted SMTP dictionary only; validate merged effective settings/TLS/outbound policy. Disabled storage metadata cannot acquire arbitrary writable fields.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F40](FRONTEND-TRACEABILITY.md#f40); /admin/integration | `IntegrationManagement / IntegrationEditor` — Integration service cards, Konfigurasi dialog, enabled checkbox, dynamic settings fields and secret keep/clear controls | Send enabled/values/clearSecrets/version for path id; returned safe Integration refreshes card/health/descriptors without echoing secrets. | [frontendx/apps/admin/src/features/IntegrationManagement.tsx](../../frontendx/apps/admin/src/features/IntegrationManagement.tsx); Generic live form uses server metadata; preview endpoint/bucket/credential examples are not selected providers |

**Query Parameters**

None. Query keys are not accepted on this operation.

**Path Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | None declared | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

Media type: `application/json`; body required: **yes**; model: [IntegrationPatch](SCHEMAS.md#integrationpatch).

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `enabled` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `values` | [SmtpValues](SCHEMAS.md#smtpvalues) | Yes | No wire default | SMTP-only patch values; omitted/blank configured password keeps existing. Validate merged effective configuration when enabled, not this partial object alone.; Additional keys: `false` |
| `clearSecrets` | array of `"password"` | Yes | No wire default | Max items: `1`; Unique items: `true` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

```json
{
  "enabled": false,
  "values": {},
  "clearSecrets": [],
  "version": 1
}
```

**Business Logic and Side Effects**

Keys host/port/tlsMode/username/password/sender; blank/omitted existing password preserves, clearSecrets=[password] clears only if valid disabled/effective state. Enable requires all merged values. VerifiedTLS only, explicit port1-65535; secret<=4096 UTF-8 bytes encrypted, never echoed.

- Keys must belong to supported provider schema; no unsupported fields.
- Omitted/blank configured secret keeps; explicit clear removes, reject required missing secret while enabled.
- No secret echoes/logs; safe field-key errors.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [Integration](SCHEMAS.md#integration) | Common response headers | Complete authorized result |

**Response Fields**

[Integration](SCHEMAS.md#integration)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `id` | [BusinessId](SCHEMAS.md#businessid) | Yes | No wire default | D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID.; Min length: `1`; Max length: `10`; Pattern: `"^(?:[1-9][0-9]{0,8}\|1[0-9]{9}\|20[0-9]{8}\|21[0-3][0-9]{7}\|214[0-6][0-9]{6}\|2147[0-3][0-9]{5}\|21474[0-7][0-9]{4}\|214748[0-2][0-9]{3}\|2147483[0-5][0-9]{2}\|21474836[0-3][0-9]\|214748364[0-7])$"` |
| `name` | `string` | Yes | No wire default | Min length: `1`; Max length: `255`; Length unit: `"unicode-code-points"`; Normalization: `"trim"` |
| `description` | `string` | Yes | No wire default | See linked model and business rules. |
| `enabled` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `health` | `string` | Yes | No wire default | Allowed: `"not_configured"`, `"ready"`, `"degraded"` |
| `version` | [Version](SCHEMAS.md#version) | Yes | No wire default | D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset.; Min: `1`; Max: `2147483647` |
| `fields` | array of [IntegrationField](SCHEMAS.md#integrationfield) | Yes | No wire default | Min items: `0`; Max items: `6` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "id": "1",
  "name": "Visitor Example",
  "description": "x",
  "enabled": false,
  "health": "not_configured",
  "version": 1,
  "fields": []
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `404` | Record/document is unavailable under current resource scope; public wrong pair/restricted/absent cases use the same safe not_found response. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `415` | A request body has unsupported media/content; this does not describe binary success responses. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/management_handler.go:UpdateIntegration` → usecase `ManagementUsecase.UpdateIntegration` → repository operations `ManagementRepository.Transaction`, `ManagementTransaction.UpdateIntegration`, `ManagementTransaction.AppendAudit`. Serializer: `ManagementSerializer.ToIntegration`. DTOs: `IntegrationPatch` → `Integration`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-INTEGRATIONS, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

### Reports

**Daftar Endpoint**

| Endpoint | Method | Frontend feature / control | Success response | Error statuses |
| --- | --- | --- | --- | --- |
| [`/v1/staff/reports/summary`](#reportsummary) | `GET` | Analytics/logs filters, summary cards and status table/meters | `200` [Summary](SCHEMAS.md#summary) | 400, 401, 403, 409, 413, 422, 429, 500, 503 |
| [`/v1/staff/reports/export`](#reportexport) | `GET` | Displayed-summary CSV download button | `200` [Export](SCHEMAS.md#export) | 400, 401, 403, 409, 413, 422, 429, 500, 503 |

---

#### reportSummary

**GET /v1/staff/reports/summary — Read scoped summary**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer plus fixed reports permission: organization scope Super Admin, assigned non-VIP scope PA, authorized scope Protocol Officer. Receptionist has no reports permission in the Phase 1 mapping. Exclude restricted before aggregation regardless of detail rights.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F41](FRONTEND-TRACEABILITY.md#f41); /admin/analytics; /admin/logs | `Reports` — Analytics/logs date range and status filters, total cards and per-status summary table/meters | Send inclusive from/to/status; filter echo/scope/generatedAt/counts/total/revision/canExport populate summary and export eligibility. Restricted records are excluded. | [frontendx/apps/admin/src/pages/Reports.tsx](../../frontendx/apps/admin/src/pages/Reports.tsx); useVisitSummary validates filter echo/sum; both tabs share operations |
| [F42](FRONTEND-TRACEABILITY.md#f42); /admin/analytics; /admin/logs | `Reports` — Unduh ringkasan CSV button and changed-summary conflict recovery | On export 409, reload the same summary filter and display its current counts/revision before another export. | [frontendx/apps/admin/src/pages/Reports.tsx](../../frontendx/apps/admin/src/pages/Reports.tsx); repository.exportSummary -> Blob download; conflict refetch; summary rows only |
| [F43](FRONTEND-TRACEABILITY.md#f43); /admin/analytics; /admin/logs; /admin/employees; /admin/roles; /admin/integration | `Settings` — Settings shell mounting read-only employees, integrations and reports; roles/editor routes deferred | Delegate to Reports summary UI; no separate logs/analytics data API. | [frontendx/apps/admin/src/pages/Settings.tsx](../../frontendx/apps/admin/src/pages/Settings.tsx); Lazy router; guard permissions in main.tsx |

**Query Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `from` | [Date](SCHEMAS.md#date) | Yes | None declared | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `to` | [Date](SCHEMAS.md#date) | Yes | None declared | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `status` | `string` | Yes | None declared | Allowed: `"all"`, `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"`, `"missed"` |

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

Inclusive dates from<=to and <=365 elapsed days. All seven nonnegative counts sum to total; specific status zeros others. Read consistent authorized ordinary-only snapshot; revision64lowercasehex binds filter/scope/content, not generatedAt.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [Summary](SCHEMAS.md#summary) | Common response headers | Complete authorized result |

**Response Fields**

[Summary](SCHEMAS.md#summary)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `filter` | [ReportFilter](SCHEMAS.md#reportfilter) | Yes | No wire default | Additional keys: `false` |
| `scope` | `string` | Yes | No wire default | Allowed: `"organization"`, `"assigned"`, `"authorized"` |
| `generatedAt` | `string` | Yes | No wire default | Format: `"date-time"` |
| `canExport` | `boolean` | Yes | No wire default | See linked model and business rules. |
| `revision` | [Revision](SCHEMAS.md#revision) | Yes | No wire default | Min length: `64`; Max length: `64`; Pattern: `"^[0-9a-f]{64}$"` |
| `counts` | `object` | Yes | No wire default | Additional keys: `false` |
| `counts.pending` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `counts.approved` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `counts.rejected` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `counts.cancelled` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `counts.checked_in` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `counts.checked_out` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `counts.missed` | `integer` | Yes, when parent present | - | Min: `0`; Max: `9007199254740991` |
| `total` | `integer` | Yes | No wire default | Min: `0`; Max: `9007199254740991` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "filter": {
    "from": "2030-01-02",
    "to": "2030-01-02",
    "status": "all"
  },
  "scope": "organization",
  "generatedAt": "2030-01-02T00:00:00Z",
  "canExport": false,
  "revision": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "counts": {
    "pending": 1,
    "approved": 0,
    "rejected": 0,
    "cancelled": 0,
    "checked_in": 0,
    "checked_out": 0,
    "missed": 0
  },
  "total": 1
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/report_handler.go:Summary` → usecase `ReportUsecase.GetSummary` → repository operations `ReportRepository.GetSummary`. Serializer: `ReportSerializer.ToSummary`. DTOs: `ReportQuery` → `Summary`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-ADMIN, BE-REPORTS, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

#### reportExport

**GET /v1/staff/reports/export — Export revision-matched summary**

**Auth Required:** Yes — Bearer access JWT plus current active employee, single database role and resource authority.

**Authorization / resource scope:** Bearer plus fixed reports and reportExport permissions for PA, Protocol Officer or Super Admin; recompute the same current authorized ordinary-only scope/filter/revision. Receptionist has no Phase 1 report export.

**Implementation Status:** Required by frontend/selected decisions; backend route missing.

**Description:** Separate export permission. Recompute authorized scope/content revision; 409 if changed. JSON CSV summary rows only, no visitor details.

**Frontend Usage**

| Feature / route | UI element and action | Request / returned-data use | Source / current integration |
| --- | --- | --- | --- |
| [F42](FRONTEND-TRACEABILITY.md#f42); /admin/analytics; /admin/logs | `Reports` — Unduh ringkasan CSV button and changed-summary conflict recovery | Send the displayed from/to/status/revision; filename/csv/revision build the local CSV download, containing summary rows only. | [frontendx/apps/admin/src/pages/Reports.tsx](../../frontendx/apps/admin/src/pages/Reports.tsx); repository.exportSummary -> Blob download; conflict refetch; summary rows only |

**Query Parameters**

| Parameter | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `from` | [Date](SCHEMAS.md#date) | Yes | None declared | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `to` | [Date](SCHEMAS.md#date) | Yes | None declared | Format: `"date"`; Min length: `10`; Max length: `10`; Pattern: `"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"` |
| `status` | `string` | Yes | None declared | Allowed: `"all"`, `"pending"`, `"approved"`, `"rejected"`, `"cancelled"`, `"checked_in"`, `"checked_out"`, `"missed"` |
| `revision` | [Revision](SCHEMAS.md#revision) | Yes | None declared | Min length: `64`; Max length: `64`; Pattern: `"^[0-9a-f]{64}$"` |

**Path Parameters**

None.

**Headers and Cookies**

JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.

| Security branch | Credential / protection | Location | Validation |
| --- | --- | --- | --- |
| 1 | `Authorization: Bearer <accessToken>` | header | Signed access JWT with sub/iat/exp/iss/aud; verified against configured algorithm/key/issuer/audience and finite expiry. Current employee active flag, exactly one database role and PA assignments are re-read for every protected request. No session ID or auth cookies. |

**Request Body**

None. No request body is defined; do not add command fields to the URL.

**Business Logic and Side Effects**

Recompute same authorized filter/content revision;409 report_changed if stale. JSON filename/csv/revision; BOM+quotedCRLF summary rows only; never personal detail.

**Success Responses**

| HTTP status | Body / model | Headers | Meaning |
| --- | --- | --- | --- |
| `200` | application/json: [Export](SCHEMAS.md#export) | Common response headers | Complete authorized result |

**Response Fields**

[Export](SCHEMAS.md#export)

| Field | Type / model | Required | Default | Validation / meaning |
| --- | --- | --- | --- | --- |
| `filename` | `string` | Yes | No wire default | Min length: `1` |
| `csv` | `string` | Yes | No wire default | See linked model and business rules. |
| `revision` | [Revision](SCHEMAS.md#revision) | Yes | No wire default | Min length: `64`; Max length: `64`; Pattern: `"^[0-9a-f]{64}$"` |

Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.

**Response (200) — synthetic example**

```json
{
  "filename": "kunjungan-ringkasan-2030-01-02-2030-01-02.csv",
  "csv": "Status,Jumlah\r\npending,1\r\n",
  "revision": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
```

**Error Responses**

JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.

| HTTP status | Condition |
| --- | --- |
| `400` | Malformed syntax/canonical ID or unsupported/duplicate query; unknown JSON fields follow D57 and unknown query keys follow D58, with rejected field paths. |
| `401` | Missing/invalid/expired Bearer credential or missing/inactive current employee; no stored-session credential is used. |
| `403` | Current fixed role, assignment, resource or field authority denies this action, or the account lacks exactly one supported role. No mutation runs. |
| `409` | A supplied version, current role revision, scoped idempotency reference, schedule commitment or report revision conflicts, only as applicable to this operation. |
| `413` | Request exceeds the selected JSON/multipart/file limit; the global request-body cap also applies to read routes. |
| `422` | Documented semantic field/state validation fails. Missing/invalid fields retain safe fieldErrors arrays; codes remain the selected common vocabulary. |
| `429` | An applicable selected request/operation budget is exceeded; Retry-After is integer seconds, with no mutation or deadline extension. |
| `500` | Unexpected internal failure; redact SQL, secrets and provider details. |
| `503` | Required dependency, applicable limiter or capability is unavailable. An uncertain write is not proof of rollback. |

**Backend Interface Flow and Evidence**

Handler: `internal/handler/report_handler.go:Export` → usecase `ReportUsecase.Export` → repository operations `ReportRepository.GetSummary`. Serializer: `ReportSerializer.ToExport`. DTOs: `ReportExportQuery` → `Export`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).

Known gaps: BE-REPORTS, BE-PHASE1-AUTH. Requirement and source evidence: Phase 1 target; backend implementation missing.

---

<!-- END GENERATED ENDPOINTS -->

## Data Models & Enums

<!-- BEGIN GENERATED MODELS -->

The exact wire schemas, including all nested models, discriminated projections, conditional constraints and server invariants, are in [SCHEMAS](SCHEMAS.md). Field tables and examples appear with each endpoint above. The canonical [OpenAPI](../BACKEND-OPENAPI-2026-11-12.json) remains the machine-readable authority.

### Enums

| Model | Allowed values |
| --- | --- |
| [Role](SCHEMAS.md#role) | `PA`, `Protocol Officer`, `Receptionist`, `Super Admin` |
| [Permission](SCHEMAS.md#permission) | `bookings`, `create`, `approve`, `reject`, `reschedule`, `cancel`, `checkin`, `checkout`, `reports`, `reportExport`, `people`, `integrations` |
| [Status](SCHEMAS.md#status) | `pending`, `approved`, `rejected`, `cancelled`, `checked_in`, `checked_out` |
| [DisplayStatus](SCHEMAS.md#displaystatus) | `pending`, `approved`, `rejected`, `cancelled`, `checked_in`, `checked_out`, `missed` |

`missed` is a derived display/report status; the persisted Status enum has six values. Restricted projection variants are privacy boundaries, not optional field bundles.

### Model directory

| Model | Shape |
| --- | --- |
| [BusinessId](SCHEMAS.md#businessid) | string |
| [Version](SCHEMAS.md#version) | integer |
| [Role](SCHEMAS.md#role) | string |
| [Permission](SCHEMAS.md#permission) | string |
| [Status](SCHEMAS.md#status) | string |
| [DisplayStatus](SCHEMAS.md#displaystatus) | string |
| [Date](SCHEMAS.md#date) | string |
| [StartTime](SCHEMAS.md#starttime) | string |
| [Email](SCHEMAS.md#email) | string |
| [Password](SCHEMAS.md#password) | string |
| [ExistingPassword](SCHEMAS.md#existingpassword) | string |
| [Reference](SCHEMAS.md#reference) | string |
| [BookingCode](SCHEMAS.md#bookingcode) | string |
| [Credential](SCHEMAS.md#credential) | string |
| [ApiError](SCHEMAS.md#apierror) | object |
| [Room](SCHEMAS.md#room) | object |
| [Officer](SCHEMAS.md#officer) | object |
| [Catalogs](SCHEMAS.md#catalogs) | object |
| [VisitTiming](SCHEMAS.md#visittiming) | object |
| [PublicEvent](SCHEMAS.md#publicevent) | object |
| [StaffEvent](SCHEMAS.md#staffevent) | object |
| [PublicBooking](SCHEMAS.md#publicbooking) | object |
| [StandardStaffBooking](SCHEMAS.md#standardstaffbooking) | object |
| [RestrictedReception](SCHEMAS.md#restrictedreception) | object |
| [RestrictedArrival](SCHEMAS.md#restrictedarrival) | object |
| [RestrictedAssignedPA](SCHEMAS.md#restrictedassignedpa) | object |
| [RestrictedFull](SCHEMAS.md#restrictedfull) | object |
| [StaffBooking](SCHEMAS.md#staffbooking) | StandardStaffBooking, RestrictedReception, RestrictedArrival, RestrictedAssignedPA, RestrictedFull |
| [ArrivalBooking](SCHEMAS.md#arrivalbooking) | StandardStaffBooking, RestrictedArrival |
| [PublicCreate](SCHEMAS.md#publiccreate) | object |
| [StandardStaffCreate](SCHEMAS.md#standardstaffcreate) | object |
| [RestrictedCreate](SCHEMAS.md#restrictedcreate) | object |
| [StaffCreate](SCHEMAS.md#staffcreate) | StandardStaffCreate, RestrictedCreate |
| [Approve](SCHEMAS.md#approve) | object |
| [Reject](SCHEMAS.md#reject) | object |
| [Reschedule](SCHEMAS.md#reschedule) | object |
| [RestrictedUpdate](SCHEMAS.md#restrictedupdate) | object |
| [StaffCancel](SCHEMAS.md#staffcancel) | object |
| [Attendance](SCHEMAS.md#attendance) | object |
| [Resolve](SCHEMAS.md#resolve) | object |
| [Pair](SCHEMAS.md#pair) | object |
| [PairVersion](SCHEMAS.md#pairversion) | object |
| [SubmissionResolve](SCHEMAS.md#submissionresolve) | object |
| [Resolution](SCHEMAS.md#resolution) | composed object |
| [Upload](SCHEMAS.md#upload) | object |
| [UploadResult](SCHEMAS.md#uploadresult) | object |
| [AdmissionResult](SCHEMAS.md#admissionresult) | object |
| [Employee](SCHEMAS.md#employee) | object |
| [EmployeePage](SCHEMAS.md#employeepage) | object |
| [IntegrationField](SCHEMAS.md#integrationfield) | composed object |
| [Integration](SCHEMAS.md#integration) | object |
| [IntegrationPatch](SCHEMAS.md#integrationpatch) | object |
| [ReportFilter](SCHEMAS.md#reportfilter) | object |
| [Revision](SCHEMAS.md#revision) | string |
| [Summary](SCHEMAS.md#summary) | object |
| [Export](SCHEMAS.md#export) | object |
| [ReadModel](SCHEMAS.md#readmodel) | object |
| [Login](SCHEMAS.md#login) | object |
| [SmtpValues](SCHEMAS.md#smtpvalues) | object |
| [CurrentUser](SCHEMAS.md#currentuser) | object |
| [AuthResult](SCHEMAS.md#authresult) | object |

<!-- END GENERATED MODELS -->

## Error Codes

<!-- BEGIN GENERATED ERRORS -->

The HTTP status is authoritative; `code` is the machine-readable error identifier. Only the statuses listed for an operation apply to it. No stacktrace, SQL, secret, provider detail or private conflicting-record data is returned.

| HTTP status | Meaning |
| --- | --- |
| `400` | Malformed request/query or unknown fields |
| `401` | Unauthenticated or generic invalid credentials |
| `403` | Forbidden or invalid/expired CSRF |
| `404` | Unavailable without disclosing private record existence |
| `409` | Version/schedule/idempotency/authority/report conflict |
| `413` | Payload too large |
| `415` | Unsupported content type |
| `422` | Semantic validation/transition/privacy error |
| `429` | Rate limited |
| `500` | Safe internal error |
| `503` | Dependency/limiter unavailable |

### Error examples

Field errors use actual form keys and arrays of messages. `429` includes an integer-seconds `Retry-After` header.

HTTP `400`

```json
{
  "code": "invalid_request",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

HTTP `401`

```json
{
  "code": "unauthenticated",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

HTTP `403`

```json
{
  "code": "forbidden",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

HTTP `404`

```json
{
  "code": "not_found",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

HTTP `409`

```json
{
  "code": "version_conflict",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

HTTP `413`

```json
{
  "code": "payload_too_large",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

HTTP `415`

```json
{
  "code": "unsupported_media_type",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

HTTP `422`

```json
{
  "code": "validation_failed",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "fieldErrors": {
    "name": [
      "Periksa nama pengunjung."
    ]
  }
}
```

HTTP `429`

```json
{
  "code": "rate_limited",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

HTTP `500`

```json
{
  "code": "internal_error",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

HTTP `503`

```json
{
  "code": "service_unavailable",
  "message": "Permintaan belum dapat diproses.",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

<!-- END GENERATED ERRORS -->

## Implementation Status

<!-- BEGIN GENERATED STATUS -->

**29 Phase 1 target operations; 1 matching source-registered route; 28 missing target routes.** The unchanged backend also has seven superseded identity routes, for eight source routes total. New login/me is not implemented. Source registration does not certify frontend, lifecycle, database or security conformance. See [GAPS](GAPS.md), [validation evidence](VALIDATION.md) and the [implementation plan](IMPLEMENTATION-PLAN.md).

| Feature group | Source registered | Backend missing |
| --- | --- | --- |
| Authentication | None | [staffLogin](#stafflogin), [staffMe](#staffme) |
| Visitor registration | [publicCatalogs](#publiccatalogs) | [publicCreateBooking](#publiccreatebooking), [resolveSubmission](#resolvesubmission) |
| Visitor status and ticket | None | [publicLookup](#publiclookup), [publicCancel](#publiccancel), [publicAdmission](#publicadmission) |
| Staff catalogs | None | [staffCatalogs](#staffcatalogs) |
| Staff bookings and dashboard | None | [staffBookings](#staffbookings), [staffBooking](#staffbooking), [staffCreateBooking](#staffcreatebooking), [staffApprove](#staffapprove), [staffReject](#staffreject), [staffReschedule](#staffreschedule), [staffRestrictedUpdate](#staffrestrictedupdate), [staffCancel](#staffcancel) |
| Reception and attendance | None | [staffResolve](#staffresolve), [staffCheckin](#staffcheckin), [staffCheckout](#staffcheckout) |
| Private attachments | None | [publicUpload](#publicupload), [staffUpload](#staffupload), [staffAttachment](#staffattachment) |
| Employee directory (read-only) | None | [staffEmployees](#staffemployees), [staffEmployee](#staffemployee) |
| Integrations | None | [staffIntegrations](#staffintegrations), [patchIntegration](#patchintegration) |
| Reports | None | [reportSummary](#reportsummary), [reportExport](#reportexport) |

### Unresolved decisions

| Issue | Affected operations | Decision needed |
| --- | --- | --- |
| [OPEN-READ-ORDER](CONTRACT-ISSUES.md#open-read-order) | staffBookings, staffEmployees | Confirm exact text comparison and tie rules for backend paging before freezing ordering fixtures. Query defaults and authorization scopes are already selected; do not silently inherit host database collation. |

<!-- END GENERATED STATUS -->

## Deferred and Replaced Operations

<!-- BEGIN GENERATED DEFERRED -->

These operations are outside the Phase 1 target. Existing session routes remain source evidence until the auth slice is replaced; their current presence is not Phase 1 support. Account/role editors must be hidden or disabled. Local Sign out has no HTTP operation.

| Previous operation | Method / path | Disposition / replacement | Reason |
| --- | --- | --- | --- |
| createSession | POST /v1/staff/sessions | replaced; [staffLogin](#stafflogin); currently source registered | Replaced by staffLogin with Bearer-only Phase 1 DTOs and no stored session. |
| getSession | GET /v1/staff/session | replaced; [staffMe](#staffme); currently source registered | Replaced by staffMe with Bearer-only Phase 1 DTOs and no stored session. |
| deleteSession | DELETE /v1/staff/session | deferred; currently source registered | Persistent session lifecycle, role switching and self-service password workflow deferred. |
| changePassword | POST /v1/staff/password | deferred; currently source registered | Persistent session lifecycle, role switching and self-service password workflow deferred. |
| selectRole | POST /v1/staff/session/role | deferred; currently source registered | Persistent session lifecycle, role switching and self-service password workflow deferred. |
| createEmployee | POST /v1/staff/employees | deferred; backend absent | Account/role writes and editable permission policy deferred; provision accounts/roles/assignments through controlled database setup. |
| patchEmployee | PATCH /v1/staff/employees/{id} | deferred; backend absent | Account/role writes and editable permission policy deferred; provision accounts/roles/assignments through controlled database setup. |
| staffRoles | GET /v1/staff/roles | deferred; backend absent | Account/role writes and editable permission policy deferred; provision accounts/roles/assignments through controlled database setup. |
| patchRole | PATCH /v1/staff/roles/{id} | deferred; backend absent | Account/role writes and editable permission policy deferred; provision accounts/roles/assignments through controlled database setup. |
| renewSession | POST /v1/staff/session/renew | deferred; currently source registered | Persistent session lifecycle, role switching and self-service password workflow deferred. |
| refreshSession | POST /v1/staff/session/refresh | deferred; currently source registered | Persistent session lifecycle, role switching and self-service password workflow deferred. |

<!-- END GENERATED DEFERRED -->

## Frontend Page-to-Endpoint Mapping

<!-- BEGIN GENERATED FRONTEND -->

These are the actual pages, dialogs, cards, selectors and shared consumers inspected in VMS. The detailed endpoint sections show which request/response fields each control uses and disclose existing integration gaps. Full callsite evidence remains in [FRONTEND-TRACEABILITY](FRONTEND-TRACEABILITY.md).

| Feature | Route / component | UI feature or control | Endpoint method and path |
| --- | --- | --- | --- |
| [F01](FRONTEND-TRACEABILITY.md#f01) | /; /?view=registration; /booking; `BookingForm` | Registration destination/official dropdown in the first booking step (phase1) | [GET /v1/public/catalogs](#publiccatalogs) |
| [F02](FRONTEND-TRACEABILITY.md#f02) | /; /booking; `BookingForm` | Three-step booking form, companion-name rows, optional supporting-document picker and submit/receipt panel (phase1) | [POST /v1/public/bookings](#publiccreatebooking); [POST /v1/public/attachments](#publicupload) |
| [F03](FRONTEND-TRACEABILITY.md#f03) | /; /booking; /status; `PrivacyNotice` | Bilingual privacy notice dialog, consent text and retry button (phase1) | [GET /v1/public/catalogs](#publiccatalogs) |
| [F04](FRONTEND-TRACEABILITY.md#f04) | /; /booking; `SubmissionRecovery` | Uncertain-submission recovery panel and email recovery action (phase1) | [POST /v1/public/submissions/resolve](#resolvesubmission) |
| [F05](FRONTEND-TRACEABILITY.md#f05) | /status; /?view=status; `Status` | Status lookup form: eight-digit code input, email input, View visit and Update buttons (phase1) | [POST /v1/public/lookup](#publiclookup) |
| [F06](FRONTEND-TRACEABILITY.md#f06) | /status; /?view=status; `Status` | Cancel visit button and confirmation dialog on the status card (phase1) | [POST /v1/public/bookings/{id}/cancel](#publiccancel); [POST /v1/public/lookup](#publiclookup) |
| [F07](FRONTEND-TRACEABILITY.md#f07) | /status; booking receipt; `AdmissionQR` | Admission QR image, expiry/countdown and Renew QR code/Try again buttons (phase1) | [POST /v1/public/bookings/{id}/admission](#publicadmission) |
| [F08](FRONTEND-TRACEABILITY.md#f08) | /status; booking receipt; `VisitTicket` | Booking receipt / printable VisitTicket card (phase1) | [POST /v1/public/lookup](#publiclookup); [POST /v1/public/bookings](#publiccreatebooking); [POST /v1/public/bookings/{id}/admission](#publicadmission) |
| [F09](FRONTEND-TRACEABILITY.md#f09) | /status; `PublicVisitHistory` | Public visit-history timeline within the status view (phase1) | [POST /v1/public/lookup](#publiclookup); [POST /v1/public/bookings/{id}/cancel](#publiccancel) |
| [F10](FRONTEND-TRACEABILITY.md#f10) | /status; `useCancellationWindow` | Time-based enabled/disabled state of the public cancellation button (phase1) | [POST /v1/public/lookup](#publiclookup); [POST /v1/public/bookings/{id}/cancel](#publiccancel) |
| [F11](FRONTEND-TRACEABILITY.md#f11) | /admin/login; `Login` | Staff login email/password form and sign-in button (phase1) | [POST /v1/staff/auth/login](#stafflogin) |
| [F12](FRONTEND-TRACEABILITY.md#f12) | /admin/*; `SessionProvider` | Workspace current-user load, fixed token expiry and local Sign out action (phase1) | [GET /v1/staff/auth/me](#staffme) |
| [F13](FRONTEND-TRACEABILITY.md#f13) | /admin/*; `RoleSelection / RoleSwitcher` | Required-role selection screen and active-role switcher (later) | Deferred in Phase 1; no API call |
| [F14](FRONTEND-TRACEABILITY.md#f14) | /admin/* forced gate; `PasswordChange` | Required password-change form: current password, new password and save button (later) | Deferred in Phase 1; no API call |
| [F15](FRONTEND-TRACEABILITY.md#f15) | /admin/*; `SessionProvider` | Workspace activity/expiry handling and access-token restoration after reload; consumer still missing (later) | Deferred in Phase 1; no API call |
| [F16](FRONTEND-TRACEABILITY.md#f16) | /admin/booking; /admin/today; `Bookings` | Janji temu / Kunjungan hari ini search box, status tabs, VIP filter, sort selector, cards/table and pagination (phase1) | [GET /v1/staff/bookings](#staffbookings) |
| [F17](FRONTEND-TRACEABILITY.md#f17) | /admin/booking?selected={id}; /admin/today?selected={id}; `Bookings / BookingDetail` | Selected booking-detail dialog opened from a row/card or selected URL parameter (phase1) | [GET /v1/staff/bookings/{id}](#staffbooking) |
| [F18](FRONTEND-TRACEABILITY.md#f18) | /admin/dashboard; `Dashboard` | Dashboard metric cards, Agenda hari ini / Menunggu kedatangan table and Perlu perhatian / Sedang berkunjung list (phase1) | [GET /v1/staff/bookings](#staffbookings) |
| [F19](FRONTEND-TRACEABILITY.md#f19) | /admin/dashboard; /admin/today; /admin/booking; /admin/check-in; /admin/check-out; `ProductionRefresh` | Visible-page refresh timer, focus refresh and manual refresh for lists/details/reception (phase1) | [GET /v1/staff/bookings](#staffbookings); [GET /v1/staff/bookings/{id}](#staffbooking) |
| [F20](FRONTEND-TRACEABILITY.md#f20) | /admin/*; `DataRefresh` | DataRefresh freshness/offline/error notice and retry controls (phase1) | [GET /v1/staff/bookings](#staffbookings); [GET /v1/staff/employees](#staffemployees); [GET /v1/staff/integrations](#staffintegrations) |
| [F21](FRONTEND-TRACEABILITY.md#f21) | /admin/booking?new=1; /admin/today?new=1; `CreateAppointment` | Staff standard-appointment dialog: official dropdown, schedule, representative/roster, note fields, document picker and submit (phase1) | [GET /v1/staff/catalogs](#staffcatalogs); [POST /v1/staff/bookings](#staffcreatebooking); [POST /v1/staff/attachments](#staffupload) |
| [F22](FRONTEND-TRACEABILITY.md#f22) | /admin/booking?new=1&kind=restricted; `RestrictedAppointment` | Kunjungan Terbatas creation dialog: date/time, optional identity/target fields, notes and create button (phase1) | [GET /v1/staff/catalogs](#staffcatalogs); [POST /v1/staff/bookings](#staffcreatebooking) |
| [F23](FRONTEND-TRACEABILITY.md#f23) | /admin/booking?selected={id}; `RestrictedAppointment` | Restricted visit edit dialog and Simpan perubahan button (phase1) | [POST /v1/staff/bookings/{id}/update](#staffrestrictedupdate); [GET /v1/staff/catalogs](#staffcatalogs) |
| [F24](FRONTEND-TRACEABILITY.md#f24) | /admin/booking; /admin/today; `BookingDetail` | Approve dialog: room dropdown, duration input, note fields and confirmation (phase1) | [POST /v1/staff/bookings/{id}/approve](#staffapprove); [GET /v1/staff/catalogs](#staffcatalogs) |
| [F25](FRONTEND-TRACEABILITY.md#f25) | /admin/booking; /admin/today; `BookingDetail` | Reject dialog: public rejection-reason textarea, private note textarea and reject button (phase1) | [POST /v1/staff/bookings/{id}/reject](#staffreject) |
| [F26](FRONTEND-TRACEABILITY.md#f26) | /admin/booking; /admin/today; `BookingDetail` | Reschedule dialog: new date/time, room dropdown, duration, reason and notes (phase1) | [POST /v1/staff/bookings/{id}/reschedule](#staffreschedule); [GET /v1/staff/catalogs](#staffcatalogs) |
| [F27](FRONTEND-TRACEABILITY.md#f27) | /admin/booking; /admin/today; `BookingDetail` | Staff cancellation dialog: public reason, internal cancellation note and confirm button (phase1) | [POST /v1/staff/bookings/{id}/cancel](#staffcancel) |
| [F28](FRONTEND-TRACEABILITY.md#f28) | staff create / approve / reschedule / restricted edit dialogs; `VisitNotesFields` | Shared Instruksi pengunjung and Catatan koordinasi internal textareas (phase1) | [POST /v1/staff/bookings](#staffcreatebooking); [POST /v1/staff/bookings/{id}/approve](#staffapprove); [POST /v1/staff/bookings/{id}/reschedule](#staffreschedule); [POST /v1/staff/bookings/{id}/update](#staffrestrictedupdate) |
| [F29](FRONTEND-TRACEABILITY.md#f29) | /admin/booking?selected={id}; `AttachmentAccess` | Private supporting-document action in BookingDetail; existing action downloads only (phase1) | [GET /v1/staff/bookings/{id}/attachment](#staffattachment) |
| [F30](FRONTEND-TRACEABILITY.md#f30) | /admin/check-in; /admin/check-out; `Reception` | Reception scheduled-arrivals / still-inside table and previous/next page controls (phase1) | [GET /v1/staff/bookings](#staffbookings) |
| [F31](FRONTEND-TRACEABILITY.md#f31) | /admin/check-in; /admin/check-out; `Reception` | Manual visit-code form or QR resolve result before attendance confirmation (phase1) | [POST /v1/staff/resolve](#staffresolve) |
| [F32](FRONTEND-TRACEABILITY.md#f32) | /admin/check-in; /admin/check-out; `Scanner` | Camera scanner preview and decoded-QR handler (phase1) | [POST /v1/staff/resolve](#staffresolve) |
| [F33](FRONTEND-TRACEABILITY.md#f33) | /admin/check-in; `Reception` | Reception identity-confirmation checkbox and check-in confirmation button (phase1) | [POST /v1/staff/bookings/{id}/checkin](#staffcheckin) |
| [F34](FRONTEND-TRACEABILITY.md#f34) | /admin/check-out; `Reception` | Reception identity-confirmation checkbox and check-out confirmation button (phase1) | [POST /v1/staff/bookings/{id}/checkout](#staffcheckout) |
| [F35](FRONTEND-TRACEABILITY.md#f35) | /admin/employees; `EmployeeDirectory` | Read-only employee directory search, cards/table, pagination and detail; editing controls deferred (phase1) | [GET /v1/staff/employees](#staffemployees); [GET /v1/staff/employees/{id}](#staffemployee) |
| [F36](FRONTEND-TRACEABILITY.md#f36) | /admin/employees; `EmployeeEditor` | Employee create/edit dialog: profile fields, active checkbox, roles, assignments and initial password on create (later) | Deferred in Phase 1; no API call |
| [F37](FRONTEND-TRACEABILITY.md#f37) | /admin/employees editor; `RoleAssignments` | Multi-role tag/search selector within the employee editor (later) | Deferred in Phase 1; no API call |
| [F38](FRONTEND-TRACEABILITY.md#f38) | /admin/employees editor; `EmployeeEditor` | PA non-VIP official-assignment checkboxes and retained-unavailable-assignment notice (later) | Deferred in Phase 1; no API call |
| [F39](FRONTEND-TRACEABILITY.md#f39) | /admin/roles; `RoleManagement / RoleEditor` | Peran operasional cards, Edit akses dialog, permission checkboxes and Simpan akses button (later) | Deferred in Phase 1; no API call |
| [F40](FRONTEND-TRACEABILITY.md#f40) | /admin/integration; `IntegrationManagement / IntegrationEditor` | Integration service cards, Konfigurasi dialog, enabled checkbox, dynamic settings fields and secret keep/clear controls (phase1) | [GET /v1/staff/integrations](#staffintegrations); [PATCH /v1/staff/integrations/{id}](#patchintegration) |
| [F41](FRONTEND-TRACEABILITY.md#f41) | /admin/analytics; /admin/logs; `Reports` | Analytics/logs date range and status filters, total cards and per-status summary table/meters (phase1) | [GET /v1/staff/reports/summary](#reportsummary) |
| [F42](FRONTEND-TRACEABILITY.md#f42) | /admin/analytics; /admin/logs; `Reports` | Unduh ringkasan CSV button and changed-summary conflict recovery (phase1) | [GET /v1/staff/reports/export](#reportexport); [GET /v1/staff/reports/summary](#reportsummary) |
| [F43](FRONTEND-TRACEABILITY.md#f43) | /admin/analytics; /admin/logs; /admin/employees; /admin/roles; /admin/integration; `Settings` | Settings shell mounting read-only employees, integrations and reports; roles/editor routes deferred (phase1) | [GET /v1/staff/employees](#staffemployees); [GET /v1/staff/integrations](#staffintegrations); [GET /v1/staff/reports/summary](#reportsummary) |
| [F44](FRONTEND-TRACEABILITY.md#f44) | /admin/*; `Shell / Guard` | Staff name/role display, fixed-permission navigation and route guards (phase1) | [GET /v1/staff/auth/me](#staffme) |
| [F45](FRONTEND-TRACEABILITY.md#f45) | public/staff submission forms; `uploadAttachment` | Shared file-picker upload/retry transport for public and staff forms (phase1) | [POST /v1/public/attachments](#publicupload); [POST /v1/staff/attachments](#staffupload) |
| [F46](FRONTEND-TRACEABILITY.md#f46) | all API consumers; `createHttpClient` | Shared Bearer-only HTTP transport and field-error/rate-limit handling (phase1) | [POST /v1/staff/auth/login](#stafflogin); [GET /v1/staff/auth/me](#staffme); [POST /v1/public/bookings](#publiccreatebooking) |
| [F47](FRONTEND-TRACEABILITY.md#f47) | all booking consumers; `createRepository` | Shared production repository used by booking/status components; preview adapter is explicit (phase1) | [GET /v1/staff/bookings](#staffbookings); [POST /v1/public/lookup](#publiclookup) |
| [F48](FRONTEND-TRACEABILITY.md#f48) | /; /booking; /status; `App` | Visitor landing shell with embedded registration/status views; hero/info content is static (phase1) | [GET /v1/public/catalogs](#publiccatalogs); [POST /v1/public/lookup](#publiclookup) |

Static landing content, ticket printing, camera capture, local filter state, identity-confirmation checkboxes and draft/preferences storage do not introduce separate business endpoints. Notes are submitted with their owning booking command; selectors share the audience-appropriate catalogs call.

<!-- END GENERATED FRONTEND -->

## References actually inspected

| Reference | Inspected evidence | What it establishes / limits |
| --- | --- | --- |
| Final uploaded `apicontract revisi2.md`, 2,294 lines | Complete current attachment; filtered commodity collection replacements, reused market/map data, enriched detail responses and proposed dropdown removals | Establishes the requested reference approach. Its unresolved combined-filter question, inconsistent routes/auth labels, malformed example JSON and stale implementation checklist are not VMS requirements. See the source comparison in the revision record. |
| Uploaded `apicontract-improved.md`, 1,096 lines | Complete current attachment, including endpoint examples, status inventory, enum/schema sections and frontend mapping | Useful document structure. Its implemented/planned commodity claims contradict one another and differ from current SIHP source; status labels were not copied into VMS. |
| `C:/Users/ASUS/go/src/github.com/thdoikn/sihp-be` | `internal/server/rest/router/{dependencies,auth_router,public_router,admin_router}.go`; auth/SIHP handlers; auth/SIHP usecase interfaces and implementations; auth/master-data/transaction-data repository interfaces; auth implementation and transactional finalization; SIHP serializer interface/implementation; `pkg/dto/sihp.go`; root/migration Makefiles; all nine SQL migrations | Router constructs repo/serializer/usecase/handler; handler depends on usecase interface; constructor returns interface; usecase validates typed DTO and coordinates repository/serializer; repository owns GORM and transaction; serializer maps fields. Actual source has public/admin audiences and aggregate detail/overview reads. |
| SIHP documentation workflow | Handler `godoc`/`@Router` annotations; `make swagger` runs `swag init -g cmd/main.go --output docs --parseDependency`; no `.github` workflow found | Reuse an explicit generation/check step. No frontend/interface/response conformance check was found. Swagger2 generation is not a compatible replacement for VMS OpenAPI3.1 union schemas. No new generator dependency is justified. |
| SIHP migrations | `migration/files/20260520...` schema migration and eight `20260521...` table migrations | Timestamped `sql-migrate` Up/Down SQL with explicit entities, FK/unique/index constraints. VMS retains this tool/layout; UUIDs, CASCADE choices and SIHP business tables are not VMS requirements. |
| VMS frontend | Both app shells; every page/feature; all `packages/data/src` adapters, validators, domain projections, session and report modules; preview boundaries | 48 feature/action rows and full source-file classification. Existing adapters are requirements evidence; a preview result does not prove backend persistence. |
| VMS backend | Router/handler/usecase/repository/serializer/DTO/auth middleware source; development docs generator and tests; three VMS SQL migrations | Only catalog plus seven identity operations exist. The previous served OpenAPI advertised only catalog. Current source already uses compatible SIHP layer conventions for these modules; there is no visit/admin/report implementation to call aligned. |

## Common wire and authority rules

All endpoint paths include `/v1` on the configured audience origin. JSON uses camelCase and direct records/arrays/pages. Positive PostgreSQL INTEGER business IDs use canonical decimal strings 1-2147483647; versions are positive integers. Eight-digit codes, phone, NIP and service IDs are text, preserving leading zeroes. UUID submission/upload references are distinct.

Dates are real YYYY-MM-DD and times HH:mm in Asia/Makassar. New starts must be future and within 08:00-16:00 inclusive; calculated end may exceed 16:00. Creation duration is 60 minutes; selected schedule mutations accept 15-240. Stored states remain pending/approved/rejected/cancelled/checked_in/checked_out. Missed is derived only when approved and now is after end. Check-in includes [start-30 minutes,end]; checkout can be late or on a later date.

Schemas define required/optional fields. Optional means omitted rather than null. Notes: omission preserves and empty clears. Restricted optional identity/target updates retain their selected preserve/clear semantics. Reschedule replaces the full schedule, so missing roomId clears; approval omission preserves an existing room but cannot bypass a required active standard-public room selection. IDs, actor, role/permissions and computed values are server owned.

Reject unknown JSON/query keys and repeated query keys with 400 before database work: D57 unexpected_field and D58 unexpected_query_parameter retain safe rejected-field paths. Semantic fields use 422 with string-array fieldErrors. Human text is trimmed/counts Unicode code points; existing passwords preserve bytes. New operator-provisioned passwords retain the selected 15-code-point minimum/72-UTF-8-byte maximum. Email is one normalized ASCII mailbox up to 254 bytes. Ordinary representative/roster names are 2-120 code points, organization 2-255, purpose 5-1000, phone 8-20 digits, roster 1-20 with representative first. No ordinary identity number/image is collected. Restricted visits require schedule but do not acquire ordinary contact/roster/upload requirements.

Staff commands derive authority from verified AccessProof and current database account/single-role/PA assignments, then recheck resource/field scope within write transactions. Fixed permissions never grant unrestricted ordinary or restricted data. No stored login session, editable policy graph, role/password gate or credential/role revision is needed. Read-only employee data and safe integration settings remain separate from role provisioning.

Public email/code proof stays in POST bodies and authorizes only matching standard records. Restricted records never enter public lookup/history/reporting. Scope and field projection precede search/sort/count/page/aggregation. Restricted reception/arrival/assigned-pa/full variants remain separate privacy boundaries. Private notes/staff actors never become public history or visitor email; full restricted read does not automatically grant coordination-note authority. Ordinary additive response fields remain compatible under D57; serializers still return authorized fields only, and restricted projections require raw allowlist verification before stripping unknown fields.

Request/media caps remain JSON 262144 bytes, multipart 6291456 bytes, file 5242880 bytes with verified PDF/JPEG/PNG. Unsupported media 415; excess size 413; malformed syntax/unknown keys 400; semantic fields 422; current mutation/resource/idempotency conflicts 409. Errors are ApiError without SQL, stacktrace, vendor secrets or private conflict data. An abort/timeout is not evidence of rollback.

Retain the existing public 120/min/trusted-IP and staff 120/min/current-employee ceilings plus applicable pair/create/recovery/file budgets. Login retains 5 failed attempts/account/15m and 100 requests/IP/15m; refresh/password-operation budgets are deferred with those operations. Applicable dependency/limiter failure is 503, never empty success. Browser origin settings and HTTPS remain deployment basics; auth cookies and signed-CSRF/gateway initialization are outside Phase 1.

## Consolidation decisions

| Shared need / consuming features | Contract choice | Authorization / performance constraints | Schema implication |
| --- | --- | --- | --- |
| Visitor destination selectors + bilingual privacy; staff room/official/assignment selectors (F01,F03,F21–24,F38) | One Catalogs request per audience, cached by authority; include related officer/room/notice data | Public active safe data; staff authorized retained selections only. Separate public/staff paths prevent private catalog leakage. No requests per dropdown/notice. Directory cardinality has no invented truncation: current lists are finite but unpaginated; measure before adding search/paging. | Reuse Pejabat/Jabatan/Ruangan and configuration; no catalog table. |
| Booking/today lists, reception pages and dashboard (F16–19,F30) | One GET staff/bookings with mutually exclusive list/reception/dashboard query schemas | Scope first; max100 rows. Dashboard requires complete metrics plus first5 agenda/first4 attention union from one snapshot. No fetch-all or per-card endpoints. Separate detail request only when a selected record is opened or not on the page. | Existing planned visit aggregate; no dashboard/read-model table. |
| Staff ordinary + restricted creation (F21,F22) | One POST staff/bookings with discriminated body | Separate required fields and role/resource checks; no forced ordinary PII for restricted visit. Public create remains separate because authority and initial state differ. | Same Kunjungan/Detail/Pengunjung ownership; restricted visitor snapshot nullable. |
| Notes with approval/reschedule/restricted edits (F23–28) | Notes are fields on the existing stateful command | Omit preserves, empty clears; public/private fields filtered separately. No unrelated standalone note request. | Reuse two distinct detail-note columns. |
| Manual code + camera QR (F31–34) | One POST staff/resolve; distinct explicit checkin/checkout commands | Resolve has no attendance side effect. State/credential rules differ between admission and late checkout, so mutations remain separate. Camera image stays local. | Reuse QR Kunjungan and selected attendance events; no scan/image table. |
| Analytics + logs (F41,F42) | Same report summary and revision-bound export operations | Full authorized ordinary-only aggregate, not current page; export independent permission, current revision. No visitor-detail CSV. | Read-only aggregation; no report table. |
| Attachment View + Download (F29,F45) | Same private bytes route with disposition enum; public/staff upload share internal implementation, distinct audience paths | Same field authority, verified content, no-store; default inline; explicit download attachment. Optional new upload disabled until supported. | One File Storage metadata relationship; no separate preview/download table. |
| Create/replay/recovery + notifications (F02,F04,F21,F22) | Existing create result returns complete booking; recovery returns outcome+booking; notify via transaction/outbox | No second lookup after known successful create, no polling notification endpoint, no resending logical email on replay. Resolve missing public reference must close it durably. | Existing approved idempotency/outbox support; no table per action. |

Separate endpoints remain for unrelated authorization/state transitions. A single HTTP aggregate may use multiple bounded SQL statements in one snapshot. Request consolidation is not a claim that all reads fit one SQL query or should share public/private credentials.


## Enforcement workflow

From the repository root:

```powershell
python docs/contracts/render.py --write
python docs/contracts/check.py --runtime --consumers
python docs/contracts/check.py --strict --runtime --consumers --output docs/contracts/strict-validation-results.json
```

Baseline checks reject undocumented drift while reporting reviewed gaps. Strict checks require all 29 Phase 1 target operations and active conformance gaps/decisions to be complete; deferred lifecycle/account-role work does not block that gate. Source-only identity routes must be removed/replaced with their owning auth implementation and no longer exposed as a Phase 1 fallback. OPEN-READ-ORDER remains active; OPEN-LOGOUT-PROOF is deferred because local logout has no cookie/server operation.

Canonical OpenAPI/manifest/examples generate the marked human sections. The current-source snapshot, AST interfaces/DTOs/routes and existing served documentation are checked separately. `vms-dev swagger` generates documentation for the unchanged current binary from the explicit source baseline, including legacy identity routes; it must not pretend login/me exists or silently hide source routes. Advance that reviewed source baseline with each actual implementation slice, alongside runtime tests. It is evidence, never an alternate Phase 1 authority.

Runtime captures exercise current handlers/decoding/errors with injected usecases. Their legacy session samples are source regressions only; they do not prove Phase 1 authentication. Consumer probes exercise actual existing parsers/HTTP headers and reviewed TypeScript request callsites, reporting the new endpoint/DTO, Bearer, read-only directory and other existing mismatches. No check starts a server/database/container or applies migrations.

[INTERFACES](INTERFACES.md) keeps the SIHP interface-first sequence and separates current-source inventory from proposed Phase 1 signatures. [IMPLEMENTATION-PLAN](IMPLEMENTATION-PLAN.md) orders auth/frontend replacement before the business slices. [SCHEMA-IMPLICATIONS](SCHEMA-IMPLICATIONS.md) records retained tables and deferred structures. [GAPS](GAPS.md), [CONTRACT-ISSUES](CONTRACT-ISSUES.md) and [VALIDATION](VALIDATION.md) distinguish documentation/source checks from real database/browser acceptance. No new dependency, app endpoint, migration, live-data action or deployment is part of this revision.
