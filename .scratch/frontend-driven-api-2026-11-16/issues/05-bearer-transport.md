# Choose staff authentication transport

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Question

Preserve the frontend's cookie-only requests or adopt the backend's Bearer token transport without changing screens?

## Answer

16 November 2026: user selected Bearer tokens, keeping backend approach and documenting later frontend connection changes. This does not revert the confirmed multiple-role, role-switching, session/logout and password-change workflows to older Phase 1 restrictions. It does not itself settle automatic refresh, token persistence, or expiry behavior.

## Evidence

`frontendx/packages/data/src/http.ts`, `session.tsx`; `backend/internal/server/rest/middleware/auth.go`; `backend/pkg/dto/auth.go`; `backend/internal/handler/auth_handler.go`.
