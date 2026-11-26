# Visitor Management System

VMS manages appointments and visitor access for the Nusantara Capital Authority (OIKN). It connects a public visitor portal with a staff workspace so a visit can be requested, reviewed, checked in and out, and reported on in one system.

## What people can do

- **Visitors** choose an official and a time, submit the visitor roster and required files, and use their booking code and email to check the result. Visitors can request an admission QR after approval. Visitors can cancel an eligible booking before it starts.
- **Pimpinan’s Assistants and Protocol Officers** manage visits within their assignments and permissions. They can create, approve, reject, reschedule or cancel eligible visits. The API checks for overlapping reservations for the same official or room.
- **Receptionists** resolve admission credentials and record check-in and checkout. The same QR can be used for checkout after a successful check-in, including for an overnight visit.
- **Super Admins** manage staff accounts and role permissions, and use the visit log to review and export records. Reports and visit details follow the active role’s access rules.

## How the system fits together

| Part | What it does |
| --- | --- |
| [Visitor and staff apps](frontendx/README.md) | React and TypeScript apps for public registration and staff operations. The local preview uses fictional data. |
| [VMS API](backend/README.md) | Go and Fiber service that enforces permissions, booking rules, admission, and file access. It serves 40 operations under `/v1`. |
| [Database](docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md) | PostgreSQL 18 stores visits, staff access, audit events, and operational records. Attachments are kept in private storage and read through the API. |

The backend implements the [40-operation REST contract](docs/contracts/FRONTEND-DRIVEN-REST-API-2026-11-16.md). [Validation](docs/contracts/VALIDATION.md) records the Go, PostgreSQL, and API checks. The local Docker launcher serves both browser apps through the real API. The [frontend adapter tickets](docs/specs/frontend-adapter-2026-11-19/map.md) track remaining workflow changes, including room selection, reception and report filters.

## Local development

- Start the [frontend preview](frontendx/README.md#run-locally) to try the visitor and staff flows with demo data.
- On Windows, open [Start VMS.cmd](Start%20VMS.cmd) to start PostgreSQL, the API and both browser apps. See the [Docker launcher guide](backend/docs/LOCAL-DEVELOPMENT.md#persistent-docker-on-windows) for testing and stopping them.
- Follow the [backend setup](backend/README.md#manual-setup) to configure PostgreSQL, run the migrations and start the API.
- See the [documentation index](docs/README.md) for requirements, the API contract and verification.
