package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/internal/validators"
	"github.com/rera/property-platform/backend/pkg/logger"
)

// AccountService covers profile, preference and notification management for
// customers and builders alike.
type AccountService struct {
	users    repositories.UserRepository
	accounts repositories.AccountRepository
	engage   repositories.EngagementRepository
	recs     *RecommendationService
}

func NewAccountService(
	users repositories.UserRepository,
	accounts repositories.AccountRepository,
	engage repositories.EngagementRepository,
	recs *RecommendationService,
) *AccountService {
	return &AccountService{users: users, accounts: accounts, engage: engage, recs: recs}
}

// UpdateProfile edits the caller's own account. Changing a verified contact
// resets its verified flag so the new address or number must be confirmed.
func (s *AccountService) UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest) (*dto.UserResponse, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Account not found")
		}
		return nil, Internal("load user", err)
	}

	fields := map[string]any{}
	if name := strings.TrimSpace(req.FullName); name != "" && name != user.FullName {
		fields["full_name"] = name
	}
	if req.AvatarURL != "" {
		fields["avatar_url"] = req.AvatarURL
	}
	if email := strings.ToLower(strings.TrimSpace(req.Email)); email != "" && email != user.Email {
		exists, err := s.users.EmailExists(ctx, email)
		if err != nil {
			return nil, Internal("check email", err)
		}
		if exists {
			return nil, Conflict("That email address is already in use")
		}
		fields["email"] = email
		fields["email_verified"] = false
	}
	if mobile := validators.NormalizeMobile(req.Mobile); mobile != "" && mobile != user.Mobile {
		exists, err := s.users.MobileExists(ctx, mobile)
		if err != nil {
			return nil, Internal("check mobile", err)
		}
		if exists {
			return nil, Conflict("That mobile number is already in use")
		}
		fields["mobile"] = mobile
		fields["mobile_verified"] = false
	}
	if len(fields) > 0 {
		if err := s.users.UpdateFields(ctx, userID, fields); err != nil {
			return nil, Internal("update profile", err)
		}
	}

	updated, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, Internal("load user", err)
	}
	resp := ToUserResponse(updated)
	if client, err := s.accounts.GetClientByUserID(ctx, userID); err == nil {
		resp.ClientID = &client.ID
	}
	return &resp, nil
}

// UpdateConsent records a change to marketing permissions, including full
// withdrawal, with a timestamp.
func (s *AccountService) UpdateConsent(ctx context.Context, userID uint, req dto.UpdateConsentRequest) (*dto.ConsentResponse, error) {
	if err := s.users.UpdateFields(ctx, userID, map[string]any{
		"consent_property_updates":   req.PropertyUpdates,
		"consent_new_project_alerts": req.NewProjectAlerts,
		"consent_promotional":        req.Promotional,
		"consent_whatsapp":           req.WhatsApp,
		"consent_email":              req.Email,
		"consent_updated_at":         time.Now(),
	}); err != nil {
		return nil, Internal("update consent", err)
	}
	return &dto.ConsentResponse{
		PropertyUpdates: req.PropertyUpdates, NewProjectAlerts: req.NewProjectAlerts,
		Promotional: req.Promotional, WhatsApp: req.WhatsApp, Email: req.Email,
	}, nil
}

// Unsubscribe switches off every non-essential channel in one call, backing the
// opt-out link in marketing messages.
func (s *AccountService) Unsubscribe(ctx context.Context, userID uint) error {
	if err := s.users.UpdateFields(ctx, userID, map[string]any{
		"consent_property_updates":   false,
		"consent_new_project_alerts": false,
		"consent_promotional":        false,
		"consent_whatsapp":           false,
		"consent_updated_at":         time.Now(),
	}); err != nil {
		return Internal("unsubscribe", err)
	}
	return nil
}

