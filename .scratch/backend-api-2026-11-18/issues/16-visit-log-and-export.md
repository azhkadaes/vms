# 16: Visit log and export (ops 24, 25)

**What to build:** Super Admin gets the visit log:
- search; repeatable status, kind, official and room filters;
- sorting and paging;
- restricted visits as limited rows.

They can also export a complete snapshot, which the browser turns into CSV or PDF. The export only succeeds when its revision matches the loaded data.

Context:
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-08, V-09, V-10(b), A-13
- [Super Admin visit log](../../../docs/contracts/SUPER-ADMIN-VISIT-LOG.md)
- decisions 04, 24
- open decision D-05 uses its default

**Blocked by:** 09.

**Status:** ready-for-agent

- [ ] **Access:** Super Admin with the reports permission only; anyone else → 403
- [ ] **Filters:**
  - [ ] `from` and `to` are optional; `q` is at most 200 characters
  - [ ] status, kind, officer and room may repeat: OR within a field, AND across fields
  - [ ] an empty value means no filter
  - [ ] room filters by ID
  - [ ] `filter` is echoed exactly as received
- [ ] **Sorting and paging:**
  - [ ] sort keys: date, name, organization, host, status, kind, visitors
  - [ ] `pageSize` is 10, 25 or 50; `page` is clamped
- [ ] **Rows:**
  - [ ] fields follow the contract
  - [ ] `checkedInAt` and `checkedOutAt` are nullable UTC `Z`
  - [ ] `roomId` is not emitted until the frontend room-ID change ships (strict parser)
- [ ] **Restricted rows:**
  - [ ] reference `TRB-<id>`; name falls back to "Kunjungan terbatas"; blank purpose; `visitors` null
  - [ ] no contacts, service ID, credentials or notes
- [ ] **Options:** officer and room options come from the report scope and include inactive historical references
- [ ] **Export:**
  - [ ] uses the same filters plus `revision`; a mismatch → 409 `revision_conflict`
  - [ ] item count equals `summary.total`
  - [ ] more than 10,000 rows → 422 `export_too_large`
- [ ] **Exact fields:** output decodes with the frontend's strict visit-log schema
