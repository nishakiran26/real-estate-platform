package dto

type CreateCityRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=100"`
	State        string  `json:"state" validate:"required,max=100"`
	StateCode    string  `json:"state_code" validate:"omitempty,max=10"`
	Latitude     float64 `json:"latitude" validate:"omitempty,gte=-90,lte=90"`
	Longitude    float64 `json:"longitude" validate:"omitempty,gte=-180,lte=180"`
	IsPopular    bool    `json:"is_popular"`
	DisplayOrder int     `json:"display_order"`
}

type CreateLocalityRequest struct {
	CityID       uint    `json:"city_id" validate:"required"`
	Name         string  `json:"name" validate:"required,min=2,max=200"`
	Pincode      string  `json:"pincode" validate:"omitempty,len=6"`
	Latitude     float64 `json:"latitude" validate:"omitempty,gte=-90,lte=90"`
	Longitude    float64 `json:"longitude" validate:"omitempty,gte=-180,lte=180"`
	IsPopular    bool    `json:"is_popular"`
	DisplayOrder int     `json:"display_order"`
}

type CreateAmenityRequest struct {
	Name         string `json:"name" validate:"required,max=100"`
	Icon         string `json:"icon" validate:"omitempty,max=255"`
	Category     string `json:"category" validate:"required,oneof=GENERAL SPORTS LEISURE SAFETY CONVENIENCE ENVIRONMENT"`
	DisplayOrder int    `json:"display_order"`
}

// UpdateUserStatusRequest suspends or reactivates an account.
type UpdateUserStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=ACTIVE INACTIVE SUSPENDED"`
	Reason string `json:"reason" validate:"omitempty,max=500"`
}

type AssignRoleRequest struct {
	RoleIDs []uint `json:"role_ids" validate:"required,min=1"`
}

type UserListQuery struct {
	Page    int    `query:"page"`
	PerPage int    `query:"per_page"`
	Role    string `query:"role"`
	Status  string `query:"status"`
	Search  string `query:"q"`
}

type CreateFeaturedPropertyRequest struct {
	PropertyID uint   `json:"property_id" validate:"required"`
	Position   int    `json:"position"`
	StartDate  string `json:"start_date" validate:"omitempty,datetime=2006-01-02"`
	EndDate    string `json:"end_date" validate:"omitempty,datetime=2006-01-02"`
}

type CreateBannerRequest struct {
	Title     string `json:"title" validate:"required,min=2,max=300"`
	Subtitle  string `json:"subtitle" validate:"omitempty,max=500"`
	ImageURL  string `json:"image_url" validate:"required,max=512"`
	LinkURL   string `json:"link_url" validate:"omitempty,max=512"`
	CTAText   string `json:"cta_text" validate:"omitempty,max=100"`
	CTALink   string `json:"cta_link" validate:"omitempty,max=512"`
	Position  int    `json:"position"`
	IsActive  bool   `json:"is_active"`
	StartDate string `json:"start_date" validate:"omitempty,datetime=2006-01-02"`
	EndDate   string `json:"end_date" validate:"omitempty,datetime=2006-01-02"`
}

type BannerResponse struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle"`
	ImageURL  string `json:"image_url"`
	LinkURL   string `json:"link_url"`
	CTAText   string `json:"cta_text"`
	CTALink   string `json:"cta_link"`
	Position  int    `json:"position"`
	IsActive  bool   `json:"is_active"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}

type FeaturedPropertyResponse struct {
	ID        uint          `json:"id"`
	Position  int           `json:"position"`
	IsActive  bool          `json:"is_active"`
	StartDate string        `json:"start_date,omitempty"`
	EndDate   string        `json:"end_date,omitempty"`
	Property  *PropertyCard `json:"property,omitempty"`
}

// StaticPageRequest edits an informational page such as the RERA disclaimer.
type StaticPageRequest struct {
	Title           string `json:"title" validate:"required,min=2,max=300"`
	Slug            string `json:"slug" validate:"required,min=2,max=300"`
	Content         string `json:"content" validate:"required"`
	MetaTitle       string `json:"meta_title" validate:"omitempty,max=300"`
	MetaDescription string `json:"meta_description" validate:"omitempty,max=500"`
	IsPublished     bool   `json:"is_published"`
}

type StaticPageResponse struct {
	ID              uint   `json:"id"`
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Content         string `json:"content"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	IsPublished     bool   `json:"is_published"`
	UpdatedAt       string `json:"updated_at"`
}

