package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/notification"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/internal/validators"
	"github.com/rera/property-platform/backend/pkg/logger"
)

// viewDedupWindow stops a page refresh from inflating the view counter.
const viewDedupWindow = 30 * time.Minute

// Visitor identifies whoever is acting, logged in or not.
type Visitor struct {
	UserID    *uint
	SessionID string
	IP        string
	UserAgent string
	Referrer  string
}

// EngagementService owns leads, favourites and the engagement event stream.
type EngagementService struct {
	engage   repositories.EngagementRepository
	props    repositories.PropertyRepository
	accounts repositories.AccountRepository
	users    repositories.UserRepository
	media    repositories.MediaRepository
	notifier *notification.Dispatcher
	cfg      *config.Config
}

func NewEngagementService(
	engage repositories.EngagementRepository,
	props repositories.PropertyRepository,
	accounts repositories.AccountRepository,
	users repositories.UserRepository,
	media repositories.MediaRepository,
	notifier *notification.Dispatcher,
	cfg *config.Config,
) *EngagementService {
	return &EngagementService{engage: engage, props: props, accounts: accounts,
		users: users, media: media, notifier: notifier, cfg: cfg}
}

// CreateInquiry stores a lead against a published property and notifies the
// builder. Anonymous visitors may submit; consent is mandatory.
func (s *EngagementService) CreateInquiry(ctx context.Context, req dto.CreateInquiryRequest, v Visitor) (*dto.InquiryResponse, error) {
	if !req.ConsentContact {
		return nil, Invalid("Please confirm that we may contact you about this enquiry")
	}
	p, err := s.props.GetByID(ctx, req.PropertyID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}
	if !p.IsPublic() {
		return nil, NotFound("Property not found")
	}
	// A configuration, if given, must belong to this property.
	if req.ConfigurationID != nil {
		found := false
		for _, c := range p.Configurations {
			if c.ID == *req.ConfigurationID {
				found = true
				break
			}
		}
		if !found {
			return nil, Invalid("The selected configuration is not part of this property")
		}
	}

	source := req.Source
	if source == "" {
		source = "WEB"
	}
	in := &models.Inquiry{
		PropertyID: p.ID, ClientID: p.ClientID, UserID: v.UserID,
		Name: strings.TrimSpace(req.Name), Mobile: validators.NormalizeMobile(req.Mobile),
		Email: strings.ToLower(strings.TrimSpace(req.Email)), Message: req.Message,
		ConfigurationID: req.ConfigurationID, Budget: req.Budget,
		PreferredTime: req.PreferredTime, SessionID: v.SessionID,
		Status: "NEW", Source: source, IPAddress: v.IP,
		// The lead type is derived from the property, never from the caller: a
		// visitor cannot mislabel a rental enquiry as a purchase one, and lead
		// reporting never has to join back to find out what it was about.
		LeadType: leadTypeFor(p.Purpose),
	}
	if err := s.engage.CreateInquiry(ctx, in); err != nil {
		return nil, Internal("create inquiry", err)
	}

	s.recordEvent(ctx, models.EventInquiry, &p.ID, v, "")
	s.notifyBuilderOfLead(ctx, p, in)

	full, err := s.engage.GetInquiry(ctx, in.ID)
	if err != nil {
		return nil, Internal("load inquiry", err)
	}
	resp := ToInquiryResponse(full)
	return &resp, nil
}

// notifyBuilderOfLead sends an in-app notification and an email to the builder.
// Delivery problems are logged, never surfaced to the visitor.
func (s *EngagementService) notifyBuilderOfLead(ctx context.Context, p *models.Property, in *models.Inquiry) {
	client, err := s.accounts.GetClient(ctx, p.ClientID)
	if err != nil {
		logger.Log.Warn("could not load client for lead notification", "client_id", p.ClientID, "error", err)
		return
	}
	title := "New enquiry for " + p.Name
	body := fmt.Sprintf("%s (%s) enquired about %s.", in.Name, in.Mobile, p.Name)

	if err := s.accounts.CreateNotification(ctx, &models.Notification{
		UserID: client.UserID, Type: models.NotificationTypeInApp,
		Title: title, Message: body, Link: "/client/leads",
	}); err != nil {
		logger.Log.Warn("could not create lead notification", "error", err)
	}

	pref, err := s.accounts.GetNotificationPreference(ctx, client.UserID)
	if err == nil && !pref.EmailEnabled {
		return
	}
	to := client.ContactEmail
	if to == "" && client.User.ID != 0 {
		to = client.User.Email
	}
	if to == "" {
		return
	}
	if err := s.notifier.Send(ctx, notification.ChannelEmail, notification.Message{
		To: to, Subject: title,
		Body: body + "\n\nSign in to your builder dashboard to respond.",
	}); err != nil {
		logger.Log.Warn("lead email failed", "error", err)
	}
}

