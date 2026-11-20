# Send room IDs while displaying room names

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Question

Use stable room IDs or resolve submitted room names against the database?

## Answer

17 November 2026: user selected roomId in the API, displaying names in the unchanged dropdown. The later frontend adapter must consume typed room options and submit IDs. Room remains optional; supplied ID maps to detail_kunjungan.id_ruangan. No name-based lookup or uniqueness assumption. Explicit clearing/omission semantics remain to freeze; report-filter migration must be documented separately from the selector decision.