// DashboardStats is the headline count block on the admin dashboard.
type DashboardStats struct {
	TotalUsers         int64 `json:"total_users"`
	TotalClients       int64 `json:"total_clients"`
	TotalCustomers     int64 `json:"total_customers"`
	TotalProperties    int64 `json:"total_properties"`
	ActiveProperties   int64 `json:"active_properties"`
	PendingProperties  int64 `json:"pending_properties"`
	ApprovedProperties int64 `json:"approved_properties"`
	RejectedProperties int64 `json:"rejected_properties"`
	DraftProperties    int64 `json:"draft_properties"`
	TotalInquiries     int64 `json:"total_inquiries"`
	NewInquiries       int64 `json:"new_inquiries"`
	WhatsAppLeads      int64 `json:"whatsapp_leads"`
	TotalViews         int64 `json:"total_views"`
	UniqueVisitors     int64 `json:"unique_visitors"`
	// The catalogue carries two purposes, so the headline block splits them.
	// A single "properties" figure hides which half of the business is growing.
	SaleProperties   int64 `json:"sale_properties"`
	RentalProperties int64 `json:"rental_properties"`
	RentedProperties int64 `json:"rented_properties"`
	BuyLeads         int64 `json:"buy_leads"`
	RentLeads        int64 `json:"rent_leads"`
	OpenReports      int64 `json:"open_reports"`
}

// AdminDashboardResponse bundles the stats, charts and activity lists.
type AdminDashboardResponse struct {
	Stats               DashboardStats     `json:"stats"`
	MostViewed          []PropertyStatDTO  `json:"most_viewed"`
	MostContacted       []PropertyStatDTO  `json:"most_contacted"`
	PopularCities       []LabelCountDTO    `json:"popular_cities"`
	SearchedLocalities  []LabelCountDTO    `json:"searched_localities"`
	RecentRegistrations []UserResponse     `json:"recent_registrations"`
	RecentInquiries     []InquiryResponse  `json:"recent_inquiries"`
	Charts              AnalyticsChartsDTO `json:"charts"`
}

// ClientDashboardResponse is the builder-facing equivalent, scoped to their own
// portfolio.
type ClientDashboardResponse struct {
	Stats           ClientStats        `json:"stats"`
	TopProperties   []PropertyStatDTO  `json:"top_properties"`
	RecentInquiries []InquiryResponse  `json:"recent_inquiries"`
	Charts          AnalyticsChartsDTO `json:"charts"`
}

type ClientStats struct {
	TotalProperties    int64   `json:"total_properties"`
	ActiveProperties   int64   `json:"active_properties"`
	PendingProperties  int64   `json:"pending_properties"`
	RejectedProperties int64   `json:"rejected_properties"`
	DraftProperties    int64   `json:"draft_properties"`
	TotalViews         int64   `json:"total_views"`
	UniqueVisitors     int64   `json:"unique_visitors"`
	TotalInquiries     int64   `json:"total_inquiries"`
	WhatsAppLeads      int64   `json:"whatsapp_leads"`
	PhoneClicks        int64   `json:"phone_clicks"`
	BrochureDownloads  int64   `json:"brochure_downloads"`
	Favorites          int64   `json:"favorites"`
	ConversionRate     float64 `json:"conversion_rate"`
	// An owner with both a flat for sale and one to let needs them apart.
	SaleProperties   int64 `json:"sale_properties"`
	RentalProperties int64 `json:"rental_properties"`
	RentedProperties int64 `json:"rented_properties"`
	BuyLeads         int64 `json:"buy_leads"`
	RentLeads        int64 `json:"rent_leads"`
	PendingVisits    int64 `json:"pending_visits"`
	ExpiringSoon     int64 `json:"expiring_soon"`
}

