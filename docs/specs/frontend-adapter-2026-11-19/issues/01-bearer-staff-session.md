# 01: Bearer staff session: login, restore and logout

**What to build:** Staff sign in, reload and sign out against the real backend.

- **Login:** captures the TokenResult (`accessToken`, `tokenType`, `accessExpiresAt`, `session`) instead of discarding it. The access token lives only in memory.
- **Authorization header:** every staff API request carries `Authorization: Bearer <token>`. Public visitor requests and third-party URLs never do.
- **Lifecycle routes:** login, refresh and logout follow the backend's cookie and `X-CSRF-Token` rules, including the anonymous initialization the backend expects before login.
- **Reload:** restores the session through the refresh operation while the server session is valid. Simultaneous requests share one refresh, and a retried uncertain refresh reuses its `refreshRequestId`.
- **Expiry and logout:** an expired server session returns to login. Logout clears the token and private caches, and a reload after logout cannot restore the session.
- **Unchanged:** screens and workflows. The preview mock backend moves to the same transport.

Context:
- decisions 05 and 06 in `docs/specs/frontend-driven-api-2026-11-16/issues/`
- the Authentication section of `docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md` (session restoration and acceptance cases)
- `docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md` §6

**Blocked by:** None (can start immediately).

**Backend needed:** ops 32, 33, 35 and 39 already exist; backend tickets 00–02 are merged.

**Status:** ready-for-agent

- [ ] Login stores the access token and expiry in memory only, never in localStorage, sessionStorage or cookies set by JavaScript
- [ ] Staff API requests send the Bearer header, and public and third-party requests never do (tested)
- [ ] Login, refresh and logout send the CSRF token and complete against the backend's origin and cookie checks
- [ ] Reload with a valid server session restores the user without a login screen
- [ ] Parallel requests during token expiry trigger exactly one refresh, and a retried refresh keeps its `refreshRequestId` (tested)
- [ ] An expired or revoked server session leads to the login screen, and private caches are cleared
- [ ] Logout prevents restoration on the next reload
- [ ] The preview mock backend implements the same transport, and the existing auth browser and integration tests pass
