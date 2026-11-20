# 06: Room IDs and officer/room requiredness in booking forms

**What to build:** Staff booking forms pick rooms by ID.

- **Room catalogs:** `{id, name, active}` objects on both catalogs. Dropdowns show the name and submit `roomId`.
- **Existing visits** (approve, reschedule, restricted edit):
  - leaving the room unchanged omits `roomId`, which keeps the room;
  - choosing "no room" sends `null`, which clears it;
  - picking a room sends its ID.
- **New visits:** treat omitted and `null` the same, meaning no room.
- **Requiredness** (decision 10):
  - the officer is required for every visit, including Terbatas, and can no longer be cleared;
  - the room is optional for every visit and stage, including approval.
- **Display:** booking views keep showing the room name from `room`.

Context:
- decisions 10, 14 and 15
- the room semantics in `docs/contracts/APPROVED-SCHEMA-TARGET-2026-11-18.md`

**Blocked by:** 05 (both change the booking parsers).

**Backend needed:** backend tickets 04 (staff catalog), 10 (create), 11 (approve) and 12 (reschedule, restricted update).

**Status:** ready-for-agent

- [ ] The public and staff catalogs parse room objects, and the room dropdowns submit IDs (tested)
- [ ] Approve, reschedule and restricted edit:
  - [ ] send no `roomId` when unchanged
  - [ ] send `null` when cleared
  - [ ] send the ID when picked (tested)
- [ ] The restricted form requires an officer, and the ordinary forms no longer require a room
- [ ] An empty string is never sent as a room ID
- [ ] The preview mock backend applies the same omit/null/ID rules, and the reservations and restricted tests pass
