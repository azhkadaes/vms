# 09: Staff booking reads (ops 09, 10)

**What to build:** PA, Protocol Officer and Receptionist get one scoped operational list, and PA and Protocol Officer also get booking detail.

The list has three views:
- `list`: filters, sorts and status tab counts;
- `dashboard`: metrics;
- `reception`: the check-in and checkout queues.

Booking detail is projected per role. Restricted records carry exactly the keys their view allows.

The policy gains scope, ownership and projection rules mirroring the UI's own rules. It also gains StaffBooking fields the contract lacked:
- file availability;
- visitor count;
- creator;
- last four characters of the service ID;
- reference;
- restricted view.

Context:
- [validation](../../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): V-01, V-02, V-04, V-14, A-09, A-10, A-20 and the §4 wire invariants
- decisions 03, 12, 23, 30
- open decisions D-01 and D-02 use their defaults

**Blocked by:** 04, 06.

**Status:** ready-for-agent

- [ ] **Query allow-list:** view, q, status, date, vip, sort, dir, page, pageSize, mode.
  - [ ] unknown key → 400
  - [ ] pageSize 1–100
  - [ ] a page past the end → 200 with empty items
- [ ] **Scope:**
  - [ ] mirrors the UI scope rules: PA sees assigned non-VIP visits plus restricted visits in the assigned-pa view; Receptionist gets reception projections
  - [ ] booking detail is refused for Receptionist (403)
  - [ ] out-of-scope records → 404
- [ ] **`view=list`:**
  - [ ] sorts: priority, name, organization, host, time, status, with stable ties
  - [ ] `vip=0` means no filter; `vip=1` means VIP ordinary only
  - [ ] search matches only fields visible to the caller
  - [ ] `total ≤ totalScoped`, where `totalScoped` ignores q, status, vip and date
  - [ ] `statusCounts` includes missed and ignores the status filter
- [ ] **`view=dashboard`:** metrics per A-09 (`inside` spans dates) and rows per A-09
- [ ] **`view=reception`:**
  - [ ] `mode=checkin`: approved, scheduled on the date, not missed
  - [ ] `mode=checkout`: checked_in on any date
- [ ] **StaffBooking fields:** includes the V-01 fields
  - [ ] `reference` is `TRB-<id>` for restricted visits
  - [ ] `createdBy` comes from the creation audit event: the staff actor's name and the role recorded there
  - [ ] `visitorCount` is `jumlah_pengunjung`; `vip` and `visitKind` come from `tipe_kunjungan`
- [ ] **Restricted allow-list:** exact key-set tests for the reception, arrival, assigned-pa and full views
- [ ] **Consistency:**
  - [ ] version is identical across list and detail
  - [ ] one server timestamp drives timing, missed, counts and metrics in a response
- [ ] **Serializer check:** output decodes with the frontend's staff schema (apart from known-deferred fields)
