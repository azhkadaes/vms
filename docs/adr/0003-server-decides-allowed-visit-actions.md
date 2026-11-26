# ADR-0003: The server decides which visit commands a staff member may run

Status: accepted (2026-11-26), not yet implemented

## Context

The frontend's `can()` in `packages/data/src/domain.ts` re-derives the Go visit policy (`internal/policy/visit.go`), including the 7-day window for rescheduling a missed visit. The server already sends `timing.canCheckIn`, `canCheckOut` and `canCancel`, but not approve or reschedule. Per-role field visibility for restricted visits is also copied three times (Go serializer, `wire.ts`, `domain.ts`).

## Decision

Staff visit responses carry `allowedActions: VisitCommandName[]`, computed from the Go policy. The client checks `allowedActions.includes(...)` and no longer re-derives policy. Field visibility for restricted visits is defined by one table in Go; the frontend only checks the response shape.

## Consequences

- This changes the API contract (`backend/docs/openapi.json` and `wire.ts`).
- `can()` and the policy parts of `toStaffBooking` move into the preview server as fake behaviour (see ADR-0001).
