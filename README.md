# RERA Property Discovery Platform

A production-grade property platform for residential real estate in India, Pune
first. It carries two catalogues in one application: RERA-registered projects
**to buy**, and flats and houses **to rent**. Buyers search and compare verified
projects, renters filter by rent, furnishing and move-in date and request a
viewing, builders and individual owners list and manage what they have posted,
and administrators review every listing before it goes live.

- **Backend** — Go 1.26, Echo v4, GORM, MySQL 8, Redis, JWT with refresh-token
  rotation, RBAC.
- **Frontend** — Vue 3 (Composition API), Vue Router, Pinia, Axios, Vite.
- **Infrastructure** — Docker and Docker Compose, environment-based
  configuration, structured logging.

---

## Contents

- [Quick start](#quick-start)
- [What you get out of the box](#what-you-get-out-of-the-box)
- [Architecture](#architecture)
- [Project layout](#project-layout)
- [Configuration](#configuration)
- [Database](#database)
- [Running the tests](#running-the-tests)
- [Documentation](#documentation)

---

## Quick start

### Option A — Docker Compose (everything in one command)

```bash
cp .env.example .env      # then edit the passwords
docker compose up --build
```

| Service  | URL                            |
|----------|--------------------------------|
| Frontend | <http://localhost:5173>        |
| API      | <http://localhost:8080>        |
| Health   | <http://localhost:8080/health> |
| MySQL    | `localhost:3306`               |
| Redis    | `localhost:6379`               |

The API waits for MySQL and Redis to report healthy, then migrates and seeds
before it starts serving.

### Option B — run the services directly

Prerequisites: Go 1.26+, Node 18+ (Node 14.18+ also works with the pinned Vite
4 toolchain), MySQL 8, and optionally Redis.

```bash
cp .env.example .env      # set DB_USER / DB_PASSWORD for your MySQL

# Backend — creates the schema, migrates and seeds on first run
cd backend
go run ./cmd/server

# Frontend, in a second terminal
cd frontend
npm install
npm run dev
```

The backend reads `.env` from the repository root (or `backend/`) when it is
present; real environment variables always win, so Docker and any deployment
target stay authoritative.

### Signing in

The seeded accounts come from your `.env`. With the defaults:

| Role     | Email                            | Password        |
|----------|----------------------------------|-----------------|
| Admin    | `admin@reraproperties.local`     | `Admin@12345`   |
| Builder  | `sales@greenvalleydevelopers.example` | `Builder@12345` |
| Owner    | `owner@reraproperties.local`     | `Builder@12345` |
| Buyer    | `buyer@example.com`              | `Buyer@12345`   |

The owner account is an individual letting two flats — the same kind of listing
account as the builder, with `profile_type = OWNER`. Sign in as it to see the
rental side of the portal: availability, renewal and viewing requests.

Change `ADMIN_PASSWORD` before any deployment that is reachable from outside
your machine. The builder, owner and buyer logins exist only while
`SEED_DEMO_DATA` is true, which the configuration refuses to allow in production.

### OTP codes in development

`MAIL_DRIVER` and `SMS_DRIVER` default to `log`, so verification codes are
printed to the API server output instead of being sent. Watch the terminal
running the backend when you test a mobile OTP login or a password reset.

---

## What you get out of the box

Seeding creates three Pune builders with six published RERA-registered projects
across Baner, Wakad, Hinjawadi, Kharadi, Viman Nagar and Hadapsar, plus an
individual owner letting two flats, cities, localities, 16 amenities, homepage
banners and the informational pages (including the RERA disclaimer). A fresh
install has something to browse on both the Buy and the Rent pages immediately.

---

## Buying and renting in one application

A listing carries a `purpose` of `BUY` or `RENT`. Everything else — the
approval workflow, media, leads, favourites, analytics, RBAC — is shared, so a
rental is a property with a `rental_details` row rather than a parallel system.

- `/properties` searches the sale catalogue; `/rent` searches rentals, with rent
  band, furnishing, tenant preference and availability replacing budget and RERA.
- An individual owner is a listing account with `profile_type = OWNER`, not a
  fourth role. One account can buy, rent and rent out.
- Rentals carry an availability status (`AVAILABLE`, `RESERVED`, `RENTED`) and
  an expiry date, because rental stock goes stale in a way sale listings do not.
- Renters request a viewing rather than only enquiring; the owner confirms,
  reschedules or declines.

Owners manage what they have posted without going back through approval: mark a
flat rented, relist it, renew a listing that is about to lapse. Stale rentals
are retired automatically and the owner is told, because a listing nobody has
confirmed in two months is the fastest way for a rental site to lose trust.
Anyone can report a wrong listing, and administrators work through a moderation
queue; an upheld complaint takes the listing down and tells the owner why.

See `docs/API.md` for the endpoints and `docs/DATABASE.md` for the schema.

---

## Architecture

```
Vue 3 SPA  ──HTTPS/JSON──►  Go + Echo API  ──►  MySQL 8
   │                             │              (GORM)
   │                             ├──►  Redis   (OTP, rate limits, cache)
   │                             ├──►  Storage (local disk | S3 | any provider)
   └── Pinia stores              └──►  Notifications (email | SMS | WhatsApp)
```

Every request follows one path, enforced by the package layout:

```
Router → Middleware (session, CORS, auth, RBAC, rate limit)
       → Handler   (bind, validate, render the response envelope)
       → Service   (business rules, authorisation, transactions)
       → Repository (GORM queries)
       → MySQL
```

Handlers never touch the database. Services never see `echo.Context`.
Repositories never return HTTP types. See [ARCHITECTURE.md](ARCHITECTURE.md) for
the full design, the role and permission matrix, and the approval workflow.

### Graceful degradation

Redis is optional. When it is unreachable, caching falls back to direct queries
and rate limiting falls back to an in-process counter, so an outage slows the
platform without taking it offline.

---

## Project layout

```
.
├── backend/
│   ├── cmd/server/          # entrypoint: config, database, shutdown
│   ├── internal/
│   │   ├── server/          # dependency graph (used by main and by tests)
│   │   ├── routes/          # URL → handler mapping, middleware per group
│   │   ├── handlers/        # HTTP controllers
│   │   ├── services/        # business logic and authorisation
│   │   ├── repositories/    # the only layer that talks to MySQL
│   │   ├── models/          # GORM models
│   │   ├── dto/             # request and response contracts
│   │   ├── middleware/      # auth, RBAC, rate limiting, sessions, logging
│   │   ├── database/        # connection, migrations, seed data
│   │   ├── storage/         # provider-agnostic file storage
│   │   ├── notification/    # provider-agnostic delivery channels
│   │   ├── validators/      # request validation and field errors
│   │   ├── cache/           # fail-soft Redis client
│   │   └── config/          # environment configuration
│   ├── migrations/          # SQL schema and reference data
│   └── pkg/                 # JWT, password hashing, logging, helpers
├── frontend/
│   └── src/
│       ├── views/           # public / auth / customer / client / admin
│       ├── components/      # property cards, filters, gallery, feedback
│       ├── layouts/         # public, auth and dashboard shells
│       ├── stores/          # Pinia: auth, property, ui
│       ├── services/        # axios instance and API modules
│       ├── router/          # routes and role guards
│       ├── composables/     # useAsync, useSubmit, useSeo
│       └── utils/           # formatting and constants
├── docs/                    # API, database and deployment documentation
├── docker-compose.yml
└── .env.example
```

---

## Configuration

Every setting is an environment variable, documented in
[.env.example](.env.example) and in [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md).
Nothing reads `os.Getenv` outside `internal/config`.

Production is validated at startup. The server refuses to boot when
`APP_ENV=production` and any of the following is true:

- `JWT_SECRET` is still the development default or shorter than 32 characters
- `DB_PASSWORD` is empty
- `CORS_ORIGINS` contains a wildcard
- `ADMIN_PASSWORD` is still the development default
- `SEED_DEMO_DATA` is true

---

## Database

The ORM owns the schema. `DB_AUTO_MIGRATE=true` creates and updates tables on
startup, and adds the composite indexes that back the search and analytics
queries.

For teams that apply migrations explicitly, `backend/migrations/` holds the same
schema as SQL:

```bash
mysql -u root -p rera_properties < backend/migrations/000001_init_schema.up.sql
mysql -u root -p rera_properties < backend/migrations/000002_seed_reference_data.up.sql
```

Table-by-table documentation, relationships and indexing notes are in
[docs/DATABASE.md](docs/DATABASE.md).

---

## Running the tests

```bash
cd backend && go test ./...       # unit tests plus the full API integration suite
cd frontend && npm test           # stores, router guards, components, formatting
```

The backend integration suite drives the real HTTP stack against MySQL and
covers registration, login, OTP, the approval workflow, lead isolation between
builders, and the RBAC matrix. It uses its own schema (`TEST_DB_NAME`, default
`rera_properties_test`) and **skips itself** when MySQL is unreachable, so
`go test ./...` stays green without a database.

---

## Documentation

| Document | Contents |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | System design, ER relationships, auth flow, permission matrix, approval workflow, roadmap |
| [docs/API.md](docs/API.md) | Every endpoint, request and response shapes, error format, pagination and filtering |
| [docs/DATABASE.md](docs/DATABASE.md) | Schema reference, relationships, indexes, soft deletes |
| [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md) | Every environment variable and what it does |
| [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) | Local setup, conventions, how to add a feature end to end |
