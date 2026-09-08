package dto

// CreatePropertyRequest is the builder-facing payload for a new listing. Nested
// configurations, amenities and RERA details are written in one transaction.
type CreatePropertyRequest struct {
	Name               string  `json:"name" validate:"required,min=3,max=300"`
	ProjectName        string  `json:"project_name" validate:"omitempty,max=300"`
	BuilderName        string  `json:"builder_name" validate:"omitempty,max=300"`
	PropertyType       string  `json:"property_type" validate:"required,oneof=APARTMENT FLAT VILLA ROW_HOUSE PLOT PENTHOUSE STUDIO_APARTMENT COMMERCIAL_OFFICE COMMERCIAL_SHOP OTHER"`
	ConstructionStatus string  `json:"construction_status" validate:"required,oneof=UPCOMING UNDER_CONSTRUCTION READY_TO_MOVE SOLD_OUT"`
	CityID             uint    `json:"city_id" validate:"required"`
	LocalityID         uint    `json:"locality_id" validate:"required"`
	Address            string  `json:"address" validate:"omitempty,max=1000"`
	Pincode            string  `json:"pincode" validate:"omitempty,len=6"`
	Latitude           float64 `json:"latitude" validate:"omitempty,gte=-90,lte=90"`
	Longitude          float64 `json:"longitude" validate:"omitempty,gte=-180,lte=180"`
	Description        string  `json:"description" validate:"omitempty,max=20000"`
	PossessionDate     string  `json:"possession_date" validate:"omitempty,datetime=2006-01-02"`

	Configurations []ConfigurationRequest `json:"configurations" validate:"omitempty,dive"`
	AmenityIDs     []uint                 `json:"amenity_ids"`
	RERA           *RERARequest           `json:"rera"`

	// Purpose decides which of the two blocks below applies. It defaults to BUY,
	// so an existing client that does not send it keeps posting sale listings.
	Purpose string         `json:"purpose" validate:"omitempty,oneof=BUY RENT"`
	Rental  *RentalRequest `json:"rental"`

	MetaTitle       string `json:"meta_title" validate:"omitempty,max=300"`
	MetaDescription string `json:"meta_description" validate:"omitempty,max=500"`
}

// UpdatePropertyRequest uses pointers so an omitted field keeps its value.
type UpdatePropertyRequest struct {
	Name               *string  `json:"name" validate:"omitempty,min=3,max=300"`
	ProjectName        *string  `json:"project_name" validate:"omitempty,max=300"`
	BuilderName        *string  `json:"builder_name" validate:"omitempty,max=300"`
	PropertyType       *string  `json:"property_type" validate:"omitempty,oneof=APARTMENT FLAT VILLA ROW_HOUSE PLOT PENTHOUSE STUDIO_APARTMENT COMMERCIAL_OFFICE COMMERCIAL_SHOP OTHER"`
	ConstructionStatus *string  `json:"construction_status" validate:"omitempty,oneof=UPCOMING UNDER_CONSTRUCTION READY_TO_MOVE SOLD_OUT"`
	CityID             *uint    `json:"city_id"`
	LocalityID         *uint    `json:"locality_id"`
	Address            *string  `json:"address" validate:"omitempty,max=1000"`
	Pincode            *string  `json:"pincode" validate:"omitempty,len=6"`
	Latitude           *float64 `json:"latitude" validate:"omitempty,gte=-90,lte=90"`
	Longitude          *float64 `json:"longitude" validate:"omitempty,gte=-180,lte=180"`
	Description        *string  `json:"description" validate:"omitempty,max=20000"`
	PossessionDate     *string  `json:"possession_date" validate:"omitempty,datetime=2006-01-02"`

	// A nil slice leaves the collection untouched; an empty slice clears it.
	Configurations []ConfigurationRequest `json:"configurations" validate:"omitempty,dive"`
	AmenityIDs     []uint                 `json:"amenity_ids"`
	RERA           *RERARequest           `json:"rera"`
	Rental         *RentalRequest         `json:"rental"`

	MetaTitle       *string `json:"meta_title" validate:"omitempty,max=300"`
	MetaDescription *string `json:"meta_description" validate:"omitempty,max=500"`
}

