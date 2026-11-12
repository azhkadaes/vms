# Frontend deployment and acceptance

Scope: frontendx only. No deployment is performed by this audit. A new backend at D:/oikn/VMS/backend still needs implementation against API-CONTRACT.md.

Build both workspaces with npm run build. Serve visitor dist at / and staff dist at /admin/ on the same HTTPS origin. Staff base is compiled as /admin/. Direct visitor task URLs fall back to visitor index.html; /admin/* staff routes fall back to staff index.html. Missing assets must return 404 instead of HTML. Redirect /admin to /admin/ before SPA routing.

Route /v1/* to the new service before either SPA fallback. API 401/403/409/422/429/5xx must remain API responses, with no login HTML or synthetic success substituted by the host. Default public VITE_API_BASE_PATH=/v1 can be changed consistently at build/host time; it must remain a safe same-origin path. No VITE_* secrets.

Hash-named static assets may use long immutable caching; HTML and private API/files must not. Configure MIME types, compression, TLS, Secure/HttpOnly/SameSite cookies, appropriate CSP/referrer/nosniff/frame policy, and CSRF/Origin validation together with the service. Verify actual headers in the deployed browser. The local contract fixture is not a security-header deployment test.

Visitor image delivery uses AVIF, then WebP, then the preserved JPEG/PNG fallback. Serve `.avif` as `image/avif` and `.webp` as `image/webp`. The built visitor HTML contains a small inline, landing-only responsive image preload before stylesheets; include the exact generated script's SHA-256 hash in a restrictive CSP rather than enabling unrestricted inline scripts. Recompute the hash after every build that changes its asset URLs. Confirm that one hero candidate loads on `/` and none loads on direct `/booking` or `/status` navigation. Enable Brotli/gzip for HTML/JS/CSS at the host and verify Content-Encoding and Vary: Accept-Encoding; precompressed image formats do not benefit from another compression pass.

Before enabling genuine submissions, publish the versioned ID/EN privacy notice and approve submitted-data/file/audit retention and rights contacts. The public form blocks creation when the service has no notice. Apply the G19 initial capacity/file limits on both client and server. Review upload quarantine/content inspection, orphan cleanup and protected download. Inspect logs to ensure credentials, raw tokens, passwords and file contents are absent.

Acceptance must include real session lifecycle/reassignment, version conflicts and concurrent approvals, durable submission recovery, notification outbox/retries, actual storage/access denial, admission revocation/single use, service-time arrival/cancellation, backups and restart recovery. Test the built frontend against these services, not only the disposable fixture.

Run npm run check from a clean install. The included GitHub Actions workflow assumes frontendx is the repository root; if the surrounding VMS repository is used, move the workflow to its root and set working-directory/cache/artifact paths deliberately. The workflow has been provided, not executed on a remote runner in this audit.

Perform real-device/narrow-screen, keyboard, screen-reader, actual browser zoom, Safari and physical camera/scanner acceptance where applicable. Record real performance under the intended connection/device. Automated axe and local bundle sizes do not establish full WCAG conformance or field Core Web Vitals.

G14 draft recovery is deferred. G20 staff cancellation is implemented in the frontend/fixture and still requires real-service permission, audit, version and admission-revocation acceptance. Do not call release scope complete or activate live operations while required behavior/integration acceptance remains unresolved.
