# Confirm restricted visits in Super Admin report exports

Type: grilling
Label: wayfinder:grilling
Status: resolved
Assignee: current review agent
Parent: ../map.md
Blocked by: none

## Question

Should the Super Admin visit-log export include the same projected restricted rows as its current table, including name/organization/host/schedule, while omitting contacts, service IDs, admission credentials and internal notes? Current frontend supports this, but older documentation prohibits general export of restricted visits. Aggregate analytics independently excludes restricted visits.

## Evidence

`frontendx/packages/data/src/visit-log.ts` projectVisitLog and canReadVisitLog; `frontendx/apps/admin/src/pages/VisitLog.tsx` export flow; `frontendx/packages/data/src/reports.ts` aggregateVisits; older `docs/frontend/PRODUCT.md` privacy boundary.

## Answer

16 November 2026: user confirmed limited restricted rows may be exported by authorized Super Admin, matching current frontend. This supersedes the older general export prohibition for this specific report. Contacts, service IDs, admission credentials and internal notes remain excluded. Aggregate analytics' separate restricted exclusion is unchanged. The decision is not authorization to export or disclose actual records during this review.
