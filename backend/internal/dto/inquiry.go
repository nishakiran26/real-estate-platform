package dto

// CreateInquiryRequest is the lead-capture payload. It is available to
// anonymous visitors, so the contact fields are validated strictly.
type CreateInquiryRequest struct {
	PropertyID      uint    `json:"property_id" validate:"required"`
	Name            string  `json:"name" validate:"required,min=2,max=255"`
	Mobile          string  `json:"mobile" validate:"required,mobile"`
	Email           string  `json:"email" validate:"omitempty,email"`
	Message         string  `json:"message" validate:"omitempty,max=2000"`
	ConfigurationID *uint   `json:"configuration_id"`
	Budget          float64 `json:"budget" validate:"gte=0"`
	PreferredTime   string  `json:"preferred_time" validate:"omitempty,max=100"`
	Source          string  `json:"source" validate:"omitempty,max=50"`
	// ConsentContact records that the visitor agreed to be contacted about
	// this enquiry; it is required for a lead to be stored.
	ConsentContact bool `json:"consent_contact" validate:"required"`
}

type InquiryResponse struct {
	ID            uint             `json:"id"`
	PropertyID    uint             `json:"property_id"`
	Property      *PropertySummary `json:"property,omitempty"`
	ClientID      uint             `json:"client_id"`
	Name          string           `json:"name"`
	Mobile        string           `json:"mobile"`
	Email         string           `json:"email"`
	Message       string           `json:"message"`
	Configuration string           `json:"configuration,omitempty"`
	Budget        float64          `json:"budget"`
	PreferredTime string           `json:"preferred_time"`
	Status        string           `json:"status"`
	Source        string           `json:"source"`
	CreatedAt     string           `json:"created_at"`
}

type InquiryListQuery struct {
	Page       int    `query:"page"`
	PerPage    int    `query:"per_page"`
	Status     string `query:"status"`
	PropertyID uint   `query:"property_id"`
	Search     string `query:"q"`
	From       string `query:"from"`
	To         string `query:"to"`
}

type UpdateInquiryStatusRequest struct {
	Status  string `json:"status" validate:"required,oneof=NEW CONTACTED SITE_VISIT_SCHEDULED NEGOTIATING CLOSED_WON CLOSED_LOST"`
	Remarks string `json:"remarks" validate:"omitempty,max=1000"`
}

// TrackEventRequest records an engagement event from the browser. It works for
// anonymous visitors, who are identified by session ID.
type TrackEventRequest struct {
	EventType  string `json:"event_type" validate:"required,oneof=VIEW FAVORITE UNFAVORITE INQUIRY WHATSAPP_CLICK PHONE_CLICK CALL_CLICK EMAIL_CLICK BROCHURE_DOWNLOAD VISIT_REQUEST SEARCH"`
	PropertyID *uint  `json:"property_id"`
	Metadata   string `json:"metadata" validate:"omitempty,max=2000"`
}

// WhatsAppLinkResponse returns the prefilled deep link for a property, so the
// message template lives on the server rather than in the frontend.
type WhatsAppLinkResponse struct {
	URL     string `json:"url"`
	Message string `json:"message"`
	Phone   string `json:"phone"`
}

// CreateVisitRequest asks to view a property in person. Like an inquiry, it is
// open to visitors who have not signed in, so the contact fields are validated
// strictly and consent is explicit.
type CreateVisitRequest struct {
	// PropertyID is taken from the request path, not the body, so a visitor
	// cannot book a viewing of one listing while looking at another. It carries
	// no `required` tag because the handler fills it in after binding.
	PropertyID uint   `json:"-"`
	Name       string `json:"name" validate:"required,min=2,max=255"`
	Mobile     string `json:"mobile" validate:"required,mobile"`
	Email      string `json:"email" validate:"omitempty,email"`
	Message    string `json:"message" validate:"omitempty,max=1000"`

	// VisitDate and TimeSlot are what the visitor asked for; the owner confirms
	// or proposes another time.
	VisitDate string `json:"visit_date" validate:"required,datetime=2006-01-02"`
	TimeSlot  string `json:"time_slot" validate:"omitempty,oneof=MORNING AFTERNOON EVENING ANY"`

	ConsentContact bool `json:"consent_contact" validate:"required"`
}

// UpdateVisitRequest is the owner or builder responding to a request.
type UpdateVisitRequest struct {
	Status string `json:"status" validate:"required,oneof=CONFIRMED RESCHEDULED COMPLETED CANCELLED"`
	// ScheduledDate is required when confirming or proposing a new time.
	ScheduledDate string `json:"scheduled_date" validate:"omitempty,datetime=2006-01-02"`
	TimeSlot      string `json:"time_slot" validate:"omitempty,oneof=MORNING AFTERNOON EVENING ANY"`
	Remarks       string `json:"remarks" validate:"omitempty,max=1000"`
}

