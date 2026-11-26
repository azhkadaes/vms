# 02: Role switch, password change and activity renewal adopt server tokens

**What to build:**

- **Role switch and password change:** both return a fresh TokenResult. The client adopts the new token before any later request, takes UI state from `result.session`, and clears role-scoped caches. Old-role credentials must never pair with the new role's screens.
- **Activity renewal:** real user activity renews the server's inactivity window through the renew operation, coalesced to at most once a minute. Polling, background refresh, window visibility and timers never count as activity.
- **Wrong current password:** shows a field error on the current-password input, from the 422 `fieldErrors.currentPassword`. It no longer signs the user out.
- **Absolute expiry:** at the 24-hour limit the user returns to login. A preserved form is never submitted under a new identity or role.

Context:
- decision 07
- the "Confirmed session restoration" and session-limits parts of `docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md`
- `docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md` V-10(a)

**Blocked by:** 01.

**Backend needed:** ops 34, 36 and 40 exist; backend ticket 01 (422 `currentPassword`) is merged.

**Status:** ready-for-agent

- [ ] After a role switch, the next request uses the new token, and no cached data from the old role is shown (tested)
- [ ] After a password change, the new token is used, and the mandatory-password-change restriction lifts only when the server says so
- [ ] A wrong current password shows the field error and keeps the user signed in (tested)
- [ ] Renewal is sent only for real interaction, at most once a minute; idle tabs and background refresh never renew (tested with fake timers)
- [ ] At absolute expiry the user must sign in again, and unsent forms are not replayed
- [ ] The preview mock backend returns TokenResult for role and password, and the related browser tests pass