// GetPreferences returns a buyer's search preferences.
func (s *AccountService) GetPreferences(ctx context.Context, userID uint) (*dto.CustomerPreferenceResponse, error) {
	p, err := s.accounts.GetCustomerProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return &dto.CustomerPreferenceResponse{Purpose: "BUY"}, nil
		}
		return nil, Internal("load preferences", err)
	}
	return &dto.CustomerPreferenceResponse{
		PreferredCity: p.PreferredCity, PreferredLocality: p.PreferredLocality,
		MinBudget: p.MinBudget, MaxBudget: p.MaxBudget,
		PropertyType: p.PropertyType, BHK: p.BHK, Purpose: p.Purpose,
		LookingToBuy: p.LookingToBuy, LookingToRent: p.LookingToRent,
		MonthlyRentMin: p.MonthlyRentMin, MonthlyRentMax: p.MonthlyRentMax,
		PreferredFurnishing: string(p.PreferredFurnishing),
		FamilyType:          string(p.FamilyType),
	}, nil
}

// SavePreferences stores what the buyer is looking for.
func (s *AccountService) SavePreferences(ctx context.Context, userID uint, req dto.CustomerPreferenceRequest) (*dto.CustomerPreferenceResponse, error) {
	if req.MaxBudget > 0 && req.MinBudget > req.MaxBudget {
		return nil, Invalid("The minimum budget cannot exceed the maximum budget")
	}
	if req.MonthlyRentMax > 0 && req.MonthlyRentMin > req.MonthlyRentMax {
		return nil, Invalid("The minimum rent cannot exceed the maximum rent")
	}

	// Start from what is stored. The upsert overwrites every column it lists,
	// so building the profile from the request alone would wipe whatever this
	// form does not carry — which is how a renter silently became interested in
	// neither buying nor renting.
	profile := &models.CustomerProfile{UserID: userID}
	if existing, err := s.accounts.GetCustomerProfile(ctx, userID); err == nil {
		profile = existing
	} else if !errors.Is(err, repositories.ErrNotFound) {
		return nil, Internal("load preferences", err)
	}

	profile.UserID = userID
	profile.PreferredCity = req.PreferredCity
	profile.PreferredLocality = req.PreferredLocality
	profile.MinBudget, profile.MaxBudget = req.MinBudget, req.MaxBudget
	profile.PropertyType, profile.BHK = req.PropertyType, req.BHK
	profile.MonthlyRentMin, profile.MonthlyRentMax = req.MonthlyRentMin, req.MonthlyRentMax
	profile.PreferredFurnishing = models.FurnishingType(req.PreferredFurnishing)
	profile.FamilyType = models.TenantPreference(req.FamilyType)

	if req.Purpose != "" {
		profile.Purpose = req.Purpose
	} else if profile.Purpose == "" {
		profile.Purpose = "BUY"
	}

	// Intents are only touched when the caller actually sent them.
	if req.LookingToBuy != nil {
		profile.LookingToBuy = *req.LookingToBuy
	}
	if req.LookingToRent != nil {
		profile.LookingToRent = *req.LookingToRent
	}
	// An account interested in neither would be recommended nothing, so the
	// last one switched off falls back to the older single-valued purpose.
	if !profile.LookingToBuy && !profile.LookingToRent {
		if strings.EqualFold(profile.Purpose, "RENT") {
			profile.LookingToRent = true
		} else {
			profile.LookingToBuy = true
		}
	}

	if err := s.accounts.UpsertCustomerProfile(ctx, profile); err != nil {
		return nil, Internal("save preferences", err)
	}
	return s.GetPreferences(ctx, userID)
}

// GetNotificationPreferences returns the delivery channel settings.
func (s *AccountService) GetNotificationPreferences(ctx context.Context, userID uint) (*dto.NotificationPreferenceResponse, error) {
	p, err := s.accounts.GetNotificationPreference(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return &dto.NotificationPreferenceResponse{
				EmailEnabled: true, SMSEnabled: true, InAppEnabled: true,
			}, nil
		}
		return nil, Internal("load notification preferences", err)
	}
	return &dto.NotificationPreferenceResponse{
		EmailEnabled: p.EmailEnabled, SMSEnabled: p.SMSEnabled,
		WhatsAppEnabled: p.WhatsAppEnabled, InAppEnabled: p.InAppEnabled,
	}, nil
}

