package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/pkg/auth"
	"github.com/rera/property-platform/backend/pkg/utils"
	"gorm.io/gorm"
)

// demoBuilder is a sample builder account plus the projects it owns.
type demoBuilder struct {
	email       string
	mobile      string
	ownerName   string
	companyName string
	companyDesc string
	reraRegNo   string
	// profileType is BUILDER when empty. An individual owner letting a flat is
	// the same kind of account with a different profile type, not a new role.
	profileType models.ProfileType
	projects    []demoProject
}

// demoProject describes one seeded listing. Prices are in rupees.
type demoProject struct {
	name           string
	propertyType   models.PropertyType
	construction   models.ConstructionStatus
	citySlug       string
	localitySlug   string
	address        string
	pincode        string
	latitude       float64
	longitude      float64
	description    string
	possessionIn   int // months from today; 0 means ready to move
	reraNumber     string
	reraAuthority  string
	reraVerified   bool
	featured       bool
	amenitySlugs   []string
	configurations []demoConfiguration
	// images are remote URLs for the demo gallery: the first is the cover.
	// Real listings upload files through the media endpoints instead.
	images []demoImage
	// rental, when set, makes this a listing to let rather than to buy: no RERA
	// record, an availability status and an expiry date.
	rental *demoRental
}

// demoRental carries the terms of a seeded letting. Amounts are in rupees.
type demoRental struct {
	rent            float64
	deposit         float64
	maintenance     float64
	maintenanceIncl bool
	availableInDays int
	minLeaseMonths  int
	noticeDays      int
	furnishing      models.FurnishingType
	tenant          models.TenantPreference
	petAllowed      bool
	includedItems   []string
}

type demoImage struct {
	url       string
	mediaType models.MediaType
}

type demoConfiguration struct {
	name       string
	carpet     float64
	builtUp    float64
	superBuilt float64
	bedrooms   int
	bathrooms  int
	balconies  int
	price      float64
	units      int
}

// seedDemoPortfolio creates sample builders and published projects so a fresh
// installation has browsable content. It is skipped when SEED_DEMO_DATA is off,
// which production deployments are required to do.
func seedDemoPortfolio(db *gorm.DB, cfg *config.Config) error {
	if !cfg.Seed.DemoData {
		return nil
	}
	for _, builder := range demoBuilders() {
		client, err := ensureDemoBuilder(db, builder)
		if err != nil {
			return err
		}
		for _, project := range builder.projects {
			if err := ensureDemoProject(db, client, project); err != nil {
				return err
			}
		}
		if err := refreshPropertyCount(db, client.ID); err != nil {
			return err
		}
	}
	return seedDemoCustomer(db)
}

