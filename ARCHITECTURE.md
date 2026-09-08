# RERA Property Discovery Platform — System Architecture

A production-grade, RERA-focused real-estate discovery platform for India (Pune-first).
Modular monolith designed so services can be extracted later.

---

## 1. High-Level Architecture

```
                        ┌──────────────────────────────┐
                        │        Vue 3 SPA (Vite)       │
                        │ Tailwind · Pinia · Vue Router │
                        │  Public / Customer / Client / │
                        │  Admin  (role-guarded views)  │
                        └──────────────┬───────────────┘
                                       │ HTTPS / JSON  (Access Token in Authorization header)
                                       ▼
                        ┌──────────────────────────────┐
                        │      Go 1.22 + Echo v4        │
                        │  Router → Middleware chain    │
                        │  Handler → Service → Repo     │
                        └───────┬──────────────┬────────┘
                                │              │
                     ┌──────────▼───┐   ┌──────▼──────────────┐
                     │   MySQL 8    │   │       Redis         │
                     │  (GORM)      │   │ OTP · RateLimit ·   │
                     │ persistent   │   │ Cache · Tokens      │
                     └──────────────┘   └─────────────────────┘
                                │
                     ┌──────────▼─────────────┐
                     │ Storage abstraction     │  local / S3 / Cloudinary
                     │ Notification providers  │  smtp / sms / whatsapp / inapp
                     └────────────────────────┘
```

Request flow (strict, enforced by code layout):
`Router → Middleware(CORS, AuthN, RBAC, RateLimit) → Handler(Controller) → Service(business logic, DTOs, validation) → Repository(GORM queries, transactions) → MySQL`

Handlers never touch the DB; services never touch `echo.Context` beyond DTO binding; repositories return models/DTOs, not HTTP.

---

## 2. Database ER (key relationships)

```
users ──< user_roles >── roles
  │  │
  │  ├──< refresh_tokens
  │  ├──< login_history
  │  ├──< otp_requests
  │  ├──< favorites >── properties
  │  ├──< inquiries ──> properties (nullable user_id)
  │  ├──< notifications / notification_preferences
  │  ├──< user_property_events ──> properties
  │  ├──< property_views ──> properties
  │  ├──1:1 clients (role CLIENT)
  │  └──1:1 customer_profiles (role CUSTOMER) ── customer_preferences
  │
cities ──< localities ──< properties
properties ──< property_configurations
properties ──< property_amenities >── amenities
properties ──< property_media
properties ──< property_documents
properties ──1:1 rera_details
properties ──< property_status_history
properties ──< featured_properties
audit_logs ──> users (actor)
homepage_banners / static_pages / seo (CMS)
```

Full DDL lives in `backend/migrations/` and `docs/DATABASE.md`.

---

## 3. Property Approval Workflow

```
        (client creates)
             │
             ▼
          DRAFT ──────────(edit)──────────┐
             │ submit                      │
             ▼                             │
        SUBMITTED ──(admin picks up)──► UNDER_REVIEW
             │                             │
      admin reject                    admin approve
             ▼                             ▼
         REJECTED ──(client corrects)──► APPROVED ──(auto/manual)──► PUBLISHED
             │                            │
             └────── resubmit ─────────────┘
```
Only `PUBLISHED` properties are publicly visible. Every transition is written to `property_status_history` (actor + reason) and, for admin actions, to `audit_logs`.

---

## 4. Authentication Flow

1. **Register** (email+password or mobile+OTP) → `POST /auth/register` → creates user (unverified) → dispatch email/mobile OTP.
2. **Verify OTP** → `POST /auth/verify-otp` (expiry + attempt limit, stored in Redis).
3. **Login** by email+password **or** mobile+OTP → issues `Access Token` (short-lived, HS256) + `Refresh Token` (opaque, persisted in `refresh_tokens`).
4. **Refresh rotation**: `POST /auth/refresh` invalidates old refresh, issues new pair. Reuse of a rotated token ⇒ family revoked.
5. **Logout**: current device (revoke one refresh) or all devices (revoke all). `login_history` records login/logout, IP, UA, method.
6. Passwords hashed with **bcrypt**. Forgot/reset via signed token + OTP.

---

## 5. Role & Permission Matrix

| Capability | CUSTOMER | CLIENT (Builder) | ADMIN |
|---|:--:|:--:|:--:|
| Browse published properties | ✓ | ✓ | ✓ |
| Favorites / compare / inquiries | ✓ | – | – |
| Own profile + preferences | ✓ | ✓ | ✓ |
| Create/edit/delete own properties | – | ✓ (own) | ✓ (all) |
| Submit property for approval | – | ✓ | – |
| View leads for own properties | – | ✓ (own) | ✓ (all) |
| Approve/reject properties | – | – | ✓ |
| Manage users/cities/amenities/CMS | – | – | ✓ |
| Full analytics | – | own-property | global |

