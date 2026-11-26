# ADR-0001: The preview server mirrors the production HTTP contract

Status: accepted (2026-11-26)

## Context

Preview mode (`DEV` on localhost) used its own request format: `?role=` impersonation, `/__preview` paths, room names instead of `roomId`, and `/checkin` instead of `/attendance`. The frontend covered the differences with `if (preview)` checks in about 31 files. Two production bugs (ebc271c, 15a1653) sat in those checks, and preview hid one of them by accepting requests the Go backend rejects.

## Decision

The HTTP contract is the only seam between the frontend and a backend. The Go backend and `tooling/preview.ts` are the two adapters behind it. The preview server accepts the same paths, bodies and rules as the Go backend, so frontend code needs no preview checks. The two may differ only in the base URL and in how a staff identity is established (to be closed in a later step).

## Consequences

- Each feature or data module that drops its preview checks shrinks the surface where bugs specific to one adapter can hide.
- `preview.ts` must follow every contract change. The same integration tests should run against both adapters.
- Preview-only conveniences (such as the role switcher) must work through the contract, not through checks in feature code.
