package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rera/property-platform/backend/internal/cache"
	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/notification"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/internal/validators"
	"github.com/rera/property-platform/backend/pkg/auth"
	"github.com/rera/property-platform/backend/pkg/logger"
	"github.com/rera/property-platform/backend/pkg/utils"
)

// RequestMeta carries the transport details a handler knows and the service
// needs for audit trails, without leaking the HTTP context into the service.
type RequestMeta struct {
	IP        string
	UserAgent string
}

// AuthService owns registration, login, OTP and token lifecycle.
type AuthService struct {
	users    repositories.UserRepository
	authRepo repositories.AuthRepository
	accounts repositories.AccountRepository
	jwt      *auth.JWTService
	cache    *cache.Client
	notifier *notification.Dispatcher
	cfg      *config.Config
}

func NewAuthService(
	users repositories.UserRepository,
	authRepo repositories.AuthRepository,
	accounts repositories.AccountRepository,
	jwt *auth.JWTService,
	cache *cache.Client,
	notifier *notification.Dispatcher,
	cfg *config.Config,
) *AuthService {
	return &AuthService{users: users, authRepo: authRepo, accounts: accounts,
		jwt: jwt, cache: cache, notifier: notifier, cfg: cfg}
}

// selfServiceRoles are the only roles a visitor may claim at registration.
// Administrators are provisioned by seed or by another administrator.
var selfServiceRoles = map[string]string{
	"customer": "CUSTOMER",
	"client":   "CLIENT",
}

// registration describes what a self-service signup resolves to: one system
// role, and for listing accounts the kind of account it is. Builders, owners and
// agents deliberately share the CLIENT role — they differ only in ProfileType,
// so every ownership, lead and analytics path serves all three unchanged.
type registration struct {
	role        string
	profileType models.ProfileType
	buys        bool
	rents       bool
}

var registrationIntents = map[string]registration{
	"BUYER":   {role: "CUSTOMER", buys: true},
	"RENTER":  {role: "CUSTOMER", rents: true},
	"OWNER":   {role: "CLIENT", profileType: models.ProfileOwner},
	"BUILDER": {role: "CLIENT", profileType: models.ProfileBuilder},
	"AGENT":   {role: "CLIENT", profileType: models.ProfileAgent},
}

// resolveRegistration turns what the caller said into a role and account shape,
// accepting both the newer ProfileType and the original RoleCode field.
func resolveRegistration(req dto.RegisterRequest) (registration, error) {
	if pt := strings.ToUpper(strings.TrimSpace(req.ProfileType)); pt != "" {
		reg, ok := registrationIntents[pt]
		if !ok {
			return registration{}, Invalid("Unrecognised profile type")
		}
		// A customer may be looking to buy, to rent, or both. The profile type
		// sets the default; explicit intents override it.
		if reg.role == "CUSTOMER" && len(req.Intents) > 0 {
			reg.buys, reg.rents = false, false
			for _, intent := range req.Intents {
				switch strings.ToUpper(strings.TrimSpace(intent)) {
				case "BUY":
					reg.buys = true
				case "RENT":
					reg.rents = true
				}
			}
			if !reg.buys && !reg.rents {
				return registration{}, Invalid("Choose whether you are looking to buy, to rent, or both")
			}
		}
		return reg, nil
	}

	// Fall back to the original field so existing clients keep working.
	switch role := selfServiceRoles[strings.ToLower(strings.TrimSpace(req.RoleCode))]; role {
	case "CUSTOMER":
		return registration{role: "CUSTOMER", buys: true}, nil
	case "CLIENT":
		return registration{role: "CLIENT", profileType: models.ProfileBuilder}, nil
	default:
		return registration{}, Invalid("Tell us what you are looking to do")
	}
}