// PropertyStatDTO mirrors the repository stat row for API responses.
type PropertyStatDTO struct {
	PropertyID        uint    `json:"property_id"`
	Name              string  `json:"name"`
	Slug              string  `json:"slug"`
	Views             int64   `json:"views"`
	UniqueVisitors    int64   `json:"unique_visitors"`
	Inquiries         int64   `json:"inquiries"`
	WhatsAppClicks    int64   `json:"whatsapp_clicks"`
	PhoneClicks       int64   `json:"phone_clicks"`
	Favorites         int64   `json:"favorites"`
	BrochureDownloads int64   `json:"brochure_downloads"`
	ConversionRate    float64 `json:"conversion_rate"`
}

type LabelCountDTO struct {
	Label string `json:"label"`
	Slug  string `json:"slug"`
	Count int64  `json:"count"`
}

type TimeSeriesDTO struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// AnalyticsChartsDTO holds the daily series rendered on the dashboards.
type AnalyticsChartsDTO struct {
	Views          []TimeSeriesDTO `json:"views"`
	Visitors       []TimeSeriesDTO `json:"visitors"`
	Inquiries      []TimeSeriesDTO `json:"inquiries"`
	Registrations  []TimeSeriesDTO `json:"registrations"`
	WhatsAppClicks []TimeSeriesDTO `json:"whatsapp_clicks"`
}

// AnalyticsQuery scopes a dashboard or analytics request.
type AnalyticsQuery struct {
	Period     string `query:"period"` // today | 7d | 30d | custom
	From       string `query:"from"`
	To         string `query:"to"`
	CityID     uint   `query:"city_id"`
	ClientID   uint   `query:"client_id"`
	PropertyID uint   `query:"property_id"`
}

// AuditLogResponse is one administrative action record.
type AuditLogResponse struct {
	ID        uint   `json:"id"`
	ActorID   uint   `json:"actor_id"`
	Action    string `json:"action"`
	Entity    string `json:"entity"`
	EntityID  uint   `json:"entity_id"`
	Details   string `json:"details"`
	IPAddress string `json:"ip_address"`
	CreatedAt string `json:"created_at"`
}

// HomepageResponse is the single payload the homepage renders from.
type HomepageResponse struct {
	Banners     []BannerResponse `json:"banners"`
	Featured    []PropertyCard   `json:"featured"`
	NewLaunches []PropertyCard   `json:"new_launches"`
	ReadyToMove []PropertyCard   `json:"ready_to_move"`
	Upcoming    []PropertyCard   `json:"upcoming"`
	// Rentals is a separate strip on purpose: a monthly rent shown beside sale
	// prices reads as a sale price.
	Rentals           []PropertyCard  `json:"rentals"`
	PopularCities     []LabelCountDTO `json:"popular_cities"`
	PopularLocalities []LabelCountDTO `json:"popular_localities"`
}

// SetVerificationRequest records an administrator's decision about how far a
// listing account has been checked. The level is ordered, so raising it is a
// deliberate act with a note attached rather than a toggle.
type SetVerificationRequest struct {
	VerificationLevel string `json:"verification_level" validate:"required,oneof=UNVERIFIED MOBILE_VERIFIED EMAIL_VERIFIED DOCUMENT_VERIFIED ADMIN_VERIFIED"`
	Note              string `json:"note" validate:"omitempty,max=500"`
}

// RentGuideResponse answers the question every renter and every owner asks
// first: what does a flat here actually go for? It is derived from live
// listings, so an empty area honestly reports no data rather than a made-up
// figure.
type RentGuideResponse struct {
	Listings    int64   `json:"listings"`
	AverageRent float64 `json:"average_rent"`
	MinRent     float64 `json:"min_rent"`
	MaxRent     float64 `json:"max_rent"`
	CityID      uint    `json:"city_id,omitempty"`
	LocalityID  uint    `json:"locality_id,omitempty"`
	BHK         int     `json:"bhk,omitempty"`
}

type RentGuideQuery struct {
	CityID     uint `query:"city_id"`
	LocalityID uint `query:"locality_id"`
	BHK        int  `query:"bhk"`
}
