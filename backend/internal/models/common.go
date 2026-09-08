package models

import (
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Status string

const (
	StatusActive    Status = "ACTIVE"
	StatusInactive  Status = "INACTIVE"
	StatusSuspended Status = "SUSPENDED"
	StatusDeleted   Status = "DELETED"
)

type PropertyStatus string

const (
	PropertyStatusDraft       PropertyStatus = "DRAFT"
	PropertyStatusSubmitted   PropertyStatus = "SUBMITTED"
	PropertyStatusUnderReview PropertyStatus = "UNDER_REVIEW"
	PropertyStatusApproved    PropertyStatus = "APPROVED"
	PropertyStatusRejected    PropertyStatus = "REJECTED"
	PropertyStatusPublished   PropertyStatus = "PUBLISHED"
	PropertyStatusSoldOut     PropertyStatus = "SOLD_OUT"
)

type PropertyType string

const (
	PropertyTypeApartment        PropertyType = "APARTMENT"
	PropertyTypeFlat             PropertyType = "FLAT"
	PropertyTypeVilla            PropertyType = "VILLA"
	PropertyTypeRowHouse         PropertyType = "ROW_HOUSE"
	PropertyTypePlot             PropertyType = "PLOT"
	PropertyTypePenthouse        PropertyType = "PENTHOUSE"
	PropertyTypeStudioApartment  PropertyType = "STUDIO_APARTMENT"
	PropertyTypeCommercialOffice PropertyType = "COMMERCIAL_OFFICE"
	PropertyTypeCommercialShop   PropertyType = "COMMERCIAL_SHOP"
	PropertyTypeOther            PropertyType = "OTHER"
)

type ConstructionStatus string

const (
	ConstructionUpcoming          ConstructionStatus = "UPCOMING"
	ConstructionUnderConstruction ConstructionStatus = "UNDER_CONSTRUCTION"
	ConstructionReadyToMove       ConstructionStatus = "READY_TO_MOVE"
	ConstructionSoldOut           ConstructionStatus = "SOLD_OUT"
)

type RERAStatus string

const (
	RERAStatusPendingVerification RERAStatus = "PENDING_VERIFICATION"
	RERAStatusVerified            RERAStatus = "VERIFIED"
	RERAStatusRejected            RERAStatus = "REJECTED"
	RERAStatusNotAvailable        RERAStatus = "NOT_AVAILABLE"
)

type OTPPurpose string

const (
	OTPPurposeEmailVerify    OTPPurpose = "EMAIL_VERIFY"
	OTPPurposeMobileVerify   OTPPurpose = "MOBILE_VERIFY"
	OTPPurposeLogin          OTPPurpose = "LOGIN"
	OTPPurposeForgotPassword OTPPurpose = "FORGOT_PASSWORD"
)

type LoginMethod string

const (
	LoginMethodEmailPassword LoginMethod = "EMAIL_PASSWORD"
	LoginMethodMobileOTP     LoginMethod = "MOBILE_OTP"
)

type EventType string

const (
	EventView             EventType = "VIEW"
	EventFavorite         EventType = "FAVORITE"
	EventUnfavorite       EventType = "UNFAVORITE"
	EventInquiry          EventType = "INQUIRY"
	EventWhatsappClick    EventType = "WHATSAPP_CLICK"
	EventPhoneClick       EventType = "PHONE_CLICK"
	EventEmailClick       EventType = "EMAIL_CLICK"
	EventBrochureDownload EventType = "BROCHURE_DOWNLOAD"
	EventSearch           EventType = "SEARCH"
	EventLogin            EventType = "LOGIN"
	EventRegister         EventType = "REGISTER"
)

type NotificationType string

const (
	NotificationTypeEmail    NotificationType = "EMAIL"
	NotificationTypeSMS      NotificationType = "SMS"
	NotificationTypeWhatsApp NotificationType = "WHATSAPP"
	NotificationTypeInApp    NotificationType = "IN_APP"
)

type ConsentPurpose string

const (
	ConsentPropertyUpdates  ConsentPurpose = "PROPERTY_UPDATES"
	ConsentNewProjectAlerts ConsentPurpose = "NEW_PROJECT_ALERTS"
	ConsentPromotional      ConsentPurpose = "PROMOTIONAL"
	ConsentWhatsApp         ConsentPurpose = "WHATSAPP"
	ConsentEmail            ConsentPurpose = "EMAIL"
)

type MediaType string

const (
	MediaCoverImage        MediaType = "COVER_IMAGE"
	MediaPropertyImage     MediaType = "PROPERTY_IMAGE"
	MediaSocietyImage      MediaType = "SOCIETY_IMAGE"
	MediaFloorPlan         MediaType = "FLOOR_PLAN"
	MediaMasterPlan        MediaType = "MASTER_PLAN"
	MediaLocationMap       MediaType = "LOCATION_MAP"
	MediaConstructionImage MediaType = "CONSTRUCTION_IMAGE"
	MediaRERACertificate   MediaType = "RERA_CERTIFICATE"
)

type DocType string

const (
	DocBrochure        DocType = "BROCHURE"
	DocRERACertificate DocType = "RERA_CERTIFICATE"
	DocOther           DocType = "OTHER"
)

// --- Rental marketplace -----------------------------------------------------
//
// The platform serves one catalogue with two purposes rather than two catalogues.
// PropertyPurpose is the discriminator; everything below it describes rental
// terms that have no meaning for a sale listing.

type PropertyPurpose string

const (
	PurposeBuy  PropertyPurpose = "BUY"
	PurposeRent PropertyPurpose = "RENT"
)

// PostedByType records who created a listing, which is separate from who owns
// it: an agent or an administrator may post on an owner's behalf.
type PostedByType string

const (
	PostedByOwner  PostedByType = "OWNER"
	PostedByClient PostedByType = "CLIENT"
	PostedByAgent  PostedByType = "AGENT"
	PostedByAdmin  PostedByType = "ADMIN"
)

// AvailabilityStatus is deliberately separate from PropertyStatus. Approval and
// availability are independent facts: a listing can be both PUBLISHED and
// RENTED, and collapsing them would make "rented" indistinguishable from
// "rejected" in the status history.
type AvailabilityStatus string

const (
	AvailabilityAvailable AvailabilityStatus = "AVAILABLE"
	AvailabilityReserved  AvailabilityStatus = "RESERVED"
	AvailabilityRented    AvailabilityStatus = "RENTED"
	AvailabilityInactive  AvailabilityStatus = "INACTIVE"
)

type FurnishingType string

const (
	FurnishingUnfurnished    FurnishingType = "UNFURNISHED"
	FurnishingSemiFurnished  FurnishingType = "SEMI_FURNISHED"
	FurnishingFullyFurnished FurnishingType = "FULLY_FURNISHED"
)

// TenantPreference describes the household an owner will let to. There is
// deliberately no gender field: it invites discrimination claims and the
// household type carries the signal owners actually act on.
type TenantPreference string

const (
	TenantFamily       TenantPreference = "FAMILY"
	TenantBachelor     TenantPreference = "BACHELOR"
	TenantCompanyLease TenantPreference = "COMPANY_LEASE"
	TenantAnyone       TenantPreference = "ANYONE"
)

// ProfileType distinguishes the kinds of listing account. It is a property of
// the account, not an authentication role: builders, owners and agents all hold
// the CLIENT role and differ only here.
type ProfileType string

const (
	ProfileBuilder ProfileType = "BUILDER"
	ProfileOwner   ProfileType = "OWNER"
	ProfileAgent   ProfileType = "AGENT"
)

// VerificationLevel is ordered: each level implies the ones before it. A badge
// is shown only for the level actually reached.
type VerificationLevel string

const (
	VerificationUnverified VerificationLevel = "UNVERIFIED"
	VerificationMobile     VerificationLevel = "MOBILE_VERIFIED"
	VerificationEmail      VerificationLevel = "EMAIL_VERIFIED"
	VerificationDocument   VerificationLevel = "DOCUMENT_VERIFIED"
	VerificationAdmin      VerificationLevel = "ADMIN_VERIFIED"
)

type LeadType string

const (
	LeadBuy  LeadType = "BUY_LEAD"
	LeadRent LeadType = "RENT_LEAD"
)

type VisitStatus string

const (
	VisitRequested   VisitStatus = "REQUESTED"
	VisitConfirmed   VisitStatus = "CONFIRMED"
	VisitRescheduled VisitStatus = "RESCHEDULED"
	VisitCompleted   VisitStatus = "COMPLETED"
	VisitCancelled   VisitStatus = "CANCELLED"
)

type ReportReason string

const (
	ReportAlreadyRented     ReportReason = "ALREADY_RENTED"
	ReportFakeListing       ReportReason = "FAKE_LISTING"
	ReportIncorrectPrice    ReportReason = "INCORRECT_PRICE"
	ReportIncorrectLocation ReportReason = "INCORRECT_LOCATION"
	ReportIncorrectPhotos   ReportReason = "INCORRECT_PHOTOS"
	ReportDuplicate         ReportReason = "DUPLICATE"
	ReportFraud             ReportReason = "FRAUD"
	ReportOther             ReportReason = "OTHER"
)

type ReportStatus string

const (
	ReportOpen      ReportStatus = "OPEN"
	ReportReviewing ReportStatus = "REVIEWING"
	ReportActioned  ReportStatus = "ACTIONED"
	ReportDismissed ReportStatus = "DISMISSED"
)

type AlertFrequency string

const (
	AlertInstant AlertFrequency = "INSTANT"
	AlertDaily   AlertFrequency = "DAILY"
	AlertWeekly  AlertFrequency = "WEEKLY"
)

// Rental photo buckets, added to the existing media vocabulary so a rental
// gallery reuses the same table, upload path and cover-image rules.
const (
	MediaLivingRoom       MediaType = "LIVING_ROOM"
	MediaBedroom          MediaType = "BEDROOM"
	MediaKitchen          MediaType = "KITCHEN"
	MediaBathroom         MediaType = "BATHROOM"
	MediaBalcony          MediaType = "BALCONY"
	MediaBuildingExterior MediaType = "BUILDING_EXTERIOR"
	MediaParking          MediaType = "PARKING"
	MediaAmenities        MediaType = "AMENITIES"
)

// Engagement events added for rentals. Views, favourites and inquiries already
// exist and are reused unchanged.
const (
	EventCallClick    EventType = "CALL_CLICK"
	EventVisitRequest EventType = "VISIT_REQUEST"
)
