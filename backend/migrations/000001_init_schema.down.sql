-- Reverses 000001_init_schema.up.sql. Tables are dropped children-first so the
-- foreign keys never block the teardown.

SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS static_pages;
DROP TABLE IF EXISTS home_banners;
DROP TABLE IF EXISTS featured_properties;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS property_views;
DROP TABLE IF EXISTS user_property_events;
DROP TABLE IF EXISTS favorites;
DROP TABLE IF EXISTS inquiries;
DROP TABLE IF EXISTS property_status_histories;
DROP TABLE IF EXISTS rera_details;
DROP TABLE IF EXISTS property_documents;
DROP TABLE IF EXISTS property_media;
DROP TABLE IF EXISTS property_amenities;
DROP TABLE IF EXISTS property_configurations;
DROP TABLE IF EXISTS properties;
DROP TABLE IF EXISTS amenities;
DROP TABLE IF EXISTS localities;
DROP TABLE IF EXISTS cities;
DROP TABLE IF EXISTS customer_profiles;
DROP TABLE IF EXISTS clients;
DROP TABLE IF EXISTS otp_requests;
DROP TABLE IF EXISTS login_histories;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;

SET FOREIGN_KEY_CHECKS = 1;