// Register creates an unverified account and dispatches verification OTPs.
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest, meta RequestMeta) (*dto.LoginResponse, error) {
	reg, err := resolveRegistration(req)
	if err != nil {
		return nil, err
	}
	roleCode := reg.role
	email := strings.ToLower(strings.TrimSpace(req.Email))
	mobile := validators.NormalizeMobile(req.Mobile)

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, Internal("hash password", err)
	}

	user := &models.User{
		Email:        email,
		Mobile:       mobile,
		PasswordHash: hash,
		FullName:     strings.TrimSpace(req.FullName),
		Status:       models.StatusActive,
		// Consent is opt-in: only what the user explicitly agreed to is stored.
		ConsentPropertyUpdates:  req.ConsentPropertyUpdates,
		ConsentNewProjectAlerts: req.ConsentNewProjectAlerts,
		ConsentPromotional:      req.ConsentPromotional,
		ConsentWhatsApp:         req.ConsentWhatsApp,
		ConsentEmail:            true, // transactional email is required to run the account
	}
	now := time.Now()
	user.ConsentUpdatedAt = &now

	if err := s.users.Create(ctx, user, []string{roleCode}); err != nil {
		switch {
		case errors.Is(err, repositories.ErrEmailExists):
			return nil, Conflict("An account with this email already exists")
		case errors.Is(err, repositories.ErrMobileExists):
			return nil, Conflict("An account with this mobile number already exists")
		}
		return nil, Internal("create user", err)
	}

	// Anyone who lists property — builder, owner or agent — needs a listing
	// account before they can post. An individual owner rarely has a company
	// name, so their own name stands in.
	if roleCode == "CLIENT" {
		company := strings.TrimSpace(req.CompanyName)
		if company == "" {
			company = user.FullName
		}
		client := &models.Client{
			UserID: user.ID, CompanyName: company,
			ContactEmail: user.Email, ContactPhone: user.Mobile,
			ProfileType:       reg.profileType,
			VerificationLevel: models.VerificationUnverified,
		}
		if err := s.accounts.CreateClient(ctx, client); err != nil {
			return nil, Internal("create client profile", err)
		}
	}

	// A buyer or renter gets a profile immediately so their stated intent is
	// recorded, rather than waiting until they open the preferences screen.
	if roleCode == "CUSTOMER" {
		profile := &models.CustomerProfile{
			UserID:        user.ID,
			LookingToBuy:  reg.buys,
			LookingToRent: reg.rents,
			Purpose:       "BUY",
		}
		if reg.rents && !reg.buys {
			profile.Purpose = "RENT"
		}
		if err := s.accounts.UpsertCustomerProfile(ctx, profile); err != nil {
			// A missing profile is recoverable from the preferences screen, so
			// this must not fail an otherwise successful registration.
			logger.Log.Warn("could not create customer profile", "user_id", user.ID, "error", err)
		}
	}

	// Verification codes are best-effort: a delivery failure must not undo a
	// successful registration, since the user can request a resend.
	if _, err := s.issueOTP(ctx, user, email, models.OTPPurposeEmailVerify, meta); err != nil {
		logger.Log.Warn("could not send email verification OTP", "user_id", user.ID, "error", err)
	}
	if _, err := s.issueOTP(ctx, user, mobile, models.OTPPurposeMobileVerify, meta); err != nil {
		logger.Log.Warn("could not send mobile verification OTP", "user_id", user.ID, "error", err)
	}

	return s.grantSession(ctx, user, models.LoginMethodEmailPassword, meta)
}

// Login authenticates with email or mobile plus password.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, meta RequestMeta) (*dto.LoginResponse, error) {
	identifier := strings.ToLower(strings.TrimSpace(req.Email))
	if identifier == "" {
		identifier = validators.NormalizeMobile(req.Mobile)
	}
	if identifier == "" {
		return nil, Invalid("Provide an email address or mobile number")
	}

	user, err := s.users.GetByEmailOrMobile(ctx, identifier)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			// Same message for unknown accounts and wrong passwords so the API
			// cannot be used to enumerate registered users.
			return nil, Unauthorized("Invalid credentials")
		}
		return nil, Internal("lookup user", err)
	}
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		return nil, Unauthorized("Invalid credentials")
	}
	if err := s.assertUsable(user); err != nil {
		return nil, err
	}
	return s.grantSession(ctx, user, models.LoginMethodEmailPassword, meta)
}

