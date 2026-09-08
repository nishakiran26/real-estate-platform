package database

import (
	"fmt"
	"strings"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// AllModels is the authoritative list of tables managed by the ORM. The SQL
// files under migrations/ describe the same schema for teams that prefer to
// apply migrations manually.
func AllModels() []any {
	return []any{
		&models.Role{},
		&models.User{},
		&models.UserRole{},
		&models.RefreshToken{},
		&models.LoginHistory{},
		&models.OTPRequest{},
		&models.Client{},
		&models.CustomerProfile{},
		&models.City{},
		&models.Locality{},
		&models.Amenity{},
		&models.Property{},
		&models.PropertyConfiguration{},
		&models.PropertyMedia{},
		&models.PropertyDocument{},
		&models.RERADetails{},
		&models.RentalDetails{},
		&models.PropertyStatusHistory{},
		&models.PropertyVisit{},
		&models.PropertyReport{},
		&models.RentalAlert{},
		&models.Inquiry{},
		&models.Favorite{},
		&models.UserPropertyEvent{},
		&models.PropertyView{},
		&models.Notification{},
		&models.NotificationPreference{},
		&models.AuditLog{},
		&models.FeaturedProperty{},
		&models.HomeBanner{},
		&models.StaticPage{},
	}
}

// AutoMigrate creates or updates every table, then adds the composite indexes
// that the struct tags cannot express.
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(AllModels()...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return createCompositeIndexes(db)
}

// compositeIndexes back the hot query paths: the public search filters on
// status plus location and price, and the analytics queries aggregate events by
// property and date.
var compositeIndexes = []struct{ name, statement string }{
	{"idx_properties_status_city", "CREATE INDEX idx_properties_status_city ON properties (property_status, city_id, locality_id)"},
	{"idx_properties_status_price", "CREATE INDEX idx_properties_status_price ON properties (property_status, min_price, max_price)"},
	{"idx_properties_status_created", "CREATE INDEX idx_properties_status_created ON properties (property_status, created_at)"},
	{"idx_properties_client_status", "CREATE INDEX idx_properties_client_status ON properties (client_id, property_status)"},
	{"idx_configs_property_bedrooms", "CREATE INDEX idx_configs_property_bedrooms ON property_configurations (property_id, bedrooms)"},
	{"idx_events_property_type_date", "CREATE INDEX idx_events_property_type_date ON user_property_events (property_id, event_type, created_at)"},
	{"idx_views_property_date", "CREATE INDEX idx_views_property_date ON property_views (property_id, created_at)"},
	{"idx_inquiries_client_status", "CREATE INDEX idx_inquiries_client_status ON inquiries (client_id, status, created_at)"},
	{"idx_localities_city_slug", "CREATE UNIQUE INDEX idx_localities_city_slug ON localities (city_id, slug)"},

	// Rental search adds purpose as a second discriminator after status, so the
	// hot search paths need purpose-aware siblings rather than relying on the
	// sale indexes above.
	{"idx_properties_purpose_city", "CREATE INDEX idx_properties_purpose_city ON properties (property_status, purpose, city_id, locality_id)"},
	{"idx_properties_purpose_avail", "CREATE INDEX idx_properties_purpose_avail ON properties (property_status, purpose, availability_status)"},
	{"idx_properties_expiry", "CREATE INDEX idx_properties_expiry ON properties (purpose, listing_expiry_date)"},
	{"idx_rental_rent", "CREATE INDEX idx_rental_rent ON rental_details (monthly_rent)"},
	{"idx_rental_available", "CREATE INDEX idx_rental_available ON rental_details (available_from)"},
	{"idx_rental_furnishing", "CREATE INDEX idx_rental_furnishing ON rental_details (furnishing_type, tenant_preference)"},
	{"idx_visits_property_status", "CREATE INDEX idx_visits_property_status ON property_visits (property_id, status, scheduled_at)"},
	{"idx_visits_client_status", "CREATE INDEX idx_visits_client_status ON property_visits (client_id, status, requested_at)"},
	{"idx_reports_status", "CREATE INDEX idx_reports_status ON property_reports (status, created_at)"},
	{"idx_inquiries_lead_type", "CREATE INDEX idx_inquiries_lead_type ON inquiries (client_id, lead_type, created_at)"},
}

// createCompositeIndexes adds each index if it is missing. Creating an index
// that already exists is not an error worth failing startup over.
func createCompositeIndexes(db *gorm.DB) error {
	for _, idx := range compositeIndexes {
		if db.Migrator().HasIndex(tableOf(idx.statement), idx.name) {
			continue
		}
		if err := db.Exec(idx.statement).Error; err != nil {
			return fmt.Errorf("create index %s: %w", idx.name, err)
		}
	}
	return nil
}

// tableOf extracts the table name from a CREATE INDEX statement so the migrator
// can be asked whether the index already exists.
func tableOf(statement string) string {
	const marker = " ON "
	start := strings.Index(statement, marker)
	if start < 0 {
		return ""
	}
	rest := statement[start+len(marker):]
	end := strings.Index(rest, " ")
	if end < 0 {
		return rest
	}
	return rest[:end]
}
