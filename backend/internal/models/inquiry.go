package models

import (
	"time"

	"gorm.io/gorm"
)

type Client struct {
	BaseModel
	UserID          uint   `gorm:"uniqueIndex;not null" json:"user_id"`
	CompanyName     string `gorm:"size:300;not null" json:"company_name"`
	CompanyLogoURL  string `gorm:"size:512" json:"company_logo_url"`
	CompanyDesc     string `gorm:"type:text" json:"company_description"`
	Website         string `gorm:"size:500" json:"website"`
	ContactPhone    string `gorm:"size:15" json:"contact_phone"`
	ContactEmail    string `gorm:"size:255" json:"contact_email"`
	Address         string `gorm:"type:text" json:"address"`
	City            string `gorm:"size:100" json:"city"`
	State           string `gorm:"size:100" json:"state"`
	Pincode         string `gorm:"size:10" json:"pincode"`
	GSTIN           string `gorm:"size:20" json:"gstin"`
	RERARegNo       string `gorm:"size:100" json:"rera_reg_no"`
	TotalProperties int    `gorm:"default:0" json:"total_properties"`

	// ProfileType distinguishes a builder from an individual owner or an agent.
	// All three are listing accounts under the CLIENT role, which is what lets a
	// rental owner reuse the ownership, lead and analytics paths unchanged.
	ProfileType ProfileType `gorm:"size:20;index;default:BUILDER;not null" json:"profile_type"`

	// VerificationLevel gates the trust badge. A badge is shown only for the
	// level actually reached, never as decoration.
	VerificationLevel VerificationLevel `gorm:"size:30;index;default:UNVERIFIED;not null" json:"verification_level"`
	VerifiedAt        *time.Time        `json:"verified_at,omitempty"`
	VerifiedBy        *uint             `json:"verified_by,omitempty"`

	User       User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Properties []Property `gorm:"foreignKey:ClientID" json:"properties,omitempty"`
}

type CustomerProfile struct {
	BaseModel
	UserID            uint    `gorm:"uniqueIndex;not null" json:"user_id"`
	PreferredCity     string  `gorm:"size:100" json:"preferred_city"`
	PreferredLocality string  `gorm:"size:200" json:"preferred_locality"`
	MinBudget         float64 `gorm:"default:0" json:"min_budget"`
	MaxBudget         float64 `gorm:"default:0" json:"max_budget"`
	PropertyType      string  `gorm:"size:50" json:"property_type"`
	BHK               int     `gorm:"default:0" json:"bhk"`
	Purpose           string  `gorm:"size:20;default:BUY" json:"purpose"`

	// A buyer and a renter are the same account with different intents, not two
	// logins. Both can be true. They are indexed columns rather than a JSON list
	// because administrators filter the user list on them.
	//
	// Neither carries a GORM `default` tag on purpose: GORM substitutes the
	// column default for any zero value, so `default:true` would silently make a
	// renter-only profile look like a buyer. The application sets both fields
	// explicitly, and the migration backfills existing rows.
	LookingToBuy  bool `gorm:"index" json:"looking_to_buy"`
	LookingToRent bool `gorm:"index" json:"looking_to_rent"`

	// Rental-specific preferences. They stay nil-valued for a pure buyer, and
	// live here rather than in a separate table because they are one-to-one with
	// the profile and read on every recommendation.
	MonthlyRentMin       float64          `gorm:"default:0" json:"monthly_rent_min"`
	MonthlyRentMax       float64          `gorm:"default:0" json:"monthly_rent_max"`
	PreferredFurnishing  FurnishingType   `gorm:"size:30" json:"preferred_furnishing"`
	PreferredLeaseMonths int              `gorm:"default:0" json:"preferred_lease_months"`
	MoveInDate           *time.Time       `json:"move_in_date"`
	FamilyType           TenantPreference `gorm:"size:30" json:"family_type"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type Inquiry struct {
	BaseModel
	PropertyID      uint    `gorm:"index;not null" json:"property_id"`
	ClientID        uint    `gorm:"index;not null" json:"client_id"`
	UserID          *uint   `gorm:"index" json:"user_id"`
	Name            string  `gorm:"size:255;not null" json:"name"`
	Mobile          string  `gorm:"size:15;not null" json:"mobile"`
	Email           string  `gorm:"size:255" json:"email"`
	Message         string  `gorm:"type:text" json:"message"`
	ConfigurationID *uint   `json:"configuration_id"`
	Budget          float64 `gorm:"default:0" json:"budget"`
	PreferredTime   string  `gorm:"size:100" json:"preferred_time"`
	SessionID       string  `gorm:"size:64;index" json:"session_id"`
	Status          string  `gorm:"size:30;default:NEW;not null" json:"status"`
	Source          string  `gorm:"size:50;default:WEB" json:"source"`

	// LeadType is derived from the property at capture time so lead reporting
	// does not have to join back to find out what the enquiry was about. Leads
	// that predate the rental marketplace are BUY_LEAD.
	LeadType LeadType `gorm:"size:20;index;default:BUY_LEAD;not null" json:"lead_type"`

	// AssignedToUserID lets a lead be routed to a specific person: the owner who
	// listed the flat, an agent handling it, or an administrator. Nil means the
	// whole listing account sees it, which is today's behaviour.
	AssignedToUserID *uint      `gorm:"index" json:"assigned_to_user_id,omitempty"`
	AssignedAt       *time.Time `json:"assigned_at,omitempty"`
	AssignedBy       *uint      `json:"assigned_by,omitempty"`
	IPAddress        string     `gorm:"size:45" json:"ip_address"`
	FollowedUpAt     *time.Time `json:"followed_up_at"`
	FollowedBy       *uint      `json:"followed_by"`

	Property      Property               `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	Configuration *PropertyConfiguration `gorm:"foreignKey:ConfigurationID" json:"configuration,omitempty"`
}

