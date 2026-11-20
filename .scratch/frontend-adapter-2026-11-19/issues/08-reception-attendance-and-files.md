# 08: Reception uses the single attendance route and credential-bound file reads

**What to build:**

- **Attendance:** check-in and check-out go through one attendance command. It sends `{action: "check_in" | "check_out", credential, version}` after the operator confirms identity (decision 31). The two separate action paths are removed.
- **Static QR:** the admission QR is static, so the same QR the visitor showed at check-in is scanned again at check-out, even on a later day.
- **Files at reception:** the Nota Dinas and visitor photo are opened with a POST carrying the already-entered code or QR as `{credential}` (decision 08). It is resent on every file request, with no extra screen. The credential never appears in a URL.
- **Unchanged:** the resolve step and its arrival projection (roster, purpose, file availability) stay as they are.

Context:
- decisions 08 and 31
- the static QR (`docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md`, decision 4)
- `docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md` V-03

**Blocked by:** 05 (both change the reception booking parser).

**Backend needed:** backend tickets 13 (resolve, attendance) and 14 (file reads).

**Status:** ready-for-agent

- [ ] Check-in and check-out both call the single attendance route with the right `action` (tested)
- [ ] A QR used for check-in checks the same visit out on a later day (tested against the preview mock backend)
- [ ] Reception file views use POST with the credential; no reception GET file request remains
- [ ] On 401, 403, 404 or 409 the resolved identity and credential are discarded and the operator resolves again
- [ ] The preview mock backend implements the attendance route and the credential POST, and the restricted-arrival and visitor-photo tests pass