// LoginWithOTP completes a passwordless mobile login.
func (s *AuthService) LoginWithOTP(ctx context.Context, mobile, code string, meta RequestMeta) (*dto.LoginResponse, error) {
	mobile = validators.NormalizeMobile(mobile)
	user, err := s.users.GetByMobile(ctx, mobile)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("No account is registered with this mobile number")
		}
		return nil, Internal("lookup user", err)
	}
	if err := s.assertUsable(user); err != nil {
		return nil, err
	}
	if err := s.consumeOTP(ctx, mobile, code, models.OTPPurposeLogin); err != nil {
		return nil, err
	}
	// A successful OTP login proves control of the number.
	if !user.MobileVerified {
		if err := s.users.UpdateFields(ctx, user.ID, map[string]any{"mobile_verified": true}); err != nil {
			logger.Log.Warn("could not flag mobile verified", "user_id", user.ID, "error", err)
		}
		user.MobileVerified = true
	}
	return s.grantSession(ctx, user, models.LoginMethodMobileOTP, meta)
}

// assertUsable rejects suspended or deactivated accounts.
func (s *AuthService) assertUsable(u *models.User) error {
	switch u.Status {
	case models.StatusSuspended:
		return Forbidden("This account has been suspended. Please contact support.")
	case models.StatusInactive, models.StatusDeleted:
		return Forbidden("This account is no longer active")
	}
	return nil
}

// SendOTP issues a code for any supported purpose, throttled per identifier.
func (s *AuthService) SendOTP(ctx context.Context, req dto.SendOTPRequest, meta RequestMeta) (*dto.OTPResponse, error) {
	identifier, err := otpIdentifier(req.Email, req.Mobile)
	if err != nil {
		return nil, err
	}
	purpose := models.OTPPurpose(strings.ToUpper(req.Purpose))

	user, err := s.users.GetByEmailOrMobile(ctx, identifier)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return nil, Internal("lookup user", err)
	}
	// Login and password-reset codes require an existing account; verification
	// codes are also only meaningful for one.
	if user == nil {
		if purpose == models.OTPPurposeLogin || purpose == models.OTPPurposeForgotPassword {
			return nil, NotFound("No account is registered with these details")
		}
		return nil, NotFound("No account is registered with these details")
	}
	return s.issueOTP(ctx, user, identifier, purpose, meta)
}

// issueOTP generates, stores and dispatches a one-time code.
func (s *AuthService) issueOTP(ctx context.Context, user *models.User, identifier string, purpose models.OTPPurpose, meta RequestMeta) (*dto.OTPResponse, error) {
	// Throttle: at most 3 codes per identifier per 15 minutes. The database
	// count keeps the limit effective even when Redis is unavailable.
	since := time.Now().Add(-15 * time.Minute)
	recent, err := s.authRepo.CountRecentOTPs(ctx, identifier, since)
	if err != nil {
		return nil, Internal("count recent OTPs", err)
	}
	if recent >= 3 {
		return nil, TooMany("Too many verification codes requested. Please try again in a few minutes.")
	}

	code := utils.GenerateOTP(6)
	codeHash, err := auth.HashPassword(code)
	if err != nil {
		return nil, Internal("hash OTP", err)
	}

	req := &models.OTPRequest{
		UserID:      user.ID,
		CodeHash:    codeHash,
		Purpose:     purpose,
		MaxAttempts: s.cfg.OTP.MaxAttempts,
		ExpiresAt:   time.Now().Add(time.Duration(s.cfg.OTP.TTL) * time.Second),
		IPAddress:   meta.IP,
	}
	if strings.Contains(identifier, "@") {
		req.Email = identifier
	} else {
		req.Mobile = identifier
	}
	if err := s.authRepo.CreateOTP(ctx, req); err != nil {
		return nil, Internal("store OTP", err)
	}

	// Mirror into Redis for fast verification; the database row remains the
	// source of truth if the cache is cold.
	s.cache.SetJSON(ctx, otpCacheKey(identifier, purpose),
		map[string]any{"id": req.ID, "hash": codeHash},
		time.Duration(s.cfg.OTP.TTL)*time.Second)

	s.dispatchOTP(ctx, user, identifier, code, purpose)

	return &dto.OTPResponse{
		OTPRequestID: req.ID,
		ExpiresIn:    s.cfg.OTP.TTL,
		SentTo:       maskIdentifier(identifier),
		Purpose:      string(purpose),
	}, nil
}