Enforced by `middleware.RequireAuth` + `middleware.RequireRole(...)` and by ownership checks inside services (e.g., a client can only read/update a property whose `client_id` matches).

---

## 6. Backend Folder Structure

```
backend/
├── cmd/server/main.go            # startup: config, database, graceful shutdown
├── internal/
│   ├── server/                   # dependency graph (shared by main and tests)
│   ├── config/                   # env config
│   ├── database/                 # mysql, redis, migrations, seed
│   ├── models/                   # GORM models
│   ├── repositories/             # data access (interfaces + gorm impl)
│   ├── services/                 # business logic
│   ├── handlers/                 # echo controllers
│   ├── middleware/               # auth, rbac, ratelimit, recovery, logger
│   ├── routes/                   # route registration
│   ├── dto/                      # request/response DTOs + envelope
│   ├── validators/               # request validation + field errors
│   ├── cache/                    # fail-soft Redis client
│   ├── storage/                  # storage abstraction (local/s3)
│   └── notification/             # notification provider interfaces
├── migrations/                   # SQL migrations + seed
├── pkg/                          # jwt, password hashing, logger, slug helpers
├── Dockerfile
└── go.mod
```

## 7. Frontend Folder Structure

```
frontend/src/
├── main.js
├── router/          # routes + role guards
├── stores/          # pinia (auth, property, ui)
├── services/        # axios instance + api modules
├── layouts/         # Public / App / Admin layouts
├── components/      # PropertyCard, Filters, Gallery, Toast, etc.
├── views/           # public/ customer/ client/ admin/
├── composables/     # useAuth, useToast, useApi
├── utils/           # formatters, constants
└── assets/          # styles
```

---

## 8. API Surface (v1, consistent envelope)

All responses: `{ success, message, data?, errors? }`. Auth via `Authorization: Bearer <access>`.

- **Auth**: register, login, mobile-otp, send-otp, verify-otp, refresh, logout, forgot-password, reset-password, change-password, resend-otp, me
- **Public**: `/meta` (cities/localities/types/amenities/banners), `/properties` (list+filter+sort+paginate), `/properties/:slug`, `/properties/:id/similar`, `/properties/compare`, `/events` (track), `sitemap.xml`, `robots.txt`
- **Client**: `/client/profile`, `/client/properties` CRUD + `/submit`, `/client/properties/:id/analytics`, `/client/inquiries`
- **Customer**: `/customer/profile`, `/customer/preferences`, `/customer/favorites[/:pid]`, `/customer/recently-viewed`, `/customer/inquiries`, `/customer/notifications[/:id/read]`, `/customer/notification-preferences`
- **Admin**: `/admin/dashboard`, `/admin/analytics`, `/admin/properties[/:id/approve|/reject]`, `/admin/users|clients|customers`, `/admin/inquiries`, `/admin/cities|localities|amenities`, `/admin/banners|featured|pages`, `/admin/audit-logs`, user `status`/`role` ops
- **Rental**: `/properties?purpose=rent` (rent band, furnishing, tenant preference, availability), `/properties/:key/visits`, `/properties/:key/report`, `/rent-guide`, `/client/properties/:id/availability|renew`, `/client/expiring-listings`, `/client/visits`, `/admin/reports`, `/admin/listings/sweep-expiry`, `/admin/clients/:id/verification|properties`

Full list in `docs/API.md`.

---

## 9. Caching, Redis, Security, SEO, Events
- **Redis**: OTP codes (TTL), rate-limit buckets, hot property/search cache. Redis failures are non-fatal (graceful degradation).
- **Security**: bcrypt, JWT+rotating refresh, RBAC, per-route ownership checks, rate limiting, input validation, GORM parameterisation (SQLi-safe), configurable CORS, upload MIME/size checks, unique storage keys.
- **SEO**: server-rendered metadata via `/seo/:slug`, dynamic meta/OG/canonical on frontend, sitemap + robots endpoints, slugs like `green-valley-residency-baner-pune`.
- **Analytics**: generic event tracking (`user_property_events`) powers admin/client dashboards and recommendations.

## 10. Roadmap (phases 1–8, then the rental extension)
Foundation → User mgmt → Property mgmt → Approval → Customer experience → Leads → Analytics → Notifications.
Each phase lands working code with migrations + seed data before the next.

The rental marketplace was added on top of that, additively, in nine further
phases: schema → search dimension → posting → visits → profiles → rental UI →
owner portal → admin rental management → analytics and trust. Nothing existing
was rebuilt: a rental is a property with `purpose = RENT` and a `rental_details`
row, so it inherits the approval workflow, media, leads, favourites, analytics
and RBAC unchanged. See `docs/API.md` and `docs/DATABASE.md`.

## 11. Deployment
`docker-compose.yml` runs `mysql`, `redis`, `backend` (Go, auto-migrate + seed), and `frontend` (Vite dev server / static). Config via `.env` (see `.env.example`). No secrets committed.