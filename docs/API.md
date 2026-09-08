# API Reference

Base URL: `http://localhost:8080/api/v1`

---

## Response format

Every endpoint returns the same envelope.

**Success**

```json
{
  "success": true,
  "message": "Property fetched successfully",
  "data": { }
}
```

**Paginated success** — list endpoints add `meta`:

```json
{
  "success": true,
  "message": "Properties fetched successfully",
  "data": [ ],
  "meta": { "page": 1, "per_page": 20, "total": 137, "total_pages": 7 }
}
```

**Error**

```json
{
  "success": false,
  "message": "Validation failed",
  "errors": [
    { "field": "email", "message": "Invalid email address" }
  ]
}
```

`errors` is present only for validation failures, and names the JSON field the
client sent.

### Status codes

| Code | Meaning |
|---|---|
| 200 | Request succeeded |
| 201 | Resource created |
| 302 | Redirect to a stored file (brochure download) |
| 400 | Malformed request body |
| 401 | Missing or invalid access token |
| 403 | Authenticated, but not allowed to perform this action |
| 404 | Resource does not exist, or is not visible to this caller |
| 409 | Conflict with the current state (duplicate email, wrong workflow state) |
| 422 | Validation failed; see `errors` |
| 429 | Rate limit exceeded; see `Retry-After` |
| 500 | Unexpected server error (details are logged, not returned) |

---

## Authentication

Send the access token as `Authorization: Bearer <token>`.

Access tokens are short-lived (15 minutes by default). Refresh tokens are
opaque, stored only as a hash, and **rotated on every use**: presenting a
rotated token revokes its whole family, which contains the damage if a token
leaks.

Anonymous visitors are tracked with a session cookie (`rera_sid`) so views,
WhatsApp clicks and inquiries can be attributed without a login. The SPA also
sends `X-Session-Id` to keep one session across tabs.

### Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/auth/register` | — | Create a customer or builder account; returns a session |
| POST | `/auth/login` | — | Email or mobile plus password |
| POST | `/auth/login-otp` | — | Passwordless mobile login |
| POST | `/auth/send-otp` | — | Dispatch a verification code |
| POST | `/auth/resend-otp` | — | Dispatch a fresh code |
| POST | `/auth/verify-otp` | — | Confirm a code; returns a reset token for `FORGOT_PASSWORD` |
| POST | `/auth/forgot-password` | — | Start a password reset |
| POST | `/auth/reset-password` | — | Complete a reset with the token from `verify-otp` |
| POST | `/auth/refresh-token` | — | Rotate the token pair |
| POST | `/auth/change-password` | Any | Change password; revokes other sessions |
| POST | `/auth/logout` | Any | Revoke this session, or all with `all_devices` |
| GET | `/auth/me` | Any | Current user profile |
| GET | `/auth/login-history` | Any | Recent sign-in activity |

**Register**

```http
POST /api/v1/auth/register
{
  "full_name": "Priya Sharma",
  "email": "priya@example.com",
  "mobile": "9876543210",
  "password": "Password@123",
  "role_code": "customer",
  "company_name": "",
  "consent_property_updates": true,
  "consent_new_project_alerts": true,
  "consent_promotional": false,
  "consent_whatsapp": false
}
```

`role_code` accepts `customer` or `client` only; administrators are provisioned
separately. Each consent is stored separately with a timestamp.

**Login response**

```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "tokens": {
      "access_token": "eyJhbGciOi...",
      "refresh_token": "8Fk2...",
      "expires_in": 900,
      "token_type": "Bearer"
    },
    "user": {
      "id": 12, "full_name": "Priya Sharma", "email": "priya@example.com",
      "mobile": "9876543210", "email_verified": true, "mobile_verified": false,
      "roles": [{ "id": 3, "code": "CUSTOMER", "name": "Property Buyer" }],
      "client_id": null,
      "consent": { "property_updates": true, "new_project_alerts": true,
                   "promotional": false, "whatsapp": false, "email": true }
    }
  }
}
```

**OTP purposes**: `EMAIL_VERIFY`, `MOBILE_VERIFY`, `LOGIN`, `FORGOT_PASSWORD`.
Codes expire after `OTP_TTL` seconds and allow `OTP_MAX_ATTEMPTS` attempts.