type ConfigurationRequest struct {
	Name             string  `json:"name" validate:"required,max=100"`
	CarpetArea       float64 `json:"carpet_area" validate:"gte=0"`
	BuiltUpArea      float64 `json:"built_up_area" validate:"gte=0"`
	SuperBuiltUpArea float64 `json:"super_built_up_area" validate:"gte=0"`
	Bedrooms         int     `json:"bedrooms" validate:"gte=0,lte=20"`
	Bathrooms        int     `json:"bathrooms" validate:"gte=0,lte=20"`
	Balconies        int     `json:"balconies" validate:"gte=0,lte=20"`
	// Price is the sale price. A rental configuration describes the unit only,
	// with the rent held on the rental block, so zero is legitimate here and the
	// service enforces a positive price for sale listings instead.
	Price         float64 `json:"price" validate:"gte=0"`
	MinPrice      float64 `json:"min_price" validate:"gte=0"`
	MaxPrice      float64 `json:"max_price" validate:"gte=0"`
	Availability  string  `json:"availability" validate:"omitempty,oneof=AVAILABLE SOLD_OUT COMING_SOON"`
	NumberOfUnits int     `json:"number_of_units" validate:"gte=0"`
}

// RERARequest carries the registration details shown on the detail page.
type RERARequest struct {
	RegistrationNumber string `json:"registration_number" validate:"omitempty,max=100"`
	Authority          string `json:"authority" validate:"omitempty,max=200"`
	RegistrationDate   string `json:"registration_date" validate:"omitempty,datetime=2006-01-02"`
	CompletionDate     string `json:"completion_date" validate:"omitempty,datetime=2006-01-02"`
}

