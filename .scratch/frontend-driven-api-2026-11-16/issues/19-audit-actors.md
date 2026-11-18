# Represent staff, visitor and system audit actors

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Answer

17 November 2026: user confirmed explicit non-staff audit actors. Proposed actor_type values are staff, visitor, system. id_pegawai is required for staff and nullable for visitor/system actions; do not fabricate employee accounts. Actor classification and authenticated identity are set by the server, never trusted from request bodies. Public history remains an allowlisted projection, not raw audit records. No schema migration executed.
