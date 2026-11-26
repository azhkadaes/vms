# VMS environment example audit

The old Monogo example contained 35 settings. The table records what the VMS backend actually honors. The [current example](../.env.example) preserves the original settings and lists additional VMS options below them as comments. [Configuration loader](../config/config.go), [HTTP server](../internal/server/rest/server.go)

| Original key | VMS behavior |
| --- | --- |
| `APP_NAME` | Legacy key. The HTTP server is named VMS. |
| `APP_PORT` | Listener port. Defaults to 8080. |
| `APP_ENVIRONMENT` | Accepts development, test, or production. Affects database TLS and staff origin validation. |
| `APP_DEBUG` | Legacy key. VMS uses its normal error handling and logging. |
| `APP_HOST` | Listener hostname or IP without a port. Defaults to 127.0.0.1. |
| `APP_SCHEME` | Legacy key. The API listens over HTTP. Use the gateway for upstream HTTPS and `STAFF_ORIGIN` for the staff browser origin. |
| `APP_READ_TIMEOUT` | Legacy key. The normal read timeout is 15 seconds. Uploads have a 60-second receive timeout. |
| `APP_WRITE_TIMEOUT` | Legacy key. The write timeout is 20 seconds. |
| `APP_IDLE_TIMEOUT` | Legacy key. Idle HTTP connections time out after 60 seconds. |
| `APP_BODY_LIMIT` | Legacy key. JSON is capped at 256 KiB. Multipart upload routes have a separate limit. |
| `DB_HOST` | Database hostname or IP without a port. The example uses loopback for manual startup. |
| `DB_PORT` | Database port. Defaults to 5432. |
| `DB_USER` | Required database user. Use `vms_api` for the application. |
| `DB_PASSWORD` | Required database password or `DB_PASSWORD_FILE`. Enter the raw value and the loader escapes it for the connection URL. |
| `DB_NAME` | Required database name. The local stack uses `vms_stage1`. |
| `DB_SSL_MODE` | Defaults to `verify-full`. `disable` is accepted on loopback or the Docker host named `db` outside production. |
| `DB_MAX_IDLE_CONNS` | Legacy key. The pool allows five idle connections. |
| `DB_MAX_OPEN_CONNS` | Legacy key. The pool allows ten open connections. |
| `DB_CONN_MAX_LIFETIME` | Legacy key. The connection lifetime is 30 minutes. |
| `LOG_LEVEL` | Legacy key. The service uses Go's default structured logger. |
| `LOG_FORMAT` | Legacy key. The running process uses the default Go logger output format. |
| `LOG_OUTPUT` | Legacy key. The backend does not select a log destination from the environment. |
| `CORS_ALLOWED_ORIGINS` | Legacy key. The HTTP server has no configurable CORS middleware. Staff browser requests use the exact `STAFF_ORIGIN`. |
| `CORS_ALLOWED_METHODS` | Legacy key. Methods follow the registered routes. |
| `CORS_ALLOWED_HEADERS` | Legacy key. There is no environment-controlled CORS allowlist for headers. |
| `CORS_ALLOW_CREDENTIALS` | Legacy key. Staff cookies and request guards follow the same-origin design. |
| `CORS_MAX_AGE` | Legacy key. There is no configurable CORS preflight cache. |
| `JWT_SECRET_KEY` | Enables staff authentication as part of the staff configuration block. Requires at least 32 random bytes or a `_FILE` equivalent. Other staff keys, origin and gateway networks must be set together. |
| `JWT_ACCESS_TOKEN_EXPIRY_IN_HOURS` | Legacy key. Access tokens last up to five minutes, bounded by session expiry. |
| `JWT_REFRESH_TOKEN_EXPIRY_IN_DAYS` | Legacy key. Staff sessions have an eight-hour idle timeout and a 24-hour absolute lifetime. |
| `JWT_TOKEN_ISSUER` | Supported when staff auth is enabled. Defaults to `vms`. |
| `JWT_TOKEN_AUDIENCE` | Supported when staff auth is enabled. Defaults to `vms-staff`. |
| `RATE_LIMIT_MAX_REQUESTS` | Legacy key. Limits depend on the operation. The required `RATE_LIMIT_KEY` is the digest key for identifiers used in limiting and booking recovery. |
| `SWAGGER_USERNAME` | Optional. Set it with the Swagger password to enable the protected docs. |
| `SWAGGER_PASSWORD` | Optional. Set it with the username or supply `SWAGGER_PASSWORD_FILE`. Both remain blank to disable Swagger. |

For working manual startup values, see the [backend setup](../README.md#manual-setup). Docker Compose injects its own settings and mounts private secrets. [Database pool](../pkg/helper/database/gorm_connection.go), [Staff authentication loader](../config/auth.go), [Session lifetimes](../internal/usecase/auth/implementation/session_usecase.go), [Docker setup](LOCAL-DEVELOPMENT.md)
