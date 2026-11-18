# 04: Staff catalogs (op 08)

**What to build:** Staff booking forms and the employee editor get the officers and rooms they may choose from, under the caller's authority:
- officers: id, name, vip, active, jobTitle, directorate, deputy
- rooms: id, name, active

This is the first operation through the protected staff route group. Scope follows A-15:
- Roles with the bookings or people permission get active officers and rooms.
- For callers with the people permission, inactive officers still assigned to an employee are also included, marked inactive.

Context:
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-05, A-15
- [operation reference](../../../docs/contracts/FRONTEND-DRIVEN-REST-API-2026-11-16.md): op 08

**Blocked by:** 01, 02, 03.

**Status:** ready-for-agent

- [ ] `GET /v1/staff/catalogs` requires a ready staff session:
  - [ ] 401 when unauthenticated
  - [ ] 403 when the session is pending
  - [ ] 403 when both the bookings and people permissions are absent
- [ ] Officers include jobTitle, directorate and deputy when known; IDs are decimal strings; empty collections are `[]`
- [ ] Rooms are `{id, name, active}` objects, and the privacy notice is present as on the public catalog
- [ ] Inactive officers with an existing assignment appear only for callers with the people permission, marked `active:false`
- [ ] Usecase, HTTP and contract-capture tests pass, and op 08 is marked implemented in the pipeline
- [ ] The operation reference is corrected:
  - [ ] op 01 example and field table include `rooms`, `directorate` and `deputy`
  - [ ] op 08 tables include `directorate` and `deputy`