type Favorite struct {
	BaseModel
	UserID     uint `gorm:"uniqueIndex:idx_user_property;not null" json:"user_id"`
	PropertyID uint `gorm:"uniqueIndex:idx_user_property;not null" json:"property_id"`
}

type UserPropertyEvent struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     *uint     `gorm:"index" json:"user_id"`
	PropertyID *uint     `gorm:"index" json:"property_id"`
	EventType  EventType `gorm:"size:30;not null;index" json:"event_type"`
	SessionID  string    `gorm:"size:64;index" json:"session_id"`
	IPAddress  string    `gorm:"size:45" json:"ip_address"`
	UserAgent  string    `gorm:"size:512" json:"user_agent"`
	Metadata   string    `gorm:"type:json" json:"metadata"`
	CreatedAt  time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

type PropertyView struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	PropertyID uint      `gorm:"index;not null" json:"property_id"`
	UserID     *uint     `gorm:"index" json:"user_id"`
	SessionID  string    `gorm:"size:64;index" json:"session_id"`
	IPAddress  string    `gorm:"size:45" json:"ip_address"`
	UserAgent  string    `gorm:"size:512" json:"user_agent"`
	Referrer   string    `gorm:"size:512" json:"referrer"`
	Duration   int       `gorm:"default:0" json:"duration"`
	CreatedAt  time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

type StaticPage struct {
	BaseModel
	Title           string `gorm:"size:300;not null" json:"title"`
	Slug            string `gorm:"size:300;uniqueIndex;not null" json:"slug"`
	Content         string `gorm:"type:longtext;not null" json:"content"`
	MetaTitle       string `gorm:"size:300" json:"meta_title"`
	MetaDescription string `gorm:"size:500" json:"meta_description"`
	IsPublished     bool   `gorm:"default:true" json:"is_published"`
}

// RentalAlert is a saved search that notifies a customer when a matching rental
// is published. The criteria are stored as JSON because they mirror the search
// filter exactly: giving each filter its own column would mean a migration every
// time the search gains a parameter.
type RentalAlert struct {
	BaseModel
	UserID uint   `gorm:"index;not null" json:"user_id"`
	Name   string `gorm:"size:200" json:"name"`

	// Criteria is a serialised PropertyListQuery. Matching is done in the
	// service layer against the same filter the search uses, so an alert can
	// never drift from what the customer would see on the site.
	Criteria string `gorm:"type:json;not null" json:"criteria"`

	// The columns below duplicate the hot criteria so a newly published property
	// can be matched with an indexed query instead of decoding every alert.
	CityID     uint    `gorm:"index" json:"city_id"`
	LocalityID uint    `gorm:"index" json:"locality_id"`
	BHK        int     `gorm:"index" json:"bhk"`
	MinRent    float64 `json:"min_rent"`
	MaxRent    float64 `json:"max_rent"`

	Frequency AlertFrequency `gorm:"size:20;default:DAILY;not null" json:"frequency"`
	// IsActive carries no `default` tag on purpose: GORM omits a false bool from
	// the INSERT and the column default would put it back, silently reviving a
	// paused alert. The service sets it explicitly instead.
	IsActive bool `gorm:"index" json:"is_active"`

	LastNotifiedAt *time.Time `json:"last_notified_at"`
	MatchCount     int64      `gorm:"default:0" json:"match_count"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

// BeforeSave keeps the event metadata valid JSON. Engagement events are written
// on a fire-and-forget path that only logs failures, so an empty string here
// would drop the event with no visible error.
func (e *UserPropertyEvent) BeforeSave(*gorm.DB) error {
	e.Metadata = normalizeJSONObject(e.Metadata)
	return nil
}

// BeforeSave guards the saved-search criteria, which are NOT NULL.
func (a *RentalAlert) BeforeSave(*gorm.DB) error {
	a.Criteria = normalizeJSONObject(a.Criteria)
	return nil
}
