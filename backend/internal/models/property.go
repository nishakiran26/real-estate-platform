package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type City struct {
	BaseModel
	Name          string  `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Slug          string  `gorm:"size:100;uniqueIndex;not null" json:"slug"`
	State         string  `gorm:"size:100;not null" json:"state"`
	StateCode     string  `gorm:"size:10" json:"state_code"`
	PincodePrefix string  `gorm:"size:10" json:"pincode_prefix"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	IsPopular     bool    `gorm:"default:false" json:"is_popular"`
	DisplayOrder  int     `gorm:"default:0" json:"display_order"`

	Localities []Locality `gorm:"foreignKey:CityID" json:"localities,omitempty"`
}

type Locality struct {
	BaseModel
	CityID          uint    `gorm:"index;not null" json:"city_id"`
	Name            string  `gorm:"size:200;not null" json:"name"`
	Slug            string  `gorm:"size:200;not null" json:"slug"`
	Pincode         string  `gorm:"size:10" json:"pincode"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	AvgPricePerSqft float64 `json:"avg_price_per_sqft"`
	IsPopular       bool    `gorm:"default:false" json:"is_popular"`
	DisplayOrder    int     `gorm:"default:0" json:"display_order"`

	City City `gorm:"foreignKey:CityID" json:"city,omitempty"`
}

type Amenity struct {
	BaseModel
	Name         string `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Slug         string `gorm:"size:100;uniqueIndex;not null" json:"slug"`
	Icon         string `gorm:"size:255" json:"icon"`
	Category     string `gorm:"size:50;not null;default:GENERAL" json:"category"`
	DisplayOrder int    `gorm:"default:0" json:"display_order"`
}

