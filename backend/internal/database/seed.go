package database

import (
	"fmt"
	"time"

	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/pkg/auth"
	"github.com/rera/property-platform/backend/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Seed loads the reference data the platform cannot start without (roles, the
// administrator account, cities, localities, amenities, static pages) plus a
// small demo portfolio so a fresh install has something to browse.
//
// It is idempotent: every insert is keyed on a natural unique column, so
// running it on an existing database changes nothing.
func Seed(db *gorm.DB, cfg *config.Config) error {
	if err := seedRoles(db); err != nil {
		return err
	}
	if err := seedAdmin(db, cfg); err != nil {
		return err
	}
	if err := seedAmenities(db); err != nil {
		return err
	}
	if err := seedCities(db); err != nil {
		return err
	}
	if err := seedStaticPages(db); err != nil {
		return err
	}
	if err := seedDemoPortfolio(db, cfg); err != nil {
		return err
	}
	return seedBanners(db)
}

func seedRoles(db *gorm.DB) error {
	roles := []models.Role{
		{Code: "ADMIN", Name: "Administrator", Description: "Full access to the platform"},
		{Code: "CLIENT", Name: "Builder / Developer", Description: "Manages their own projects and leads"},
		{Code: "CUSTOMER", Name: "Property Buyer", Description: "Browses properties, saves favourites and raises inquiries"},
	}
	for i := range roles {
		if err := db.Where("code = ?", roles[i].Code).
			Attrs(roles[i]).FirstOrCreate(&roles[i]).Error; err != nil {
			return fmt.Errorf("seed role %s: %w", roles[i].Code, err)
		}
	}
	return nil
}

// seedAdmin provisions the first administrator from the environment. The
// password is never hard-coded: it comes from ADMIN_PASSWORD.
func seedAdmin(db *gorm.DB, cfg *config.Config) error {
	var count int64
	if err := db.Model(&models.User{}).Where("email = ?", cfg.Seed.AdminEmail).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := auth.HashPassword(cfg.Seed.AdminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	now := time.Now()
	admin := models.User{
		Email:            cfg.Seed.AdminEmail,
		Mobile:           cfg.Seed.AdminMobile,
		PasswordHash:     hash,
		FullName:         "Platform Administrator",
		Status:           models.StatusActive,
		EmailVerified:    true,
		MobileVerified:   true,
		ConsentEmail:     true,
		ConsentUpdatedAt: &now,
	}
	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}
	return attachRole(db, admin.ID, "ADMIN")
}

func attachRole(db *gorm.DB, userID uint, roleCode string) error {
	var role models.Role
	if err := db.Where("code = ?", roleCode).First(&role).Error; err != nil {
		return fmt.Errorf("lookup role %s: %w", roleCode, err)
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&models.UserRole{UserID: userID, RoleID: role.ID}).Error
}

func seedAmenities(db *gorm.DB) error {
	amenities := []models.Amenity{
		{Name: "Swimming Pool", Icon: "pool", Category: "LEISURE", DisplayOrder: 1},
		{Name: "Gymnasium", Icon: "gym", Category: "SPORTS", DisplayOrder: 2},
		{Name: "Clubhouse", Icon: "clubhouse", Category: "LEISURE", DisplayOrder: 3},
		{Name: "Landscaped Garden", Icon: "garden", Category: "ENVIRONMENT", DisplayOrder: 4},
		{Name: "Children Play Area", Icon: "play", Category: "LEISURE", DisplayOrder: 5},
		{Name: "24x7 Security", Icon: "security", Category: "SAFETY", DisplayOrder: 6},
		{Name: "CCTV Surveillance", Icon: "cctv", Category: "SAFETY", DisplayOrder: 7},
		{Name: "Covered Parking", Icon: "parking", Category: "CONVENIENCE", DisplayOrder: 8},
		{Name: "Lift", Icon: "lift", Category: "CONVENIENCE", DisplayOrder: 9},
		{Name: "Power Backup", Icon: "power", Category: "CONVENIENCE", DisplayOrder: 10},
		{Name: "Jogging Track", Icon: "jogging", Category: "SPORTS", DisplayOrder: 11},
		{Name: "Indoor Games", Icon: "indoor-games", Category: "LEISURE", DisplayOrder: 12},
		{Name: "Rainwater Harvesting", Icon: "rainwater", Category: "ENVIRONMENT", DisplayOrder: 13},
		{Name: "Sewage Treatment Plant", Icon: "stp", Category: "ENVIRONMENT", DisplayOrder: 14},
		{Name: "Fire Fighting System", Icon: "fire", Category: "SAFETY", DisplayOrder: 15},
		{Name: "Multipurpose Hall", Icon: "hall", Category: "GENERAL", DisplayOrder: 16},
	}
	for i := range amenities {
		amenities[i].Slug = utils.Slugify(amenities[i].Name)
		if err := db.Where("slug = ?", amenities[i].Slug).
			Attrs(amenities[i]).FirstOrCreate(&amenities[i]).Error; err != nil {
			return fmt.Errorf("seed amenity %s: %w", amenities[i].Name, err)
		}
	}
	return nil
}

// seedCity is one city plus its localities.
type seedCity struct {
	city       models.City
	localities []models.Locality
}

func seedCities(db *gorm.DB) error {
	data := []seedCity{
		{
			city: models.City{
				Name: "Pune", State: "Maharashtra", StateCode: "MH", PincodePrefix: "411",
				Latitude: 18.5204, Longitude: 73.8567, IsPopular: true, DisplayOrder: 1,
			},
			localities: []models.Locality{
				{Name: "Baner", Pincode: "411045", Latitude: 18.5590, Longitude: 73.7868, AvgPricePerSqft: 11500, IsPopular: true, DisplayOrder: 1},
				{Name: "Wakad", Pincode: "411057", Latitude: 18.5975, Longitude: 73.7628, AvgPricePerSqft: 9800, IsPopular: true, DisplayOrder: 2},
				{Name: "Hinjawadi", Pincode: "411057", Latitude: 18.5913, Longitude: 73.7389, AvgPricePerSqft: 8900, IsPopular: true, DisplayOrder: 3},
				{Name: "Kharadi", Pincode: "411014", Latitude: 18.5515, Longitude: 73.9430, AvgPricePerSqft: 12200, IsPopular: true, DisplayOrder: 4},
				{Name: "Viman Nagar", Pincode: "411014", Latitude: 18.5679, Longitude: 73.9143, AvgPricePerSqft: 13000, IsPopular: true, DisplayOrder: 5},
				{Name: "Kothrud", Pincode: "411038", Latitude: 18.5074, Longitude: 73.8077, AvgPricePerSqft: 12800, IsPopular: false, DisplayOrder: 6},
				{Name: "Hadapsar", Pincode: "411028", Latitude: 18.5089, Longitude: 73.9260, AvgPricePerSqft: 9200, IsPopular: false, DisplayOrder: 7},
			},
		},
		{
			city: models.City{
				Name: "Mumbai", State: "Maharashtra", StateCode: "MH", PincodePrefix: "400",
				Latitude: 19.0760, Longitude: 72.8777, IsPopular: true, DisplayOrder: 2,
			},
			localities: []models.Locality{
				{Name: "Andheri West", Pincode: "400053", Latitude: 19.1364, Longitude: 72.8296, AvgPricePerSqft: 28000, IsPopular: true, DisplayOrder: 1},
				{Name: "Powai", Pincode: "400076", Latitude: 19.1176, Longitude: 72.9060, AvgPricePerSqft: 26500, IsPopular: true, DisplayOrder: 2},
				{Name: "Thane West", Pincode: "400601", Latitude: 19.2183, Longitude: 72.9781, AvgPricePerSqft: 16500, IsPopular: false, DisplayOrder: 3},
			},
		},
		{
			city: models.City{
				Name: "Bengaluru", State: "Karnataka", StateCode: "KA", PincodePrefix: "560",
				Latitude: 12.9716, Longitude: 77.5946, IsPopular: true, DisplayOrder: 3,
			},
			localities: []models.Locality{
				{Name: "Whitefield", Pincode: "560066", Latitude: 12.9698, Longitude: 77.7500, AvgPricePerSqft: 8600, IsPopular: true, DisplayOrder: 1},
				{Name: "Sarjapur Road", Pincode: "560035", Latitude: 12.9010, Longitude: 77.6874, AvgPricePerSqft: 9100, IsPopular: true, DisplayOrder: 2},
			},
		},
	}

	for _, entry := range data {
		city := entry.city
		city.Slug = utils.Slugify(city.Name)
		if err := db.Where("slug = ?", city.Slug).Attrs(city).FirstOrCreate(&city).Error; err != nil {
			return fmt.Errorf("seed city %s: %w", city.Name, err)
		}
		for _, locality := range entry.localities {
			locality.CityID = city.ID
			locality.Slug = utils.Slugify(locality.Name)
			if err := db.Where("city_id = ? AND slug = ?", city.ID, locality.Slug).
				Attrs(locality).FirstOrCreate(&locality).Error; err != nil {
				return fmt.Errorf("seed locality %s: %w", locality.Name, err)
			}
		}
	}
	return nil
}

func seedStaticPages(db *gorm.DB) error {
	pages := []models.StaticPage{
		{
			Title: "About Us", Slug: "about-us", IsPublished: true,
			MetaTitle: "About Us | RERA Property Platform",
			Content: "<p>We help home buyers discover RERA-registered residential projects across India, " +
				"starting with Pune. Every project listed on the platform is submitted by a verified builder " +
				"and reviewed by our team before it is published.</p>",
		},
		{
			Title: "Contact Us", Slug: "contact-us", IsPublished: true,
			MetaTitle: "Contact Us | RERA Property Platform",
			Content:   "<p>Write to us at support@example.com for help with a listing or an inquiry.</p>",
		},
		{
			Title: "Privacy Policy", Slug: "privacy-policy", IsPublished: true,
			MetaTitle: "Privacy Policy | RERA Property Platform",
			Content: "<p>We collect only the information needed to connect buyers with builders. " +
				"Marketing communication is sent only where you have given explicit consent, and you can " +
				"withdraw that consent at any time from your account settings.</p>",
		},
		{
			Title: "Terms and Conditions", Slug: "terms-and-conditions", IsPublished: true,
			MetaTitle: "Terms and Conditions | RERA Property Platform",
			Content:   "<p>Use of this platform is subject to these terms.</p>",
		},
		{
			Title: "RERA Disclaimer", Slug: "rera-disclaimer", IsPublished: true,
			MetaTitle: "RERA Disclaimer | RERA Property Platform",
			Content: "<p>Project information, including RERA registration details, is supplied by the " +
				"builder. While we verify registration numbers against the details submitted to us, buyers " +
				"must independently verify every project and its regulatory approvals with the relevant RERA " +
				"authority before making a purchase decision. This platform is not a party to any transaction " +
				"between a buyer and a builder.</p>",
		},
	}
	for i := range pages {
		if err := db.Where("slug = ?", pages[i].Slug).
			Attrs(pages[i]).FirstOrCreate(&pages[i]).Error; err != nil {
			return fmt.Errorf("seed page %s: %w", pages[i].Slug, err)
		}
	}
	return nil
}

func seedBanners(db *gorm.DB) error {
	banners := []models.HomeBanner{
		{
			Title: "Find your RERA-registered home in Pune", Position: 1, IsActive: true,
			Subtitle: "Verified projects from trusted builders across Baner, Wakad, Hinjawadi and Kharadi",
			ImageURL: imgBannerCity,
			CTAText:  "Explore projects", CTALink: "/properties?city=pune",
		},
		{
			Title: "Ready to move flats", Position: 2, IsActive: true,
			Subtitle: "Move in now, no waiting for possession",
			ImageURL: imgBannerReady,
			CTAText:  "See ready homes", CTALink: "/properties?status=READY_TO_MOVE",
		},
	}
	for i := range banners {
		if err := db.Where("title = ?", banners[i].Title).
			Attrs(banners[i]).FirstOrCreate(&banners[i]).Error; err != nil {
			return fmt.Errorf("seed banner: %w", err)
		}
	}
	return nil
}

// Banner artwork for the seeded homepage. Administrators replace these through
// the content manager; they exist so a fresh install is not visually empty.
const (
	imgBannerCity  = "https://images.unsplash.com/photo-1570168007204-dfb528c6958f?w=1600&q=80&auto=format&fit=crop"
	imgBannerReady = "https://images.unsplash.com/photo-1600585154340-be6161a56a0c?w=1600&q=80&auto=format&fit=crop"
)
