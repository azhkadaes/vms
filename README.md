# Visitor Management System (VMS)

VMS handles visits to OIKN officials, from booking to check-out. It has a public visitor portal and a staff backoffice.

## Features

- **Visitors** book a visit with an official, upload the required files, and check their booking with a code and email. Approved visits get an admission QR code.
- **Pimpinan's Assistants and Protocol Officers** approve, reject, reschedule or cancel visits.
- **Receptionists** check visitors in and out with the visit code or QR.
- **Super Admins** manage staff roles and export the visit log.

## Tech stack

| Part | Folder | Tech |
| --- | --- | --- |
| Visitor and staff apps | `frontendx` | React, TypeScript, Vite |
| API | `backend` | Go, Fiber, served under `/v1` |
| Database | — | PostgreSQL 18 |

In Docker, a small Node gateway serves both apps and forwards `/v1` to the API.

## Getting started

### Requirements

- Windows
- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- Node 22, for frontend development
- Go 1.27.4, for backend development

### Run with Docker

1. Open Docker Desktop.
2. Double-click **`Start VMS.cmd`**. The first time, add fictional officials and rooms with:

   ```powershell
   & '.\Start VMS.cmd' -SeedTestCatalog
   ```

3. Open the apps:

   | App | URL |
   | --- | --- |
   | Visitor | http://localhost:58101/ |
   | Staff | http://localhost:58101/admin/ |
   | API health | http://localhost:8080/health/ready |
   | Swagger | http://localhost:8080/swagger/index.html |

The first start creates the secrets in `backend/.runtime/local/`, builds the images, starts PostgreSQL, runs the migrations, and starts the API and frontend. Data stays in the Docker volumes `vms-postgres` and `vms-uploads`.

| Task | Command |
| --- | --- |
| Stop | `Stop VMS.cmd` |
| Rebuild after pulling code | `& '.\Start VMS.cmd' -Rebuild` |
| API and migration logs | `docker compose -p vms logs --tail 50 api migrate` |

### Frontend only (demo data)

```powershell
cd frontendx
npm.cmd ci
npm.cmd run dev
```

Visitor: http://127.0.0.1:3000/. Staff: http://127.0.0.1:3002/admin/. It runs without a backend and doesn't save anything.

## Configuration

Settings live in `backend/.env`. Create it with `.\vms.ps1 env-setup` from `backend`. The Docker launcher reads only these:

| Key | Default | Notes |
| --- | --- | --- |
| `APP_PORT` | `8080` | API port. `8180` and `58101` are taken by the frontend. |
| `SWAGGER_USERNAME`, `SWAGGER_PASSWORD` | blank | Set both to turn on Swagger. |

The full list is in [Environment settings](backend/docs/ENVIRONMENT-CONFIGURATION.md).

## Database migrations

The Docker launcher runs migrations on every start, before the API starts. There's no down migration.

To check them by hand, from `backend`:

```powershell
$env:VMS_PRIVATE_DIR = './.runtime/local'
$env:VMS_STAGE1_IMAGE = 'vms-backend'
docker compose -p vms -f docker-compose.yml -f docker-compose.local.yml run --rm migrate -mode status -expect-database vms_stage1
```

## Development

Backend commands, run from `backend`. Every Makefile target has a Windows version:

| Make | Windows | What it does |
| --- | --- | --- |
| `make env-setup` | `.\vms.ps1 env-setup` | Creates `.env` from `.env.example` if it's missing |
| `make build` | `.\vms.ps1 build` | Builds `bin\vms-api.exe` |
| `make run` | `.\vms.ps1 run` | Runs the API from source |
| `make start` | `.\vms.ps1 start` | `env-setup`, then build, then run |
| `make swagger` | `.\vms.ps1 swagger` | Regenerates `docs/openapi.json` |
| `make test` | `.\vms.ps1 test` | Runs the Go tests |
| `make lint` | `.\vms.ps1 lint` | Runs `go vet` |
| `make migration-plan` | `.\vms.ps1 migration-plan` | Lists migrations without a database |
| `make contract-check` | `.\vms.ps1 contract-check` | Checks the code against `docs/contracts` (needs Python and Node) |

Frontend commands, run from `frontendx`:

| Command | What it does |
| --- | --- |
| `npm.cmd run dev` | Starts both apps with demo data |
| `npm.cmd test` | Runs the unit tests |
| `npm.cmd run build` | Builds both apps |
| `npm.cmd run check` | Format, types, tests, builds, browser checks |

## Production

The Docker launcher is for local use only. It runs in development mode, doesn't encrypt the database connection, and uses `localhost` addresses. A production deployment needs:

- PostgreSQL 18 with `verify-full` TLS, and the roles `vms_migrator` and `vms_api`
- `APP_ENVIRONMENT=production` and an HTTPS `STAFF_ORIGIN`
- The frontend image behind an HTTPS proxy that keeps the original `Host` header

See the [backend README](backend/README.md#manual-setup) and the production notes in the [Docker launcher guide](backend/docs/LOCAL-DEVELOPMENT.md#persistent-docker-on-windows).

A new database has roles and permissions but no staff accounts. Add those directly in the database.

## Project structure

```text
backend/      Go API, migrations, Docker files, ops scripts
frontendx/    Visitor app, staff app, shared packages, tests
docs/         Requirements, API contract, decisions
Start VMS.cmd / Stop VMS.cmd   Local Docker launcher
```

## Documentation

- [Frontend](frontendx/README.md)
- [Backend](backend/README.md)
- [Docker launcher guide](backend/docs/LOCAL-DEVELOPMENT.md#persistent-docker-on-windows)
- [Docs index](docs/README.md)
