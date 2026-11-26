# 03: Error codes drive conflict and validation messages

**What to build:** Every screen reacts to the backend's error `code`, not only the HTTP status. The body is `{code, message, requestId, fieldErrors?}`, where `fieldErrors` maps flat form keys to lists of Indonesian messages.

- **Conflicts:** each 409 cause gets its own handling:
  - `schedule_conflict`: the slot is taken
  - `version_conflict`: the data changed, so reload
  - `idempotency_conflict`: the request was already sent with different data
  - `notice_outdated`: the privacy notice changed
  - `invalid_state`, `reschedule_window_closed`, `credential_expired` and `revision_conflict`: as their names say
- **Field errors:** every message for a field shows on that input.
- **Server messages** are shown as written. `requestId` is available for support, for example in a copyable detail.
- **429:** shows the `Retry-After` wait.

Context:
- the error-code catalogue in `docs/BACKEND-GO-CONVENTIONS.md` §5
- `docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md` V-13 and A-14

**Blocked by:** 01 (both change the shared HTTP client).

**Backend needed:** ticket 01 is merged. The booking codes appear as backend tickets 06–16 land.

**Status:** ready-for-agent

- [ ] Staff create shows "schedule conflict" only for `schedule_conflict`; other 409 codes show their own message and recovery (tested)
- [ ] Every form maps each `fieldErrors` entry to its input and shows all messages for that field
- [ ] Unknown codes fall back to the server `message`, and the UI never shows a raw code or stack
- [ ] 429 responses show the wait from `Retry-After`
- [ ] The preview mock backend emits the same codes, and the existing form tests are updated