// ensureDemoBuilder creates the builder login and company profile once.
func ensureDemoBuilder(db *gorm.DB, b demoBuilder) (*models.Client, error) {
	var user models.User
	err := db.Where("email = ?", b.email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		hash, hashErr := auth.HashPassword("Builder@12345")
		if hashErr != nil {
			return nil, fmt.Errorf("hash demo password: %w", hashErr)
		}
		user = models.User{
			Email: b.email, Mobile: b.mobile, PasswordHash: hash, FullName: b.ownerName,
			Status: models.StatusActive, EmailVerified: true, MobileVerified: true,
			ConsentEmail: true,
		}
		if err := db.Create(&user).Error; err != nil {
			return nil, fmt.Errorf("seed builder user %s: %w", b.email, err)
		}
		if err := attachRole(db, user.ID, "CLIENT"); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	profileType := b.profileType
	if profileType == "" {
		profileType = models.ProfileBuilder
	}
	client := models.Client{
		UserID: user.ID, CompanyName: b.companyName, CompanyDesc: b.companyDesc,
		ContactEmail: b.email, ContactPhone: b.mobile, City: "Pune", State: "Maharashtra",
		RERARegNo: b.reraRegNo, ProfileType: profileType,
	}
	if err := db.Where("user_id = ?", user.ID).Attrs(client).FirstOrCreate(&client).Error; err != nil {
		return nil, fmt.Errorf("seed builder profile %s: %w", b.companyName, err)
	}
	return &client, nil
}

// ensureDemoProject writes one published listing with its configurations,
// amenities and RERA record in a single transaction.
func ensureDemoProject(db *gorm.DB, client *models.Client, p demoProject) error {
	var city models.City
	if err := db.Where("slug = ?", p.citySlug).First(&city).Error; err != nil {
		return fmt.Errorf("demo project %s: city %s not seeded: %w", p.name, p.citySlug, err)
	}
	var locality models.Locality
	if err := db.Where("city_id = ? AND slug = ?", city.ID, p.localitySlug).First(&locality).Error; err != nil {
		return fmt.Errorf("demo project %s: locality %s not seeded: %w", p.name, p.localitySlug, err)
	}

	slug := utils.Slugify(fmt.Sprintf("%s %s %s", p.name, locality.Name, city.Name))
	var existing int64
	if err := db.Model(&models.Property{}).Where("slug = ?", slug).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	now := time.Now()
	var possession *time.Time
	if p.possessionIn > 0 {
		d := now.AddDate(0, p.possessionIn, 0)
		possession = &d
	}

	property := models.Property{
		ClientID: client.ID, UserID: client.UserID,
		Name: p.name, Slug: slug, ProjectName: p.name, BuilderName: client.CompanyName,
		PropertyType: p.propertyType, PropertyStatus: models.PropertyStatusPublished,
		ConstructionStatus: p.construction,
		CityID:             city.ID, LocalityID: locality.ID,
		Address: p.address, Pincode: p.pincode, Latitude: p.latitude, Longitude: p.longitude,
		Description: p.description, PossessionDate: possession,
		IsFeatured: p.featured, IsRERAVerified: p.reraVerified,
		SubmittedAt: &now, ApprovedAt: &now,
		MetaTitle: fmt.Sprintf("%s, %s %s | RERA Registered Project", p.name, locality.Name, city.Name),
		MetaDescription: fmt.Sprintf("%s in %s, %s. RERA number %s. View configurations, price, "+
			"floor plans and amenities.", p.name, locality.Name, city.Name, p.reraNumber),
		Purpose:            models.PurposeBuy,
		PostedByType:       models.PostedByClient,
		AvailabilityStatus: models.AvailabilityAvailable,
	}

	// A letting is the same property row with a different purpose: no RERA
	// registration, an availability status that search honours, and an expiry
	// date, because rental stock goes stale in a way a project does not.
	var rental *models.RentalDetails
	if p.rental != nil {
		rental = demoRentalModel(p.rental)
		property.Purpose = models.PurposeRent
		property.PostedByType = models.PostedByOwner
		expiry := now.AddDate(0, 0, 60)
		property.ListingExpiryDate = &expiry
		property.PossessionDate = nil
		property.MetaTitle = fmt.Sprintf("%s for rent in %s, %s", p.name, locality.Name, city.Name)
		property.MetaDescription = fmt.Sprintf(
			"%s in %s, %s. Monthly rent, deposit, furnishing and availability. "+
				"Contact the owner directly.", p.name, locality.Name, city.Name)
	}

	configs := make([]models.PropertyConfiguration, 0, len(p.configurations))
	for _, cfg := range p.configurations {
		config := models.PropertyConfiguration{
			Name: cfg.name, CarpetArea: cfg.carpet, BuiltUpArea: cfg.builtUp,
			SuperBuiltUpArea: cfg.superBuilt, Bedrooms: cfg.bedrooms, Bathrooms: cfg.bathrooms,
			Balconies: cfg.balconies, Price: cfg.price, MinPrice: cfg.price, MaxPrice: cfg.price,
			Availability: "AVAILABLE", NumberOfUnits: cfg.units,
		}
		if cfg.carpet > 0 {
			config.PricePerSqft = cfg.price / cfg.carpet
		}
		configs = append(configs, config)
	}
	property.Configurations = configs
	property.UpdatePricingSummary()

	reraStatus := models.RERAStatusPendingVerification
	if p.reraVerified {
		reraStatus = models.RERAStatusVerified
	}
	regDate := now.AddDate(0, -8, 0)
	completion := now.AddDate(0, p.possessionIn+3, 0)
	rera := models.RERADetails{
		RegistrationNumber: p.reraNumber, Authority: p.reraAuthority,
		RegistrationDate: &regDate, CompletionDate: &completion,
		VerificationStatus: reraStatus,
	}
	if p.reraVerified {
		rera.VerifiedAt = &now
	}

	var amenities []models.Amenity
	if len(p.amenitySlugs) > 0 {
		if err := db.Where("slug IN ?", p.amenitySlugs).Find(&amenities).Error; err != nil {
			return err
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&property).Error; err != nil {
			return fmt.Errorf("seed property %s: %w", p.name, err)
		}
		// A flat to let carries no RERA registration: RERA governs projects for
		// sale, so inventing a number here would be a lie in the data.
		if rental != nil {
			rental.PropertyID = property.ID
			if err := tx.Create(rental).Error; err != nil {
				return fmt.Errorf("seed rental terms for %s: %w", p.name, err)
			}
		} else {
			rera.PropertyID = property.ID
			if err := tx.Create(&rera).Error; err != nil {
				return fmt.Errorf("seed rera for %s: %w", p.name, err)
			}
		}
		if len(amenities) > 0 {
			if err := tx.Model(&property).Association("Amenities").Replace(amenities); err != nil {
				return fmt.Errorf("seed amenities for %s: %w", p.name, err)
			}
		}
		if err := createDemoMedia(tx, property.ID, p.images); err != nil {
			return err
		}
		history := models.PropertyStatusHistory{
			PropertyID: property.ID, FromStatus: models.PropertyStatusDraft,
			ToStatus: models.PropertyStatusPublished, ActorID: client.UserID,
			Remarks: "Seeded demo project",
		}
		return tx.Create(&history).Error
	})
}