// ListInquiriesForClient returns leads for the builder's own properties only.
func (s *EngagementService) ListInquiriesForClient(ctx context.Context, clientID uint, q dto.InquiryListQuery) ([]dto.InquiryResponse, *dto.Meta, error) {
	return s.listInquiries(ctx, repositories.InquiryFilter{ClientID: clientID}, q)
}

// ListInquiriesForAdmin returns every lead in the system.
func (s *EngagementService) ListInquiriesForAdmin(ctx context.Context, q dto.InquiryListQuery) ([]dto.InquiryResponse, *dto.Meta, error) {
	return s.listInquiries(ctx, repositories.InquiryFilter{}, q)
}

// ListInquiriesForCustomer returns the enquiries a buyer submitted.
func (s *EngagementService) ListInquiriesForCustomer(ctx context.Context, userID uint, q dto.InquiryListQuery) ([]dto.InquiryResponse, *dto.Meta, error) {
	return s.listInquiries(ctx, repositories.InquiryFilter{UserID: userID}, q)
}

func (s *EngagementService) listInquiries(ctx context.Context, base repositories.InquiryFilter, q dto.InquiryListQuery) ([]dto.InquiryResponse, *dto.Meta, error) {
	page, perPage := normalizePage(q.Page, q.PerPage)
	base.PropertyID = q.PropertyID
	base.Status = q.Status
	base.Search = q.Search
	base.Offset = (page - 1) * perPage
	base.Limit = perPage
	if q.From != "" {
		if d, err := time.Parse("2006-01-02", q.From); err == nil {
			base.From = &d
		}
	}
	if q.To != "" {
		if d, err := time.Parse("2006-01-02", q.To); err == nil {
			end := d.Add(24*time.Hour - time.Second)
			base.To = &end
		}
	}
	rows, total, err := s.engage.ListInquiries(ctx, base)
	if err != nil {
		return nil, nil, Internal("load inquiries", err)
	}
	out := make([]dto.InquiryResponse, 0, len(rows))
	for i := range rows {
		out = append(out, ToInquiryResponse(&rows[i]))
	}
	return out, newMeta(page, perPage, total), nil
}

// UpdateInquiryStatus advances a lead. clientID is non-zero for builder calls,
// which restricts the update to their own leads.
func (s *EngagementService) UpdateInquiryStatus(ctx context.Context, id, clientID, actorID uint, req dto.UpdateInquiryStatusRequest) error {
	in, err := s.engage.GetInquiry(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Enquiry not found")
		}
		return Internal("load inquiry", err)
	}
	if clientID != 0 && in.ClientID != clientID {
		return NotFound("Enquiry not found")
	}
	if err := s.engage.UpdateInquiryStatus(ctx, id, req.Status, req.Remarks, actorID); err != nil {
		return Internal("update inquiry", err)
	}
	return nil
}

// GetInquiryForClient loads one lead, scoped to its owning builder.
func (s *EngagementService) GetInquiryForClient(ctx context.Context, id, clientID uint) (*dto.InquiryResponse, error) {
	in, err := s.engage.GetInquiry(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Enquiry not found")
		}
		return nil, Internal("load inquiry", err)
	}
	if clientID != 0 && in.ClientID != clientID {
		return nil, NotFound("Enquiry not found")
	}
	resp := ToInquiryResponse(in)
	return &resp, nil
}

// TrackEvent records a browser-reported engagement event.
func (s *EngagementService) TrackEvent(ctx context.Context, req dto.TrackEventRequest, v Visitor) error {
	eventType := models.EventType(strings.ToUpper(req.EventType))
	if req.PropertyID != nil {
		p, err := s.props.GetByID(ctx, *req.PropertyID)
		if err != nil || !p.IsPublic() {
			return NotFound("Property not found")
		}
	}
	// A view goes through RecordView so it is de-duplicated and counted.
	if eventType == models.EventView && req.PropertyID != nil {
		return s.RecordView(ctx, *req.PropertyID, v)
	}
	s.recordEvent(ctx, eventType, req.PropertyID, v, req.Metadata)

	// Mirror high-value events onto the property counters.
	if req.PropertyID != nil && eventType == models.EventWhatsappClick {
		if err := s.props.IncrementCounter(ctx, *req.PropertyID, "whatsapp_click_count", 1); err != nil {
			logger.Log.Warn("could not increment whatsapp counter", "error", err)
		}
	}
	return nil
}

