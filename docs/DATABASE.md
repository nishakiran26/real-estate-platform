# Database Schema

MySQL 8, InnoDB, `utf8mb4`. The ORM owns the schema; `backend/migrations/`
carries the same definition as SQL for teams that apply migrations explicitly.

Conventions used throughout:

- `bigint unsigned` surrogate primary keys.
- `created_at` / `updated_at` on every business table.
- `deleted_at` (indexed) for soft deletes, so a removed listing does not take
  its historical leads and analytics with it.
- Event tables (`user_property_events`, `property_views`) are append-only and
  carry only `created_at`.

---

## Relationships

```
users ──< user_roles >── roles
  │
  ├──< refresh_tokens          (rotating sessions, hashed)
  ├──< login_histories         (login/logout, IP, device, method)
  ├──< otp_requests            (hashed codes, expiry, attempt counter)
  ├──< notifications
  ├──1:1 notification_preferences
  ├──1:1 clients               (builders)
  ├──1:1 customer_profiles     (buyer search preferences)
  ├──< favorites >── properties
  ├──< inquiries ──> properties        (user_id nullable: anonymous leads)
  ├──< user_property_events ──> properties
  └──< property_views ──> properties

cities ──< localities ──< properties

properties ──< property_configurations
           ──< property_amenities >── amenities
           ──< property_media
           ──< property_documents
           ──1:1 rera_details
           ──< property_status_histories
           ──< featured_properties

audit_logs ──> users (actor)
home_banners, static_pages                (CMS, no foreign keys)
```

---

## Identity and access

### `users`

The account. Email and mobile are both unique, because either can be used to
sign in.

| Column | Type | Notes |
|---|---|---|
| `email`, `mobile` | varchar | Unique; mobile stored in canonical 10-digit form |
| `password_hash` | varchar(255) | bcrypt |
| `status` | varchar(20) | `ACTIVE`, `INACTIVE`, `SUSPENDED` |
| `email_verified`, `mobile_verified` | bool | Cleared when the value changes |
| `consent_property_updates` … `consent_email` | bool | One column per purpose, never bundled |
| `consent_updated_at` | datetime | When consent last changed |
| `last_login_at` | datetime | |

Consent is stored per purpose so a withdrawal is specific and auditable.

### `roles`, `user_roles`

Three seeded roles: `ADMIN`, `CLIENT`, `CUSTOMER`. The join table allows more
than one role per user, which is what lets an administrator use the builder and
customer areas.

### `refresh_tokens`

Only `token_hash` (SHA-256) is stored, so a database leak yields no usable
tokens. `family_id` groups the tokens descended from one login: presenting a
rotated token revokes the whole family.

### `otp_requests`

`code_hash`, `purpose`, `expires_at`, `attempts` and `max_attempts`. Codes live
in Redis when it is available and in this table otherwise, so verification keeps
working during a Redis outage.

### `login_histories`

Login and logout time, IP address, user agent, derived device name and the
method used (`EMAIL_PASSWORD` or `MOBILE_OTP`).

---

## Accounts

### `clients`

The builder company: name, logo, description, website, contact details,
registered address, GSTIN and company RERA registration. `total_properties` is
a denormalised counter kept in step when listings change.

### `customer_profiles`

What a buyer is looking for — preferred city and locality, budget band, property
type, BHK and purpose (`BUY`, `INVEST`, `RENT`). This is the input to the
recommendation engine.

---

## Geography and reference data

### `cities`, `localities`

Both carry a slug, so URLs read `/city/pune/baner`. Localities are unique on
`(city_id, slug)` — two cities may each have a "Civil Lines". `is_popular` and
`display_order` drive homepage placement without a code change.

### `amenities`

Name, slug, icon key, category (`GENERAL`, `SPORTS`, `LEISURE`, `SAFETY`,
`CONVENIENCE`, `ENVIRONMENT`) and display order.

---

## Properties

### `properties`

The central table.

| Group | Columns |
|---|---|
| Ownership | `client_id`, `user_id` |
| Identity | `name`, `slug` (unique), `project_name`, `builder_name` |
| Classification | `property_type`, `property_status`, `construction_status` |
| Location | `city_id`, `locality_id`, `address`, `pincode`, `latitude`, `longitude` |
| Pricing summary | `min_price`, `max_price`, `min_area`, `max_area` — computed from configurations |
| Approval | `submitted_at`, `approved_at`, `approved_by`, `rejected_at`, `rejected_by`, `rejection_reason` |
| Flags | `is_featured`, `is_rera_verified` |
| SEO | `meta_title`, `meta_description` |
| Counters | `view_count`, `inquiry_count`, `whatsapp_click_count`, `favorite_count` |

`property_status` moves through `DRAFT → SUBMITTED → UNDER_REVIEW → APPROVED →
PUBLISHED`, with `REJECTED` as the branch back to the builder. Only `PUBLISHED`
rows are publicly visible.

The counters are denormalised on purpose: a listing card must not run an
aggregate over the event tables.

### `property_configurations`

One row per unit type (2 BHK, 3 BHK, penthouse): carpet, built-up and super
built-up area, bedrooms, bathrooms, balconies, price band, computed
`price_per_sqft`, availability and unit count.

### `property_media`, `property_documents`

Images and PDFs. Both keep `file_url` for display and `storage_key` for
deletion, which is what lets the storage provider change without touching the
data. Exactly one media row per property carries `is_cover_image`.

### `rera_details`

One row per property: registration number, authority, registration and declared
completion dates, and `verification_status` (`PENDING_VERIFICATION`, `VERIFIED`,
`REJECTED`, `NOT_AVAILABLE`) with the verifying administrator and timestamp.

### `property_status_histories`

