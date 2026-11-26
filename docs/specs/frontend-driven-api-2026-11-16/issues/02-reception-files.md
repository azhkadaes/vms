# Define reception proof for private files

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: 01

## Question

How must successful code/QR resolution authorize subsequent reception document/photo reads? The preview sends credential with the file request, while production sends an authenticated GET without it. Inspect owning backend behavior before proposing the smallest contract correction.

## Evidence

`frontendx/packages/data/src/attachments.ts` loadArrivalFile; `apps/admin/src/features/ReceptionFiles.tsx`.

## Answer

16 November 2026: user requires verified code/QR proof before reception may access the photo or Nota Dinas, matching the visible flow. Reception role alone is insufficient. The production adapter's proof omission is a later implementation fix, not an approved relaxation. Exact proof transport remains to be specified; never place credentials in URL queries. Other roles retain their independently scoped file permissions.
