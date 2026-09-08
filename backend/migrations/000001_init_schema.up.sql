-- RERA Property Platform — initial schema.
--
-- This file mirrors the GORM models in internal/models. The application can
-- create the same schema itself with DB_AUTO_MIGRATE=true; these files exist
-- for teams that apply migrations explicitly (CI, managed databases, review).
--
-- Conventions: InnoDB + utf8mb4, bigint unsigned surrogate keys, datetime(3)
-- timestamps, soft deletes through deleted_at on business tables.

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 1;

-- ---------------------------------------------------------------------------
-- Identity and access
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS roles (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at    DATETIME(3) NULL,
  updated_at    DATETIME(3) NULL,
  deleted_at    DATETIME(3) NULL,
  code          VARCHAR(50)  NOT NULL,
  name          VARCHAR(100) NOT NULL,
  description   VARCHAR(500) NULL,
  permissions   TEXT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_roles_code (code),
  KEY idx_roles_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS users (
  id                         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at                 DATETIME(3) NULL,
  updated_at                 DATETIME(3) NULL,
  deleted_at                 DATETIME(3) NULL,
  email                      VARCHAR(255) NOT NULL,
  mobile                     VARCHAR(15)  NOT NULL,
  password_hash              VARCHAR(255) NOT NULL,
  full_name                  VARCHAR(255) NOT NULL,
  avatar_url                 VARCHAR(512) NULL,
  status                     VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
  email_verified             BOOLEAN NOT NULL DEFAULT FALSE,
  mobile_verified            BOOLEAN NOT NULL DEFAULT FALSE,
  email_verify_token         VARCHAR(255) NULL,
  last_login_at              DATETIME(3) NULL,
  consent_property_updates   BOOLEAN NOT NULL DEFAULT FALSE,
  consent_new_project_alerts BOOLEAN NOT NULL DEFAULT FALSE,
  consent_promotional        BOOLEAN NOT NULL DEFAULT FALSE,
  consent_whats_app          BOOLEAN NOT NULL DEFAULT FALSE,
  consent_email              BOOLEAN NOT NULL DEFAULT TRUE,
  consent_updated_at         DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_users_email (email),
  UNIQUE KEY uq_users_mobile (mobile),
  KEY idx_users_deleted_at (deleted_at),
  KEY idx_users_status (status),
  KEY idx_users_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_roles (
  user_id BIGINT UNSIGNED NOT NULL,
  role_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (user_id, role_id),
  KEY idx_user_roles_role (role_id),
  CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  user_id    BIGINT UNSIGNED NOT NULL,
  token_hash VARCHAR(255) NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  revoked_at DATETIME(3) NULL,
  user_agent VARCHAR(512) NULL,
  ip_address VARCHAR(45)  NULL,
  family_id  VARCHAR(64)  NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_refresh_tokens_hash (token_hash),
  KEY idx_refresh_tokens_user (user_id),
  KEY idx_refresh_tokens_family (family_id),
  KEY idx_refresh_tokens_deleted_at (deleted_at),
  CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS login_histories (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at  DATETIME(3) NULL,
  updated_at  DATETIME(3) NULL,
  deleted_at  DATETIME(3) NULL,
  user_id     BIGINT UNSIGNED NOT NULL,
  method      VARCHAR(30) NOT NULL,
  ip_address  VARCHAR(45) NULL,
  user_agent  VARCHAR(512) NULL,
  login_at    DATETIME(3) NOT NULL,
  logout_at   DATETIME(3) NULL,
  device_name VARCHAR(255) NULL,
  PRIMARY KEY (id),
  KEY idx_login_histories_user (user_id),
  KEY idx_login_histories_deleted_at (deleted_at),
  CONSTRAINT fk_login_histories_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS otp_requests (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at   DATETIME(3) NULL,
  updated_at   DATETIME(3) NULL,
  deleted_at   DATETIME(3) NULL,
  user_id      BIGINT UNSIGNED NULL,
  email        VARCHAR(255) NULL,
  mobile       VARCHAR(15)  NULL,
  code_hash    VARCHAR(255) NOT NULL,
  purpose      VARCHAR(30)  NOT NULL,
  attempts     INT NOT NULL DEFAULT 0,
  max_attempts INT NOT NULL DEFAULT 5,
  expires_at   DATETIME(3) NOT NULL,
  verified_at  DATETIME(3) NULL,
  ip_address   VARCHAR(45) NULL,
  PRIMARY KEY (id),
  KEY idx_otp_requests_user (user_id),
  KEY idx_otp_requests_email (email),
  KEY idx_otp_requests_mobile (mobile),
  KEY idx_otp_requests_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ---------------------------------------------------------------------------
-- Accounts
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS clients (
  id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at          DATETIME(3) NULL,
  updated_at          DATETIME(3) NULL,
  deleted_at          DATETIME(3) NULL,
  user_id             BIGINT UNSIGNED NOT NULL,
  company_name        VARCHAR(300) NOT NULL,
  company_logo_url    VARCHAR(512) NULL,
  company_desc        TEXT NULL,
  website             VARCHAR(500) NULL,
  contact_phone       VARCHAR(15)  NULL,
  contact_email       VARCHAR(255) NULL,
  address             TEXT NULL,
  city                VARCHAR(100) NULL,
  state               VARCHAR(100) NULL,
  pincode             VARCHAR(10)  NULL,
  gstin               VARCHAR(20)  NULL,
  rerareg_no          VARCHAR(100) NULL,
  total_properties    INT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uq_clients_user (user_id),
  KEY idx_clients_deleted_at (deleted_at),
  CONSTRAINT fk_clients_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS customer_profiles (
  id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at         DATETIME(3) NULL,
  updated_at         DATETIME(3) NULL,
  deleted_at         DATETIME(3) NULL,
  user_id            BIGINT UNSIGNED NOT NULL,
  preferred_city     VARCHAR(100) NULL,
  preferred_locality VARCHAR(200) NULL,
  min_budget         DOUBLE NOT NULL DEFAULT 0,
  max_budget         DOUBLE NOT NULL DEFAULT 0,
  property_type      VARCHAR(50) NULL,
  bhk                BIGINT NOT NULL DEFAULT 0,
  purpose            VARCHAR(20) NOT NULL DEFAULT 'BUY',
  PRIMARY KEY (id),
  UNIQUE KEY uq_customer_profiles_user (user_id),
  KEY idx_customer_profiles_deleted_at (deleted_at),
  CONSTRAINT fk_customer_profiles_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ---------------------------------------------------------------------------
-- Geography and reference data
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cities (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at     DATETIME(3) NULL,
  updated_at     DATETIME(3) NULL,
  deleted_at     DATETIME(3) NULL,
  name           VARCHAR(100) NOT NULL,
  slug           VARCHAR(100) NOT NULL,
  state          VARCHAR(100) NOT NULL,
  state_code     VARCHAR(10) NULL,
  pincode_prefix VARCHAR(10) NULL,
  latitude       DOUBLE NULL,
  longitude      DOUBLE NULL,
  is_popular     BOOLEAN NOT NULL DEFAULT FALSE,
  display_order  BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uq_cities_name (name),
  UNIQUE KEY uq_cities_slug (slug),
  KEY idx_cities_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS localities (
  id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at         DATETIME(3) NULL,
  updated_at         DATETIME(3) NULL,
  deleted_at         DATETIME(3) NULL,
  city_id            BIGINT UNSIGNED NOT NULL,
  name               VARCHAR(200) NOT NULL,
  slug               VARCHAR(200) NOT NULL,
  pincode            VARCHAR(10) NULL,
  latitude           DOUBLE NULL,
  longitude          DOUBLE NULL,
  avg_price_per_sqft DOUBLE NULL,
  is_popular         BOOLEAN NOT NULL DEFAULT FALSE,
  display_order      BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY idx_localities_city_slug (city_id, slug),
  KEY idx_localities_deleted_at (deleted_at),
  CONSTRAINT fk_localities_city FOREIGN KEY (city_id) REFERENCES cities (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS amenities (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at    DATETIME(3) NULL,
  updated_at    DATETIME(3) NULL,
  deleted_at    DATETIME(3) NULL,
  name          VARCHAR(100) NOT NULL,
  slug          VARCHAR(100) NOT NULL,
  icon          VARCHAR(255) NULL,
  category      VARCHAR(50) NOT NULL DEFAULT 'GENERAL',
  display_order BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uq_amenities_name (name),
  UNIQUE KEY uq_amenities_slug (slug),
  KEY idx_amenities_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ---------------------------------------------------------------------------
-- Properties
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS properties (
  id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at            DATETIME(3) NULL,
  updated_at            DATETIME(3) NULL,
  deleted_at            DATETIME(3) NULL,
  client_id             BIGINT UNSIGNED NOT NULL,
  user_id               BIGINT UNSIGNED NOT NULL,
  name                  VARCHAR(300) NOT NULL,
  slug                  VARCHAR(300) NOT NULL,
  project_name          VARCHAR(300) NULL,
  builder_name          VARCHAR(300) NULL,
  property_type         VARCHAR(50)  NOT NULL,
  property_status       VARCHAR(30)  NOT NULL DEFAULT 'DRAFT',
  construction_status   VARCHAR(30)  NULL,
  city_id               BIGINT UNSIGNED NOT NULL,
  locality_id           BIGINT UNSIGNED NOT NULL,
  address               TEXT NULL,
  pincode               VARCHAR(10) NULL,
  latitude              DOUBLE NULL,
  longitude             DOUBLE NULL,
  description           TEXT NULL,
  min_price             DOUBLE NOT NULL DEFAULT 0,
  max_price             DOUBLE NOT NULL DEFAULT 0,
  min_area              DOUBLE NOT NULL DEFAULT 0,
  max_area              DOUBLE NOT NULL DEFAULT 0,
  possession_date       DATETIME(3) NULL,
  is_featured           BOOLEAN NOT NULL DEFAULT FALSE,
  is_reraverified       BOOLEAN NOT NULL DEFAULT FALSE,
  rejection_reason      LONGTEXT NULL,
  submitted_at          DATETIME(3) NULL,
  approved_at           DATETIME(3) NULL,
  approved_by           BIGINT UNSIGNED NULL,
  rejected_at           DATETIME(3) NULL,
  rejected_by           BIGINT UNSIGNED NULL,
  meta_title            VARCHAR(300) NULL,
  meta_description      VARCHAR(500) NULL,
  view_count            BIGINT NOT NULL DEFAULT 0,
  inquiry_count         BIGINT NOT NULL DEFAULT 0,
  whatsapp_click_count  BIGINT NOT NULL DEFAULT 0,
  favorite_count        BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uq_properties_slug (slug),
  KEY idx_properties_deleted_at (deleted_at),
  KEY idx_properties_client (client_id),
  KEY idx_properties_user (user_id),
  KEY idx_properties_type (property_type),
  KEY idx_properties_status (property_status),
  KEY idx_properties_construction (construction_status),
  KEY idx_properties_city (city_id),
  KEY idx_properties_locality (locality_id),
  KEY idx_properties_min_price (min_price),
  KEY idx_properties_max_price (max_price),
  KEY idx_properties_featured (is_featured),
  KEY idx_properties_rera (is_reraverified),
  KEY idx_properties_status_city (property_status, city_id, locality_id),
  KEY idx_properties_status_price (property_status, min_price, max_price),
  KEY idx_properties_status_created (property_status, created_at),
  KEY idx_properties_client_status (client_id, property_status),
  CONSTRAINT fk_properties_client FOREIGN KEY (client_id) REFERENCES clients (id),
  CONSTRAINT fk_properties_city FOREIGN KEY (city_id) REFERENCES cities (id),
  CONSTRAINT fk_properties_locality FOREIGN KEY (locality_id) REFERENCES localities (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS property_configurations (
  id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at          DATETIME(3) NULL,
  updated_at          DATETIME(3) NULL,
  deleted_at          DATETIME(3) NULL,
  property_id         BIGINT UNSIGNED NOT NULL,
  name                VARCHAR(100) NOT NULL,
  carpet_area         DOUBLE NULL,
  built_up_area       DOUBLE NULL,
  super_built_up_area DOUBLE NULL,
  bedrooms            BIGINT NOT NULL DEFAULT 0,
  bathrooms           BIGINT NOT NULL DEFAULT 0,
  balconies           BIGINT NOT NULL DEFAULT 0,
  price               DOUBLE NOT NULL DEFAULT 0,
  min_price           DOUBLE NOT NULL DEFAULT 0,
  max_price           DOUBLE NOT NULL DEFAULT 0,
  price_per_sqft      DOUBLE NULL,
  availability        VARCHAR(50) NOT NULL DEFAULT 'AVAILABLE',
  number_of_units     BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_configs_property (property_id),
  KEY idx_configs_price (price),
  KEY idx_configs_deleted_at (deleted_at),
  KEY idx_configs_property_bedrooms (property_id, bedrooms),
  CONSTRAINT fk_configs_property FOREIGN KEY (property_id) REFERENCES properties (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS property_amenities (
  property_id BIGINT UNSIGNED NOT NULL,
  amenity_id  BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (property_id, amenity_id),
  KEY idx_property_amenities_amenity (amenity_id),
  CONSTRAINT fk_property_amenities_property FOREIGN KEY (property_id) REFERENCES properties (id) ON DELETE CASCADE,
  CONSTRAINT fk_property_amenities_amenity FOREIGN KEY (amenity_id) REFERENCES amenities (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS property_media (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at     DATETIME(3) NULL,
  updated_at     DATETIME(3) NULL,
  deleted_at     DATETIME(3) NULL,
  property_id    BIGINT UNSIGNED NOT NULL,
  media_type     VARCHAR(50)  NOT NULL,
  file_url       VARCHAR(512) NOT NULL,
  storage_key    VARCHAR(512) NULL,
  thumbnail_url  VARCHAR(512) NULL,
  display_order  BIGINT NOT NULL DEFAULT 0,
  is_cover_image BOOLEAN NOT NULL DEFAULT FALSE,
  PRIMARY KEY (id),
  KEY idx_property_media_property (property_id),
  KEY idx_property_media_deleted_at (deleted_at),
  CONSTRAINT fk_property_media_property FOREIGN KEY (property_id) REFERENCES properties (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS property_documents (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at     DATETIME(3) NULL,
  updated_at     DATETIME(3) NULL,
  deleted_at     DATETIME(3) NULL,
  property_id    BIGINT UNSIGNED NOT NULL,
  doc_type       VARCHAR(50)  NOT NULL,
  name           VARCHAR(300) NOT NULL,
  file_url       VARCHAR(512) NOT NULL,
  storage_key    VARCHAR(512) NULL,
  file_size      BIGINT NOT NULL DEFAULT 0,
  mime_type      VARCHAR(100) NULL,
  download_count BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_property_documents_property (property_id),
  KEY idx_property_documents_deleted_at (deleted_at),
  CONSTRAINT fk_property_documents_property FOREIGN KEY (property_id) REFERENCES properties (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS rera_details (
  id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at          DATETIME(3) NULL,
  updated_at          DATETIME(3) NULL,
  deleted_at          DATETIME(3) NULL,
  property_id         BIGINT UNSIGNED NOT NULL,
  registration_number VARCHAR(100) NULL,
  authority           VARCHAR(200) NULL,
  registration_date   DATETIME(3) NULL,
  completion_date     DATETIME(3) NULL,
  verification_status VARCHAR(30) NOT NULL DEFAULT 'PENDING_VERIFICATION',
  verified_by         BIGINT UNSIGNED NULL,
  verified_at         DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_rera_details_property (property_id),
  KEY idx_rera_details_number (registration_number),
  KEY idx_rera_details_status (verification_status),
  KEY idx_rera_details_deleted_at (deleted_at),
  CONSTRAINT fk_rera_details_property FOREIGN KEY (property_id) REFERENCES properties (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS property_status_histories (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at  DATETIME(3) NULL,
  updated_at  DATETIME(3) NULL,
  deleted_at  DATETIME(3) NULL,
  property_id BIGINT UNSIGNED NOT NULL,
  from_status VARCHAR(30) NOT NULL,
  to_status   VARCHAR(30) NOT NULL,
  actor_id    BIGINT UNSIGNED NOT NULL,
  remarks     TEXT NULL,
  PRIMARY KEY (id),
  KEY idx_status_histories_property (property_id),
  KEY idx_status_histories_actor (actor_id),
  KEY idx_status_histories_deleted_at (deleted_at),
  CONSTRAINT fk_status_histories_property FOREIGN KEY (property_id) REFERENCES properties (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ---------------------------------------------------------------------------
-- Engagement: leads, favourites, events
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS inquiries (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at       DATETIME(3) NULL,
  updated_at       DATETIME(3) NULL,
  deleted_at       DATETIME(3) NULL,
  property_id      BIGINT UNSIGNED NOT NULL,
  client_id        BIGINT UNSIGNED NOT NULL,
  user_id          BIGINT UNSIGNED NULL,
  name             VARCHAR(255) NOT NULL,
  mobile           VARCHAR(15)  NOT NULL,
  email            VARCHAR(255) NULL,
  message          TEXT NULL,
  configuration_id BIGINT UNSIGNED NULL,
  budget           DOUBLE NOT NULL DEFAULT 0,
  preferred_time   VARCHAR(100) NULL,
  session_id       VARCHAR(64) NULL,
  status           VARCHAR(30) NOT NULL DEFAULT 'NEW',
  source           VARCHAR(50) NOT NULL DEFAULT 'WEB',
  ip_address       VARCHAR(45) NULL,
  followed_up_at   DATETIME(3) NULL,
  followed_by      BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  KEY idx_inquiries_property (property_id),
  KEY idx_inquiries_client (client_id),
  KEY idx_inquiries_user (user_id),
  KEY idx_inquiries_session (session_id),
  KEY idx_inquiries_deleted_at (deleted_at),
  KEY idx_inquiries_client_status (client_id, status, created_at),
  CONSTRAINT fk_inquiries_property FOREIGN KEY (property_id) REFERENCES properties (id) ON DELETE CASCADE,
  CONSTRAINT fk_inquiries_client FOREIGN KEY (client_id) REFERENCES clients (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS favorites (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at  DATETIME(3) NULL,
  updated_at  DATETIME(3) NULL,
  deleted_at  DATETIME(3) NULL,
  user_id     BIGINT UNSIGNED NOT NULL,
  property_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_user_property (user_id, property_id),
  KEY idx_favorites_deleted_at (deleted_at),
  CONSTRAINT fk_favorites_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_favorites_property FOREIGN KEY (property_id) REFERENCES properties (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_property_events (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id     BIGINT UNSIGNED NULL,
  property_id BIGINT UNSIGNED NULL,
  event_type  VARCHAR(30) NOT NULL,
  session_id  VARCHAR(64) NULL,
  ip_address  VARCHAR(45) NULL,
  user_agent  VARCHAR(512) NULL,
  metadata    JSON NULL,
  created_at  DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_events_user (user_id),
  KEY idx_events_property (property_id),
  KEY idx_events_type (event_type),
  KEY idx_events_session (session_id),
  KEY idx_events_created_at (created_at),
  KEY idx_events_property_type_date (property_id, event_type, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS property_views (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  property_id BIGINT UNSIGNED NOT NULL,
  user_id     BIGINT UNSIGNED NULL,
  session_id  VARCHAR(64) NULL,
  ip_address  VARCHAR(45) NULL,
  user_agent  VARCHAR(512) NULL,
  referrer    VARCHAR(512) NULL,
  duration    BIGINT NOT NULL DEFAULT 0,
  created_at  DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_views_property (property_id),
  KEY idx_views_user (user_id),
  KEY idx_views_session (session_id),
  KEY idx_views_created_at (created_at),
  KEY idx_views_property_date (property_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ---------------------------------------------------------------------------
-- Notifications, audit and CMS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS notifications (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  user_id    BIGINT UNSIGNED NOT NULL,
  type       VARCHAR(20)  NOT NULL,
  title      VARCHAR(255) NOT NULL,
  message    TEXT NOT NULL,
  link       VARCHAR(512) NULL,
  read_at    DATETIME(3) NULL,
  metadata   JSON NULL,
  PRIMARY KEY (id),
  KEY idx_notifications_user (user_id),
  KEY idx_notifications_deleted_at (deleted_at),
  CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS notification_preferences (
  id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at        DATETIME(3) NULL,
  updated_at        DATETIME(3) NULL,
  deleted_at        DATETIME(3) NULL,
  user_id           BIGINT UNSIGNED NOT NULL,
  email_enabled     BOOLEAN NOT NULL DEFAULT TRUE,
  smsenabled        BOOLEAN NOT NULL DEFAULT TRUE,
  whats_app_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  in_app_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
  PRIMARY KEY (id),
  UNIQUE KEY uq_notification_preferences_user (user_id),
  KEY idx_notification_preferences_deleted_at (deleted_at),
  CONSTRAINT fk_notification_preferences_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS audit_logs (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  actor_id   BIGINT UNSIGNED NOT NULL,
  action     VARCHAR(100) NOT NULL,
  entity     VARCHAR(100) NOT NULL,
  entity_id  BIGINT UNSIGNED NULL,
  details    JSON NULL,
  ip_address VARCHAR(45) NULL,
  PRIMARY KEY (id),
  KEY idx_audit_logs_actor (actor_id),
  KEY idx_audit_logs_entity (entity_id),
  KEY idx_audit_logs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS featured_properties (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at  DATETIME(3) NULL,
  updated_at  DATETIME(3) NULL,
  deleted_at  DATETIME(3) NULL,
  property_id BIGINT UNSIGNED NOT NULL,
  position    BIGINT NOT NULL DEFAULT 0,
  start_date  DATETIME(3) NULL,
  end_date    DATETIME(3) NULL,
  is_active   BOOLEAN NOT NULL DEFAULT TRUE,
  PRIMARY KEY (id),
  KEY idx_featured_properties_property (property_id),
  KEY idx_featured_properties_deleted_at (deleted_at),
  CONSTRAINT fk_featured_properties_property FOREIGN KEY (property_id) REFERENCES properties (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS home_banners (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  title      VARCHAR(300) NOT NULL,
  subtitle   VARCHAR(500) NULL,
  image_url  VARCHAR(512) NOT NULL,
  link_url   VARCHAR(512) NULL,
  ctatext    VARCHAR(100) NULL,
  ctalink    VARCHAR(512) NULL,
  position   BIGINT NOT NULL DEFAULT 0,
  is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  start_date DATETIME(3) NULL,
  end_date   DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_home_banners_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS static_pages (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at       DATETIME(3) NULL,
  updated_at       DATETIME(3) NULL,
  deleted_at       DATETIME(3) NULL,
  title            VARCHAR(300) NOT NULL,
  slug             VARCHAR(300) NOT NULL,
  content          LONGTEXT NOT NULL,
  meta_title       VARCHAR(300) NULL,
  meta_description VARCHAR(500) NULL,
  is_published     BOOLEAN NOT NULL DEFAULT TRUE,
  PRIMARY KEY (id),
  UNIQUE KEY uq_static_pages_slug (slug),
  KEY idx_static_pages_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