type Property struct {
	BaseModel
	// Ownership. ClientID is the listing account that owns this property and is
	// the key every authorisation, lead and analytics query scopes on. Builders,
	// owners and agents all hold one, so a rental owner reuses the whole path.
	ClientID uint `gorm:"index;not null" json:"client_id"`
	UserID   uint `gorm:"index;not null" json:"user_id"`

	// OwnerUserID names the actual owner when someone else posted on their
	// behalf (an agent, or an administrator entering a managed listing). It is
	// nil when the poster is the owner, which is the common case.
	OwnerUserID  *uint        `gorm:"index" json:"owner_user_id,omitempty"`
	PostedByType PostedByType `gorm:"size:20;index;default:CLIENT;not null" json:"posted_by_type"`

	// Purpose splits the single catalogue into sale and rental listings. Every
	// property that existed before the rental marketplace is BUY.
	Purpose PropertyPurpose `gorm:"size:10;index;default:BUY;not null" json:"purpose"`

	// AvailabilityStatus is independent of the approval workflow in
	// PropertyStatus: a listing can be PUBLISHED and RENTED at the same time.
	AvailabilityStatus AvailabilityStatus `gorm:"size:20;index;default:AVAILABLE;not null" json:"availability_status"`

	// ListingExpiryDate applies to rentals, which go stale; sale listings leave
	// it nil and never expire.
	ListingExpiryDate *time.Time `gorm:"index" json:"listing_expiry_date,omitempty"`

	// Basic info
	Name               string             `gorm:"size:300;not null" json:"name"`
	Slug               string             `gorm:"size:300;uniqueIndex;not null" json:"slug"`
	ProjectName        string             `gorm:"size:300" json:"project_name"`
	BuilderName        string             `gorm:"size:300" json:"builder_name"`
	PropertyType       PropertyType       `gorm:"size:50;index;not null" json:"property_type"`
	PropertyStatus     PropertyStatus     `gorm:"size:30;index;default:DRAFT;not null" json:"property_status"`
	ConstructionStatus ConstructionStatus `gorm:"size:30;index" json:"construction_status"`
	CityID             uint               `gorm:"index;not null" json:"city_id"`
	LocalityID         uint               `gorm:"index;not null" json:"locality_id"`
	Address            string             `gorm:"type:text" json:"address"`
	Pincode            string             `gorm:"size:10" json:"pincode"`
	Latitude           float64            `json:"latitude"`
	Longitude          float64            `json:"longitude"`
	Description        string             `gorm:"type:text" json:"description"`

	// Pricing summary (computed from configurations)
	MinPrice float64 `gorm:"index;default:0" json:"min_price"`
	MaxPrice float64 `gorm:"index;default:0" json:"max_price"`
	MinArea  float64 `gorm:"default:0" json:"min_area"`
	MaxArea  float64 `gorm:"default:0" json:"max_area"`

	// Possession
	PossessionDate *time.Time `json:"possession_date"`

	// Flags
	IsFeatured     bool `gorm:"default:false;index" json:"is_featured"`
	IsRERAVerified bool `gorm:"default:false;index" json:"is_rera_verified"`

	// Approval
	RejectionReason string     `json:"rejection_reason"`
	SubmittedAt     *time.Time `json:"submitted_at"`
	ApprovedAt      *time.Time `json:"approved_at"`
	ApprovedBy      *uint      `json:"approved_by"`
	RejectedAt      *time.Time `json:"rejected_at"`
	RejectedBy      *uint      `json:"rejected_by"`

	// SEO
	MetaTitle       string `gorm:"size:300" json:"meta_title"`
	MetaDescription string `gorm:"size:500" json:"meta_description"`

	// Analytics
	ViewCount          int64 `gorm:"default:0" json:"view_count"`
	InquiryCount       int64 `gorm:"default:0" json:"inquiry_count"`
	WhatsappClickCount int64 `gorm:"default:0" json:"whatsapp_click_count"`
	FavoriteCount      int64 `gorm:"default:0" json:"favorite_count"`

	// Relations
	City           City                    `gorm:"foreignKey:CityID" json:"city,omitempty"`
	Locality       Locality                `gorm:"foreignKey:LocalityID" json:"locality,omitempty"`
	Client         Client                  `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	Configurations []PropertyConfiguration `gorm:"foreignKey:PropertyID" json:"configurations,omitempty"`
	Media          []PropertyMedia         `gorm:"foreignKey:PropertyID" json:"media,omitempty"`
	Documents      []PropertyDocument      `gorm:"foreignKey:PropertyID" json:"documents,omitempty"`
	RERADetails    RERADetails             `gorm:"foreignKey:PropertyID" json:"rera_details,omitempty"`
	RentalDetails  *RentalDetails          `gorm:"foreignKey:PropertyID" json:"rental_details,omitempty"`
	Amenities      []Amenity               `gorm:"many2many:property_amenities" json:"amenities,omitempty"`
	StatusHistory  []PropertyStatusHistory `gorm:"foreignKey:PropertyID" json:"status_history,omitempty"`
}

type PropertyConfiguration struct {
	BaseModel
	PropertyID       uint    `gorm:"index;not null" json:"property_id"`
	Name             string  `gorm:"size:100;not null" json:"name"`
	CarpetArea       float64 `json:"carpet_area"`
	BuiltUpArea      float64 `json:"built_up_area"`
	SuperBuiltUpArea float64 `gorm:"column:super_built_up_area" json:"super_built_up_area"`
	Bedrooms         int     `gorm:"default:0" json:"bedrooms"`
	Bathrooms        int     `gorm:"default:0" json:"bathrooms"`
	Balconies        int     `gorm:"default:0" json:"balconies"`
	Price            float64 `gorm:"index;default:0" json:"price"`
	MinPrice         float64 `gorm:"default:0" json:"min_price"`
	MaxPrice         float64 `gorm:"default:0" json:"max_price"`
	PricePerSqft     float64 `json:"price_per_sqft"`
	Availability     string  `gorm:"size:50;default:AVAILABLE" json:"availability"`
	NumberOfUnits    int     `gorm:"default:0" json:"number_of_units"`
}

type PropertyMedia struct {
	BaseModel
	PropertyID   uint      `gorm:"index;not null" json:"property_id"`
	MediaType    MediaType `gorm:"size:50;not null" json:"media_type"`
	FileURL      string    `gorm:"size:512;not null" json:"file_url"`
	StorageKey   string    `gorm:"size:512" json:"-"`
	ThumbnailURL string    `gorm:"size:512" json:"thumbnail_url"`
	DisplayOrder int       `gorm:"default:0" json:"display_order"`
	IsCoverImage bool      `gorm:"default:false" json:"is_cover_image"`
}

type PropertyDocument struct {
	BaseModel
	PropertyID    uint    `gorm:"index;not null" json:"property_id"`
	DocType       DocType `gorm:"size:50;not null" json:"doc_type"`
	Name          string  `gorm:"size:300;not null" json:"name"`
	FileURL       string  `gorm:"size:512;not null" json:"file_url"`
	StorageKey    string  `gorm:"size:512" json:"-"`
	FileSize      int64   `gorm:"default:0" json:"file_size"`
	MimeType      string  `gorm:"size:100" json:"mime_type"`
	DownloadCount int64   `gorm:"default:0" json:"download_count"`
}

type RERADetails struct {
	BaseModel
	PropertyID         uint       `gorm:"uniqueIndex;not null" json:"property_id"`
	RegistrationNumber string     `gorm:"size:100" json:"registration_number"`
	Authority          string     `gorm:"size:200" json:"authority"`
	RegistrationDate   *time.Time `json:"registration_date"`
	CompletionDate     *time.Time `json:"completion_date"`
	VerificationStatus RERAStatus `gorm:"size:30;default:PENDING_VERIFICATION" json:"verification_status"`
	VerifiedBy         *uint      `json:"verified_by"`
	VerifiedAt         *time.Time `json:"verified_at"`
}

type PropertyStatusHistory struct {
	BaseModel
	PropertyID uint           `gorm:"index;not null" json:"property_id"`
	FromStatus PropertyStatus `gorm:"size:30;not null" json:"from_status"`
	ToStatus   PropertyStatus `gorm:"size:30;not null" json:"to_status"`
	ActorID    uint           `gorm:"index;not null" json:"actor_id"`
	Remarks    string         `gorm:"type:text" json:"remarks"`
}

type FeaturedProperty struct {
	BaseModel
	PropertyID uint       `gorm:"index;not null" json:"property_id"`
	Position   int        `gorm:"default:0" json:"position"`
	StartDate  *time.Time `json:"start_date"`
	EndDate    *time.Time `json:"end_date"`
	IsActive   bool       `gorm:"default:true" json:"is_active"`

	Property Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
}

type HomeBanner struct {
	BaseModel
	Title     string     `gorm:"size:300;not null" json:"title"`
	Subtitle  string     `gorm:"size:500" json:"subtitle"`
	ImageURL  string     `gorm:"size:512;not null" json:"image_url"`
	LinkURL   string     `gorm:"size:512" json:"link_url"`
	CTAText   string     `gorm:"size:100" json:"cta_text"`
	CTALink   string     `gorm:"size:512" json:"cta_link"`
	Position  int        `gorm:"default:0" json:"position"`
	IsActive  bool       `gorm:"default:true" json:"is_active"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
}

