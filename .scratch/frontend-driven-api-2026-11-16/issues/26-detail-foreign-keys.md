# Reference visit details through unique foreign keys

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Answer

User confirmed Kunjungan references detail_pengunjung and detail_kunjungan through unique foreign keys, following the diagram direction and preventing shared editable details across visits. Proposed names: id_detail_pengunjung and id_detail_kunjungan. Nullability for restricted visitor details remains to finalize. UNIQUE prevents sharing but does not by itself prevent orphan detail rows; creation/rollback and lifecycle handling must enforce ownership. No migration executed.
