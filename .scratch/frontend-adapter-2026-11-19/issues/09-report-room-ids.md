# 09: Report room filters use room IDs

**What to build:** The Super Admin visit log and the analytics filters send room IDs under the existing `room` key. Several rooms are sent as repeated keys. The empty value still means "no filter".

- **Options:** room options arrive as `{id, name}` and show the name.
- **Rows:** visit-log rows gain `roomId` next to the display `room`. The strict row parser is updated in the same change.
- **Export:** uses the same selected IDs and the same revision as the list.

Context:
- decision 24
- the report room filter part of `docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md`
- `docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md` V-09 and V-10(b)

**Blocked by:** None (can start immediately).

**Backend needed:** backend tickets 15 and 16. Backend 16 starts emitting row `roomId` only when this ticket ships; coordinate the release.

**Status:** ready-for-agent

- [ ] Selecting rooms sends their IDs as repeated `room` keys, and the echoed `filter.room` still matches what was sent (tested)
- [ ] Room options render from `{id, name}`, including inactive historical rooms
- [ ] Visit-log rows parse with `roomId` under the strict schema
- [ ] CSV and PDF exports use the same ID filters and revision
- [ ] The preview mock backend filters by room ID, and the visit-log and reports tests pass
