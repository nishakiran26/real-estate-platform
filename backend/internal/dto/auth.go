package dto

// RegisterRequest is the self-service signup payload. RoleCode is limited to the
// roles a visitor may claim; administrators are provisioned separately.
type RegisterRequest struct {
	FullName string `json:"full_name" validate:"required,min=2,max=255"`
	Email    string `json:"email" validate:"required,email"`
	Mobile   string `json:"mobile" validate:"required,mobile"`
	Password string `json:"password" validate:"required,min=8,max=72"`

	// RoleCode is the original two-option field, kept so existing clients keep
	// working. New callers send ProfileType instead, which is more expressive;
	// when both are absent the request is rejected.
	RoleCode string `json:"role_code" validate:"omitempty,oneof=customer client"`

	// ProfileType is what the person says they are. It resolves to a system role
	// plus, for listing accounts, a profile type — so an owner and a builder
	// share one authentication path and differ only in their account record.
	ProfileType string `json:"profile_type" validate:"omitempty,oneof=BUYER RENTER OWNER BUILDER AGENT"`

	// Intents lets a buyer also mark themselves a renter without a second
	// account. Ignored for listing accounts, which are defined by ProfileType.
	Intents []string `json:"intents" validate:"omitempty,dive,oneof=BUY RENT"`

	// CompanyName seeds the listing account. An individual owner may leave it
	// blank, in which case their own name is used.
	CompanyName string `json:"company_name" validate:"omitempty,max=300"`

	// Explicit, unbundled marketing consent (see the lead-privacy rules).
	ConsentPropertyUpdates  bool `json:"consent_property_updates"`
	ConsentNewProjectAlerts bool `json:"consent_new_project_alerts"`
	ConsentPromotional      bool `json:"consent_promotional"`
	ConsentWhatsApp         bool `json:"consent_whatsapp"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required_without=Mobile,omitempty,email"`
	Mobile   string `json:"mobile" validate:"required_without=Email,omitempty,mobile"`
	Password string `json:"password" validate:"required"`
}

// MobileOTPLoginRequest completes a passwordless login.
type MobileOTPLoginRequest struct {
	Mobile string `json:"mobile" validate:"required,mobile"`
	Code   string `json:"code" validate:"required,len=6"`
}

type SendOTPRequest struct {
	Email   string `json:"email" validate:"omitempty,email"`
	Mobile  string `json:"mobile" validate:"omitempty,mobile"`
	Purpose string `json:"purpose" validate:"required,oneof=EMAIL_VERIFY MOBILE_VERIFY LOGIN FORGOT_PASSWORD"`
}

type VerifyOTPRequest struct {
	Email   string `json:"email" validate:"omitempty,email"`
	Mobile  string `json:"mobile" validate:"omitempty,mobile"`
	Code    string `json:"code" validate:"required,len=6"`
	Purpose string `json:"purpose" validate:"required,oneof=EMAIL_VERIFY MOBILE_VERIFY LOGIN FORGOT_PASSWORD"`
}

// OTPResponse acknowledges a dispatched code without echoing the code or the
// full destination back to the caller.
type OTPResponse struct {
	OTPRequestID uint   `json:"otp_request_id"`
	ExpiresIn    int    `json:"expires_in"`
	SentTo       string `json:"sent_to"`
	Purpose      string `json:"purpose"`
}

// VerifyOTPResponse carries the outcome and, for password resets, the one-time
// token that authorises the actual password change.
type VerifyOTPResponse struct {
	Verified   bool   `json:"verified"`
	Purpose    string `json:"purpose"`
	Message    string `json:"message"`
	ResetToken string `json:"reset_token,omitempty"`
}

type ForgotPasswordRequest struct {
	Email  string `json:"email" validate:"omitempty,email"`
	Mobile string `json:"mobile" validate:"omitempty,mobile"`
}

type ResetPasswordRequest struct {
	ResetToken  string `json:"reset_token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=72"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
	AllDevices   bool   `json:"all_devices"`
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type LoginResponse struct {
	Tokens AuthTokens   `json:"tokens"`
	User   UserResponse `json:"user"`
}

type UserResponse struct {
	ID             uint            `json:"id"`
	FullName       string          `json:"full_name"`
	Email          string          `json:"email"`
	Mobile         string          `json:"mobile"`
	AvatarURL      string          `json:"avatar_url"`
	Status         string          `json:"status"`
	EmailVerified  bool            `json:"email_verified"`
	MobileVerified bool            `json:"mobile_verified"`
	Roles          []RoleResponse  `json:"roles"`
	ClientID       *uint           `json:"client_id,omitempty"`
	Consent        ConsentResponse `json:"consent"`
	CreatedAt      string          `json:"created_at,omitempty"`
	LastLoginAt    string          `json:"last_login_at,omitempty"`
}

// ConsentResponse reflects the stored marketing preferences.
type ConsentResponse struct {
	PropertyUpdates  bool `json:"property_updates"`
	NewProjectAlerts bool `json:"new_project_alerts"`
	Promotional      bool `json:"promotional"`
	WhatsApp         bool `json:"whatsapp"`
	Email            bool `json:"email"`
}

// UpdateConsentRequest lets a user change or withdraw marketing consent.
type UpdateConsentRequest struct {
	PropertyUpdates  bool `json:"property_updates"`
	NewProjectAlerts bool `json:"new_project_alerts"`
	Promotional      bool `json:"promotional"`
	WhatsApp         bool `json:"whatsapp"`
	Email            bool `json:"email"`
}

type RoleResponse struct {
	ID   uint   `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// LoginHistoryResponse is one row of the account activity list.
type LoginHistoryResponse struct {
	ID        uint   `json:"id"`
	Method    string `json:"method"`
	IPAddress string `json:"ip_address"`
	Device    string `json:"device"`
	LoginAt   string `json:"login_at"`
	LogoutAt  string `json:"logout_at,omitempty"`
}
