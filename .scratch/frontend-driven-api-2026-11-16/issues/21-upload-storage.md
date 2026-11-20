# Store uploaded files privately outside PostgreSQL

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Answer

18 November 2026: user selected private file/object storage. Uploads stores metadata and a private storage key; omit file_data from the proposed schema. Do not expose storage paths or public file URLs in ordinary DTOs. Existing authorized upload/read operations remain. Provider, storage-key column naming, lifecycle fields and migration of any existing files are not determined by this choice. No infrastructure provisioning, file deletion or database changes authorized.