// createDemoMedia attaches the gallery of a seeded project. The first image
// becomes the cover so listing cards have a thumbnail.
func createDemoMedia(tx *gorm.DB, propertyID uint, images []demoImage) error {
	for i, img := range images {
		media := models.PropertyMedia{
			PropertyID:   propertyID,
			MediaType:    img.mediaType,
			FileURL:      img.url,
			DisplayOrder: i + 1,
			IsCoverImage: i == 0,
		}
		if err := tx.Create(&media).Error; err != nil {
			return fmt.Errorf("seed media for property %d: %w", propertyID, err)
		}
	}
	return nil
}

// Demo photography. These are hot-linked stock images used only to make a fresh
// installation look real; production listings serve files uploaded by builders
// through the storage abstraction.
const (
	imgTower1    = "https://images.unsplash.com/photo-1545324418-cc1a3fa10c00?w=1200&q=80&auto=format&fit=crop"
	imgTower2    = "https://images.unsplash.com/photo-1560448204-e02f11c3d0e2?w=1200&q=80&auto=format&fit=crop"
	imgTower3    = "https://images.unsplash.com/photo-1512917774080-9991f1c4c750?w=1200&q=80&auto=format&fit=crop"
	imgTower4    = "https://images.unsplash.com/photo-1580587771525-78b9dba3b914?w=1200&q=80&auto=format&fit=crop"
	imgTower5    = "https://images.unsplash.com/photo-1613490493576-7fde63acd811?w=1200&q=80&auto=format&fit=crop"
	imgVilla     = "https://images.unsplash.com/photo-1600596542815-ffad4c1539a9?w=1200&q=80&auto=format&fit=crop"
	imgInterior1 = "https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?w=1200&q=80&auto=format&fit=crop"
	imgInterior2 = "https://images.unsplash.com/photo-1567767292278-a4f21aa2d36e?w=1200&q=80&auto=format&fit=crop"
	imgPool      = "https://images.unsplash.com/photo-1571902943202-507ec2618e8f?w=1200&q=80&auto=format&fit=crop"
	imgLobby     = "https://images.unsplash.com/photo-1600607687939-ce8a6c25118c?w=1200&q=80&auto=format&fit=crop"
	imgFloorPlan = "https://images.unsplash.com/photo-1503387762-592deb58ef4e?w=1200&q=80&auto=format&fit=crop"
)

