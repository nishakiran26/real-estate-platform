# Development Guide

## Local setup

```bash
git clone <repository>
cd RERA
cp .env.example .env        # set DB_USER / DB_PASSWORD for your MySQL

cd backend && go mod download && go run ./cmd/server
cd frontend && npm install && npm run dev
```

The backend creates the schema if it is missing, applies migrations and seeds
reference data on first run. Watch its output for OTP codes while
`MAIL_DRIVER`/`SMS_DRIVER` are set to `log`.

### Useful commands

```bash
# Backend
go build ./...                 # compile everything
go vet ./...                   # static analysis
go test ./...                  # unit + integration tests
gofmt -l ./internal ./pkg      # anything listed is unformatted

# Frontend
npm run dev                    # dev server with hot reload
npm run build                  # production build
npm test                       # vitest
```

---

## Conventions

### The layering rule

```
Router → Handler → Service → Repository → MySQL
```

Each arrow is one direction only:

- **Handlers** bind and validate the request, call one service method and render
  the envelope. No business logic, no SQL.
- **Services** hold the business rules, authorisation and transactions. They
  take plain values and DTOs; they never see `echo.Context`.
- **Repositories** are the only code that touches GORM. They return models and
  domain errors, never HTTP types.

A handler that needs data from two services is fine. A handler that needs a
`gorm.DB` is a design mistake.

### Errors

Services return `services.Error` through the constructors `Invalid`,
`NotFound`, `Conflict`, `Forbidden`, `Unauthorized`, `TooMany` and `Internal`.
`handlers.fail` maps them to status codes and logs anything 500-class in full
while returning a generic message to the client.

```go
if property.ClientID != clientID {
    // Deliberately "not found": do not confirm that another builder's
    // property exists.
    return NotFound("Property not found")
}
```

### DTOs

Requests and responses are structs in `internal/dto`. Update requests use
pointer fields so an omitted field keeps its value and an explicit `null` does
not clear it by accident. Validation lives in struct tags and is translated into
field errors by `internal/validators`.

### Frontend

- Views never call axios directly; they call a module in `src/services/api.js`.
- Shared loading, error and empty state comes from `useAsync` / `useSubmit` and
  the `PropertyGrid`, `EmptyState` and `SkeletonCard` components.
- Styling uses the design tokens in `src/assets/styles/main.css`. Component
  styles are `scoped` and only cover what is specific to that component.
- Anything that varies by deployment (cities, amenities, banners, pages) comes
  from the API, never a hard-coded list.

---

## Adding a feature end to end

Adding "site visit scheduling" would touch these files in order:

1. **Model** — `internal/models/inquiry.go`: a `SiteVisit` struct.
2. **Migration** — add it to `AllModels()` in `internal/database/migrate.go`, and
   write the matching SQL in `backend/migrations/`.
3. **Repository** — `internal/repositories/engagement_repo.go`: add the methods
   to the interface and implement them.
4. **DTO** — `internal/dto/inquiry.go`: request and response shapes with
   validation tags.
5. **Service** — `internal/services/engagement_service.go`: the rules (who may
   schedule, which slots are valid, what notification is sent).
6. **Handler** — `internal/handlers/`: bind, call the service, render.
7. **Route** — `internal/routes/routes.go`: mount it in the right group so it
   inherits the correct authentication and rate limiting.
8. **API module** — `frontend/src/services/api.js`.
9. **View** — the relevant screen under `frontend/src/views/`.
10. **Tests** — a service or integration test for the rule, and a frontend test
    if it changes state or guards.

---

## Testing

### Backend

`go test ./...` runs unit tests and the API integration suite.

The integration suite (`internal/server/integration_test.go`) drives the real
router, services and database. It uses its own schema (`TEST_DB_NAME`, default
`rera_properties_test`), truncates it before each run, and **skips itself** when
MySQL is unreachable so the suite stays green on a machine without a database.

It covers the rules that only exist once everything is wired together:

- registration, duplicate rejection, login success and failure
- OTP dispatch, and that a wrong code does not verify an account
- the approval workflow, including that a draft is never public
- rejection with a reason, and resubmission afterwards
- that a builder cannot read, edit or approve another builder's property
- that a builder sees only their own leads while an administrator sees all
- that consent is mandatory for a lead
- the public/authenticated boundary on every route group

### Frontend

`npm test` covers the auth store, the router guards (the full role matrix),
`PropertyCard` rendering and interaction, the Indian price and area formatting,
the owner availability and renewal controls, and the visit request modal.

The last two are there because both change state: marking a flat rented must
not send the listing back through approval, a refused change must not leave the
screen claiming the flat is let, and a sent visit request must not leave a form
on screen that can be submitted again.

---

## Troubleshooting

**"could not reach MySQL"** — check `DB_HOST`, `DB_USER` and `DB_PASSWORD` in
`.env`. Inside Docker Compose the host is `mysql`, not `localhost`.

