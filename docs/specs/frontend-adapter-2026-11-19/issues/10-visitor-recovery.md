# 10: Visitor app recovers from a stale QR and an outdated privacy notice

**What to build:** Two visitor-app defects found during validation.

- **QR request conflict:** when the admission QR request returns 409, the app runs the status lookup again to get the current version. Today it retries forever with the same stale version until the visitor presses "Perbarui".
- **Outdated privacy notice:** when booking submission is refused because the notice changed (`notice_outdated`), the app:
  - refetches the catalog;
  - shows the new notice;
  - clears the consent checkbox so the visitor must accept the current version;
  - keeps the rest of the form.

Context:
- `docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md` §6 (new frontend defects)
- the static QR rules (`docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md`, decision 4)

**Blocked by:** 05 (both change the public booking parser).

**Backend needed:** backend tickets 06, 07 and 08.

**Status:** ready-for-agent

- [ ] A 409 on the QR request triggers one lookup and then a QR request with the fresh version, with no loop (tested)
- [ ] A `notice_outdated` refusal refetches the catalog, shows the new notice and resets consent; resubmitting sends the new `privacyVersion` (tested)
- [ ] The static QR shows the same image across refreshes until the visit changes
- [ ] The preview mock backend reproduces both conflicts, and the public-submission and public-history tests pass