func (s *AccountService) SaveNotificationPreferences(ctx context.Context, userID uint, req dto.NotificationPreferenceRequest) (*dto.NotificationPreferenceResponse, error) {
	if err := s.accounts.UpsertNotificationPreference(ctx, &models.NotificationPreference{
		UserID: userID, EmailEnabled: req.EmailEnabled, SMSEnabled: req.SMSEnabled,
		WhatsAppEnabled: req.WhatsAppEnabled, InAppEnabled: req.InAppEnabled,
	}); err != nil {
		return nil, Internal("save notification preferences", err)
	}
	return s.GetNotificationPreferences(ctx, userID)
}

// ListNotifications returns the in-app inbox.
func (s *AccountService) ListNotifications(ctx context.Context, userID uint, unreadOnly bool, page, perPage int) ([]dto.NotificationResponse, *dto.Meta, error) {
	page, perPage = normalizePage(page, perPage)
	list, total, err := s.accounts.ListNotifications(ctx, userID, unreadOnly, (page-1)*perPage, perPage)
	if err != nil {
		return nil, nil, Internal("load notifications", err)
	}
	out := make([]dto.NotificationResponse, 0, len(list))
	for i := range list {
		out = append(out, ToNotificationResponse(&list[i]))
	}
	return out, newMeta(page, perPage, total), nil
}

func (s *AccountService) MarkNotificationRead(ctx context.Context, userID, id uint) error {
	if err := s.accounts.MarkNotificationRead(ctx, userID, id); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Notification not found")
		}
		return Internal("mark notification read", err)
	}
	return nil
}

func (s *AccountService) MarkAllNotificationsRead(ctx context.Context, userID uint) error {
	if err := s.accounts.MarkAllNotificationsRead(ctx, userID); err != nil {
		return Internal("mark notifications read", err)
	}
	return nil
}

// CustomerDashboard assembles the buyer's landing screen.
func (s *AccountService) CustomerDashboard(ctx context.Context, userID uint, sessionID string) (*dto.CustomerDashboardResponse, error) {
	resp := &dto.CustomerDashboardResponse{}

	favIDs, err := s.engage.FavoriteIDs(ctx, userID)
	if err != nil {
		return nil, Internal("load favourites", err)
	}
	resp.SavedProperties = int64(len(favIDs))

	if resp.Inquiries, err = s.countCustomerInquiries(ctx, userID); err != nil {
		return nil, err
	}
	if resp.UnreadNotifications, err = s.accounts.UnreadNotificationCount(ctx, userID); err != nil {
		return nil, Internal("count notifications", err)
	}

	viewed, err := s.engage.RecentlyViewed(ctx, &userID, sessionID, 8)
	if err != nil {
		return nil, Internal("load recently viewed", err)
	}
	resp.RecentlyViewed = ToPropertyCards(viewed)
	resp.RecentlyViewedCount = len(resp.RecentlyViewed)

	recommended, err := s.recs.ForUser(ctx, userID, 8)
	if err != nil {
		// Recommendations are a nice-to-have; the dashboard still renders.
		logger.Log.Warn("could not load recommendations", "user_id", userID, "error", err)
	} else {
		resp.Recommended = recommended
	}
	return resp, nil
}

func (s *AccountService) countCustomerInquiries(ctx context.Context, userID uint) (int64, error) {
	_, total, err := s.engage.ListInquiries(ctx, repositories.InquiryFilter{UserID: userID, Limit: 1})
	if err != nil {
		return 0, Internal("count inquiries", err)
	}
	return total, nil
}

// GetClientProfile returns the builder company record.
func (s *AccountService) GetClientProfile(ctx context.Context, clientID uint) (*dto.ClientResponse, error) {
	c, err := s.accounts.GetClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Builder profile not found")
		}
		return nil, Internal("load client", err)
	}
	resp := ToClientResponse(c)
	return &resp, nil
}

// UpdateClientProfile edits the builder's own company record.
func (s *AccountService) UpdateClientProfile(ctx context.Context, clientID uint, req dto.UpdateClientProfileRequest) (*dto.ClientResponse, error) {
	c, err := s.accounts.GetClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Builder profile not found")
		}
		return nil, Internal("load client", err)
	}
	c.CompanyName = strings.TrimSpace(req.CompanyName)
	c.CompanyLogoURL = req.CompanyLogoURL
	c.CompanyDesc = req.CompanyDesc
	c.Website = req.Website
	c.ContactPhone = validators.NormalizeMobile(req.ContactPhone)
	c.ContactEmail = strings.ToLower(strings.TrimSpace(req.ContactEmail))
	c.Address, c.City, c.State, c.Pincode = req.Address, req.City, req.State, req.Pincode
	c.GSTIN, c.RERARegNo = req.GSTIN, req.RERARegNo

	if err := s.accounts.UpdateClient(ctx, c); err != nil {
		return nil, Internal("update client", err)
	}
	resp := ToClientResponse(c)
	return &resp, nil
}