**"redis is unavailable"** — a warning, not an error. Start Redis to remove it,
or ignore it; caching and rate limiting fall back automatically.

**Where is my OTP?** — in the API server output while `SMS_DRIVER` and
`MAIL_DRIVER` are `log`.

**CORS errors in the browser** — add the SPA origin to `CORS_ORIGINS` and
restart the backend.

**A new listing is not on the site** — it is a draft until it is submitted and
an administrator approves and publishes it. Submission also requires at least
one image and a RERA number.

**Frontend build fails on an old Node** — the toolchain is pinned to Vite 4,
which supports Node 14.18+. On Node 18+ you can move to Vite 5 by bumping
`vite`, `@vitejs/plugin-vue` and `vitest` in `frontend/package.json`.

---

## Two traps this codebase has already hit

Both cost real debugging time and neither announces itself, so they are worth
knowing before you add a model or a test.

### An empty Go string is not valid JSON

MySQL rejects `''` in a `JSON` column with *"Invalid JSON text: The document is
empty."* Go's zero value for a `string` field is exactly that, so any model with
`gorm:"type:json"` fails to insert whenever the field is unset.

This is nastiest on best-effort paths. Engagement events, notifications and
audit entries are all written fire-and-forget with only a logged warning, so the
insert failed **silently** and `user_property_events` stayed empty — every
WhatsApp-click, conversion-rate and most-contacted figure on the dashboards read
zero, with nothing in the API response to suggest a problem.

Every model with a JSON column now carries a `BeforeSave` hook normalising the
zero value (`{}` for objects, `[]` for lists). **Add one whenever you add a JSON
column**, and assert the row actually lands — a test that only checks the HTTP
response will pass against a table that never received anything.

### GORM substitutes the column default for any zero value

A field tagged `gorm:"default:true"` is omitted from the INSERT when its Go
value is `false`, so the database default wins and the deliberate `false` is
lost. `CustomerProfile.LookingToBuy` hit this: a renter-only profile was written
back as a buyer.

Where the application always sets a boolean explicitly, leave the `default` tag
off the model and let the migration backfill existing rows instead.

---

## Writing integration tests

`setup(t)` builds a whole application: config, connection pool, cache, storage
and router. Two rules follow from that.

**Close what you open.** Each pool allows up to 25 connections and MySQL's
default ceiling is 151, so a suite of ~50 tests that leaks its pools exhausts the
server. The failures then land on whichever test happens to run once the limit is
crossed — producing symptoms that look like logic bugs (a 500 on registration, a
foreign-key violation from an ID left at zero) in tests that pass perfectly well
on their own. `setup` registers a `t.Cleanup` that closes the pool and the cache
client; keep it.

**Suspect the harness when failures move.** A real regression fails the same test
every run. Different tests failing on each run, or a test that passes alone and
fails in the suite, points at shared state: connections, the seeded schema, or
rate-limit buckets.

**One schema per process.** `TestMain` names the test schema after the process
id and drops it at the end. Without that, two `go test` runs on one machine
share a schema: each one's `truncateAll` deletes the other's rows mid-test and
each one's `AutoMigrate` alters tables the other has open. The symptoms never
mention isolation — a create that answers *404 Property not found* because its
rows were truncated underneath it, a register that fails with *Error 1412: Table
definition has changed* — and they land on a different test every run. A run
killed part-way leaves its schema behind; those are safe to drop by hand.

**No real vendors.** `setup` forces the mail, SMS and WhatsApp drivers to `log`,
whatever the developer's `.env` says. A suite that registers hundreds of
accounts against a real SMTP server is slow, depends on the network, and sends
real messages to real inboxes.

---

## Seed data

`SEED_DEMO_DATA=true` (the default outside production) creates three Pune
builders with six published RERA-registered projects **and** an individual owner
letting two flats, so a fresh installation has something to browse on both the
Buy and the Rent pages. Opening `/rent` on an empty catalogue tells you nothing
about whether the rental journey works.

The seeded rentals are honest records rather than copies of the sale listings:
they carry rent, deposit, furnishing and availability, they are posted by an
`OWNER` account, and they carry **no RERA registration**, because RERA governs
projects for sale and inventing a number would put a lie in the data. Seeding is
idempotent, so a restart does not duplicate anything.

Production is required to set `SEED_DEMO_DATA=false`; `config.Validate` refuses
to start otherwise.

**"ok" can mean "skipped".** When MySQL is unreachable the integration tests
skip themselves, and `go test ./...` then reports `ok` for the package in a
fraction of a second. That is the intended behaviour — the suite stays green on
a machine without a database — but it is easy to read a skipped run as a passing
one. A server-package run that finishes in under a second ran nothing: check
with `-v`, or start MySQL.