func (p *Property) IsPublic() bool {
	return p.PropertyStatus == PropertyStatusApproved || p.PropertyStatus == PropertyStatusPublished
}

func (p *Property) UpdatePricingSummary() {
	if len(p.Configurations) == 0 {
		return
	}
	minP, maxP := p.Configurations[0].Price, p.Configurations[0].Price
	minA, maxA := p.Configurations[0].CarpetArea, p.Configurations[0].CarpetArea
	for _, cfg := range p.Configurations[1:] {
		if cfg.Price < minP {
			minP = cfg.Price
		}
		if cfg.Price > maxP {
			maxP = cfg.Price
		}
		if cfg.CarpetArea < minA {
			minA = cfg.CarpetArea
		}
		if cfg.CarpetArea > maxA {
			maxA = cfg.CarpetArea
		}
	}
	p.MinPrice = minP
	p.MaxPrice = maxP
	p.MinArea = minA
	p.MaxArea = maxA
}

// RentalDetails holds the terms that only apply to a listing offered for rent.
// It is a 1:1 side table rather than columns on Property, so a sale listing does
// not carry a dozen permanently null fields. It mirrors the shape of
// RERADetails so it loads through the same preload pattern.
type RentalDetails struct {
	BaseModel
	PropertyID uint `gorm:"uniqueIndex;not null" json:"property_id"`

	// Money, in rupees.
	MonthlyRent         float64 `gorm:"index;default:0" json:"monthly_rent"`
	SecurityDeposit     float64 `gorm:"default:0" json:"security_deposit"`
	MaintenanceAmount   float64 `gorm:"default:0" json:"maintenance_amount"`
	MaintenanceIncluded bool    `gorm:"default:false" json:"maintenance_included"`

	// Availability drives the "available now / within 7 days / within 30 days"
	// filter, so it is indexed.
	AvailableFrom *time.Time `gorm:"index" json:"available_from"`

	// Lease terms. An 11-month agreement is the Indian norm, which is why the
	// minimum defaults there rather than to a year.
	MinLeaseMonths   int `gorm:"default:11" json:"min_lease_months"`
	MaxLeaseMonths   int `gorm:"default:0" json:"max_lease_months"`
	NoticePeriodDays int `gorm:"default:30" json:"notice_period_days"`

	FurnishingType   FurnishingType   `gorm:"size:30;index;default:UNFURNISHED" json:"furnishing_type"`
	TenantPreference TenantPreference `gorm:"size:30;index;default:ANYONE" json:"tenant_preference"`
	PetAllowed       bool             `gorm:"default:false" json:"pet_allowed"`

	// IncludedItems lists appliances and fittings that come with the flat
	// (geyser, air conditioner, wardrobe). Stored as JSON because the catalogue
	// is configurable and has no relational meaning of its own.
	IncludedItems string `gorm:"type:json" json:"included_items"`

	Property Property `gorm:"foreignKey:PropertyID" json:"-"`
}