// dispatchOTP delivers the code on the channel implied by the identifier.
func (s *AuthService) dispatchOTP(ctx context.Context, user *models.User, identifier, code string, purpose models.OTPPurpose) {
	subject := "Your verification code"
	if purpose == models.OTPPurposeForgotPassword {
		subject = "Reset your password"
	}
	body := fmt.Sprintf("Hello %s,\n\nYour %s verification code is %s.\nIt expires in %d minutes.\n\nIf you did not request this, you can ignore this message.",
		user.FullName, s.cfg.App.Name, code, s.cfg.OTP.TTL/60)

	channel := notification.ChannelSMS
	if strings.Contains(identifier, "@") {
		channel = notification.ChannelEmail
	}
	msg := notification.Message{
		To: identifier, Subject: subject, Body: body,
		Data: map[string]string{"code": code, "purpose": string(purpose)},
	}

	// Delivery runs detached from the request: a slow or unreachable SMTP server
	// must not hold the HTTP response open, and the send must not be cancelled
	// when the client disconnects. The code is already stored, so the user can
	// still verify once the message arrives.
	go func() {
		sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()

		if err := s.notifier.Send(sendCtx, channel, msg); err != nil {
			// Logged at error level: a failure here is invisible to the caller,
			// who is deliberately told only that a code "has been sent", so the
			// server log is the only place a misconfiguration shows up.
			logger.Log.Error("OTP delivery failed",
				"channel", channel, "to", maskIdentifier(identifier),
				"purpose", purpose, "error", err)
		}
	}()
}

// VerifyOTP validates a code and applies the side effect for its purpose.
func (s *AuthService) VerifyOTP(ctx context.Context, req dto.VerifyOTPRequest, meta RequestMeta) (*dto.VerifyOTPResponse, error) {
	identifier, err := otpIdentifier(req.Email, req.Mobile)
	if err != nil {
		return nil, err
	}
	purpose := models.OTPPurpose(strings.ToUpper(req.Purpose))
	if err := s.consumeOTP(ctx, identifier, req.Code, purpose); err != nil {
		return nil, err
	}

	user, err := s.users.GetByEmailOrMobile(ctx, identifier)
	if err != nil {
		return nil, Internal("lookup user", err)
	}

	resp := &dto.VerifyOTPResponse{Verified: true, Purpose: string(purpose)}
	switch purpose {
	case models.OTPPurposeEmailVerify:
		if err := s.users.UpdateFields(ctx, user.ID, map[string]any{"email_verified": true}); err != nil {
			return nil, Internal("mark email verified", err)
		}
		resp.Message = "Email address verified"
	case models.OTPPurposeMobileVerify:
		if err := s.users.UpdateFields(ctx, user.ID, map[string]any{"mobile_verified": true}); err != nil {
			return nil, Internal("mark mobile verified", err)
		}
		resp.Message = "Mobile number verified"
	case models.OTPPurposeForgotPassword:
		// Hand back a short-lived single-use token instead of letting the client
		// replay the OTP against the reset endpoint.
		token, err := auth.RandomToken(32)
		if err != nil {
			return nil, Internal("generate reset token", err)
		}
		if err := s.cache.Set(ctx, resetCacheKey(token), fmt.Sprint(user.ID), 15*time.Minute); err != nil {
			logger.Log.Warn("reset token cache write failed", "error", err)
		}
		if !s.cache.Available() {
			return nil, Internal("reset token store unavailable", errors.New("redis unavailable"))
		}
		resp.ResetToken = token
		resp.Message = "Code verified. You can now set a new password."
	case models.OTPPurposeLogin:
		resp.Message = "Code verified"
	}
	return resp, nil
}