// PropertyResponse is the full detail payload.
type PropertyResponse struct {
	ID                 uint                    `json:"id"`
	Name               string                  `json:"name"`
	Slug               string                  `json:"slug"`
	ProjectName        string                  `json:"project_name"`
	BuilderName        string                  `json:"builder_name"`
	PropertyType       string                  `json:"property_type"`
	PropertyStatus     string                  `json:"property_status"`
	ConstructionStatus string                  `json:"construction_status"`
	City               *NameIDResponse         `json:"city,omitempty"`
	Locality           *NameIDResponse         `json:"locality,omitempty"`
	Builder            *BuilderResponse        `json:"builder,omitempty"`
	Address            string                  `json:"address"`
	Pincode            string                  `json:"pincode"`
	Latitude           float64                 `json:"latitude"`
	Longitude          float64                 `json:"longitude"`
	Description        string                  `json:"description"`
	MinPrice           float64                 `json:"min_price"`
	MaxPrice           float64                 `json:"max_price"`
	MinArea            float64                 `json:"min_area"`
	MaxArea            float64                 `json:"max_area"`
	PossessionDate     *string                 `json:"possession_date"`
	IsFeatured         bool                    `json:"is_featured"`
	IsRERAVerified     bool                    `json:"is_rera_verified"`
	IsFavorite         bool                    `json:"is_favorite"`
	ViewCount          int64                   `json:"view_count"`
	InquiryCount       int64                   `json:"inquiry_count"`
	FavoriteCount      int64                   `json:"favorite_count"`
	CoverImage         string                  `json:"cover_image"`
	Configurations     []ConfigurationResponse `json:"configurations"`
	Media              []MediaResponse         `json:"media"`
	Documents          []DocumentResponse      `json:"documents"`
	RERADetails        *RERAResponse           `json:"rera_details,omitempty"`
	Purpose            string                  `json:"purpose"`
	AvailabilityStatus string                  `json:"availability_status"`
	PostedByType       string                  `json:"posted_by_type"`
	// ListingExpiryDate is set on rentals only. An owner cannot renew a listing
	// they cannot see the expiry of, so it belongs in the detail payload.
	ListingExpiryDate *string           `json:"listing_expiry_date,omitempty"`
	RentalDetails     *RentalResponse   `json:"rental_details,omitempty"`
	Amenities         []AmenityResponse `json:"amenities"`
	RejectionReason   string            `json:"rejection_reason,omitempty"`
	MetaTitle         string            `json:"meta_title,omitempty"`
	MetaDescription   string            `json:"meta_description,omitempty"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
}

// PropertyCard is the compact listing payload.
type PropertyCard struct {
	ID                 uint            `json:"id"`
	Name               string          `json:"name"`
	Slug               string          `json:"slug"`
	BuilderName        string          `json:"builder_name"`
	PropertyType       string          `json:"property_type"`
	PropertyStatus     string          `json:"property_status"`
	ConstructionStatus string          `json:"construction_status"`
	City               string          `json:"city"`
	CitySlug           string          `json:"city_slug"`
	Locality           string          `json:"locality"`
	LocalitySlug       string          `json:"locality_slug"`
	CoverImage         string          `json:"cover_image"`
	MinPrice           float64         `json:"min_price"`
	MaxPrice           float64         `json:"max_price"`
	MinArea            float64         `json:"min_area"`
	MaxArea            float64         `json:"max_area"`
	BHKOptions         []int           `json:"bhk_options"`
	Configurations     []ConfigSummary `json:"configurations,omitempty"`
	PossessionDate     string          `json:"possession_date,omitempty"`
	IsFeatured         bool            `json:"is_featured"`
	IsRERAVerified     bool            `json:"is_rera_verified"`
	IsFavorite         bool            `json:"is_favorite"`
	Purpose            string          `json:"purpose"`
	AvailabilityStatus string          `json:"availability_status,omitempty"`
	MonthlyRent        float64         `json:"monthly_rent,omitempty"`
	SecurityDeposit    float64         `json:"security_deposit,omitempty"`
	FurnishingType     string          `json:"furnishing_type,omitempty"`
	AvailableFrom      string          `json:"available_from,omitempty"`
	RERANumber         string          `json:"rera_number,omitempty"`
	RERAStatus         string          `json:"rera_status,omitempty"`
	ViewCount          int64           `json:"view_count"`
	FavoriteCount      int64           `json:"favorite_count"`
	CreatedAt          string          `json:"created_at"`
}

type ConfigSummary struct {
	Name       string  `json:"name"`
	Bedrooms   int     `json:"bedrooms"`
	CarpetArea float64 `json:"carpet_area"`
	Price      float64 `json:"price"`
}

type ConfigurationResponse struct {
	ID               uint    `json:"id"`
	Name             string  `json:"name"`
	CarpetArea       float64 `json:"carpet_area"`
	BuiltUpArea      float64 `json:"built_up_area"`
	SuperBuiltUpArea float64 `json:"super_built_up_area"`
	Bedrooms         int     `json:"bedrooms"`
	Bathrooms        int     `json:"bathrooms"`
	Balconies        int     `json:"balconies"`
	Price            float64 `json:"price"`
	MinPrice         float64 `json:"min_price"`
	MaxPrice         float64 `json:"max_price"`
	PricePerSqft     float64 `json:"price_per_sqft"`
	Availability     string  `json:"availability"`
	NumberOfUnits    int     `json:"number_of_units"`
}

type MediaResponse struct {
	ID           uint   `json:"id"`
	MediaType    string `json:"media_type"`
	FileURL      string `json:"file_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	DisplayOrder int    `json:"display_order"`
	IsCoverImage bool   `json:"is_cover_image"`
}

type DocumentResponse struct {
	ID            uint   `json:"id"`
	DocType       string `json:"doc_type"`
	Name          string `json:"name"`
	FileURL       string `json:"file_url"`
	FileSize      int64  `json:"file_size"`
	DownloadCount int64  `json:"download_count"`
}

type RERAResponse struct {
	ID                 uint   `json:"id"`
	RegistrationNumber string `json:"registration_number"`
	Authority          string `json:"authority"`
	RegistrationDate   string `json:"registration_date"`
	CompletionDate     string `json:"completion_date"`
	VerificationStatus string `json:"verification_status"`
	VerifiedAt         string `json:"verified_at,omitempty"`
}

