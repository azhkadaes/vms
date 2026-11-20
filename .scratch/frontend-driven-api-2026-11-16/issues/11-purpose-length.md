# Preserve the frontend purpose length

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Question

Expand diagram keperluan VARCHAR(255) to match the current frontend's 5–1000 characters, or shorten the frontend limit?

## Answer

17 November 2026: user confirmed detail_kunjungan.keperluan VARCHAR(1000), mapped to API purpose:string, trimmed length 5–1000 for ordinary visits. Restricted creation does not collect purpose; this does not introduce a new required restricted field. No schema migration performed.