// gallery builds a standard demo gallery from a cover image.
func gallery(cover string) []demoImage {
	return []demoImage{
		{cover, models.MediaCoverImage},
		{imgInterior1, models.MediaPropertyImage},
		{imgInterior2, models.MediaPropertyImage},
		{imgPool, models.MediaSocietyImage},
		{imgLobby, models.MediaSocietyImage},
		{imgFloorPlan, models.MediaFloorPlan},
	}
}

// refreshPropertyCount keeps the denormalised counter on the builder profile in
// step with the seeded listings.
func refreshPropertyCount(db *gorm.DB, clientID uint) error {
	var count int64
	if err := db.Model(&models.Property{}).Where("client_id = ?", clientID).Count(&count).Error; err != nil {
		return err
	}
	return db.Model(&models.Client{}).Where("id = ?", clientID).
		Update("total_properties", count).Error
}

// seedDemoCustomer creates a buyer login with saved preferences so the
// recommendation engine has something to work with out of the box.
func seedDemoCustomer(db *gorm.DB) error {
	const email = "buyer@example.com"
	var user models.User
	err := db.Where("email = ?", email).First(&user).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	hash, hashErr := auth.HashPassword("Buyer@12345")
	if hashErr != nil {
		return fmt.Errorf("hash demo password: %w", hashErr)
	}
	user = models.User{
		Email: email, Mobile: "9000000010", PasswordHash: hash, FullName: "Rahul Deshmukh",
		Status: models.StatusActive, EmailVerified: true, MobileVerified: true,
		ConsentEmail: true, ConsentPropertyUpdates: true, ConsentNewProjectAlerts: true,
	}
	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("seed demo customer: %w", err)
	}
	if err := attachRole(db, user.ID, "CUSTOMER"); err != nil {
		return err
	}
	profile := models.CustomerProfile{
		UserID: user.ID, PreferredCity: "Pune", PreferredLocality: "Baner",
		MinBudget: 6000000, MaxBudget: 12000000, PropertyType: string(models.PropertyTypeApartment),
		BHK: 2, Purpose: "BUY",
	}
	return db.Create(&profile).Error
}

