-- Reference data the platform needs before anyone can sign in or search.
--
-- The application seeds the same rows on startup (DB_SEED=true), including the
-- administrator account, whose password comes from ADMIN_PASSWORD and therefore
-- cannot live in a SQL file. Run this script only when seeding is disabled and
-- the reference tables are loaded by hand; create the administrator afterwards
-- with the application (DB_SEED=true, SEED_DEMO_DATA=false) or your own tooling.

INSERT INTO roles (code, name, description, created_at, updated_at) VALUES
  ('ADMIN',    'Administrator',       'Full access to the platform', NOW(3), NOW(3)),
  ('CLIENT',   'Builder / Developer', 'Manages their own projects and leads', NOW(3), NOW(3)),
  ('CUSTOMER', 'Property Buyer',      'Browses properties, saves favourites and raises inquiries', NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description);

INSERT INTO amenities (name, slug, icon, category, display_order, created_at, updated_at) VALUES
  ('Swimming Pool',          'swimming-pool',          'pool',         'LEISURE',     1,  NOW(3), NOW(3)),
  ('Gymnasium',              'gymnasium',              'gym',          'SPORTS',      2,  NOW(3), NOW(3)),
  ('Clubhouse',              'clubhouse',              'clubhouse',    'LEISURE',     3,  NOW(3), NOW(3)),
  ('Landscaped Garden',      'landscaped-garden',      'garden',       'ENVIRONMENT', 4,  NOW(3), NOW(3)),
  ('Children Play Area',     'children-play-area',     'play',         'LEISURE',     5,  NOW(3), NOW(3)),
  ('24x7 Security',          '24x7-security',          'security',     'SAFETY',      6,  NOW(3), NOW(3)),
  ('CCTV Surveillance',      'cctv-surveillance',      'cctv',         'SAFETY',      7,  NOW(3), NOW(3)),
  ('Covered Parking',        'covered-parking',        'parking',      'CONVENIENCE', 8,  NOW(3), NOW(3)),
  ('Lift',                   'lift',                   'lift',         'CONVENIENCE', 9,  NOW(3), NOW(3)),
  ('Power Backup',           'power-backup',           'power',        'CONVENIENCE', 10, NOW(3), NOW(3)),
  ('Jogging Track',          'jogging-track',          'jogging',      'SPORTS',      11, NOW(3), NOW(3)),
  ('Indoor Games',           'indoor-games',           'indoor-games', 'LEISURE',     12, NOW(3), NOW(3)),
  ('Rainwater Harvesting',   'rainwater-harvesting',   'rainwater',    'ENVIRONMENT', 13, NOW(3), NOW(3)),
  ('Sewage Treatment Plant', 'sewage-treatment-plant', 'stp',          'ENVIRONMENT', 14, NOW(3), NOW(3)),
  ('Fire Fighting System',   'fire-fighting-system',   'fire',         'SAFETY',      15, NOW(3), NOW(3)),
  ('Multipurpose Hall',      'multipurpose-hall',      'hall',         'GENERAL',     16, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE icon = VALUES(icon), category = VALUES(category);

INSERT INTO cities (name, slug, state, state_code, pincode_prefix, latitude, longitude, is_popular, display_order, created_at, updated_at) VALUES
  ('Pune',      'pune',      'Maharashtra', 'MH', '411', 18.5204, 73.8567, TRUE, 1, NOW(3), NOW(3)),
  ('Mumbai',    'mumbai',    'Maharashtra', 'MH', '400', 19.0760, 72.8777, TRUE, 2, NOW(3), NOW(3)),
  ('Bengaluru', 'bengaluru', 'Karnataka',   'KA', '560', 12.9716, 77.5946, TRUE, 3, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE state = VALUES(state), is_popular = VALUES(is_popular);

INSERT INTO localities (city_id, name, slug, pincode, latitude, longitude, avg_price_per_sqft, is_popular, display_order, created_at, updated_at)
SELECT c.id, v.name, v.slug, v.pincode, v.latitude, v.longitude, v.avg_price, v.is_popular, v.display_order, NOW(3), NOW(3)
FROM cities c
JOIN (
  SELECT 'pune' AS city_slug, 'Baner'        AS name, 'baner'        AS slug, '411045' AS pincode, 18.5590 AS latitude, 73.7868 AS longitude, 11500 AS avg_price, TRUE  AS is_popular, 1 AS display_order UNION ALL
  SELECT 'pune', 'Wakad',        'wakad',        '411057', 18.5975, 73.7628, 9800,  TRUE,  2 UNION ALL
  SELECT 'pune', 'Hinjawadi',    'hinjawadi',    '411057', 18.5913, 73.7389, 8900,  TRUE,  3 UNION ALL
  SELECT 'pune', 'Kharadi',      'kharadi',      '411014', 18.5515, 73.9430, 12200, TRUE,  4 UNION ALL
  SELECT 'pune', 'Viman Nagar',  'viman-nagar',  '411014', 18.5679, 73.9143, 13000, TRUE,  5 UNION ALL
  SELECT 'pune', 'Kothrud',      'kothrud',      '411038', 18.5074, 73.8077, 12800, FALSE, 6 UNION ALL
  SELECT 'pune', 'Hadapsar',     'hadapsar',     '411028', 18.5089, 73.9260, 9200,  FALSE, 7 UNION ALL
  SELECT 'mumbai', 'Andheri West', 'andheri-west', '400053', 19.1364, 72.8296, 28000, TRUE,  1 UNION ALL
  SELECT 'mumbai', 'Powai',        'powai',        '400076', 19.1176, 72.9060, 26500, TRUE,  2 UNION ALL
  SELECT 'mumbai', 'Thane West',   'thane-west',   '400601', 19.2183, 72.9781, 16500, FALSE, 3 UNION ALL
  SELECT 'bengaluru', 'Whitefield',    'whitefield',    '560066', 12.9698, 77.7500, 8600, TRUE, 1 UNION ALL
  SELECT 'bengaluru', 'Sarjapur Road', 'sarjapur-road', '560035', 12.9010, 77.6874, 9100, TRUE, 2
) v ON v.city_slug = c.slug
ON DUPLICATE KEY UPDATE avg_price_per_sqft = VALUES(avg_price_per_sqft), is_popular = VALUES(is_popular);

INSERT INTO static_pages (title, slug, content, meta_title, is_published, created_at, updated_at) VALUES
  ('About Us', 'about-us',
   '<p>We help home buyers discover RERA-registered residential projects across India, starting with Pune.</p>',
   'About Us | RERA Property Platform', TRUE, NOW(3), NOW(3)),
  ('Contact Us', 'contact-us',
   '<p>Write to us at support@example.com for help with a listing or an inquiry.</p>',
   'Contact Us | RERA Property Platform', TRUE, NOW(3), NOW(3)),
  ('Privacy Policy', 'privacy-policy',
   '<p>We collect only the information needed to connect buyers with builders, and send marketing communication only where you have given explicit consent.</p>',
   'Privacy Policy | RERA Property Platform', TRUE, NOW(3), NOW(3)),
  ('Terms and Conditions', 'terms-and-conditions',
   '<p>Use of this platform is subject to these terms.</p>',
   'Terms and Conditions | RERA Property Platform', TRUE, NOW(3), NOW(3)),
  ('RERA Disclaimer', 'rera-disclaimer',
   '<p>Project information, including RERA registration details, is supplied by the builder. Buyers must independently verify every project and its regulatory approvals with the relevant RERA authority before making a purchase decision.</p>',
   'RERA Disclaimer | RERA Property Platform', TRUE, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE content = VALUES(content), meta_title = VALUES(meta_title);
