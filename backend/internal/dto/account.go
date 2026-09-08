package dto

// UpdateProfileRequest edits the fields a user owns on their own account.
type UpdateProfileRequest struct {
	FullName  string `json:"full_name" validate:"omitempty,min=2,max=255"`
	AvatarURL string `json:"avatar_url" validate:"omitempty,url,max=512"`
	Email     string `json:"email" validate:"omitempty,email"`
	Mobile    string `json:"mobile" validate:"omitempty,mobile"`
}

// UpdateClientProfileRequest edits the builder company record.
type UpdateClientProfileRequest struct {
	CompanyName    string `json:"company_name" validate:"required,min=2,max=300"`
	CompanyLogoURL string `json:"company_logo_url" validate:"omitempty,max=512"`
	CompanyDesc    string `json:"company_description" validate:"omitempty,max=5000"`
	Website        string `json:"website" validate:"omitempty,url,max=500"`
	ContactPhone   string `json:"contact_phone" validate:"omitempty,mobile"`
	ContactEmail   string `json:"contact_email" validate:"omitempty,email"`
	Address        string `json:"address" validate:"omitempty,max=1000"`
	City           string `json:"city" validate:"omitempty,max=100"`
	State          string `json:"state" validate:"omitempty,max=100"`
	Pincode        string `json:"pincode" validate:"omitempty,len=6"`
	GSTIN          string `json:"gstin" validate:"omitempty,max=20"`
	RERARegNo      string `json:"rera_reg_no" validate:"omitempty,max=100"`
}

type ClientResponse struct {
	ID              uint   `json:"id"`
	UserID          uint   `json:"user_id"`
	CompanyName     string `json:"company_name"`
	CompanyLogoURL  string `json:"company_logo_url"`
	CompanyDesc     string `json:"company_description"`
	Website         string `json:"website"`
	ContactPhone    string `json:"contact_phone"`
	ContactEmail    string `json:"contact_email"`
	Address         string `json:"address"`
	City            string `json:"city"`
	State           string `json:"state"`
	Pincode         string `json:"pincode"`
	GSTIN           string `json:"gstin"`
	RERARegNo       string `json:"rera_reg_no"`
	TotalProperties int    `json:"total_properties"`
	// ProfileType tells the interface whether to speak to a builder, an
	// individual owner or an agent; VerificationLevel gates the trust badge.
	ProfileType       string `json:"profile_type"`
	VerificationLevel string `json:"verification_level"`
	OwnerName         string `json:"owner_name,omitempty"`
	OwnerEmail        string `json:"owner_email,omitempty"`
	Status            string `json:"status,omitempty"`
	CreatedAt         string `json:"created_at"`
}

// CustomerPreferenceRequest captures what a buyer is looking for; it drives the
// recommendation engine.
type CustomerPreferenceRequest struct {
	PreferredCity     string  `json:"preferred_city" validate:"omitempty,max=100"`
	PreferredLocality string  `json:"preferred_locality" validate:"omitempty,max=200"`
	MinBudget         float64 `json:"min_budget" validate:"gte=0"`
	MaxBudget         float64 `json:"max_budget" validate:"gte=0"`
	PropertyType      string  `json:"property_type" validate:"omitempty,max=50"`
	BHK               int     `json:"bhk" validate:"gte=0,lte=20"`
	Purpose           string  `json:"purpose" validate:"omitempty,oneof=BUY INVEST RENT"`

	// Intents are pointers so an omitted field keeps its stored value. They are
	// what decides which catalogue the dashboard recommends from, and a form
	// that does not send them must not silently turn a renter into a buyer.
	LookingToBuy  *bool `json:"looking_to_buy"`
	LookingToRent *bool `json:"looking_to_rent"`

	// Rental criteria, for the renting half of the account.
	MonthlyRentMin      float64 `json:"monthly_rent_min" validate:"gte=0"`
	MonthlyRentMax      float64 `json:"monthly_rent_max" validate:"gte=0"`
	PreferredFurnishing string  `json:"preferred_furnishing" validate:"omitempty,oneof=UNFURNISHED SEMI_FURNISHED FULLY_FURNISHED"`
	FamilyType          string  `json:"family_type" validate:"omitempty,oneof=FAMILY BACHELOR COMPANY_LEASE ANYONE"`
}

type CustomerPreferenceResponse struct {
	PreferredCity     string  `json:"preferred_city"`
	PreferredLocality string  `json:"preferred_locality"`
	MinBudget         float64 `json:"min_budget"`
	MaxBudget         float64 `json:"max_budget"`
	PropertyType      string  `json:"property_type"`
	BHK               int     `json:"bhk"`
	Purpose           string  `json:"purpose"`

	LookingToBuy  bool `json:"looking_to_buy"`
	LookingToRent bool `json:"looking_to_rent"`

	MonthlyRentMin      float64 `json:"monthly_rent_min"`
	MonthlyRentMax      float64 `json:"monthly_rent_max"`
	PreferredFurnishing string  `json:"preferred_furnishing,omitempty"`
	FamilyType          string  `json:"family_type,omitempty"`
}

// NotificationPreferenceRequest toggles delivery channels.
type NotificationPreferenceRequest struct {
	EmailEnabled    bool `json:"email_enabled"`
	SMSEnabled      bool `json:"sms_enabled"`
	WhatsAppEnabled bool `json:"whatsapp_enabled"`
	InAppEnabled    bool `json:"in_app_enabled"`
}

type NotificationPreferenceResponse struct {
	EmailEnabled    bool `json:"email_enabled"`
	SMSEnabled      bool `json:"sms_enabled"`
	WhatsAppEnabled bool `json:"whatsapp_enabled"`
	InAppEnabled    bool `json:"in_app_enabled"`
}

type NotificationResponse struct {
	ID        uint   `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Link      string `json:"link"`
	ReadAt    string `json:"read_at,omitempty"`
	CreatedAt string `json:"created_at"`
}

// CustomerDashboardResponse summarises a buyer's activity.
type CustomerDashboardResponse struct {
	SavedProperties     int64          `json:"saved_properties"`
	Inquiries           int64          `json:"inquiries"`
	RecentlyViewedCount int            `json:"recently_viewed_count"`
	UnreadNotifications int64          `json:"unread_notifications"`
	RecentlyViewed      []PropertyCard `json:"recently_viewed"`
	Recommended         []PropertyCard `json:"recommended"`
}

// EnableListingAccountRequest turns an existing account into one that can also
// post property. It is what keeps a buyer who decides to rent out their flat on
// a single login rather than registering again.
type EnableListingAccountRequest struct {
	ProfileType string `json:"profile_type" validate:"required,oneof=OWNER BUILDER AGENT"`
	CompanyName string `json:"company_name" validate:"omitempty,max=300"`
}
