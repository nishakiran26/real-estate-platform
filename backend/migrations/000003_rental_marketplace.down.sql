-- Reverses 000003_rental_marketplace.up.sql.
--
-- Run this through a single session (the mysql client, or a migration tool that
-- holds one connection). FOREIGN_KEY_CHECKS is session-scoped, so a pooled
-- client that spreads these statements across connections would leave the
-- disable on one connection and run the DDL on others with checks still on.
--
-- Rental listings and their leads are deleted before the purpose column goes,
-- because without that column they would be indistinguishable from sale
-- listings and would surface in the Buy search. Sale data is untouched.

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- Remove rental-only rows first, while purpose still identifies them.
DELETE FROM inquiries WHERE property_id IN (SELECT id FROM properties WHERE purpose = 'RENT');
DELETE FROM properties WHERE purpose = 'RENT';

DROP TABLE IF EXISTS rental_alerts;
DROP TABLE IF EXISTS property_reports;
DROP TABLE IF EXISTS property_visits;
DROP TABLE IF EXISTS rental_details;

DROP INDEX idx_properties_purpose       ON properties;
DROP INDEX idx_properties_posted_by     ON properties;
DROP INDEX idx_properties_owner_user    ON properties;
DROP INDEX idx_properties_availability  ON properties;
DROP INDEX idx_properties_purpose_city  ON properties;
DROP INDEX idx_properties_purpose_avail ON properties;
DROP INDEX idx_properties_expiry        ON properties;

DROP INDEX idx_clients_profile_type ON clients;
DROP INDEX idx_clients_verification ON clients;

DROP INDEX idx_inquiries_assigned  ON inquiries;
DROP INDEX idx_inquiries_lead_type ON inquiries;

ALTER TABLE properties
  DROP COLUMN purpose,
  DROP COLUMN posted_by_type,
  DROP COLUMN owner_user_id,
  DROP COLUMN availability_status,
  DROP COLUMN listing_expiry_date;

ALTER TABLE clients
  DROP COLUMN profile_type,
  DROP COLUMN verification_level,
  DROP COLUMN verified_at,
  DROP COLUMN verified_by;

ALTER TABLE inquiries
  DROP COLUMN lead_type,
  DROP COLUMN assigned_to_user_id,
  DROP COLUMN assigned_at,
  DROP COLUMN assigned_by;

DROP INDEX idx_profiles_looking_to_buy  ON customer_profiles;
DROP INDEX idx_profiles_looking_to_rent ON customer_profiles;

ALTER TABLE customer_profiles
  DROP COLUMN looking_to_buy,
  DROP COLUMN looking_to_rent,
  DROP COLUMN monthly_rent_min,
  DROP COLUMN monthly_rent_max,
  DROP COLUMN preferred_furnishing,
  DROP COLUMN preferred_lease_months,
  DROP COLUMN move_in_date,
  DROP COLUMN family_type;

SET FOREIGN_KEY_CHECKS = 1;
