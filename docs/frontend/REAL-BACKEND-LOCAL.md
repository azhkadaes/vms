# Use the real backend locally

**Current local setup, 11 November 2026:** the approved backend upgrade is complete and the site runs at `http://127.0.0.1:3010/admin/`, using the actual Docker API on 8081. Its sign-in page was browser-checked with no preview requests. To restart this particular site after Docker/backend startup, use `npm.cmd run serve:backend -- --api http://127.0.0.1:8081 --port 3010`. The generic selection instructions below also cover other isolated test backends; do not restart the completed database rehearsal drivers.

Goal: provide the existing visitor and staff screens at one local address using the selected real API, so administrator settings such as SMTP can be saved privately. Reuse production builds, the existing `/v1` contract and the existing forms. No UI, session, data-layer or dependency change is needed.

The launcher serves the two built applications on loopback and forwards `/v1` requests to an explicitly selected loopback API. It does not start a database, apply migrations, bootstrap accounts, start a worker or enable email. All decisions and records belong to that API/database; choose it deliberately.

## Before starting

- Review the backend/database version pair. The existing Docker stack was observed on migrations 001–006, while current source requires 001–009. Do not use this command as a substitute for the reviewed backend upgrade.
- Have a compatible backend running. Its `VMS_ORIGIN` must match the launcher address exactly, normally `http://127.0.0.1:3000`. A backend configured for another origin needs its own reviewed configuration change.
- Build both frontend workspaces. The built apps use actual API requests; the normal `npm run dev` remains the demo workflow on localhost.
- A local setup intended for real SMTP must have a reviewed notification queue. Do not enable delivery against retained rehearsal fixtures with invalid recipients.

## Run

From `D:\oikn\VMS\frontendx`:

```powershell
npm.cmd run build
```

Once a compatible backend has been started on the agreed API port, select it explicitly. For example, **only if that backend is actually running on 8082**:

```powershell
npm.cmd run serve:backend -- --api http://127.0.0.1:8082
```

Open `http://127.0.0.1:3000/admin/` for staff or `http://127.0.0.1:3000/` for visitors. Keep the terminal open; Ctrl+C stops only this frontend launcher. If using `--port 5175`, the backend origin must instead be exactly `http://127.0.0.1:5175`.

The API argument is required. The launcher rejects remote/credential-bearing upstream URLs, listens only on 127.0.0.1, and refuses a port already in use. It preserves API errors, request bodies, browser Origin, cookies and binary files. It has no preview-data endpoint and no request/credential logging. Missing assets return 404. It is a local setup utility; production needs the reviewed HTTPS gateway and operating configuration.

## Enter the email password

After staff bootstrap/login and Super Admin role selection, open **Integrasi → Email SMTP → Konfigurasi**. Enter the selected host, port 465, sender and username. Paste the generated Google app password locally into **SMTP password**. Stored secrets are not displayed again. Save/enable only when the chosen backend/worker configuration and notification queue have been reviewed.

## Verification scope

`npm.cmd run test:backend-site` checks transport preservation (Origin, cookies, request marker, binary upload/download and API errors), local-only host/upstream restrictions, asset failures and visitor/staff routing against a disposable local HTTP server and temporary synthetic files. It does not claim real SMTP delivery, database acceptance or production readiness. Existing real-API rehearsal evidence remains in the backend documentation.
