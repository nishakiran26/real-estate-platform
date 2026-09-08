-- Rental marketplace — schema extension.
--
-- Additive only. No column is dropped, renamed or retyped, so existing Buy and
-- RERA listings keep working while this is applied and after.
--
-- The order matters: columns are added nullable, existing rows are backfilled,
-- the backfill is verified, and only then are defaults and NOT NULL applied. A
-- migration interrupted halfway therefore cannot leave rows violating a new
-- constraint.

SET NAMES utf8mb4;

-- ---------------------------------------------------------------------------
-- Step 1 — add the new columns, nullable for now
-- ---------------------------------------------------------------------------

ALTER TABLE properties
  ADD COLUMN purpose             VARCHAR(10)  NULL AFTER slug,
  ADD COLUMN posted_by_type      VARCHAR(20)  NULL AFTER user_id,
  ADD COLUMN owner_user_id       BIGINT UNSIGNED NULL AFTER user_id,
  ADD COLUMN availability_status VARCHAR(20)  NULL AFTER property_status,
  ADD COLUMN listing_expiry_date DATETIME(3)  NULL AFTER possession_date;

ALTER TABLE clients
  ADD COLUMN profile_type       VARCHAR(20) NULL AFTER company_name,
  ADD COLUMN verification_level VARCHAR(30) NULL AFTER profile_type,
  ADD COLUMN verified_at        DATETIME(3) NULL,
  ADD COLUMN verified_by        BIGINT UNSIGNED NULL;

ALTER TABLE inquiries
  ADD COLUMN lead_type           VARCHAR(20) NULL AFTER source,
  ADD COLUMN assigned_to_user_id BIGINT UNSIGNED NULL,
  ADD COLUMN assigned_at         DATETIME(3) NULL,
  ADD COLUMN assigned_by         BIGINT UNSIGNED NULL;

-- ---------------------------------------------------------------------------
-- Step 2 — backfill existing rows
--
-- Everything that existed before the rental marketplace is a sale listing
-- posted by a builder, and every lead against it is a buy lead.
-- ---------------------------------------------------------------------------

UPDATE properties SET purpose             = 'BUY'       WHERE purpose IS NULL;
UPDATE properties SET posted_by_type      = 'CLIENT'    WHERE posted_by_type IS NULL;
UPDATE properties SET availability_status = 'AVAILABLE' WHERE availability_status IS NULL;

-- A sold-out sale listing is unavailable, not merely published. This keeps the
-- two columns telling a consistent story from the first day.
UPDATE properties
   SET availability_status = 'INACTIVE'
 WHERE property_status = 'SOLD_OUT';

UPDATE clients SET profile_type       = 'BUILDER'    WHERE profile_type IS NULL;
UPDATE clients SET verification_level = 'UNVERIFIED' WHERE verification_level IS NULL;

-- Existing builder accounts whose owning user has already confirmed both
-- channels start at EMAIL_VERIFIED rather than being demoted to UNVERIFIED.
UPDATE clients c
  JOIN users u ON u.id = c.user_id
   SET c.verification_level = 'EMAIL_VERIFIED'
 WHERE u.email_verified = TRUE
   AND u.mobile_verified = TRUE;

UPDATE inquiries SET lead_type = 'BUY_LEAD' WHERE lead_type IS NULL;

-- ---------------------------------------------------------------------------
-- Step 3 — verify the backfill before adding constraints
--
-- MySQL can only raise an error from a stored program, so the check runs inside
-- a throwaway procedure. If any row was missed, SIGNAL aborts the script here
-- with a readable message, before Step 4 tries to add NOT NULL and fails with a
-- far less obvious one.
-- ---------------------------------------------------------------------------

DROP PROCEDURE IF EXISTS assert_rental_backfill;

DELIMITER //
CREATE PROCEDURE assert_rental_backfill()
BEGIN
  DECLARE missing_properties INT DEFAULT 0;
  DECLARE missing_clients    INT DEFAULT 0;
  DECLARE missing_inquiries  INT DEFAULT 0;
  DECLARE msg TEXT;

  SELECT COUNT(*) INTO missing_properties FROM properties
   WHERE purpose IS NULL OR posted_by_type IS NULL OR availability_status IS NULL;

  SELECT COUNT(*) INTO missing_clients FROM clients
   WHERE profile_type IS NULL OR verification_level IS NULL;

  SELECT COUNT(*) INTO missing_inquiries FROM inquiries
   WHERE lead_type IS NULL;

  IF missing_properties > 0 OR missing_clients > 0 OR missing_inquiries > 0 THEN
    SET msg = CONCAT('Rental backfill incomplete — properties: ', missing_properties,
                     ', clients: ', missing_clients,
                     ', inquiries: ', missing_inquiries,
                     '. Re-run step 2 before applying constraints.');
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = msg;
  END IF;
END //
DELIMITER ;