// consumeOTP validates a code against the stored hash, enforcing expiry and the
// per-request attempt limit.
func (s *AuthService) consumeOTP(ctx context.Context, identifier, code string, purpose models.OTPPurpose) error {
	otp, err := s.authRepo.LatestOTP(ctx, identifier, purpose)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return Invalid("This code has expired or is not valid. Please request a new one.")
		}
		return Internal("load OTP", err)
	}
	if time.Now().After(otp.ExpiresAt) {
		return Invalid("This code has expired. Please request a new one.")
	}
	if otp.Attempts >= otp.MaxAttempts {
		return TooMany("Too many incorrect attempts. Please request a new code.")
	}
	if !auth.CheckPassword(code, otp.CodeHash) {
		if err := s.authRepo.IncrementOTPAttempts(ctx, otp.ID); err != nil {
			logger.Log.Warn("could not record OTP attempt", "otp_id", otp.ID, "error", err)
		}
		remaining := otp.MaxAttempts - otp.Attempts - 1
		if remaining <= 0 {
			return TooMany("Too many incorrect attempts. Please request a new code.")
		}
		return Invalid(fmt.Sprintf("Incorrect code. %d attempt(s) remaining.", remaining))
	}
	if err := s.authRepo.MarkOTPVerified(ctx, otp.ID); err != nil {
		return Internal("mark OTP verified", err)
	}
	_ = s.cache.Del(ctx, otpCacheKey(identifier, purpose))
	return nil
}

// ForgotPassword sends a reset code without revealing whether the account exists.
func (s *AuthService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest, meta RequestMeta) error {
	identifier, err := otpIdentifier(req.Email, req.Mobile)
	if err != nil {
		return err
	}
	user, err := s.users.GetByEmailOrMobile(ctx, identifier)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil // silent success: do not disclose registration status
		}
		return Internal("lookup user", err)
	}
	if _, err := s.issueOTP(ctx, user, identifier, models.OTPPurposeForgotPassword, meta); err != nil {
		var se *Error
		if errors.As(err, &se) && se.Kind == KindTooMany {
			return err
		}
		return Internal("send reset code", err)
	}
	return nil
}

// ResetPassword sets a new password using the token issued by VerifyOTP and
// revokes every existing session.
func (s *AuthService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	raw, err := s.cache.Get(ctx, resetCacheKey(req.ResetToken))
	if err != nil {
		return Invalid("This reset link has expired. Please start again.")
	}
	var userID uint
	if _, err := fmt.Sscanf(raw, "%d", &userID); err != nil {
		return Invalid("This reset link is not valid")
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return Internal("hash password", err)
	}
	if err := s.users.UpdateFields(ctx, userID, map[string]any{"password_hash": hash}); err != nil {
		return Internal("update password", err)
	}
	_ = s.cache.Del(ctx, resetCacheKey(req.ResetToken))
	if err := s.authRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		logger.Log.Warn("could not revoke sessions after reset", "user_id", userID, "error", err)
	}
	return nil
}

// ChangePassword updates the password for a signed-in user.
func (s *AuthService) ChangePassword(ctx context.Context, userID uint, req dto.ChangePasswordRequest) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return Internal("load user", err)
	}
	if !auth.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		return Invalid("Your current password is incorrect")
	}
	if req.CurrentPassword == req.NewPassword {
		return Invalid("The new password must differ from the current one")
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return Internal("hash password", err)
	}
	if err := s.users.UpdateFields(ctx, userID, map[string]any{"password_hash": hash}); err != nil {
		return Internal("update password", err)
	}
	// Other devices keep stale credentials, so drop their sessions.
	if err := s.authRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		logger.Log.Warn("could not revoke sessions after password change", "user_id", userID, "error", err)
	}
	return nil
}

