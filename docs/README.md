# Project documentation

- [REST API operation reference](contracts/FRONTEND-DRIVEN-REST-API-2026-11-16.md): the current 40-operation frontend-driven contract, with [field rules](contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md). Each operation's implemented/missing status is in [the contract manifest](contracts/manifest.json), the implemented shapes in [the source baseline](contracts/source-baseline.json), and the gate in [contract validation](contracts/VALIDATION.md).
- [Phase 1 API contract (superseded)](contracts/API-CONTRACT.md): the earlier 29-operation target, kept as a record.
- [Super Admin Log Kunjungan addition](contracts/SUPER-ADMIN-VISIT-LOG.md): combined visit log, filtered analytics, and complete CSV/PDF export; includes the new reporting projection and production endpoint requirements.
- [UI ↔ contract validation, 18 November 2026](contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): every reachable UI data need checked against the 40-operation frontend-driven contract; gaps, defaults adopted and open product decisions.
- [Physical schema and per-table migrations, 18 November 2026](contracts/PHYSICAL-SCHEMA-2026-11-18.md): every table following the user's DB diagram column for column, each agreed departure cited, migration order and the stage-1 reset procedure.
- [Deferred email notification spec](../.scratch/email-notifications-2026-11-18/spec.md): the later plan for visitor emails through a PostgreSQL outbox, a worker and a mailgrid SMTP adapter.
- [Backend Go conventions](BACKEND-GO-CONVENTIONS.md): the module template, layering, transactions, errors and test seams new backend code follows. The implementation spec and proposed tickets are in `.scratch/backend-api-2026-11-18/spec.md`.
- [Frontend documentation](frontend/): requirements, API contract, deployment, design, verification, and historical reviews.
- [Manual setup guide](MANUAL-SETUP-GUIDE.md): local setup for the application.
- Backend planning, API contracts, checkpoints, and supporting evidence are in this folder and its evidence subfolders.

The frontend and backend READMEs remain in their respective project directories as entry points. The backend's generated API documentation remains in `backend/docs/` because it may be used by the Go application.
