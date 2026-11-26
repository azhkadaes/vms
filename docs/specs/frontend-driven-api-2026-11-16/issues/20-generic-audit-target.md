# Reference audit targets by entity type and ID

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Answer

User selected generic entity_type/entity_id references instead of a dedicated id_kunjungan FK. Recorded 18 November 2026. Server assigns a controlled entity type and target identifier; visit history selects the pair for the owning visit. This is separate from actor_type/id_pegawai, which describe who acted. A polymorphic reference does not provide an ordinary database FK to every target table; application validation and transactional writes must enforce target correctness. Exact SQL identifier type and controlled entity-type values remain to freeze. No audit CRUD API, migration or application edits are authorized by this choice.
