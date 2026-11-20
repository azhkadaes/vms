# 05: Booking responses use `pejabat.nama` and the new staff fields

**What to build:** The public and staff booking parsers and screens read the official's name from `pejabat.nama` instead of `host` (decision 30). The label "Pejabat" stays.

The staff booking model accepts the fields the backend now defines:
- `attachmentAvailable`, `visitorPhotoAvailable`, `visitorCount`
- `createdBy {name, role}`, `serviceIdLast4`, `reference`, `restrictedView`, `visitKind`
- the notes and decision reasons, `ndName` and `serviceId`

Each field is used where its view allows. The restricted-record allow-list check stays strict, using the new key names.

The public booking model accepts `pejabat`, `roomId` and the fully defined history element `{at, action, publicMessage?}`. Timestamps are UTC `Z`. `room` is `""` when there is no room.

Context:
- decision 30
- `docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md` V-01, V-02, V-07 and §4 wire invariants
- the API decisions table in `.scratch/backend-api-2026-11-18/spec.md`

**Blocked by:** None (can start immediately).

**Backend needed:** backend tickets 06 (public) and 09 (staff).

**Status:** ready-for-agent

- [ ] Every booking view shows the official from `pejabat.nama`, and no parser still requires `host`
- [ ] The Nota Dinas and photo buttons, "Dibuat oleh…" and the approve/reject visibility all work from the new fields
- [ ] Restricted records for each view (`reception`, `arrival`, `assigned-pa`, `full`) parse with exactly their allowed keys; one extra key is still rejected (tested)
- [ ] Public lookup, receipt and history render from the new public model
- [ ] The preview mock backend returns the same shapes, and the contract and alignment tests pass
