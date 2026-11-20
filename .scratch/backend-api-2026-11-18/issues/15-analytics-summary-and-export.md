# 15: Analytics summary and CSV export (ops 22, 23)

**What to build:** A date-range analytics summary, scoped by role:
- counts per stored status plus derived missed, summing to the total;
- restricted visits excluded;
- a revision that changes whenever any counted visit changes.

Users with the export permission can download a summary CSV, but only when its revision matches what they viewed.

Context:
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-08, V-09, V-10(b), A-11, A-12
- decision 12

**Blocked by:** 09.

**Status:** done (merge 772eef9 on vms/backend-api)

**Verification:** PostgreSQL summary scope and revision tests, plus strict frontend summary and CSV parsers.

- [ ] **Input:** `from` and `to` are inclusive WITA dates, at most 366 days apart; `status` may repeat
- [x] **Filter echo:** `filter` is echoed exactly as received (string for one value, array for 2+, click order kept)
- [x] **Scope:** follows A-11 on the WITA scheduled-date basis, with restricted visits excluded
- [x] **Counts:** stored statuses plus missed sum to `total`, all evaluated at one server time
- [ ] **Metadata:**
  - [x] `generatedAt` is UTC `Z`
  - [x] `canExport` reflects the reportExport permission
  - [x] `revision` changes when any matched visit's version or display status changes
- [ ] **Export:**
  - [x] requires the same filters plus `revision`; a mismatch → 409 `revision_conflict`
  - [ ] missing reportExport → 403
  - [x] returns `{filename, csv, revision}`; the filename matches `kunjungan-ringkasan-…csv` and the CSV follows the preview format
- [x] **Exact fields:** output decodes with the frontend's strict report schema