// BuilderResponse is the public builder summary shown on a property page.
type BuilderResponse struct {
	ID              uint   `json:"id"`
	CompanyName     string `json:"company_name"`
	LogoURL         string `json:"logo_url"`
	Description     string `json:"description"`
	Website         string `json:"website"`
	RERARegNo       string `json:"rera_reg_no"`
	TotalProperties int    `json:"total_properties"`
	// Who posted it and how far they have been checked. A renter dealing with
	// an individual owner is in a different situation from one dealing with a
	// builder, and the verification level an administrator sets is worth
	// nothing if it never reaches the page.
	ProfileType       string `json:"profile_type,omitempty"`
	VerificationLevel string `json:"verification_level,omitempty"`
}

type NameIDResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type AmenityResponse struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Icon         string `json:"icon"`
	Category     string `json:"category"`
	DisplayOrder int    `json:"display_order"`
}

type CityResponse struct {
	ID           uint    `json:"id"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	State        string  `json:"state"`
	StateCode    string  `json:"state_code"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	IsPopular    bool    `json:"is_popular"`
	DisplayOrder int     `json:"display_order"`
}

type LocalityResponse struct {
	ID              uint    `json:"id"`
	CityID          uint    `json:"city_id"`
	CityName        string  `json:"city_name,omitempty"`
	CitySlug        string  `json:"city_slug,omitempty"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	Pincode         string  `json:"pincode"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	AvgPricePerSqft float64 `json:"avg_price_per_sqft"`
	IsPopular       bool    `json:"is_popular"`
	DisplayOrder    int     `json:"display_order"`
}

// PropertyListQuery is bound from the URL so a search is shareable as a link.
type PropertyListQuery struct {
	// Purpose selects sale or rental listings. It is omitted by the existing Buy
	// pages, which is why the service treats an empty value as BUY; "all"
	// searches both catalogues.
	Purpose string `query:"purpose"`

	// Rental filters. They are ignored unless the search is for rentals.
	MinRent          float64 `query:"min_rent"`
	MaxRent          float64 `query:"max_rent"`
	Furnishing       string  `query:"furnishing"`
	TenantPreference string  `query:"tenant_preference"`
	PetAllowed       string  `query:"pet_allowed"`
	// AvailableFrom accepts a date, or the shorthand "now", "7d" and "30d" that
	// the availability filter on the rent page offers.
	AvailableFrom string `query:"available_from"`

	Page               int     `query:"page"`
	PerPage            int     `query:"per_page"`
	CitySlug           string  `query:"city"`
	LocalitySlug       string  `query:"locality"`
	CityID             uint    `query:"city_id"`
	LocalityID         uint    `query:"locality_id"`
	PropertyType       string  `query:"property_type"`
	ConstructionStatus string  `query:"status"`
	MinPrice           float64 `query:"min_price"`
	MaxPrice           float64 `query:"max_price"`
	MinArea            float64 `query:"min_area"`
	MaxArea            float64 `query:"max_area"`
	BHK                int     `query:"bhk"`
	Amenities          string  `query:"amenities"` // comma-separated amenity IDs
	Builder            string  `query:"builder"`
	PossessionBefore   string  `query:"possession_before"`
	Featured           string  `query:"featured"`
	RERAVerified       string  `query:"rera_verified"`
	Search             string  `query:"q"`
	SortBy             string  `query:"sort"`
	SortOrder          string  `query:"order"`
}

type AdminPropertyListQuery struct {
	Page     int    `query:"page"`
	PerPage  int    `query:"per_page"`
	Status   string `query:"status"`
	ClientID uint   `query:"client_id"`
	CityID   uint   `query:"city_id"`
	Search   string `query:"q"`
	SortBy   string `query:"sort"`
	// Purpose splits the review queue. With two catalogues in one application an
	// administrator reviewing rentals should not have to read past sale listings.
	// Empty means both, so the existing queue is unchanged.
	Purpose      string `query:"purpose"`
	Availability string `query:"availability"`
}

type ApprovePropertyRequest struct {
	Remarks string `json:"remarks" validate:"omitempty,max=1000"`
	Publish bool   `json:"publish"`
}

type RejectPropertyRequest struct {
	Reason string `json:"reason" validate:"required,min=5,max=1000"`
}

// VerifyRERARequest records an administrator decision on a registration number.
type VerifyRERARequest struct {
	Status string `json:"status" validate:"required,oneof=PENDING_VERIFICATION VERIFIED REJECTED NOT_AVAILABLE"`
}

