# Super Admin visit log

This additive frontend contract replaces Super Admin's separate Kunjungan, Analitik, and Laporan kunjungan navigation with **Log Kunjungan** at `/admin/logs`. Old `/admin/booking`, `/admin/today`, and `/admin/analytics` links redirect to the log. Other roles retain their existing pages and scopes.

Implementation status: the Go backend serves both report endpoints below. The frontend repository, runtime schemas, preview API, CSV export, and PDF export already support the log. The frontend adapter still needs to use room IDs and send Bearer credentials when it connects to the Go backend. The operational booking list has a separate Super Admin scope for restricted visits.

## Authority and projection

Both endpoints require an authenticated Super Admin with `reports`; export additionally requires `reportExport`. Resolve identity, role, and permissions from the authenticated account. Never accept a caller-selected role in production. Send `Cache-Control: no-store`.

The log is read-only and includes regular, VIP, and restricted visits. This is an explicit new Super Admin reporting capability, separate from operational booking access. It does not grant create/update/approval/check-in permissions or change other roles' scope. Existing aggregate-only summary/export endpoints retain their current behavior, including excluding restricted visits.

Return only the allowlisted reporting projection defined by `visitLogRowSchema` in `frontendx/packages/data/src/visit-log.ts`. Do not include service IDs, restricted attendance codes, email, phone, identity documents, attachments, private notes, or raw history. For restricted visits, return a non-credential reference, an empty purpose, and `visitors: null`. Standard references use the existing booking reference/code. Unknown attendance timestamps are null, never inferred from status.

## List: `GET /v1/staff/reports/visits`

| Query | Default | Validation / behavior |
| --- | --- | --- |
| `from`, `to` | empty | Optional inclusive calendar dates (`YYYY-MM-DD`) in WITA, using scheduled visit date. Open-ended and all-time ranges are supported. If both present, `from <= to`. |
| `q` | empty | Trimmed search, max 200 characters, case-insensitive match on reference, name, organization, host, room, or purpose. |
| `status` | `all` | `all`, `pending`, `approved`, `rejected`, `cancelled`, `checked_in`, `checked_out`, `missed`. Resolve missed using the server's attendance timing rules; a missed visit must not also count as approved. |
| `kind` | `all` | `all`, `standard`, `vip`, `restricted`. Categories are mutually exclusive; `standard` excludes VIP. |
| `officer` | empty | Exact destination officer ID. |
| `room` | empty | Exact room ID. |
| `sort` | `date` | `date`, `name`, `organization`, `host`, `status`, `kind`, `visitors`. `date` includes scheduled time. |
| `dir` | `desc` | `asc` or `desc`. Stable ordering uses the visit ID as the final tie-breaker. Unknown visitor counts sort below zero. |
| `page` | `1` | Integer 1–100000; clamp to the final available page after filtering. |
| `pageSize` | `10` | One of 10, 25, 50. |

Filter and sort on the server before slicing the page. Reject invalid filters with HTTP 422 and `fieldErrors` where applicable. The strict response schema is `visitLogResponseSchema`:

- `filter`: normalized filters, without pagination.
- `items`: the requested page of reporting rows.
- `total`, `summary.total`: count of **all** matching visits.
- `summary.visitors`: sum of known visitor counts across all matching visits; restricted counts are excluded.
- `summary.inside`, `summary.completed`, `summary.pending`: filtered counts of `checked_in`, `checked_out`, `pending`.
- `page`, `pageSize`: effective pagination.
- `officers`: unique `{ id, name }` destinations across the authorized log, before applying filters.
- `rooms`: unique nonempty room values across the authorized log, before applying filters.
- `canExport`: current server-authorized export permission.
- `generatedAt`: ISO timestamp; presentation uses Asia/Makassar.
- `revision`: SHA-256 revision of normalized filters, **all matching reporting rows**, their deterministic order, and current authority. Page changes must not alter the revision.

Each reporting row contains `id`, `reference`, `name`, `organization`, `officerId`, `host`, `date`, `time`, `duration`, `room`, `purpose`, `kind`, `visitors`, `status`, `checkedInAt`, `checkedOutAt`. Attendance timestamps come from the recorded check-in/out events, as ISO timestamps or null.

## Export: `GET /v1/staff/reports/visits/export`

Accept the same filters/sort plus required `revision`, without pagination. Reauthorize and rebuild the complete filtered snapshot. Return HTTP 409 if the revision differs, requiring the user to review refreshed data before exporting. Return HTTP 403 if report or export authority has been removed.

Return `visitLogSnapshotSchema`: `filter`, **all** matching `items` in the same order, `summary`, `generatedAt`, `revision`. No pagination or option fields. The frontend checks the filter, revision, and complete item count before generating the selected file locally.

CSV includes a UTF-8 BOM, consistent quoted columns, escaped quotes, and spreadsheet formula neutralization. PDF includes applied filters, sort order, summary counts, a paginated landscape A4 table with repeated headers, generation timestamp, and page numbers. Both include all filtered records, independent of the visible table page. PDF dependencies load only when exporting PDF.

HTTP 401 follows the existing authentication-loss behavior. HTTP 403 hides cached rows/cards. Transient refresh errors mark cached data as old and disable export; initial errors/offline states never display fabricated zero counts. Filter and sort state are preserved in the URL, while unapplied form edits disable export.

## Verification

From `frontendx`:

```powershell
node node_modules/typescript/bin/tsc --noEmit
node --experimental-strip-types --test tests/visit-log.integration.mjs tests/visit-log.browser.mjs
node node_modules/vite/bin/vite.js build --config apps/admin/vite.config.ts
```

Browser verification uses the local admin development server on port 3002 and an isolated preview API. It checks navigation redirects, filters, sorting, pagination, read-only detail, complete CSV/PDF downloads, responsive layout, and stale/denied states.