### Rate limits

| Bucket | Default | Applies to |
|---|---|---|
| `auth` | 10 / minute | Register, login, verify, reset, refresh |
| `otp` | 20 / hour | Send and resend OTP, forgot password |
| `lead` | 10 / hour | Inquiry submission |
| `public` | 120 / minute | All public read endpoints |

Responses carry `X-RateLimit-Limit` and `X-RateLimit-Remaining`; a 429 adds
`Retry-After`.

---

## Public endpoints

No token required. Sending one personalises the response (for example
`is_favorite` on cards).

| Method | Path | Description |
|---|---|---|
| GET | `/homepage` | Banners, featured, new launches, ready to move, upcoming, popular locations |
| GET | `/meta` | Cities, localities, amenities and filter enumerations |
| GET | `/properties` | Search and filter (see below) |
| GET | `/properties/:key` | One published property by numeric id **or** slug |
| GET | `/properties/:key/similar` | Similar projects nearby |
| GET | `/properties/compare?ids=1,2,3` | Full detail for up to four projects |
| GET | `/properties/recently-viewed` | Works for anonymous visitors via the session |
| GET | `/properties/recommended` | Rule-based suggestions |
| GET | `/properties/:key/whatsapp` | Prefilled WhatsApp deep link; records the click |
| GET | `/properties/:key/documents/:doc_id/download` | Records the download, redirects to the file |
| GET | `/cities`, `/cities/:slug` | Cities |
| GET | `/localities?city_id=` | Localities |
| GET | `/cities/:slug/localities/:locality_slug` | One locality |
| GET | `/amenities` | Amenity catalogue |
| GET | `/builders/:id` | Public builder profile |
| GET | `/pages`, `/pages/:slug` | Published informational pages |
| POST | `/inquiries` | Submit a lead |
| POST | `/events` | Record an engagement event |
| GET | `/sitemap.xml`, `/robots.txt` | Crawler artefacts (served at the root) |

### Property search

`GET /api/v1/properties`

| Parameter | Type | Notes |
|---|---|---|
| `city`, `locality` | string | Slugs, for shareable URLs |
| `city_id`, `locality_id` | int | Alternative to slugs |
| `property_type` | enum | `APARTMENT`, `FLAT`, `VILLA`, `ROW_HOUSE`, `PLOT`, `PENTHOUSE`, `STUDIO_APARTMENT`, `COMMERCIAL_OFFICE`, `COMMERCIAL_SHOP`, `OTHER` |
| `status` | enum | `UPCOMING`, `UNDER_CONSTRUCTION`, `READY_TO_MOVE`, `SOLD_OUT` |
| `min_price`, `max_price` | number | Rupees |
| `min_area`, `max_area` | number | Carpet area in sq.ft. |
| `bhk` | int | Bedrooms |
| `amenities` | string | Comma-separated amenity ids |
| `builder` | string | Builder name |
| `possession_before` | date | `YYYY-MM-DD` |
| `rera_verified` | bool | `true` restricts to verified projects |
| `featured` | bool | Featured only |
| `q` | string | Free text over project, builder and locality |
| `sort` | enum | `relevance`, `price_asc`, `price_desc`, `newest`, `popular`, `views` |
| `page`, `per_page` | int | `per_page` is capped at 100 |

Example:

```
/api/v1/properties?city=pune&bhk=2&min_price=6000000&max_price=9000000
  &status=UNDER_CONSTRUCTION&rera_verified=true&sort=price_asc
```

Only `PUBLISHED` properties are ever returned here.

### Submit an inquiry

```http
POST /api/v1/inquiries
{
  "property_id": 12,
  "name": "Rahul Deshmukh",
  "mobile": "9812345678",
  "email": "rahul@example.com",
  "message": "Please share the price list.",
  "configuration_id": 34,
  "budget": 10000000,
  "preferred_time": "EVENING",
  "source": "WEB",
  "consent_contact": true
}
```

`consent_contact` must be `true`: a lead is never stored without explicit
consent to be contacted.

### Track an event