// RecordView stores a property view, de-duplicated per visitor and window.
func (s *EngagementService) RecordView(ctx context.Context, propertyID uint, v Visitor) error {
	seen, err := s.engage.ViewedRecently(ctx, propertyID, v.UserID, v.SessionID, viewDedupWindow)
	if err != nil {
		return Internal("check recent view", err)
	}
	if err := s.engage.RecordView(ctx, &models.PropertyView{
		PropertyID: propertyID, UserID: v.UserID, SessionID: v.SessionID,
		IPAddress: v.IP, UserAgent: v.UserAgent, Referrer: v.Referrer,
	}); err != nil {
		return Internal("record view", err)
	}
	if seen {
		return nil // history is kept, but the headline counter is not inflated
	}
	s.recordEvent(ctx, models.EventView, &propertyID, v, "")
	if err := s.props.IncrementCounter(ctx, propertyID, "view_count", 1); err != nil {
		logger.Log.Warn("could not increment view counter", "error", err)
	}
	return nil
}

// recordEvent writes to the generic event stream. Failures are logged only:
// analytics must never break a user-facing action.
func (s *EngagementService) recordEvent(ctx context.Context, t models.EventType, propertyID *uint, v Visitor, metadata string) {
	if err := s.engage.RecordEvent(ctx, &models.UserPropertyEvent{
		UserID: v.UserID, PropertyID: propertyID, EventType: t,
		SessionID: v.SessionID, IPAddress: v.IP, UserAgent: v.UserAgent,
		Metadata: metadata,
	}); err != nil {
		logger.Log.Warn("could not record event", "type", t, "error", err)
	}
}

// AddFavorite saves a property for a buyer.
func (s *EngagementService) AddFavorite(ctx context.Context, userID, propertyID uint, v Visitor) error {
	p, err := s.props.GetByID(ctx, propertyID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Property not found")
		}
		return Internal("load property", err)
	}
	if !p.IsPublic() {
		return NotFound("Property not found")
	}
	if _, err := s.engage.AddFavorite(ctx, userID, propertyID); err != nil {
		return Internal("save favourite", err)
	}
	s.recordEvent(ctx, models.EventFavorite, &propertyID, v, "")
	return nil
}

// RemoveFavorite removes a saved property.
func (s *EngagementService) RemoveFavorite(ctx context.Context, userID, propertyID uint, v Visitor) error {
	if _, err := s.engage.RemoveFavorite(ctx, userID, propertyID); err != nil {
		return Internal("remove favourite", err)
	}
	s.recordEvent(ctx, models.EventUnfavorite, &propertyID, v, "")
	return nil
}

// ListFavorites returns a buyer's saved properties.
func (s *EngagementService) ListFavorites(ctx context.Context, userID uint, page, perPage int) ([]dto.PropertyCard, *dto.Meta, error) {
	page, perPage = normalizePage(page, perPage)
	props, total, err := s.engage.ListFavorites(ctx, userID, (page-1)*perPage, perPage)
	if err != nil {
		return nil, nil, Internal("load favourites", err)
	}
	cards := ToPropertyCards(props)
	for i := range cards {
		cards[i].IsFavorite = true
	}
	return cards, newMeta(page, perPage, total), nil
}

// RecentlyViewed returns the visitor's browsing history, for logged-in users and
// anonymous sessions alike.
func (s *EngagementService) RecentlyViewed(ctx context.Context, v Visitor, limit int) ([]dto.PropertyCard, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	props, err := s.engage.RecentlyViewed(ctx, v.UserID, v.SessionID, limit)
	if err != nil {
		return nil, Internal("load recently viewed", err)
	}
	return ToPropertyCards(props), nil
}