// demoBuilders is the sample portfolio: three Pune builders with published
// RERA-registered projects across the localities the platform launches with.
func demoBuilders() []demoBuilder {
	const authority = "MahaRERA"
	return []demoBuilder{
		{
			email: "sales@greenvalleydevelopers.example", mobile: "9000000002",
			ownerName: "Suresh Patil", companyName: "Green Valley Developers",
			companyDesc: "Pune-based developer with 18 years of experience delivering " +
				"residential townships across the western corridor.",
			reraRegNo: "A52100000001",
			projects: []demoProject{
				{
					name:   "Green Valley Residency",
					images: gallery(imgTower1), propertyType: models.PropertyTypeApartment,
					construction: models.ConstructionUnderConstruction,
					citySlug:     "pune", localitySlug: "baner",
					address: "Survey No 142, Baner Road, Baner",
					pincode: "411045", latitude: 18.5601, longitude: 73.7799,
					possessionIn: 18, reraNumber: "P52100047821", reraAuthority: authority,
					reraVerified: true, featured: true,
					description: "Green Valley Residency is a RERA-registered gated development of " +
						"2 and 3 BHK homes on Baner Road, walking distance from the Mumbai-Bengaluru " +
						"highway and ten minutes from Balewadi High Street.",
					amenitySlugs: []string{"swimming-pool", "gymnasium", "clubhouse", "landscaped-garden",
						"children-play-area", "24x7-security", "covered-parking", "power-backup"},
					configurations: []demoConfiguration{
						{name: "2 BHK", carpet: 850, builtUp: 980, superBuilt: 1120, bedrooms: 2, bathrooms: 2, balconies: 1, price: 7500000, units: 48},
						{name: "3 BHK", carpet: 1200, builtUp: 1380, superBuilt: 1560, bedrooms: 3, bathrooms: 3, balconies: 2, price: 10500000, units: 32},
					},
				},
				{
					name:   "Green Valley Heights",
					images: gallery(imgTower2), propertyType: models.PropertyTypeApartment,
					construction: models.ConstructionReadyToMove,
					citySlug:     "pune", localitySlug: "wakad",
					address: "Datta Mandir Road, Wakad",
					pincode: "411057", latitude: 18.5989, longitude: 73.7601,
					possessionIn: 0, reraNumber: "P52100031204", reraAuthority: authority,
					reraVerified: true,
					description: "Ready-to-move 2 and 3 BHK apartments in Wakad with possession " +
						"available immediately and an occupancy certificate in place.",
					amenitySlugs: []string{"gymnasium", "clubhouse", "jogging-track", "cctv-surveillance",
						"lift", "power-backup", "rainwater-harvesting"},
					configurations: []demoConfiguration{
						{name: "2 BHK", carpet: 780, builtUp: 900, superBuilt: 1030, bedrooms: 2, bathrooms: 2, balconies: 1, price: 6900000, units: 24},
						{name: "3 BHK", carpet: 1105, builtUp: 1270, superBuilt: 1440, bedrooms: 3, bathrooms: 3, balconies: 2, price: 9600000, units: 18},
					},
				},
			},
		},
		{
			email: "connect@skylinehabitats.example", mobile: "9000000003",
			ownerName: "Meera Kulkarni", companyName: "Skyline Habitats",
			companyDesc: "Boutique developer focused on IT-corridor housing in Hinjawadi and Kharadi.",
			reraRegNo:   "A52100000002",
			projects: []demoProject{
				{
					name:   "Skyline Elmwood",
					images: gallery(imgTower3), propertyType: models.PropertyTypeFlat,
					construction: models.ConstructionUnderConstruction,
					citySlug:     "pune", localitySlug: "hinjawadi",
					address: "Phase 2, Hinjawadi Rajiv Gandhi Infotech Park",
					pincode: "411057", latitude: 18.5908, longitude: 73.7385,
					possessionIn: 24, reraNumber: "P52100052117", reraAuthority: authority,
					reraVerified: true, featured: true,
					description: "1, 2 and 3 BHK flats built for the Hinjawadi IT corridor, with a " +
						"dedicated shuttle bay and co-working lounge inside the society.",
					amenitySlugs: []string{"swimming-pool", "gymnasium", "indoor-games", "24x7-security",
						"covered-parking", "fire-fighting-system", "multipurpose-hall"},
					configurations: []demoConfiguration{
						{name: "1 BHK", carpet: 480, builtUp: 560, superBuilt: 640, bedrooms: 1, bathrooms: 1, balconies: 1, price: 4200000, units: 60},
						{name: "2 BHK", carpet: 720, builtUp: 830, superBuilt: 950, bedrooms: 2, bathrooms: 2, balconies: 1, price: 6300000, units: 72},
						{name: "3 BHK", carpet: 1050, builtUp: 1210, superBuilt: 1380, bedrooms: 3, bathrooms: 3, balconies: 2, price: 9200000, units: 40},
					},
				},
				{
					name:   "Skyline Riverfront",
					images: gallery(imgTower4), propertyType: models.PropertyTypeApartment,
					construction: models.ConstructionUpcoming,
					citySlug:     "pune", localitySlug: "kharadi",
					address: "Riverside Road, Kharadi",
					pincode: "411014", latitude: 18.5528, longitude: 73.9412,
					possessionIn: 36, reraNumber: "P52100058930", reraAuthority: authority,
					reraVerified: false,
					description: "An upcoming riverside development in Kharadi with 3 and 4 BHK " +
						"residences overlooking the Mula-Mutha river, minutes from EON IT Park.",
					amenitySlugs: []string{"swimming-pool", "clubhouse", "landscaped-garden",
						"jogging-track", "sewage-treatment-plant", "lift"},
					configurations: []demoConfiguration{
						{name: "3 BHK", carpet: 1180, builtUp: 1360, superBuilt: 1540, bedrooms: 3, bathrooms: 3, balconies: 2, price: 14200000, units: 36},
						{name: "4 BHK", carpet: 1620, builtUp: 1860, superBuilt: 2100, bedrooms: 4, bathrooms: 4, balconies: 3, price: 19800000, units: 12},
					},
				},
			},
		},
		{
			email: "info@sunriseabodes.example", mobile: "9000000004",
			ownerName: "Anil Joshi", companyName: "Sunrise Abodes",
			companyDesc: "Developer of premium low-density projects in east Pune.",
			reraRegNo:   "A52100000003",
			projects: []demoProject{
				{
					name:   "Sunrise Aster",
					images: gallery(imgTower5), propertyType: models.PropertyTypeApartment,
					construction: models.ConstructionReadyToMove,
					citySlug:     "pune", localitySlug: "viman-nagar",
					address: "Nagar Road, Viman Nagar",
					pincode: "411014", latitude: 18.5671, longitude: 73.9155,
					possessionIn: 0, reraNumber: "P52100029744", reraAuthority: authority,
					reraVerified: true,
					description: "Ready-to-move 2 and 3 BHK homes in Viman Nagar, close to Phoenix " +
						"Marketcity and Pune airport.",
					amenitySlugs: []string{"gymnasium", "clubhouse", "children-play-area",
						"24x7-security", "cctv-surveillance", "covered-parking", "lift", "power-backup"},
					configurations: []demoConfiguration{
						{name: "2 BHK", carpet: 895, builtUp: 1030, superBuilt: 1180, bedrooms: 2, bathrooms: 2, balconies: 2, price: 11200000, units: 28},
						{name: "3 BHK", carpet: 1290, builtUp: 1480, superBuilt: 1680, bedrooms: 3, bathrooms: 3, balconies: 2, price: 15800000, units: 20},
					},
				},
				{
					name:   "Sunrise Meadows",
					images: gallery(imgVilla), propertyType: models.PropertyTypeVilla,
					construction: models.ConstructionUnderConstruction,
					citySlug:     "pune", localitySlug: "hadapsar",
					address: "Magarpatta Annexe, Hadapsar",
					pincode: "411028", latitude: 18.5102, longitude: 73.9284,
					possessionIn: 20, reraNumber: "P52100061455", reraAuthority: authority,
					reraVerified: true,
					description: "A gated cluster of 4 BHK villas in Hadapsar with private gardens " +
						"and covered parking for two cars.",
					amenitySlugs: []string{"landscaped-garden", "24x7-security", "covered-parking",
						"rainwater-harvesting", "jogging-track"},
					configurations: []demoConfiguration{
						{name: "4 BHK Villa", carpet: 2100, builtUp: 2420, superBuilt: 2750, bedrooms: 4, bathrooms: 5, balconies: 3, price: 27500000, units: 14},
					},
				},
			},
		},
		// The rental marketplace needs an example too, or a fresh installation
		// opens the Rent page on an empty catalogue.
		demoOwner(),
	}
}

