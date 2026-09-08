-- Removes the reference rows inserted by 000002_seed_reference_data.up.sql.
-- Localities are deleted before cities, and nothing here touches user or
-- property data.

DELETE FROM localities WHERE slug IN (
  'baner', 'wakad', 'hinjawadi', 'kharadi', 'viman-nagar', 'kothrud', 'hadapsar',
  'andheri-west', 'powai', 'thane-west', 'whitefield', 'sarjapur-road'
);

DELETE FROM cities WHERE slug IN ('pune', 'mumbai', 'bengaluru');

DELETE FROM amenities WHERE slug IN (
  'swimming-pool', 'gymnasium', 'clubhouse', 'landscaped-garden', 'children-play-area',
  '24x7-security', 'cctv-surveillance', 'covered-parking', 'lift', 'power-backup',
  'jogging-track', 'indoor-games', 'rainwater-harvesting', 'sewage-treatment-plant',
  'fire-fighting-system', 'multipurpose-hall'
);

DELETE FROM static_pages WHERE slug IN (
  'about-us', 'contact-us', 'privacy-policy', 'terms-and-conditions', 'rera-disclaimer'
);

DELETE FROM roles WHERE code IN ('ADMIN', 'CLIENT', 'CUSTOMER');
