# Add visitor detail, identity type, duration and roster storage

Type: grilling
Label: wayfinder:grilling
Status: claimed
Assignee: current review agent
Parent: ../map.md
Blocked by: none

## Question

Specify the user-requested detail_pengunjung table, identity_type enum, visit_duration and nama_rombongan array without duplicating visitor ownership or changing frontend semantics.

## Confirmed input

17 November 2026 WITA: user requested `detail_pengunjung`, `identity_type enum`, `visit_duration time / double`, and `nama_rombongan array`. This authorizes documenting schema additions in the planning-first contract. It does not settle columns/FKs/cardinality, duration storage type, array owner, or database migration execution. Record version remains unresolved; it was not included in this answer.

## Remaining decisions

- Duration RESOLVED: INTEGER minutes, range 15–240, mapped to API duration; user confirmed detail_kunjungan owns it together with officer/date/time/room. Duplicate room FK on Kunjungan is omitted in proposed design only.
- detail_pengunjung ownership RESOLVED: user confirmed one detail record per visit, containing representative details and ordered nama_rombongan array. No separate row per companion. User subsequently confirmed it replaces Pengunjung's role in the proposed diagram; no reusable visitor profile and no table deletion/migration authorized.
- identity_type RESOLVED: detail_pengunjung stores the ordinary visitor document-type enum KTP/Paspor/SIM. User confirmed omitting ordinary nomor_identitas storage. Restricted serviceId/No ID Dinas remains a separate optional text concept.
- nama_rombongan owner RESOLVED: detail_pengunjung. Ordered names include representative first, maximum 20 including representative, following current frontend. Proposed PostgreSQL representation is TEXT[] with element validation; exact DDL has not been executed.