// Refresh rotates a refresh token. Replaying an already-rotated token revokes
// the whole family, which contains the damage from a stolen token.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string, meta RequestMeta) (*dto.AuthTokens, error) {
	hash := auth.HashToken(refreshToken)
	stored, err := s.authRepo.GetRefreshToken(ctx, hash)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, Unauthorized("Your session has expired. Please sign in again.")
		}
		return nil, Internal("load refresh token", err)
	}
	if stored.RevokedAt != nil {
		if err := s.authRepo.RevokeTokenFamily(ctx, stored.FamilyID); err != nil {
			logger.Log.Error("could not revoke token family after reuse", "family", stored.FamilyID, "error", err)
		}
		logger.Log.Warn("refresh token reuse detected", "user_id", stored.UserID, "family", stored.FamilyID)
		return nil, Unauthorized("Your session has expired. Please sign in again.")
	}
	if time.Now().After(stored.ExpiresAt) {
		return nil, Unauthorized("Your session has expired. Please sign in again.")
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, Internal("load user", err)
	}
	if err := s.assertUsable(user); err != nil {
		return nil, err
	}
	if err := s.authRepo.RevokeRefreshToken(ctx, stored.ID); err != nil {
		return nil, Internal("rotate refresh token", err)
	}
	tokens, err := s.issueTokens(ctx, user, stored.FamilyID, meta)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// Logout revokes the presented refresh token, or every session when allDevices
// is set.
func (s *AuthService) Logout(ctx context.Context, userID uint, refreshToken string, allDevices bool) error {
	if allDevices {
		if err := s.authRepo.RevokeAllUserTokens(ctx, userID); err != nil {
			return Internal("revoke sessions", err)
		}
	} else if refreshToken != "" {
		stored, err := s.authRepo.GetRefreshToken(ctx, auth.HashToken(refreshToken))
		if err == nil && stored.UserID == userID {
			if err := s.authRepo.RevokeRefreshToken(ctx, stored.ID); err != nil {
				return Internal("revoke session", err)
			}
		}
	}
	if err := s.authRepo.RecordLogout(ctx, userID); err != nil {
		logger.Log.Warn("could not record logout", "user_id", userID, "error", err)
	}
	return nil
}

// Me returns the profile for the authenticated user.
func (s *AuthService) Me(ctx context.Context, userID uint) (*dto.UserResponse, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Account not found")
		}
		return nil, Internal("load user", err)
	}
	resp := ToUserResponse(user)
	if client, err := s.accounts.GetClientByUserID(ctx, userID); err == nil {
		resp.ClientID = &client.ID
	}
	return &resp, nil
}

// LoginHistory lists recent sessions for the account settings page.
func (s *AuthService) LoginHistory(ctx context.Context, userID uint, limit int) ([]models.LoginHistory, error) {
	list, err := s.authRepo.ListLoginHistory(ctx, userID, limit)
	if err != nil {
		return nil, Internal("load login history", err)
	}
	return list, nil
}

// grantSession issues tokens and records the login.
func (s *AuthService) grantSession(ctx context.Context, user *models.User, method models.LoginMethod, meta RequestMeta) (*dto.LoginResponse, error) {
	family := uuid.NewString()
	tokens, err := s.issueTokens(ctx, user, family, meta)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if err := s.authRepo.RecordLogin(ctx, &models.LoginHistory{
		UserID: user.ID, Method: method, IPAddress: meta.IP,
		UserAgent: meta.UserAgent, LoginAt: now,
		DeviceName: deviceName(meta.UserAgent),
	}); err != nil {
		logger.Log.Warn("could not record login", "user_id", user.ID, "error", err)
	}
	if err := s.users.UpdateFields(ctx, user.ID, map[string]any{"last_login_at": now}); err != nil {
		logger.Log.Warn("could not update last login", "user_id", user.ID, "error", err)
	}

	resp := &dto.LoginResponse{Tokens: *tokens, User: ToUserResponse(user)}
	if client, err := s.accounts.GetClientByUserID(ctx, user.ID); err == nil {
		resp.User.ClientID = &client.ID
	}
	return resp, nil
}