Every workflow transition: `from_status`, `to_status`, `actor_id` and remarks.
This is the record a builder sees when a listing is rejected.

---

## Engagement

### `inquiries`

A lead. `user_id` is nullable because anonymous visitors can enquire;
`session_id` attributes those. `client_id` is denormalised from the property so
lead queries never need a join to enforce ownership. `status` follows the
follow-up pipeline from `NEW` to `CLOSED_WON` / `CLOSED_LOST`.

### `favorites`

Unique on `(user_id, property_id)`, so saving twice is idempotent.

### `user_property_events`

The generic event stream: `event_type`, optional `user_id` and `property_id`,
`session_id`, IP, user agent and a JSON `metadata` column. It powers both
dashboards and the recommendation engine, and works for anonymous visitors.

### `property_views`

Views are recorded separately from the event stream because they are the
highest-volume signal and are queried differently (unique visitors per day per
property). A repeat view from the same session within 30 minutes is not counted
again.

---

## Notifications, audit and CMS

- `notifications` — in-app messages with a read timestamp.
- `notification_preferences` — one row per user, one switch per channel.
- `audit_logs` — administrative actions: actor, action, entity, entity id, JSON
  details and IP address.
- `home_banners`, `featured_properties`, `static_pages` — homepage and
  informational content, so none of it is hard-coded in the frontend.

---

## Indexing

Single-column indexes cover the foreign keys and the columns filtered on
directly: `city_id`, `locality_id`, `property_status`, `property_type`,
`min_price`, `max_price`, `is_featured`, `is_rera_verified`, `created_at`.

Composite indexes back the hot paths, and are created by the migrator after the
tables exist:

| Index | Query it serves |
|---|---|
| `idx_properties_status_city (property_status, city_id, locality_id)` | The public search, which always filters on status plus location |
| `idx_properties_status_price (property_status, min_price, max_price)` | Budget filtering and price sorting |
| `idx_properties_status_created (property_status, created_at)` | "Newest first" and new-launch strips |
| `idx_properties_client_status (client_id, property_status)` | The builder portfolio table |
| `idx_configs_property_bedrooms (property_id, bedrooms)` | BHK filtering |
| `idx_events_property_type_date (property_id, event_type, created_at)` | Per-property engagement analytics |
| `idx_views_property_date (property_id, created_at)` | View counts and daily series |
| `idx_inquiries_client_status (client_id, status, created_at)` | The builder lead inbox |
| `idx_localities_city_slug (city_id, slug)` UNIQUE | Locality resolution from a URL |

---

## Scaling notes

MySQL plus Redis is the first version, and the repository interfaces are the
seam that keeps it changeable. `PropertyRepository` takes a `PropertyFilter`
struct rather than exposing a query builder, so an Elasticsearch or OpenSearch
implementation can be added behind the same interface without touching the
services, handlers or frontend.

---

## Rental marketplace

Added additively in migration `000003`. No existing column was dropped, renamed
or retyped, so Buy and RERA data kept working throughout.

### Columns added to existing tables

| Table | Column | Purpose |
|---|---|---|
| `properties` | `purpose` | `BUY` / `RENT`. Backfilled so every pre-existing row is `BUY` |
| `properties` | `posted_by_type` | `OWNER` / `CLIENT` / `AGENT` / `ADMIN`, derived from the posting account |
| `properties` | `owner_user_id` | The real owner when an agent or administrator posted on their behalf |
| `properties` | `availability_status` | `AVAILABLE` / `RESERVED` / `RENTED` / `INACTIVE` |
| `properties` | `listing_expiry_date` | Rentals go stale; sale listings leave it null |
| `clients` | `profile_type` | `BUILDER` / `OWNER` / `AGENT` |
| `clients` | `verification_level` | Gates the trust badge; ordered, each level implies the last |
| `inquiries` | `lead_type` | `BUY_LEAD` / `RENT_LEAD`, derived from the property |
| `inquiries` | `assigned_to_user_id` | Routes a lead to one person; null means the whole account |
| `customer_profiles` | `looking_to_buy`, `looking_to_rent` | One account, both intents |
| `customer_profiles` | rental preferences | Rent band, furnishing, lease, move-in date, household type |

**Why `availability_status` is separate from `property_status`.** They are
independent facts: a listing can be `PUBLISHED` *and* `RENTED`. Collapsing them
would make "rented" indistinguishable from "rejected" in the status history and
would silently change the meaning of every existing status query.

**Why `client_id` still carries ownership.** An owner is a listing account with
`profile_type = OWNER`, not a fourth role. Because `client_id` is the key every
authorisation check, lead query, analytics rollup and media guard already uses,
owners reuse all of it unchanged. A separate `owner_id` would have meant nullable
ownership columns in six tables and a second branch in every check.

### New tables

- **`rental_details`** — 1:1 with a property, shaped like `rera_details`. Rent,
  deposit, maintenance, availability, lease terms, furnishing, tenant preference,
  pets and a JSON list of included items. Rent lives here, not on `properties`,
  so a sale listing carries no permanently null columns.
- **`property_visits`** — viewing requests with their own status machine. Keeps
  both what the visitor asked for and what the owner confirmed, so the
  negotiation is preserved.
- **`property_reports`** — abuse reports. Needed the moment the public can post.
- **`rental_alerts`** — saved rental searches. Criteria are stored as the
  search query itself so an alert always reruns exactly the search that created
  it; the hot fields (city, locality, BHK, rent band) are duplicated as indexed
  columns so a newly published rental is matched with one query rather than by
  decoding every alert. `is_active` deliberately carries no GORM `default` tag:
  the column default would revive a paused alert whenever the service wrote
  `false`.
