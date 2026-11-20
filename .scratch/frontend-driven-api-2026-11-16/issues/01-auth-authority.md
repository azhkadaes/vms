# Reconcile current frontend authentication with Phase 1

Type: grilling
Label: wayfinder:grilling
Status: resolved
Assignee: current review agent
Parent: ../map.md
Blocked by: none

## Question

Should the new full-frontend contract describe current session login/read/logout, multiple assigned roles with one active role, role selection/switching and password-change workflows, superseding Phase 1's single-role fixed Bearer target? Full frontend coverage is confirmed, but the incompatible authentication designs must not be mixed.

## Evidence

`frontendx/packages/data/src/session.tsx`, `session-contract.ts`, `management-contract.ts`; `docs/contracts/API-CONTRACT.md` Phase 1 Precedence.

## Answer

16 November 2026: user explicitly selected current frontend login, session, logout, password-change and role-switching workflows. Multiple assigned roles are supported with one active role at a time. The new contract flags the older Phase 1 single-role authentication target as conflicting. This approves the contract direction, not application implementation or unobserved token/session security details.