```http
POST /api/v1/events
{ "event_type": "PHONE_CLICK", "property_id": 12, "metadata": "{}" }
```

Event types: `VIEW`, `FAVORITE`, `UNFAVORITE`, `INQUIRY`, `WHATSAPP_CLICK`,
`PHONE_CLICK`, `EMAIL_CLICK`, `BROCHURE_DOWNLOAD`, `SEARCH`.

---

## Customer endpoints

Requires `CUSTOMER` (or `ADMIN`). Every endpoint is scoped to the caller.

| Method | Path | Description |
|---|---|---|
| GET | `/customer/dashboard` | Counts, recently viewed, recommendations |
| PUT | `/customer/profile` | Update name, avatar, email, mobile |
| GET/PUT | `/customer/preferences` | Search preferences that drive recommendations |
| PUT | `/customer/consent` | Update marketing consent |
| POST | `/customer/unsubscribe` | Withdraw all non-essential consent |
| GET | `/customer/favorites` | Saved properties |
| POST/DELETE | `/customer/favorites/:property_id` | Save or unsave |
| GET | `/customer/recently-viewed` | Recently opened listings |
| GET | `/customer/recommendations` | Personalised suggestions |
| GET | `/customer/inquiries` | Inquiries this customer submitted |
| GET | `/customer/notifications` | In-app notifications |
| PUT | `/customer/notifications/:id/read` | Mark one as read |
| PUT | `/customer/notifications/read-all` | Clear the unread badge |
| GET/PUT | `/customer/notification-preferences` | Per-channel delivery switches |

Changing an email or mobile clears its verified flag until the new value is
confirmed.

---

## Builder (client) endpoints

Requires `CLIENT` (or `ADMIN`). **A builder can only ever reach their own
properties and their own leads**; ownership is checked in the service layer, not
just in the route.

| Method | Path | Description |
|---|---|---|
| GET | `/client/dashboard` | Portfolio statistics and charts |
| GET/PUT | `/client/profile` | Company profile |
| POST | `/client/logo` | Upload a company logo (multipart `file`) |
| GET | `/client/properties` | Own listings in every workflow state |
| POST | `/client/properties` | Create a draft |
| GET/PUT/DELETE | `/client/properties/:id` | Read, update, soft-delete |
| POST | `/client/properties/:id/submit` | Submit for review |
| GET | `/client/properties/:id/status-history` | Approval trail |
| GET | `/client/properties/:id/analytics` | Engagement metrics |
| GET/POST | `/client/properties/:id/media` | List or upload gallery images |
| DELETE | `/client/properties/:id/media/:media_id` | Remove an image |
| PUT | `/client/properties/:id/media/cover` | Set the cover image |
| PUT | `/client/properties/:id/media/reorder` | Reorder the gallery |
| GET/POST | `/client/properties/:id/documents` | List or upload PDFs |
| DELETE | `/client/properties/:id/documents/:doc_id` | Remove a document |
| GET | `/client/inquiries` | Leads on own properties only |
| GET | `/client/inquiries/:id` | One lead |
| PUT | `/client/inquiries/:id/status` | Move a lead along the pipeline |

### Create a property

```http
POST /api/v1/client/properties
{
  "name": "Green Valley Residency",
  "property_type": "APARTMENT",
  "construction_status": "UNDER_CONSTRUCTION",
  "city_id": 1,
  "locality_id": 3,
  "address": "Survey No 142, Baner Road",
  "pincode": "411045",
  "latitude": 18.5601, "longitude": 73.7799,
  "description": "A RERA-registered gated development...",
  "possession_date": "2027-06-01",
  "configurations": [
    { "name": "2 BHK", "carpet_area": 850, "built_up_area": 980,
      "bedrooms": 2, "bathrooms": 2, "balconies": 1,
      "price": 7500000, "number_of_units": 48 }
  ],
  "amenity_ids": [1, 2, 8],
  "rera": {
    "registration_number": "P52100047821",
    "authority": "MahaRERA",
    "registration_date": "2025-01-15",
    "completion_date": "2027-09-30"
  }
}
```

The property, its configurations, amenities and RERA record are written in one
transaction. A slug is generated as
`green-valley-residency-baner-pune`, and price and area summaries are computed
from the configurations.