// issueTokens mints an access/refresh pair and persists the refresh hash.
func (s *AuthService) issueTokens(ctx context.Context, user *models.User, familyID string, meta RequestMeta) (*dto.AuthTokens, error) {
	roles := roleCodes(user)
	var clientID *uint
	if client, err := s.accounts.GetClientByUserID(ctx, user.ID); err == nil {
		clientID = &client.ID
	}

	pair, err := s.jwt.GenerateTokenPair(user.ID, user.Email, user.Mobile, user.FullName, roles, clientID)
	if err != nil {
		return nil, Internal("generate tokens", err)
	}
	// The refresh token is opaque; only its digest is stored.
	opaque, err := auth.RandomToken(48)
	if err != nil {
		return nil, Internal("generate refresh token", err)
	}
	record := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: auth.HashToken(opaque),
		ExpiresAt: time.Now().Add(s.jwt.RefreshTTL),
		UserAgent: meta.UserAgent,
		IPAddress: meta.IP,
		FamilyID:  familyID,
	}
	if err := s.authRepo.CreateRefreshToken(ctx, record); err != nil {
		return nil, Internal("store refresh token", err)
	}
	return &dto.AuthTokens{
		AccessToken:  pair.AccessToken,
		RefreshToken: opaque,
		ExpiresIn:    pair.ExpiresIn,
		TokenType:    "Bearer",
	}, nil
}

func roleCodes(u *models.User) []string {
	codes := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		codes = append(codes, r.Code)
	}
	return codes
}

// ToUserResponse maps the user model onto its API representation, omitting the
// password hash and other internal fields.
func ToUserResponse(u *models.User) dto.UserResponse {
	roles := make([]dto.RoleResponse, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, dto.RoleResponse{ID: r.ID, Code: r.Code, Name: r.Name})
	}
	return dto.UserResponse{
		ID: u.ID, FullName: u.FullName, Email: u.Email, Mobile: u.Mobile,
		AvatarURL: u.AvatarURL, Status: string(u.Status),
		EmailVerified: u.EmailVerified, MobileVerified: u.MobileVerified,
		Roles: roles,
		Consent: dto.ConsentResponse{
			PropertyUpdates:  u.ConsentPropertyUpdates,
			NewProjectAlerts: u.ConsentNewProjectAlerts,
			Promotional:      u.ConsentPromotional,
			WhatsApp:         u.ConsentWhatsApp,
			Email:            u.ConsentEmail,
		},
	}
}

func otpIdentifier(email, mobile string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email != "" {
		return email, nil
	}
	mobile = validators.NormalizeMobile(mobile)
	if mobile != "" {
		return mobile, nil
	}
	return "", Invalid("Provide an email address or mobile number")
}

func otpCacheKey(identifier string, purpose models.OTPPurpose) string {
	return fmt.Sprintf("otp:%s:%s", purpose, identifier)
}

func resetCacheKey(token string) string { return "pwreset:" + token }

// maskIdentifier hides most of an email or number in API responses so a code
// can be confirmed as sent without echoing contact details back.
func maskIdentifier(identifier string) string {
	if at := strings.Index(identifier, "@"); at > 0 {
		local, domain := identifier[:at], identifier[at:]
		if len(local) <= 2 {
			return local[:1] + "***" + domain
		}
		return local[:2] + strings.Repeat("*", len(local)-2) + domain
	}
	if len(identifier) > 4 {
		return strings.Repeat("*", len(identifier)-4) + identifier[len(identifier)-4:]
	}
	return "****"
}

// deviceName extracts a human-readable client label from a user agent string.
func deviceName(ua string) string {
	switch {
	case ua == "":
		return "Unknown device"
	case strings.Contains(ua, "Android"):
		return "Android device"
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"):
		return "iOS device"
	case strings.Contains(ua, "Edg/"):
		return "Microsoft Edge"
	case strings.Contains(ua, "Chrome/"):
		return "Chrome browser"
	case strings.Contains(ua, "Firefox/"):
		return "Firefox browser"
	case strings.Contains(ua, "Safari/"):
		return "Safari browser"
	default:
		return "Web browser"
	}
}