type VisitResponse struct {
	ID         uint             `json:"id"`
	PropertyID uint             `json:"property_id"`
	Property   *PropertySummary `json:"property,omitempty"`
	Name       string           `json:"name"`
	// Contact details are present only for the account that owns the listing and
	// for administrators; the customer view of their own visit includes them too.
	Mobile      string `json:"mobile,omitempty"`
	Email       string `json:"email,omitempty"`
	Message     string `json:"message"`
	RequestedAt string `json:"requested_at"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
	TimeSlot    string `json:"time_slot,omitempty"`
	Status      string `json:"status"`
	Remarks     string `json:"remarks,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type VisitListQuery struct {
	Page       int    `query:"page"`
	PerPage    int    `query:"per_page"`
	Status     string `query:"status"`
	PropertyID uint   `query:"property_id"`
	From       string `query:"from"`
	To         string `query:"to"`
}

// CreateReportRequest flags a listing as wrong or abusive. It is open to
// visitors who have not signed in, because the people best placed to notice a
// flat that is already let are the ones still searching for it.
type CreateReportRequest struct {
	// PropertyID comes from the path, as with a visit request.
	PropertyID uint   `json:"-"`
	Reason     string `json:"reason" validate:"required,oneof=ALREADY_RENTED FAKE_LISTING INCORRECT_PRICE INCORRECT_LOCATION INCORRECT_PHOTOS DUPLICATE FRAUD OTHER"`
	Details    string `json:"details" validate:"omitempty,max=1000"`
}

// ResolveReportRequest is an administrator closing a complaint.
type ResolveReportRequest struct {
	Status     string `json:"status" validate:"required,oneof=REVIEWING ACTIONED DISMISSED"`
	ActionNote string `json:"action_note" validate:"omitempty,max=1000"`
	// Unpublish takes the listing down in the same step, which is the usual
	// outcome of an upheld complaint.
	Unpublish bool `json:"unpublish"`
}

type ReportResponse struct {
	ID         uint             `json:"id"`
	PropertyID uint             `json:"property_id"`
	Property   *PropertySummary `json:"property,omitempty"`
	Reason     string           `json:"reason"`
	Details    string           `json:"details,omitempty"`
	Status     string           `json:"status"`
	// OpenReports counts the complaints still open against the same listing, so
	// the queue shows which listing is being reported repeatedly.
	OpenReports int64  `json:"open_reports"`
	ActionNote  string `json:"action_note,omitempty"`
	ReviewedAt  string `json:"reviewed_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type ReportListQuery struct {
	Page       int    `query:"page"`
	PerPage    int    `query:"per_page"`
	Status     string `query:"status"`
	Reason     string `query:"reason"`
	PropertyID uint   `query:"property_id"`
}

// CreateRentalAlertRequest saves the search a renter is running so they hear
// about matching flats without checking the site every day. The criteria mirror
// the rent filters exactly, because an alert that matched something different
// from what the site shows would be worse than no alert at all.
type CreateRentalAlertRequest struct {
	Name       string  `json:"name" validate:"omitempty,max=200"`
	CityID     uint    `json:"city_id"`
	LocalityID uint    `json:"locality_id"`
	BHK        int     `json:"bhk" validate:"omitempty,gte=1,lte=10"`
	MinRent    float64 `json:"min_rent" validate:"omitempty,gte=0"`
	MaxRent    float64 `json:"max_rent" validate:"omitempty,gte=0"`
	// Furnishing and TenantPreference are carried in the criteria so the saved
	// link reproduces the search, even though matching does not filter on them.
	Furnishing       string `json:"furnishing" validate:"omitempty,oneof=UNFURNISHED SEMI_FURNISHED FULLY_FURNISHED"`
	TenantPreference string `json:"tenant_preference" validate:"omitempty,oneof=FAMILY BACHELOR COMPANY_LEASE ANYONE"`

	// There is deliberately no frequency here. Matching alerts are notified as
	// soon as a listing is published, and nothing in this platform batches them:
	// a daily or weekly digest needs a scheduler that does not exist yet. Offering
	// the choice would be a setting that silently does nothing.
}

// RentalAlertResponse is a saved search as the customer sees it.
type RentalAlertResponse struct {
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	CityID     uint    `json:"city_id"`
	LocalityID uint    `json:"locality_id"`
	BHK        int     `json:"bhk"`
	MinRent    float64 `json:"min_rent"`
	MaxRent    float64 `json:"max_rent"`
	Frequency  string  `json:"frequency"`
	IsActive   bool    `json:"is_active"`
	MatchCount int64   `json:"match_count"`
	// SearchQuery is the query string that reruns this search on the site, so
	// the saved alert and the live results can never disagree.
	SearchQuery    string `json:"search_query"`
	LastNotifiedAt string `json:"last_notified_at,omitempty"`
	CreatedAt      string `json:"created_at"`
}