### Submission requirements

`POST /client/properties/:id/submit` returns 422 unless the listing has at least
one configuration with a price, a description, a site address, at least one
property image, and a RERA registration number.

### File uploads

Multipart, field name `file`.

| Kind | Types | Limit | Extra fields |
|---|---|---|---|
| Images | JPG, PNG, WEBP | 5 MB | `media_type` |
| Documents | PDF | 20 MB | `doc_type`, `name` |

The content type is **sniffed from the file**, not taken from the request, so a
renamed executable is rejected. Keys are randomised and the first image
uploaded becomes the cover automatically.

---

## Admin endpoints

Requires `ADMIN`.

| Method | Path | Description |
|---|---|---|
| GET | `/admin/dashboard` | Platform statistics, charts, recent activity |
| GET | `/admin/analytics` | Same payload with explicit range and scope filters |
| GET | `/admin/audit-logs` | Administrative action trail |
| GET | `/admin/properties` | Every listing in every state |
| GET | `/admin/properties/:id` | Full detail for review |
| GET | `/admin/properties/:id/status-history` | Approval trail |
| PUT | `/admin/properties/:id/review` | Move `SUBMITTED` → `UNDER_REVIEW` |
| PUT | `/admin/properties/:id/approve` | Approve, optionally publish |
| PUT | `/admin/properties/:id/reject` | Reject with a reason |
| PUT | `/admin/properties/:id/rera` | Set the RERA verification status |
| PUT | `/admin/properties/:id/featured?featured=true` | Toggle featured |
| GET | `/admin/users`, `/customers`, `/clients` | Account management |
| PUT | `/admin/users/:id/status` | Suspend or reactivate |
| PUT | `/admin/users/:id/roles` | Replace the role set |
| GET | `/admin/roles` | Assignable roles |
| GET | `/admin/inquiries` | Every lead |
| POST/PUT/DELETE | `/admin/cities[/:id]` | Cities |
| POST/PUT/DELETE | `/admin/localities[/:id]` | Localities |
| POST/PUT/DELETE | `/admin/amenities[/:id]` | Amenities |
| GET/POST/PUT/DELETE | `/admin/banners[/:id]` | Homepage banners |
| GET/POST/DELETE | `/admin/featured[/:id]` | Featured properties |
| GET/POST/DELETE | `/admin/pages[/:id]` | Informational pages |
| POST | `/admin/uploads` | Upload console artwork |

**Approve**

```http
PUT /api/v1/admin/properties/12/approve
{ "publish": true, "remarks": "Verified against the MahaRERA portal" }
```

**Reject**

```http
PUT /api/v1/admin/properties/12/reject
{ "reason": "The RERA number does not match the project name." }
```

The builder is notified, sees the reason on the listing, and can correct it and
resubmit.

**Analytics filters**: `period` (`today`, `7d`, `30d`, `custom`), `from`, `to`,
`city_id`, `client_id`, `property_id`.

Approvals, rejections, RERA decisions, suspensions and featured changes are all
written to the audit log with the actor and IP address.

---

## Business rules enforced by the API

1. Only `PUBLISHED` properties appear on public endpoints.
2. A builder can read, edit and delete only their own properties.
3. A builder sees only the leads raised on their own properties.
4. Customers browse without logging in; a lead requires explicit contact consent.
5. Only an administrator can approve, reject, verify RERA or feature a listing.
6. Every state transition is recorded in `property_status_histories`.
7. Administrative actions are recorded in `audit_logs`.
8. Customer contact details are never exposed on public endpoints.

---

# Rental marketplace

The platform serves one catalogue with two purposes. Sale and rental listings
share the same table, endpoints, media, favourites, leads and analytics; only
the terms differ.

## Purpose

`GET /api/v1/properties` defaults to `purpose=BUY`, because `/properties` is the
Buy page. The rent pages pass `purpose=rent`; `purpose=all` searches both.

There is deliberately **no** separate `/rent/properties` endpoint — it would
duplicate pagination, sorting, favourite-marking, caching and event tracking,
and drift from the sale search. The pretty URLs `/rent/pune/wakad` are frontend
routes that resolve to this one parameter.

