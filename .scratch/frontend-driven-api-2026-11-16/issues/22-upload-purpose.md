# Bind uploads to document or visitor-photo purpose

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Answer

18 November 2026: user confirmed an explicit purpose field with values document/visitorPhoto in upload requests and metadata. Validate file content against purpose; bind issued upload token to that purpose and reject cross-purpose attachment. Keep one upload endpoint per audience and existing screens. No application/schema migration performed.
