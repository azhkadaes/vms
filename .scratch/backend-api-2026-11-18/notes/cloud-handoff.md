# Cloud handoff: implement tickets 02–18

Instructions for a Claude Code cloud session on branch `vms/backend-api` of `azhkadaes/vms`. The user approved this work on 18 November 2026:
- Implement all remaining tickets, in parallel where the dependency graph allows, with **Opus** subagents.
- Use one branch per ticket, merged into an integration branch.
- Pushing to GitHub is allowed.

## State

- `backend/` holds the real VMS backend: the baseline plus tickets **00** (per-table migrations) and **01** (error and transport prefactor), both done.
- On the user's PC this code lives in a nested repo at commit `402e94f`. In this repo it is plain files.
- Tickets **02–18** are open:
  - ticket files: `.scratch/backend-api-2026-11-18/issues/`
  - dependency graph: the ticket table in `.scratch/backend-api-2026-11-18/spec.md` (each ticket's "Blocked by" line is authoritative)
- Reference:
  - `docs/BACKEND-GO-CONVENTIONS.md`
  - `docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md`
  - `docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md`
  - `.scratch/backend-api-2026-11-18/notes/build-environment.md`, which is written for the Windows PC (see "Adapt first")

## Adapt first (the PC setup does not exist here)

1. **Rewrite `build-environment.md` for Linux:**
   - Install Go **1.27.4** (exact; `go.mod` requires it) and PostgreSQL **18**.
   - Give each agent a throwaway cluster (`initdb` in a temp dir, trust auth, `127.0.0.1`, port `56000 + ticket`), with the roles and databases the notes list.
   - Run the contract checker with `python3`.
   - Drop the Windows paths, `GOCACHE`/`TMP` on D:, the cached-toolchain PATH trick, and the "nested backend repo" layout.
2. **Git layout:**
   - The integration branch is `vms/backend-api` of this repo.
   - Implementers use worktrees of this repo, branch `ticket/<NN>-<slug>`, and work inside `<worktree>/backend`.
   - The contract tooling lives in `docs/contracts` of the same tree, so each worktree has its own copy. Contract updates can then be part of each ticket branch, which removes the PC-era rule that mergers apply them.
3. **Adapt `.claude/workflows/implement-vms-backend-spec.js`** to these paths and to the layout above. Keep the design:
   - an Opus implementer per ready ticket, using the `tdd` skill;
   - serialized Opus mergers that run the full gate (gofmt, vet, vet with the integration tag, `go test`, integration suites against a throwaway cluster, contract check);
   - a final `code-review` pass with one fix pass, then cleanup.

   Run it with `done = ["00", "01"]`.

## Constraints

- No real credentials, no stage-1 database, no `.env`. Use only throwaway clusters.
- Ticket **03** rebases the contract pipeline. It must also apply the "Contract tooling follow-ups" listed in `build-environment.md`.
- Open product decisions D-01 to D-05 stay at their documented defaults. D-06 (email) is deferred to `.scratch/email-notifications-2026-11-18/spec.md`.
- Mark each ticket done in its file: `**Status:** done (merge <hash>)`, and tick only the verified criteria.
- Push the integration branch when the gate is green. Do not open a PR unless the user asks.