// WhatsAppLink builds the prefilled deep link for a property and records the
// click, so WhatsApp leads are tracked separately from form enquiries.
func (s *EngagementService) WhatsAppLink(ctx context.Context, propertyID uint, configuration string, v Visitor) (*dto.WhatsAppLinkResponse, error) {
	p, err := s.props.GetByID(ctx, propertyID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Property not found")
		}
		return nil, Internal("load property", err)
	}
	if !p.IsPublic() {
		return nil, NotFound("Property not found")
	}

	// Prefer the builder's own number; fall back to the platform desk.
	phone := s.cfg.WhatsApp.DefaultNumber
	if client, err := s.accounts.GetClient(ctx, p.ClientID); err == nil && client.ContactPhone != "" {
		phone = client.ContactPhone
	}
	if phone == "" {
		return nil, NotFound("No contact number is available for this property")
	}
	phone = strings.TrimPrefix(validators.NormalizeMobile(phone), "+")
	if len(phone) == 10 {
		phone = s.cfg.WhatsApp.CountryCode + phone
	}

	location := p.Locality.Name
	if p.City.Name != "" {
		location = strings.TrimPrefix(location+", "+p.City.Name, ", ")
	}
	// A renter and a buyer are asking different questions, so the prefilled
	// message says the right thing for each rather than one generic line.
	isRental := p.Purpose == models.PurposeRent
	opening := fmt.Sprintf("Hi, I am interested in %s.", p.Name)
	if isRental {
		opening = fmt.Sprintf("Hi, I am interested in renting %s.", p.Name)
	}

	lines := []string{opening, "", "Property: " + p.Name}
	if configuration != "" {
		lines = append(lines, "Configuration: "+configuration)
	}
	lines = append(lines, "Location: "+location)

	if isRental && p.RentalDetails != nil {
		lines = append(lines, "Monthly rent: "+formatRupees(p.RentalDetails.MonthlyRent))
		if p.RentalDetails.FurnishingType != "" {
			lines = append(lines, "Furnishing: "+humanizeEnum(string(p.RentalDetails.FurnishingType)))
		}
	}

	closing := "Please share more information."
	if isRental {
		closing = "Please share more details and let me know when I could visit."
	}
	lines = append(lines,
		"Link: "+s.cfg.App.FrontendURL+"/properties/"+p.Slug,
		"",
		closing)
	message := strings.Join(lines, "\n")

	s.recordEvent(ctx, models.EventWhatsappClick, &p.ID, v, "")
	if err := s.props.IncrementCounter(ctx, p.ID, "whatsapp_click_count", 1); err != nil {
		logger.Log.Warn("could not increment whatsapp counter", "error", err)
	}

	return &dto.WhatsAppLinkResponse{
		URL:     fmt.Sprintf("https://wa.me/%s?text=%s", phone, url.QueryEscape(message)),
		Message: message,
		Phone:   phone,
	}, nil
}

// DownloadDocument records a brochure download and returns the file location.
func (s *EngagementService) DownloadDocument(ctx context.Context, propertyID, documentID uint, v Visitor) (*models.PropertyDocument, error) {
	doc, err := s.media.GetDocument(ctx, documentID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Document not found")
		}
		return nil, Internal("load document", err)
	}
	if doc.PropertyID != propertyID {
		return nil, NotFound("Document not found")
	}
	p, err := s.props.GetByID(ctx, propertyID)
	if err != nil || !p.IsPublic() {
		return nil, NotFound("Property not found")
	}
	if err := s.media.IncrementDownloads(ctx, doc.ID); err != nil {
		logger.Log.Warn("could not increment download count", "error", err)
	}
	s.recordEvent(ctx, models.EventBrochureDownload, &propertyID, v, doc.Name)
	return doc, nil
}

// leadTypeFor maps a property's purpose onto the lead vocabulary, so a rental
// enquiry is reported as a rental lead without the caller saying so.
func leadTypeFor(purpose models.PropertyPurpose) models.LeadType {
	if purpose == models.PurposeRent {
		return models.LeadRent
	}
	return models.LeadBuy
}

// formatRupees renders an amount the way an Indian message would read it:
// thousands for a rent, lakhs and crores for a price.
func formatRupees(amount float64) string {
	switch {
	case amount <= 0:
		return "On request"
	case amount >= 10000000:
		return fmt.Sprintf("₹%.2f Cr", amount/10000000)
	case amount >= 100000:
		return fmt.Sprintf("₹%.2f L", amount/100000)
	default:
		return "₹" + addThousandSeparators(int64(amount))
	}
}

// addThousandSeparators groups digits the Indian way: the last three, then
// pairs. 2800000 reads as 28,00,000 rather than 2,800,000.
func addThousandSeparators(n int64) string {
	if n < 0 {
		return "-" + addThousandSeparators(-n)
	}
	digits := strconv.FormatInt(n, 10)
	if len(digits) <= 3 {
		return digits
	}
	head, tail := digits[:len(digits)-3], digits[len(digits)-3:]

	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return strings.Join(parts, ",") + "," + tail
}

// humanizeEnum turns SEMI_FURNISHED into "Semi Furnished" for message text.
func humanizeEnum(value string) string {
	words := strings.Split(strings.ToLower(value), "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
