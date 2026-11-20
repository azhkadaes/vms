# 07: Booking tabs and dashboard counts come from the server

**What to build:**

- **Status tabs:** the booking list's tab counts come from the server's `statusCounts` (all, each stored status, and missed). They are no longer counted from the current page of at most seven rows. "Terlewat" is correct for the first time.
- **Dashboard cards:** use the server `metrics` for every role. The Receptionist's "Sedang berkunjung" card shows all checked-in visits, including overnight ones, so it matches the checkout queue it links to.
- **Fallback:** the page-based fallback stays only for a missing field, so a partly deployed backend still renders.

Context:
- `docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md` V-04, A-09, D-01, D-02 and §6 (new frontend defects)
- decision 03

**Blocked by:** 05 (both change the booking list parser).

**Backend needed:** backend ticket 09.

**Status:** ready-for-agent

- [ ] Tab counts match `statusCounts` across pages and filters, and the missed count is non-zero when missed visits exist (tested)
- [ ] Receptionist "Sedang berkunjung" equals `metrics.inside`, including visits checked in on an earlier day
- [ ] Dashboard cards for PA and Protocol Officer use `metrics.today`, `pending`, `inside` and `completed`
- [ ] The preview mock backend returns `statusCounts` and `metrics`, and the dashboard and booking tests pass
