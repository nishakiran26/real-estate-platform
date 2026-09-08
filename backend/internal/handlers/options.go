package handlers

import "github.com/rera/property-platform/backend/internal/models"

// option is a value/label pair the frontend renders directly in a select.
type option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// rangeOption is a preset budget band on the search form.
type rangeOption struct {
	Label string  `json:"label"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

// The lists below describe enumerations that live in the type system rather
// than the database, so they are served from code. Everything that is business
// data (cities, localities, amenities, banners) comes from MySQL instead.

func propertyTypeOptions() []option {
	return []option{
		{string(models.PropertyTypeApartment), "Apartment"},
		{string(models.PropertyTypeFlat), "Flat"},
		{string(models.PropertyTypeVilla), "Villa"},
		{string(models.PropertyTypeRowHouse), "Row House"},
		{string(models.PropertyTypePlot), "Plot"},
		{string(models.PropertyTypePenthouse), "Penthouse"},
		{string(models.PropertyTypeStudioApartment), "Studio Apartment"},
		{string(models.PropertyTypeCommercialOffice), "Commercial Office"},
		{string(models.PropertyTypeCommercialShop), "Commercial Shop"},
		{string(models.PropertyTypeOther), "Other"},
	}
}

func constructionStatusOptions() []option {
	return []option{
		{string(models.ConstructionUpcoming), "Upcoming"},
		{string(models.ConstructionUnderConstruction), "Under Construction"},
		{string(models.ConstructionReadyToMove), "Ready to Move"},
		{string(models.ConstructionSoldOut), "Sold Out"},
	}
}

func reraStatusOptions() []option {
	return []option{
		{string(models.RERAStatusPendingVerification), "Pending Verification"},
		{string(models.RERAStatusVerified), "Verified"},
		{string(models.RERAStatusRejected), "Rejected"},
		{string(models.RERAStatusNotAvailable), "Not Available"},
	}
}

func propertyStatusFlow() []option {
	return []option{
		{string(models.PropertyStatusDraft), "Draft"},
		{string(models.PropertyStatusSubmitted), "Submitted"},
		{string(models.PropertyStatusUnderReview), "Under Review"},
		{string(models.PropertyStatusApproved), "Approved"},
		{string(models.PropertyStatusPublished), "Published"},
		{string(models.PropertyStatusRejected), "Rejected"},
		{string(models.PropertyStatusSoldOut), "Sold Out"},
	}
}

func inquiryStatuses() []option {
	return []option{
		{"NEW", "New"},
		{"CONTACTED", "Contacted"},
		{"SITE_VISIT_SCHEDULED", "Site Visit Scheduled"},
		{"NEGOTIATING", "Negotiating"},
		{"CLOSED_WON", "Closed - Won"},
		{"CLOSED_LOST", "Closed - Lost"},
	}
}

func sortOptions() []option {
	return []option{
		{"relevance", "Relevance"},
		{"price_asc", "Price: Low to High"},
		{"price_desc", "Price: High to Low"},
		{"newest", "Newest First"},
		{"popular", "Most Popular"},
		{"views", "Most Viewed"},
	}
}

// budgetRanges are the quick-pick bands used on the hero search, expressed in
// rupees (1 lakh = 100000, 1 crore = 10000000).
func budgetRanges() []rangeOption {
	return []rangeOption{
		{"Under 50 Lakh", 0, 5000000},
		{"50 Lakh - 75 Lakh", 5000000, 7500000},
		{"75 Lakh - 1 Crore", 7500000, 10000000},
		{"1 Crore - 1.5 Crore", 10000000, 15000000},
		{"1.5 Crore - 2.5 Crore", 15000000, 25000000},
		{"Above 2.5 Crore", 25000000, 0},
	}
}

// --- Rental vocabulary ------------------------------------------------------

func furnishingOptions() []option {
	return []option{
		{string(models.FurnishingUnfurnished), "Unfurnished"},
		{string(models.FurnishingSemiFurnished), "Semi Furnished"},
		{string(models.FurnishingFullyFurnished), "Fully Furnished"},
	}
}

func tenantPreferenceOptions() []option {
	return []option{
		{string(models.TenantAnyone), "Anyone"},
		{string(models.TenantFamily), "Family"},
		{string(models.TenantBachelor), "Bachelors"},
		{string(models.TenantCompanyLease), "Company Lease"},
	}
}

// availabilityOptions are the shorthand values the rent page sends as
// available_from; the service resolves them to a date.
func availabilityOptions() []option {
	return []option{
		{"now", "Available now"},
		{"7d", "Within 7 days"},
		{"30d", "Within 30 days"},
	}
}

// rentRanges are the quick-pick bands on the rent search, in rupees per month.
func rentRanges() []rangeOption {
	return []rangeOption{
		{"Under ₹15,000", 0, 15000},
		{"₹15,000 - ₹25,000", 15000, 25000},
		{"₹25,000 - ₹40,000", 25000, 40000},
		{"₹40,000 - ₹60,000", 40000, 60000},
		{"₹60,000 - ₹1,00,000", 60000, 100000},
		{"Above ₹1,00,000", 100000, 0},
	}
}

// rentalPhotoTypes are the gallery buckets an owner uploads a flat into.
func rentalPhotoTypes() []option {
	return []option{
		{string(models.MediaLivingRoom), "Living Room"},
		{string(models.MediaBedroom), "Bedroom"},
		{string(models.MediaKitchen), "Kitchen"},
		{string(models.MediaBathroom), "Bathroom"},
		{string(models.MediaBalcony), "Balcony"},
		{string(models.MediaBuildingExterior), "Building Exterior"},
		{string(models.MediaParking), "Parking"},
		{string(models.MediaAmenities), "Amenities"},
		{string(models.MediaSocietyImage), "Society"},
		{string(models.MediaFloorPlan), "Floor Plan"},
	}
}

// includedItemOptions is the configurable catalogue of what a let flat may come
// with. It is served rather than hard-coded in the frontend so it can grow
// without a deploy.
func includedItemOptions() []string {
	return []string{
		"Gas Pipeline", "Air Conditioner", "Refrigerator", "Washing Machine",
		"Television", "Geyser", "Microwave", "Dining Table", "Bed", "Wardrobe",
		"Sofa", "Water Purifier", "Modular Kitchen", "Curtains", "Chimney",
	}
}