type PropertySummary struct {
	ID       uint    `json:"id"`
	Name     string  `json:"name"`
	Slug     string  `json:"slug"`
	MinPrice float64 `json:"min_price"`
}

// StatusHistoryResponse is one row of the approval audit trail.
type StatusHistoryResponse struct {
	ID         uint   `json:"id"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	ActorID    uint   `json:"actor_id"`
	Remarks    string `json:"remarks"`
	CreatedAt  string `json:"created_at"`
}

// SetCoverRequest and ReorderMediaRequest manage a property gallery.
type SetCoverRequest struct {
	MediaID uint `json:"media_id" validate:"required"`
}

type ReorderMediaRequest struct {
	MediaIDs []uint `json:"media_ids" validate:"required,min=1"`
}

// UploadedFileResponse is returned by the generic image upload endpoint used
// for avatars, company logos and banner artwork.
type UploadedFileResponse struct {
	Key      string `json:"key"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

// RentalResponse is the rental block on a property payload. It is present only
// for listings whose purpose is RENT, so a sale listing is unchanged.
type RentalResponse struct {
	MonthlyRent         float64  `json:"monthly_rent"`
	SecurityDeposit     float64  `json:"security_deposit"`
	MaintenanceAmount   float64  `json:"maintenance_amount"`
	MaintenanceIncluded bool     `json:"maintenance_included"`
	AvailableFrom       string   `json:"available_from,omitempty"`
	MinLeaseMonths      int      `json:"min_lease_months"`
	MaxLeaseMonths      int      `json:"max_lease_months"`
	NoticePeriodDays    int      `json:"notice_period_days"`
	FurnishingType      string   `json:"furnishing_type"`
	TenantPreference    string   `json:"tenant_preference"`
	PetAllowed          bool     `json:"pet_allowed"`
	IncludedItems       []string `json:"included_items,omitempty"`
}

// RentalRequest is the builder- or owner-facing payload for rental terms. It
// accompanies a property whose purpose is RENT.
type RentalRequest struct {
	MonthlyRent         float64  `json:"monthly_rent" validate:"required,gt=0"`
	SecurityDeposit     float64  `json:"security_deposit" validate:"gte=0"`
	MaintenanceAmount   float64  `json:"maintenance_amount" validate:"gte=0"`
	MaintenanceIncluded bool     `json:"maintenance_included"`
	AvailableFrom       string   `json:"available_from" validate:"omitempty,datetime=2006-01-02"`
	MinLeaseMonths      int      `json:"min_lease_months" validate:"gte=0,lte=120"`
	MaxLeaseMonths      int      `json:"max_lease_months" validate:"gte=0,lte=240"`
	NoticePeriodDays    int      `json:"notice_period_days" validate:"gte=0,lte=365"`
	FurnishingType      string   `json:"furnishing_type" validate:"omitempty,oneof=UNFURNISHED SEMI_FURNISHED FULLY_FURNISHED"`
	TenantPreference    string   `json:"tenant_preference" validate:"omitempty,oneof=FAMILY BACHELOR COMPANY_LEASE ANYONE"`
	PetAllowed          bool     `json:"pet_allowed"`
	IncludedItems       []string `json:"included_items" validate:"omitempty,dive,max=60"`
}

// UpdateAvailabilityRequest takes a rental off the market or puts it back. It
// is deliberately separate from the approval workflow: an owner marking a flat
// RENTED must not send the listing back for review.
type UpdateAvailabilityRequest struct {
	AvailabilityStatus string `json:"availability_status" validate:"required,oneof=AVAILABLE RESERVED RENTED INACTIVE"`
	// AvailableFrom re-dates the listing when it comes back on the market,
	// because a flat freed in March should not still advertise January.
	AvailableFrom string `json:"available_from" validate:"omitempty,datetime=2006-01-02"`
}

// ExpiringListing is one row of the owner's "these lapse soon" panel.
type ExpiringListing struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	ExpiresOn   string  `json:"expires_on"`
	DaysLeft    int     `json:"days_left"`
	MonthlyRent float64 `json:"monthly_rent"`
	HasExpired  bool    `json:"has_expired"`
}
