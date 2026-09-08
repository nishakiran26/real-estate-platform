package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type User struct {
	BaseModel
	Email                   string     `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Mobile                  string     `gorm:"size:15;uniqueIndex;not null" json:"mobile"`
	PasswordHash            string     `gorm:"size:255;not null" json:"-"`
	FullName                string     `gorm:"size:255;not null" json:"full_name"`
	AvatarURL               string     `gorm:"size:512" json:"avatar_url"`
	Status                  Status     `gorm:"size:20;default:ACTIVE;not null" json:"status"`
	EmailVerified           bool       `gorm:"default:false" json:"email_verified"`
	MobileVerified          bool       `gorm:"default:false" json:"mobile_verified"`
	EmailVerifyToken        string     `gorm:"size:255" json:"-"`
	LastLoginAt             *time.Time `json:"last_login_at"`
	ConsentPropertyUpdates  bool       `gorm:"default:false" json:"consent_property_updates"`
	ConsentNewProjectAlerts bool       `gorm:"default:false" json:"consent_new_project_alerts"`
	ConsentPromotional      bool       `gorm:"default:false" json:"consent_promotional"`
	ConsentWhatsApp         bool       `gorm:"default:false" json:"consent_whatsapp"`
	ConsentEmail            bool       `gorm:"default:true" json:"consent_email"`
	ConsentUpdatedAt        *time.Time `json:"consent_updated_at"`

	Roles         []Role         `gorm:"many2many:user_roles;" json:"roles,omitempty"`
	RefreshTokens []RefreshToken `json:"-"`
	LoginHistory  []LoginHistory `json:"-"`
	OTPRequests   []OTPRequest   `json:"-"`
}

type Role struct {
	BaseModel
	Code        string `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Name        string `gorm:"size:100;not null" json:"name"`
	Description string `gorm:"size:500" json:"description"`
	Permissions string `gorm:"type:text" json:"permissions,omitempty"`
}

type UserRole struct {
	UserID uint `gorm:"primaryKey" json:"user_id"`
	RoleID uint `gorm:"primaryKey" json:"role_id"`
}

type RefreshToken struct {
	BaseModel
	UserID    uint       `gorm:"index;not null" json:"user_id"`
	TokenHash string     `gorm:"size:255;uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	UserAgent string     `gorm:"size:512" json:"user_agent"`
	IPAddress string     `gorm:"size:45" json:"ip_address"`
	FamilyID  string     `gorm:"size:64;index" json:"family_id"`
}

type LoginHistory struct {
	BaseModel
	UserID     uint        `gorm:"index;not null" json:"user_id"`
	Method     LoginMethod `gorm:"size:30;not null" json:"method"`
	IPAddress  string      `gorm:"size:45" json:"ip_address"`
	UserAgent  string      `gorm:"size:512" json:"user_agent"`
	LoginAt    time.Time   `gorm:"not null" json:"login_at"`
	LogoutAt   *time.Time  `json:"logout_at"`
	DeviceName string      `gorm:"size:255" json:"device_name"`
}

type OTPRequest struct {
	BaseModel
	UserID      uint       `gorm:"index" json:"user_id"`
	Email       string     `gorm:"size:255;index" json:"email"`
	Mobile      string     `gorm:"size:15;index" json:"mobile"`
	CodeHash    string     `gorm:"size:255;not null" json:"-"`
	Purpose     OTPPurpose `gorm:"size:30;not null" json:"purpose"`
	Attempts    int        `gorm:"default:0" json:"attempts"`
	MaxAttempts int        `gorm:"default:5" json:"max_attempts"`
	ExpiresAt   time.Time  `gorm:"not null" json:"expires_at"`
	VerifiedAt  *time.Time `json:"verified_at"`
	IPAddress   string     `gorm:"size:45" json:"ip_address"`
}

type Notification struct {
	BaseModel
	UserID   uint             `gorm:"index;not null" json:"user_id"`
	Type     NotificationType `gorm:"size:20;not null" json:"type"`
	Title    string           `gorm:"size:255;not null" json:"title"`
	Message  string           `gorm:"type:text;not null" json:"message"`
	Link     string           `gorm:"size:512" json:"link"`
	ReadAt   *time.Time       `json:"read_at"`
	Metadata string           `gorm:"type:json" json:"metadata"`
}

type NotificationPreference struct {
	BaseModel
	UserID          uint `gorm:"uniqueIndex;not null" json:"user_id"`
	EmailEnabled    bool `gorm:"default:true" json:"email_enabled"`
	SMSEnabled      bool `gorm:"default:true" json:"sms_enabled"`
	WhatsAppEnabled bool `gorm:"default:false" json:"whatsapp_enabled"`
	InAppEnabled    bool `gorm:"default:true" json:"in_app_enabled"`
}

type AuditLog struct {
	BaseModel
	ActorID   uint   `gorm:"index;not null" json:"actor_id"`
	Action    string `gorm:"size:100;not null" json:"action"`
	Entity    string `gorm:"size:100;not null" json:"entity"`
	EntityID  uint   `gorm:"index" json:"entity_id"`
	Details   string `gorm:"type:json" json:"details"`
	IPAddress string `gorm:"size:45" json:"ip_address"`
}

// MySQL rejects an empty string in a JSON column, and Go's zero value for these
// fields is exactly that. Without these hooks every notification and audit entry
// written without metadata fails on insert — and because both are written on
// best-effort paths that only log a warning, the loss is silent.

func (n *Notification) BeforeSave(*gorm.DB) error {
	n.Metadata = normalizeJSONObject(n.Metadata)
	return nil
}

func (a *AuditLog) BeforeSave(*gorm.DB) error {
	a.Details = normalizeJSONObject(a.Details)
	return nil
}

// normalizeJSONObject turns an unset value into an empty JSON object so the
// column always holds something MySQL will accept.
func normalizeJSONObject(value string) string {
	if strings.TrimSpace(value) == "" {
		return "{}"
	}
	return value
}