// PropertyVisit is a request to view a property in person. Renting rarely
// converts without a viewing, so this is first-class rather than a note on an
// inquiry.
type PropertyVisit struct {
	BaseModel
	PropertyID uint  `gorm:"index;not null" json:"property_id"`
	ClientID   uint  `gorm:"index;not null" json:"client_id"`
	UserID     *uint `gorm:"index" json:"user_id"`

	// Contact details are captured directly: a visit can be requested by a
	// visitor who has not signed in, exactly like an inquiry.
	Name      string `gorm:"size:255;not null" json:"name"`
	Mobile    string `gorm:"size:15;not null" json:"mobile"`
	Email     string `gorm:"size:255" json:"email"`
	Message   string `gorm:"type:text" json:"message"`
	SessionID string `gorm:"size:64;index" json:"session_id"`

	// RequestedAt is what the visitor asked for; ScheduledAt is what the owner
	// confirmed. Keeping both preserves the negotiation.
	RequestedAt time.Time  `gorm:"index;not null" json:"requested_at"`
	ScheduledAt *time.Time `gorm:"index" json:"scheduled_at"`
	TimeSlot    string     `gorm:"size:50" json:"time_slot"`

	Status  VisitStatus `gorm:"size:30;index;default:REQUESTED;not null" json:"status"`
	Remarks string      `gorm:"type:text" json:"remarks"`

	// HandledBy is the account member who confirmed or cancelled the visit.
	HandledBy   *uint      `json:"handled_by"`
	HandledAt   *time.Time `json:"handled_at"`
	CompletedAt *time.Time `json:"completed_at"`

	Property Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
}

// PropertyReport is an abuse report raised against a listing. Once the public
// can post properties, a moderation path is a launch requirement rather than a
// refinement.
type PropertyReport struct {
	BaseModel
	PropertyID uint  `gorm:"index;not null" json:"property_id"`
	ReporterID *uint `gorm:"index" json:"reporter_id"`

	Reason    ReportReason `gorm:"size:40;index;not null" json:"reason"`
	Details   string       `gorm:"type:text" json:"details"`
	SessionID string       `gorm:"size:64" json:"session_id"`
	IPAddress string       `gorm:"size:45" json:"ip_address"`

	Status     ReportStatus `gorm:"size:20;index;default:OPEN;not null" json:"status"`
	ReviewedBy *uint        `json:"reviewed_by"`
	ReviewedAt *time.Time   `json:"reviewed_at"`
	ActionNote string       `gorm:"type:text" json:"action_note"`

	Property Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
}

// BeforeSave keeps IncludedItems valid JSON. MySQL rejects an empty string in a
// JSON column, and Go's zero value for the field is exactly that — so a rental
// saved without any listed appliances would fail on insert.
func (r *RentalDetails) BeforeSave(*gorm.DB) error {
	if strings.TrimSpace(r.IncludedItems) == "" {
		r.IncludedItems = "[]"
	}
	return nil
}