// demoRentalModel turns seeded terms into the stored record. Defaults match the
// Indian norm the platform assumes elsewhere: an 11-month lease and a month's
// notice.
func demoRentalModel(r *demoRental) *models.RentalDetails {
	availableFrom := time.Now().AddDate(0, 0, r.availableInDays)
	minLease := r.minLeaseMonths
	if minLease == 0 {
		minLease = 11
	}
	notice := r.noticeDays
	if notice == 0 {
		notice = 30
	}
	items := "[]"
	if len(r.includedItems) > 0 {
		if encoded, err := json.Marshal(r.includedItems); err == nil {
			items = string(encoded)
		}
	}
	return &models.RentalDetails{
		MonthlyRent: r.rent, SecurityDeposit: r.deposit,
		MaintenanceAmount: r.maintenance, MaintenanceIncluded: r.maintenanceIncl,
		AvailableFrom:  &availableFrom,
		MinLeaseMonths: minLease, NoticePeriodDays: notice,
		FurnishingType: r.furnishing, TenantPreference: r.tenant,
		PetAllowed: r.petAllowed, IncludedItems: items,
	}
}

// demoOwner is an individual letting out two flats: the account type the rental
// marketplace exists for, and the one a fresh installation needs an example of.
func demoOwner() demoBuilder {
	return demoBuilder{
		email:       "owner@reraproperties.local",
		mobile:      "9000000021",
		ownerName:   "Sunita Deshpande",
		companyName: "Sunita Deshpande",
		companyDesc: "Letting out two flats in Pune directly, without an agent.",
		profileType: models.ProfileOwner,
		projects: []demoProject{
			{
				name:         "2 BHK in Sai Radhe Residency",
				propertyType: models.PropertyTypeFlat,
				construction: models.ConstructionReadyToMove,
				citySlug:     "pune", localitySlug: "baner",
				address: "Sai Radhe Residency, Baner Road", pincode: "411045",
				latitude: 18.5590, longitude: 73.7868,
				description: "A bright east-facing 2 BHK on the fourth floor with covered parking, " +
					"a five-minute walk from Baner Road. The society has a lift, a generator " +
					"and round-the-clock security.",
				amenitySlugs: []string{"lift", "power-backup", "security", "parking"},
				configurations: []demoConfiguration{
					{name: "2 BHK", carpet: 820, builtUp: 960, superBuilt: 1050,
						bedrooms: 2, bathrooms: 2, balconies: 1, units: 1},
				},
				images: gallery("https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?w=1200&q=80"),
				rental: &demoRental{
					rent: 26000, deposit: 78000, maintenance: 1500,
					availableInDays: 7, furnishing: models.FurnishingSemiFurnished,
					tenant:        models.TenantFamily,
					includedItems: []string{"Wardrobes", "Modular kitchen", "Geysers", "Fans and lights"},
				},
			},
			{
				name:         "1 BHK near Hinjawadi Phase 1",
				propertyType: models.PropertyTypeFlat,
				construction: models.ConstructionReadyToMove,
				citySlug:     "pune", localitySlug: "hinjawadi",
				address: "Lakeview Society, Hinjawadi Phase 1", pincode: "411057",
				latitude: 18.5913, longitude: 73.7389,
				description: "A compact fully furnished 1 BHK ten minutes from the Phase 1 IT park. " +
					"Suited to someone working in Hinjawadi who would rather not commute.",
				amenitySlugs: []string{"lift", "security", "parking"},
				configurations: []demoConfiguration{
					{name: "1 BHK", carpet: 480, builtUp: 560, superBuilt: 620,
						bedrooms: 1, bathrooms: 1, balconies: 1, units: 1},
				},
				images: gallery("https://images.unsplash.com/photo-1502672260266-1c1ef2d93688?w=1200&q=80"),
				rental: &demoRental{
					rent: 17500, deposit: 35000, maintenanceIncl: true,
					availableInDays: 0, furnishing: models.FurnishingFullyFurnished,
					tenant: models.TenantAnyone, petAllowed: true,
					includedItems: []string{"Bed", "Wardrobe", "Fridge", "Washing machine", "Air conditioner"},
				},
			},
		},
	}
}