### Rental filters

Applied only when `purpose=rent`; on a sale search they are ignored, since a
sale listing has no rental record to join and would otherwise return nothing.

| Parameter | Notes |
|---|---|
| `min_rent`, `max_rent` | Rupees per month |
| `furnishing` | `UNFURNISHED`, `SEMI_FURNISHED`, `FULLY_FURNISHED` |
| `tenant_preference` | `FAMILY`, `BACHELOR`, `COMPANY_LEASE`, `ANYONE`. Listings open to anyone always match |
| `pet_allowed` | Boolean |
| `available_from` | A date, or the shorthand `now`, `7d`, `30d` |
| `sort` | Adds `rent_asc` and `rent_desc` |

Rented and withdrawn listings never appear, whatever their approval status.

`GET /meta` gains `rent_ranges`, `furnishing`, `tenant_preference`,
`availability`, `rental_photo_types` and `included_items`, so the filter
vocabulary can change without a frontend deploy.

## Posting a rental

`POST /api/v1/client/properties` with `purpose: "RENT"` and a `rental` block:

```json
{
  "name": "Rahul 2 BHK Wakad",
  "purpose": "RENT",
  "property_type": "FLAT",
  "construction_status": "READY_TO_MOVE",
  "city_id": 1, "locality_id": 2,
  "address": "Datta Mandir Road, Wakad",
  "description": "Bright 2 BHK with covered parking.",
  "configurations": [
    { "name": "2 BHK", "carpet_area": 850, "bedrooms": 2, "bathrooms": 2 }
  ],
  "rental": {
    "monthly_rent": 28000,
    "security_deposit": 84000,
    "maintenance_amount": 1500,
    "available_from": "2026-10-01",
    "furnishing_type": "SEMI_FURNISHED",
    "tenant_preference": "FAMILY",
    "pet_allowed": true,
    "included_items": ["Geyser", "Wardrobe"]
  }
}
```

Notes that differ from a sale listing:

- **A configuration needs no price.** The rent lives on the rental block; the
  configuration describes the unit. A *sale* listing still requires a price on
  every configuration.
- **RERA is not required to submit.** An individual letting a flat has no project
  registration. Submission requires the monthly rent instead. A sale listing
  still requires its RERA number.
- Room-level photo types are accepted: `LIVING_ROOM`, `BEDROOM`, `KITCHEN`,
  `BATHROOM`, `BALCONY`, `BUILDING_EXTERIOR`, `PARKING`, `AMENITIES`. Any of
  them satisfies the "at least one photo" rule and can become the cover.
- Defaults are applied where an owner is unlikely to have an opinion: an
  11-month lease and 30 days' notice.
- A rental gets a `listing_expiry_date` (`RENTAL_LISTING_DAYS`, default 60).
  Sale listings never expire.
- `posted_by_type` is derived from the account's profile type — an owner's
  listing records `OWNER` — not taken from the request.

## Registration and profile types

`POST /api/v1/auth/register` accepts `profile_type` in place of the older
`role_code`, which still works:

| `profile_type` | Role | Listing account |
|---|---|---|
| `BUYER`, `RENTER` | `CUSTOMER` | — |
| `OWNER`, `BUILDER`, `AGENT` | `CLIENT` | `profile_type` set accordingly |

A customer may also send `intents: ["BUY", "RENT"]` — one account covers both,
and no one ever needs a second login.

`POST /api/v1/customer/listing-account` upgrades an existing customer to a
listing account (`{"profile_type": "OWNER"}`). They gain the `CLIENT` role
*alongside* `CUSTOMER` and keep their saved properties and enquiries. The call
is idempotent. Their current token still holds the old role set, so the client
must refresh before the builder endpoints admit them.

## Visits