CALL assert_rental_backfill();
DROP PROCEDURE assert_rental_backfill;

-- ---------------------------------------------------------------------------
-- Step 4 — apply defaults and NOT NULL now that every row has a value
-- ---------------------------------------------------------------------------

ALTER TABLE properties
  MODIFY COLUMN purpose             VARCHAR(10) NOT NULL DEFAULT 'BUY',
  MODIFY COLUMN posted_by_type      VARCHAR(20) NOT NULL DEFAULT 'CLIENT',
  MODIFY COLUMN availability_status VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE';

ALTER TABLE clients
  MODIFY COLUMN profile_type       VARCHAR(20) NOT NULL DEFAULT 'BUILDER',
  MODIFY COLUMN verification_level VARCHAR(30) NOT NULL DEFAULT 'UNVERIFIED';

ALTER TABLE inquiries
  MODIFY COLUMN lead_type VARCHAR(20) NOT NULL DEFAULT 'BUY_LEAD';

-- ---------------------------------------------------------------------------
-- Step 5 — new tables
-- ---------------------------------------------------------------------------

-- Rental terms live in a side table so a sale listing does not carry a dozen
-- permanently null columns. Shaped like rera_details so it loads the same way.
CREATE TABLE IF NOT EXISTS rental_details (
  id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at           DATETIME(3) NULL,
  updated_at           DATETIME(3) NULL,
  deleted_at           DATETIME(3) NULL,
  property_id          BIGINT UNSIGNED NOT NULL,
  monthly_rent         DOUBLE NOT NULL DEFAULT 0,
  security_deposit     DOUBLE NOT NULL DEFAULT 0,
  maintenance_amount   DOUBLE NOT NULL DEFAULT 0,
  maintenance_included BOOLEAN NOT NULL DEFAULT FALSE,
  available_from       DATETIME(3) NULL,
  min_lease_months     BIGINT NOT NULL DEFAULT 11,
  max_lease_months     BIGINT NOT NULL DEFAULT 0,
  notice_period_days   BIGINT NOT NULL DEFAULT 30,
  furnishing_type      VARCHAR(30) NOT NULL DEFAULT 'UNFURNISHED',
  tenant_preference    VARCHAR(30) NOT NULL DEFAULT 'ANYONE',
  pet_allowed          BOOLEAN NOT NULL DEFAULT FALSE,
  included_items       JSON NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_rental_details_property (property_id),
  KEY idx_rental_details_deleted_at (deleted_at),
  KEY idx_rental_rent (monthly_rent),
  KEY idx_rental_available (available_from),
  KEY idx_rental_furnishing (furnishing_type, tenant_preference),
  CONSTRAINT fk_rental_details_property FOREIGN KEY (property_id)
    REFERENCES properties (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Visit requests. Renting rarely converts without a viewing, so this is a
-- first-class record rather than a note on an inquiry.
CREATE TABLE IF NOT EXISTS property_visits (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at   DATETIME(3) NULL,
  updated_at   DATETIME(3) NULL,
  deleted_at   DATETIME(3) NULL,
  property_id  BIGINT UNSIGNED NOT NULL,
  client_id    BIGINT UNSIGNED NOT NULL,
  user_id      BIGINT UNSIGNED NULL,
  name         VARCHAR(255) NOT NULL,
  mobile       VARCHAR(15)  NOT NULL,
  email        VARCHAR(255) NULL,
  message      TEXT NULL,
  session_id   VARCHAR(64) NULL,
  requested_at DATETIME(3) NOT NULL,
  scheduled_at DATETIME(3) NULL,
  time_slot    VARCHAR(50) NULL,
  status       VARCHAR(30) NOT NULL DEFAULT 'REQUESTED',
  remarks      TEXT NULL,
  handled_by   BIGINT UNSIGNED NULL,
  handled_at   DATETIME(3) NULL,
  completed_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_visits_property (property_id),
  KEY idx_visits_client (client_id),
  KEY idx_visits_user (user_id),
  KEY idx_visits_session (session_id),
  KEY idx_visits_status (status),
  KEY idx_visits_deleted_at (deleted_at),
  KEY idx_visits_property_status (property_id, status, scheduled_at),
  KEY idx_visits_client_status (client_id, status, requested_at),
  CONSTRAINT fk_visits_property FOREIGN KEY (property_id)
    REFERENCES properties (id) ON DELETE CASCADE,
  CONSTRAINT fk_visits_client FOREIGN KEY (client_id)
    REFERENCES clients (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Abuse reports. Once the public can post listings, a moderation path is a
-- launch requirement, not a refinement.
CREATE TABLE IF NOT EXISTS property_reports (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at  DATETIME(3) NULL,
  updated_at  DATETIME(3) NULL,
  deleted_at  DATETIME(3) NULL,
  property_id BIGINT UNSIGNED NOT NULL,
  reporter_id BIGINT UNSIGNED NULL,
  reason      VARCHAR(40) NOT NULL,
  details     TEXT NULL,
  session_id  VARCHAR(64) NULL,
  ip_address  VARCHAR(45) NULL,
  status      VARCHAR(20) NOT NULL DEFAULT 'OPEN',
  reviewed_by BIGINT UNSIGNED NULL,
  reviewed_at DATETIME(3) NULL,
  action_note TEXT NULL,
  PRIMARY KEY (id),
  KEY idx_reports_property (property_id),
  KEY idx_reports_reporter (reporter_id),
  KEY idx_reports_reason (reason),
  KEY idx_reports_deleted_at (deleted_at),
  KEY idx_reports_status (status, created_at),
  CONSTRAINT fk_reports_property FOREIGN KEY (property_id)
    REFERENCES properties (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Saved rental searches. Criteria are JSON so the alert always matches whatever
-- the live search filter supports; the hot fields are duplicated as columns so
-- a newly published property can be matched with an indexed query.
CREATE TABLE IF NOT EXISTS rental_alerts (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at       DATETIME(3) NULL,
  updated_at       DATETIME(3) NULL,
  deleted_at       DATETIME(3) NULL,
  user_id          BIGINT UNSIGNED NOT NULL,
  name             VARCHAR(200) NULL,
  criteria         JSON NOT NULL,
  city_id          BIGINT UNSIGNED NULL,
  locality_id      BIGINT UNSIGNED NULL,
  bhk              BIGINT NULL,
  min_rent         DOUBLE NULL,
  max_rent         DOUBLE NULL,
  frequency        VARCHAR(20) NOT NULL DEFAULT 'DAILY',
  is_active        BOOLEAN NOT NULL DEFAULT TRUE,
  last_notified_at DATETIME(3) NULL,
  match_count      BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_alerts_user (user_id),
  KEY idx_alerts_city (city_id),
  KEY idx_alerts_locality (locality_id),
  KEY idx_alerts_bhk (bhk),
  KEY idx_alerts_active (is_active),
  KEY idx_alerts_deleted_at (deleted_at),
  CONSTRAINT fk_alerts_user FOREIGN KEY (user_id)
    REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ---------------------------------------------------------------------------
-- Step 6 — indexes on the extended tables
-- ---------------------------------------------------------------------------

CREATE INDEX idx_properties_purpose        ON properties (purpose);
CREATE INDEX idx_properties_posted_by      ON properties (posted_by_type);
CREATE INDEX idx_properties_owner_user     ON properties (owner_user_id);
CREATE INDEX idx_properties_availability   ON properties (availability_status);
CREATE INDEX idx_properties_purpose_city   ON properties (property_status, purpose, city_id, locality_id);
CREATE INDEX idx_properties_purpose_avail  ON properties (property_status, purpose, availability_status);
CREATE INDEX idx_properties_expiry         ON properties (purpose, listing_expiry_date);

CREATE INDEX idx_clients_profile_type      ON clients (profile_type);
CREATE INDEX idx_clients_verification      ON clients (verification_level);

CREATE INDEX idx_inquiries_assigned        ON inquiries (assigned_to_user_id);
CREATE INDEX idx_inquiries_lead_type       ON inquiries (client_id, lead_type, created_at);

-- ---------------------------------------------------------------------------
-- Step 7 — customer intent
--
-- Everyone who registered before the rental marketplace signed up to buy, so
-- their existing profile is marked accordingly. LookingToBuy carries no column
-- default: the application always sets it explicitly, because a DEFAULT would
-- make GORM overwrite a deliberate "false" for a renter-only profile.
-- ---------------------------------------------------------------------------

ALTER TABLE customer_profiles
  ADD COLUMN looking_to_buy         BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN looking_to_rent        BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN monthly_rent_min       DOUBLE NOT NULL DEFAULT 0,
  ADD COLUMN monthly_rent_max       DOUBLE NOT NULL DEFAULT 0,
  ADD COLUMN preferred_furnishing   VARCHAR(30) NULL,
  ADD COLUMN preferred_lease_months BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN move_in_date           DATETIME(3) NULL,
  ADD COLUMN family_type            VARCHAR(30) NULL;

UPDATE customer_profiles SET looking_to_buy  = TRUE WHERE purpose IN ('BUY', 'INVEST');
UPDATE customer_profiles SET looking_to_rent = TRUE WHERE purpose = 'RENT';

-- A profile with no usable purpose still wants to see something, so it defaults
-- to buying rather than to an empty feed.
UPDATE customer_profiles
   SET looking_to_buy = TRUE
 WHERE looking_to_buy = FALSE AND looking_to_rent = FALSE;

CREATE INDEX idx_profiles_looking_to_buy  ON customer_profiles (looking_to_buy);
CREATE INDEX idx_profiles_looking_to_rent ON customer_profiles (looking_to_rent);
