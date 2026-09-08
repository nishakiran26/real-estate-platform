# Environment Variables

Every setting is read once, in `backend/internal/config`. Nothing else in the
application calls `os.Getenv`, so a deployment is configured in exactly one
place.

Precedence: **real environment variables always win** over a `.env` file. The
`.env` loader exists so a local `go run` behaves like a container; Docker,
Kubernetes and CI keep full control.

Copy [`.env.example`](../.env.example) to `.env` to start.

---

## Application

| Variable | Default | Purpose |
|---|---|---|
| `APP_NAME` | `RERA Property Platform` | Shown in emails and page titles |
| `APP_ENV` | `development` | `development` or `production`; production enables strict validation |
| `APP_URL` | `http://localhost:8080` | Public URL of the API; used in robots.txt and storage URLs |
| `FRONTEND_URL` | `http://localhost:5173` | Public URL of the SPA; used in emails, share links and the sitemap |
| `PORT` | `8080` | Listen port |
| `SERVER_READ_TIMEOUT` | `30` | Seconds |
| `SERVER_WRITE_TIMEOUT` | `60` | Seconds |

## MySQL

| Variable | Default | Purpose |
|---|---|---|
| `DB_HOST` | `localhost` | Inside Docker Compose this is the service name, `mysql` |
| `DB_PORT` | `3306` | |
| `DB_USER` | `root` | |
| `DB_PASSWORD` | *(empty)* | Required in production |
| `DB_NAME` | `rera_properties` | Created automatically if missing |
| `DB_AUTO_MIGRATE` | `true` | Create and update tables on startup |
| `DB_SEED` | `true` | Load roles, the administrator, cities, amenities and pages |
| `MYSQL_ROOT_PASSWORD` | — | Used by the MySQL container only |

## Redis

Redis is optional. When it is unreachable the platform keeps serving: caching
falls back to direct queries and rate limiting to an in-process counter.

| Variable | Default |
|---|---|
| `REDIS_ADDR` | `localhost:6379` |
| `REDIS_PASSWORD` | *(empty)* |
| `REDIS_DB` | `0` |

## Authentication

| Variable | Default | Purpose |
|---|---|---|
| `JWT_SECRET` | development placeholder | **Must** be at least 32 characters in production |
| `JWT_ACCESS_TTL` | `15` | Access token lifetime, minutes |
| `JWT_REFRESH_TTL` | `7` | Refresh token lifetime, days |
| `JWT_ISSUER` | `rera-property-platform` | |
| `OTP_TTL` | `300` | Seconds a code stays valid |
| `OTP_MAX_ATTEMPTS` | `5` | Verification attempts per code |

## First administrator

Created once, on the first seeded startup.

| Variable | Default |
|---|---|
| `ADMIN_EMAIL` | `admin@reraproperties.local` |
| `ADMIN_PASSWORD` | `Admin@12345` — must be changed in production |
| `ADMIN_MOBILE` | `9000000001` |
| `SEED_DEMO_DATA` | `true` — sample builders, projects and a demo buyer; must be `false` in production |

## CORS and rate limiting

| Variable | Default | Purpose |
|---|---|---|
| `CORS_ORIGINS` | `http://localhost:5173,http://localhost:3000` | Comma-separated; a wildcard is rejected in production |
| `RATE_LIMIT_AUTH` | `10` | Login and registration requests per minute per caller |
| `RATE_LIMIT_OTP` | `20` | OTP requests per hour per caller |
| `RATE_LIMIT_PUBLIC` | `120` | Public API requests per minute per caller |

The caller key is the authenticated user id when present, otherwise the client
IP.

## File storage

The application depends on a `Storage` interface. Switching provider is a
configuration change plus one case in `buildStorage()`.

| Variable | Default | Purpose |
|---|---|---|
| `STORAGE_DRIVER` | `local` | `local` today; `s3` is the next implementation |
| `STORAGE_LOCAL_DIR` | `./uploads` | Where files are written under the local driver |
| `STORAGE_BASE_URL` | `<APP_URL>/uploads` | Public prefix for stored files |
| `S3_BUCKET`, `S3_REGION`, `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_PUBLIC_BASE_URL` | *(empty)* | Reserved for the S3 driver |

Upload limits are enforced in code: images (JPG, PNG, WEBP) up to 5 MB,
documents (PDF) up to 20 MB, with the content type sniffed from the file rather
than trusted from the request.

## Email, SMS and WhatsApp

No vendor is hard-coded. Implement `notification.Sender` and register it in
`BuildNotifier()` to plug one in.

| Variable | Default | Purpose |
|---|---|---|
| `MAIL_DRIVER` | `log` | `log` prints to the server log; `smtp` delivers |
| `MAIL_HOST`, `MAIL_PORT`, `MAIL_USERNAME`, `MAIL_PASSWORD` | — | SMTP credentials |
| `MAIL_FROM`, `MAIL_FROM_NAME` | `no-reply@example.com` | Envelope sender |
| `SMS_DRIVER` | `log` | Vendor implementations plug in here |
| `WHATSAPP_DRIVER` | `log` | |
| `WHATSAPP_NUMBER` | *(empty)* | Fallback number when a builder has none |
| `WHATSAPP_COUNTRY_CODE` | `91` | Prefix for deep links |

With the `log` driver, OTP codes appear in the API server output — which is what
you want locally, and never what you want in production.

## Rental marketplace

| Variable | Default | Notes |
|---|---|---|
| `RENTAL_LISTING_DAYS` | `60` | How long a new rental stays live before the owner must renew it. Sale listings never expire. |
| `RENTAL_SWEEP_INTERVAL_HOURS` | `6` | How often stale rentals are retired and owners of lapsing listings warned. `0` switches the timer off, leaving `POST /admin/listings/sweep-expiry` as the only trigger. |

The sweep also runs once at startup, so a server that has been down over a
weekend does not serve stale listings until the first tick.

## Frontend (build time)

Vite inlines these at build time, so they must be set before `npm run build`.

| Variable | Default |
|---|---|
| `VITE_API_BASE_URL` | `http://localhost:8080/api/v1` |
| `VITE_APP_NAME` | `RERA Property Platform` |

---

## Production checklist

The server refuses to start when `APP_ENV=production` and any of these is true:

- [ ] `JWT_SECRET` is still the default, or shorter than 32 characters
- [ ] `DB_PASSWORD` is empty
- [ ] `CORS_ORIGINS` contains `*`
- [ ] `ADMIN_PASSWORD` is still the development default
- [ ] `SEED_DEMO_DATA` is `true`

Also review before going live:

- `MAIL_DRIVER=smtp` with working credentials, and a real SMS provider
- `STORAGE_DRIVER` pointing at durable storage rather than a container filesystem
- `DB_AUTO_MIGRATE=false` if migrations are applied by a separate step
- `FRONTEND_URL` and `APP_URL` set to the real hostnames, since they appear in
  emails, the sitemap and share links