// PublicBuilder returns the public company profile shown on property pages.
func (s *AccountService) PublicBuilder(ctx context.Context, clientID uint) (*dto.BuilderResponse, error) {
	c, err := s.accounts.GetClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Builder not found")
		}
		return nil, Internal("load client", err)
	}
	// Deliberately narrow: the public view never exposes GSTIN, address or the
	// owner's personal contact details.
	return &dto.BuilderResponse{
		ID: c.ID, CompanyName: c.CompanyName, LogoURL: c.CompanyLogoURL,
		Description: c.CompanyDesc, Website: c.Website,
		RERARegNo: c.RERARegNo, TotalProperties: c.TotalProperties,
	}, nil
}

// LoginHistory maps recent sessions for the settings screen.
func (s *AccountService) LoginHistory(ctx context.Context, entries []models.LoginHistory) []dto.LoginHistoryResponse {
	out := make([]dto.LoginHistoryResponse, 0, len(entries))
	for i := range entries {
		out = append(out, ToLoginHistoryResponse(&entries[i]))
	}
	return out
}

// EnableListingAccount gives an existing account the ability to post property.
// This is what keeps a buyer who decides to rent out their flat on one login:
// they gain the CLIENT role alongside CUSTOMER and a listing account record,
// rather than registering a second time.
//
// The caller's current access token still carries the old role set, so the
// client must refresh its token before the builder endpoints will admit it.
func (s *AccountService) EnableListingAccount(
	ctx context.Context, userID uint, req dto.EnableListingAccountRequest,
) (*dto.ClientResponse, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, notFoundOr(err, "Account not found")
	}

	// Already a listing account: return it rather than creating a duplicate,
	// which the unique index on clients.user_id would reject anyway.
	if existing, err := s.accounts.GetClientByUserID(ctx, userID); err == nil && existing != nil {
		resp := ToClientResponse(existing)
		return &resp, nil
	}

	company := strings.TrimSpace(req.CompanyName)
	if company == "" {
		company = user.FullName
	}

	client := &models.Client{
		UserID:            userID,
		CompanyName:       company,
		ContactEmail:      user.Email,
		ContactPhone:      user.Mobile,
		ProfileType:       models.ProfileType(req.ProfileType),
		VerificationLevel: verificationFor(user),
	}
	if err := s.accounts.CreateClient(ctx, client); err != nil {
		return nil, Internal("create listing account", err)
	}

	if err := s.grantRole(ctx, user, "CLIENT"); err != nil {
		return nil, err
	}

	resp := ToClientResponse(client)
	return &resp, nil
}

// grantRole adds a role while keeping the ones the user already holds, so
// becoming an owner never costs someone their customer access.
func (s *AccountService) grantRole(ctx context.Context, user *models.User, code string) error {
	role, err := s.users.RoleByCode(ctx, code)
	if err != nil {
		return Internal("look up role", err)
	}

	ids := make([]uint, 0, len(user.Roles)+1)
	for _, existing := range user.Roles {
		if existing.Code == code {
			return nil // already granted
		}
		ids = append(ids, existing.ID)
	}
	ids = append(ids, role.ID)

	if err := s.users.SetRoles(ctx, user.ID, ids); err != nil {
		return Internal("assign role", err)
	}
	return nil
}

// verificationFor reports the level a new listing account starts at, based on
// what the user has actually confirmed. A badge is never granted for a step that
// did not happen.
func verificationFor(user *models.User) models.VerificationLevel {
	switch {
	case user.EmailVerified && user.MobileVerified:
		return models.VerificationEmail
	case user.MobileVerified:
		return models.VerificationMobile
	default:
		return models.VerificationUnverified
	}
}
