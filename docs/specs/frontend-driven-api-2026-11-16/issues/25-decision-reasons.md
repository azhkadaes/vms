# Store current decision reasons separately from appointment notes

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Answer

User confirmed dedicated Kunjungan fields for latest rejection/cancellation reasons and internal decision notes, with Audit Log retaining earlier events. Keep them separate from detail_kunjungan appointment notes. Field types follow current validation: reasons up to 1000 characters; internal notes up to 2000. Exact SQL names below are proposed implementation naming, not existing diagram columns. No migration or application edits executed.