Renting rarely converts without a viewing, so this is a first-class flow.

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/properties/:key/visits` | — | Request a viewing (rate limited like a lead) |
| GET | `/customer/visits` | Customer | Visits this person asked for |
| GET | `/client/visits` | Client | Requests on this account's own properties |
| PUT | `/client/visits/:id` | Client | Confirm, reschedule, complete or cancel |

```http
POST /api/v1/properties/rahul-2-bhk-wakad-wakad-pune/visits
{
  "name": "Anjali Rao", "mobile": "9812340001", "email": "anjali@example.com",
  "visit_date": "2026-09-14", "time_slot": "EVENING",
  "message": "Could I see it this weekend?",
  "consent_contact": true
}
```

The property is taken from the path, so a visitor cannot book a viewing of one
listing while looking at another.

Status runs `REQUESTED → CONFIRMED | RESCHEDULED → COMPLETED | CANCELLED`, and
the rules are enforced server-side:

- a flat already `RENTED` or `INACTIVE` cannot be viewed
- dates in the past, or more than three months ahead, are refused
- confirming or rescheduling requires committing to a date
- a completed or cancelled visit cannot be reopened
- the same number cannot book the same property twice within six hours

`lead_type` on an inquiry is derived from the property (`BUY_LEAD` /
`RENT_LEAD`), never trusted from the caller.

The WhatsApp deep link is rental-aware: it opens with "interested in renting",
carries the monthly rent in Indian digit grouping and the furnishing, and closes
by asking about a visit.

Two engagement events are added: `CALL_CLICK` and `VISIT_REQUEST`.

---

## Owner portal, moderation and rent statistics

### Availability and renewal

Rental stock turns over in weeks. Two operations keep a listing honest without
sending it back through approval, because an owner confirming a flat is still
free has not changed anything an administrator needs to re-check.

```
PUT  /api/v1/client/properties/:id/availability
POST /api/v1/client/properties/:id/renew
GET  /api/v1/client/expiring-listings?within_days=14
```

`PUT .../availability` takes `{"availability_status": "AVAILABLE|RESERVED|RENTED|INACTIVE"}`
and an optional `available_from`, because a flat freed in March should not still
advertise January. It applies to rentals only: `400` on a sale listing.

Availability is independent of `property_status`. A let flat stays `PUBLISHED`
and keeps its history, its leads and its analytics; it simply leaves the search
results. Relisting is one call and it reappears.

`POST .../renew` extends `listing_expiry_date` by `RENTAL_LISTING_DAYS` **from
today**, not from the old expiry, so a listing renewed late still gets a full
period. A listing already retired by the expiry sweep returns to `AVAILABLE` in
the same call.

`listing_expiry_date` is now part of the property detail payload — an owner
cannot renew a listing whose expiry they cannot see.

### Expiry sweep

```
POST /api/v1/admin/listings/sweep-expiry    → {"expired": n, "warned": n}
```

Retires rentals whose expiry has passed and warns the owners of listings lapsing
within three days. It also runs on a timer every `RENTAL_SWEEP_INTERVAL_HOURS`
(default 6; `0` disables the timer and leaves this endpoint as the only trigger),
including once at startup so a server that was down over a weekend does not
serve stale listings until the first tick.

Repeated runs are safe: a retired listing is no longer returned as expiring, so
it is neither retired nor announced twice.

### Reporting a listing

```
POST /api/v1/properties/:key/report
```

Open to visitors who have not signed in, and rate limited like a lead. The
people best placed to notice a flat that is already let are the ones still
searching for it, and they will not create an account to say so.

Body: `reason` (`ALREADY_RENTED`, `FAKE_LISTING`, `INCORRECT_PRICE`,
`INCORRECT_LOCATION`, `INCORRECT_PHOTOS`, `DUPLICATE`, `FRAUD`, `OTHER`) and
optional `details`.

- Only a publicly visible listing can be reported. A draft answers `404`, so a
  report cannot be used to confirm that a listing exists at an id.
- The same reporter filing again on the same listing within 24 hours gets `409`.
  A refresh must not inflate the count that drives moderation priority.

### Moderation queue

```
GET /api/v1/admin/reports?status=OPEN&reason=&property_id=
PUT /api/v1/admin/reports/:id
```

Administrators only — showing an owner the complaints against their own flat
would tell them who reported it. Oldest first, which is the opposite of every
other listing in the API: the complaint that has waited longest is the one the
queue should surface. Each row carries `open_reports`, the number of complaints
still open against that listing, so one reported repeatedly stands out.

`PUT` takes `{"status": "REVIEWING|ACTIONED|DISMISSED", "action_note": "...",
"unpublish": true}`. With `unpublish`, the listing goes back to the owner as
`REJECTED` with the note as the reason, so they can see what was wrong and fix
it rather than the listing silently vanishing. The decision is written to the
audit log.

### Verification and posting on behalf of an owner

```
PUT  /api/v1/admin/clients/:id/verification
POST /api/v1/admin/clients/:id/properties
```

`verification_level` is ordered — `UNVERIFIED`, `MOBILE_VERIFIED`,
`EMAIL_VERIFIED`, `DOCUMENT_VERIFIED`, `ADMIN_VERIFIED` — and gates the trust
badge on every listing that account posts. It is an administrator decision with
an audit entry and a notification to the account, never a self-service claim.

**No identity documents are collected or stored.** The endpoint records the
outcome of a check made elsewhere. Collecting documents would need consent,
retention and access controls this platform does not have, and adding the field
without them would be worse than not having the feature.

`POST /admin/clients/:id/properties` posts a listing into that account — details
taken over the phone, most often. The listing is the owner's: it carries their
`client_id`, so their leads, analytics and edits all work normally. The
provenance does not pretend otherwise: `posted_by_type` is `ADMIN` and
`owner_user_id` records whose flat it is.

### Purpose-split analytics

Both dashboards now split the catalogue, because a single "properties" figure
hides which half of the business is growing:

- `stats.sale_properties`, `stats.rental_properties`, `stats.rented_properties`
- `stats.buy_leads`, `stats.rent_leads`
- Client only: `stats.pending_visits`, `stats.expiring_soon`
- Admin only: `stats.open_reports`

Every figure that existed before is unchanged.

### Rent guide

```
GET /api/v1/rent-guide?city_id=&locality_id=&bhk=
→ {"listings": 3, "average_rent": 33667, "min_rent": 18000, "max_rent": 55000}
```

What a flat in an area actually goes for — the question a renter has before they
can judge any single listing. Computed from **live listings only**: published,
`AVAILABLE`, and not soft-deleted. A rent asked six months ago on a flat since
let is not a guide to anything.

`listings` is returned so a thin sample is visible as one. An area with nothing
in it reports zeros rather than a made-up figure, and the frontend hides the
panel below three listings rather than presenting one flat as the market rate.

### Saved rental searches

A renter who has not found the flat yet should not have to check the site every
day. `POST /customer/rental-alerts` saves the current rent search; the renter is
notified when a matching rental is **published**, not merely approved — an
approved-but-unpublished listing is not yet openable, so announcing it would
send them to a page they cannot see.

```
GET    /api/v1/customer/rental-alerts        list saved searches
POST   /api/v1/customer/rental-alerts        save the current search
PUT    /api/v1/customer/rental-alerts/:id    pause or resume  { "is_active": false }
DELETE /api/v1/customer/rental-alerts/:id    delete
```

The saved criteria are `city_id`, `locality_id`, `bhk`, `min_rent`, `max_rent`,
`furnishing` and `tenant_preference`. Matching uses the first five; furnishing
and tenant preference are carried so the returned `search_query` reproduces the
renter's search exactly:

```json
{
  "id": 4,
  "name": "2 BHK in Baner under ₹30,000",
  "search_query": "bhk=2&city_id=1&max_rent=30000&purpose=rent",
  "is_active": true,
  "match_count": 3
}
```

Three rules are deliberate. A zero criterion means "no constraint", the same
convention `PropertyFilter` uses, so an alert saved for a city matches on city
alone. An unnamed search is labelled from what was searched for, so a list of
saved searches does not read as "Untitled" three times over. And each account is
capped at 20 saved searches, because an unbounded alert list is a subscription
to the whole catalogue rather than a watch for a flat.

Pausing is offered alongside deleting: a renter who has tuned a rent band and a
locality should be able to stop the notices for a fortnight without rebuilding
the search.

Delivery is immediate: a matching renter is notified when the listing is
published. There is deliberately no daily or weekly digest option — nothing in
this platform batches notices, so offering the choice would be a setting that
silently did nothing. A digest would need a scheduler, which does not exist yet.
