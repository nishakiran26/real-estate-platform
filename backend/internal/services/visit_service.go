package services

import (
	"context"
	"errors"
	"fmt"
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

// duplicateVisitWindow collapses repeat requests from the same number for the
// same property. Someone tapping twice has asked once.
const duplicateVisitWindow = 6 * time.Hour

// VisitService owns property viewing requests. Renting rarely converts without
// a viewing, so this is a first-class flow rather than a note on a lead.
type VisitService struct {
	visits   repositories.VisitRepository
	props    repositories.PropertyRepository
	accounts repositories.AccountRepository
	engage   repositories.EngagementRepository
	notifier *notification.Dispatcher
	cfg      *config.Config
}

func NewVisitService(
	visits repositories.VisitRepository,
	props repositories.PropertyRepository,
	accounts repositories.AccountRepository,
	engage repositories.EngagementRepository,
	notifier *notification.Dispatcher,
	cfg *config.Config,
) *VisitService {
	return &VisitService{visits: visits, props: props, accounts: accounts,
		engage: engage, notifier: notifier, cfg: cfg}
}

// Request records a viewing request against a published property.
func (s *VisitService) Request(ctx context.Context, req dto.CreateVisitRequest, v Visitor) (*dto.VisitResponse, error) {
	if !req.ConsentContact {
		return nil, Invalid("Please confirm that we may contact you about this visit")
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
	// A flat already let cannot be viewed.
	if p.AvailabilityStatus == models.AvailabilityRented ||
		p.AvailabilityStatus == models.AvailabilityInactive {
		return nil, Conflict("This property is no longer available to view")
	}

	visitDate, err := time.Parse("2006-01-02", req.VisitDate)
	if err != nil {
		return nil, Invalid("Enter a valid visit date")
	}
	// Compare against the start of today so a request for later today is valid.
	today := time.Now().Truncate(24 * time.Hour)
	if visitDate.Before(today) {
		return nil, Invalid("Choose a visit date that has not already passed")
	}
	if visitDate.After(time.Now().AddDate(0, 3, 0)) {
		return nil, Invalid("Visits can be requested up to three months ahead")
	}

	mobile := validators.NormalizeMobile(req.Mobile)
	recent, err := s.visits.RecentForProperty(ctx, p.ID, mobile, duplicateVisitWindow)
	if err == nil && recent > 0 {
		return nil, Conflict("You have already requested a visit for this property. The owner will be in touch.")
	}

	slot := req.TimeSlot
	if slot == "" {
		slot = "ANY"
	}

	visit := &models.PropertyVisit{
		PropertyID: p.ID, ClientID: p.ClientID, UserID: v.UserID,
		Name:        strings.TrimSpace(req.Name),
		Mobile:      mobile,
		Email:       strings.ToLower(strings.TrimSpace(req.Email)),
		Message:     req.Message,
		SessionID:   v.SessionID,
		RequestedAt: visitDate,
		TimeSlot:    slot,
		Status:      models.VisitRequested,
	}
	if err := s.visits.Create(ctx, visit); err != nil {
		return nil, Internal("create visit request", err)
	}

	s.recordVisitEvent(ctx, p.ID, v)
	s.notifyOwner(ctx, p, visit)

	resp := toVisitResponse(visit, true)
	return &resp, nil
}

// ListForClient returns viewing requests on the caller's own properties.
func (s *VisitService) ListForClient(ctx context.Context, clientID uint, q dto.VisitListQuery) ([]dto.VisitResponse, *dto.Meta, error) {
	return s.list(ctx, repositories.VisitFilter{ClientID: clientID}, q, true)
}

// ListForAdmin returns every viewing request.
func (s *VisitService) ListForAdmin(ctx context.Context, q dto.VisitListQuery) ([]dto.VisitResponse, *dto.Meta, error) {
	return s.list(ctx, repositories.VisitFilter{}, q, true)
}

// ListForCustomer returns the visits this customer asked for.
func (s *VisitService) ListForCustomer(ctx context.Context, userID uint, q dto.VisitListQuery) ([]dto.VisitResponse, *dto.Meta, error) {
	return s.list(ctx, repositories.VisitFilter{UserID: userID}, q, true)
}

func (s *VisitService) list(ctx context.Context, base repositories.VisitFilter, q dto.VisitListQuery, withContact bool) ([]dto.VisitResponse, *dto.Meta, error) {
	page, perPage := normalizePage(q.Page, q.PerPage)
	base.Status = q.Status
	base.PropertyID = q.PropertyID
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

	rows, total, err := s.visits.List(ctx, base)
	if err != nil {
		return nil, nil, Internal("load visits", err)
	}
	out := make([]dto.VisitResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toVisitResponse(&rows[i], withContact))
	}
	return out, newMeta(page, perPage, total), nil
}

// Respond confirms, reschedules, completes or cancels a request. clientID is
// non-zero for owner and builder calls, which restricts it to their own
// properties; administrators pass zero.
func (s *VisitService) Respond(ctx context.Context, id, clientID, actorID uint, req dto.UpdateVisitRequest) (*dto.VisitResponse, error) {
	visit, err := s.visits.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Visit request not found")
		}
		return nil, Internal("load visit", err)
	}
	// Deliberately "not found": do not confirm that another account's visit
	// request exists.
	if clientID != 0 && visit.ClientID != clientID {
		return nil, NotFound("Visit request not found")
	}

	status := models.VisitStatus(req.Status)
	if visit.Status == models.VisitCompleted || visit.Status == models.VisitCancelled {
		return nil, Conflict("This visit has already been closed")
	}

	// Confirming or rescheduling means committing to a time.
	if status == models.VisitConfirmed || status == models.VisitRescheduled {
		if req.ScheduledDate == "" {
			return nil, Invalid("Give the date you are confirming for")
		}
		scheduled, err := time.Parse("2006-01-02", req.ScheduledDate)
		if err != nil {
			return nil, Invalid("Enter a valid date")
		}
		visit.ScheduledAt = &scheduled
		if req.TimeSlot != "" {
			visit.TimeSlot = req.TimeSlot
		}
	}
	if status == models.VisitCompleted {
		now := time.Now()
		visit.CompletedAt = &now
	}

	now := time.Now()
	visit.Status = status
	visit.Remarks = req.Remarks
	visit.HandledBy = &actorID
	visit.HandledAt = &now

	if err := s.visits.Update(ctx, visit); err != nil {
		return nil, Internal("update visit", err)
	}

	s.notifyVisitor(ctx, visit)

	resp := toVisitResponse(visit, true)
	return &resp, nil
}

// recordVisitEvent adds the request to the engagement stream so it shows up in
// owner analytics beside views and WhatsApp clicks.
func (s *VisitService) recordVisitEvent(ctx context.Context, propertyID uint, v Visitor) {
	event := &models.UserPropertyEvent{
		UserID: v.UserID, PropertyID: &propertyID,
		EventType: models.EventVisitRequest,
		SessionID: v.SessionID, IPAddress: v.IP, UserAgent: v.UserAgent,
	}
	if err := s.engage.RecordEvent(ctx, event); err != nil {
		logger.Log.Warn("could not record visit event", "property_id", propertyID, "error", err)
	}
}

// notifyOwner tells the listing account a viewing has been requested.
func (s *VisitService) notifyOwner(ctx context.Context, p *models.Property, visit *models.PropertyVisit) {
	client, err := s.accounts.GetClient(ctx, p.ClientID)
	if err != nil {
		logger.Log.Warn("could not load client for visit notification",
			"client_id", p.ClientID, "error", err)
		return
	}

	title := "Visit requested for " + p.Name
	body := fmt.Sprintf("%s would like to view %s on %s (%s).",
		visit.Name, p.Name, visit.RequestedAt.Format("2 Jan 2006"), humanizeEnum(visit.TimeSlot))

	if err := s.accounts.CreateNotification(ctx, &models.Notification{
		UserID: client.UserID, Type: models.NotificationTypeInApp,
		Title: title, Message: body, Link: "/client/visits",
	}); err != nil {
		logger.Log.Warn("could not create visit notification", "error", err)
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
		Body: body + "\n\nSign in to confirm or propose another time.",
	}); err != nil {
		logger.Log.Warn("visit email failed", "error", err)
	}
}

// notifyVisitor tells a signed-in visitor what the owner decided. An anonymous
// visitor left only a phone number, so they are contacted directly instead.
func (s *VisitService) notifyVisitor(ctx context.Context, visit *models.PropertyVisit) {
	if visit.UserID == nil {
		return
	}
	when := ""
	if visit.ScheduledAt != nil {
		when = " on " + visit.ScheduledAt.Format("2 Jan 2006")
	}

	var title, body string
	switch visit.Status {
	case models.VisitConfirmed:
		title = "Your visit is confirmed"
		body = "The owner confirmed your visit" + when + "."
	case models.VisitRescheduled:
		title = "Your visit was rescheduled"
		body = "The owner proposed a new time" + when + "."
	case models.VisitCancelled:
		title = "Your visit was cancelled"
		body = "The owner cancelled the visit."
	default:
		return
	}
	if visit.Remarks != "" {
		body += "\n\n" + visit.Remarks
	}

	if err := s.accounts.CreateNotification(ctx, &models.Notification{
		UserID: *visit.UserID, Type: models.NotificationTypeInApp,
		Title: title, Message: body, Link: "/account/visits",
	}); err != nil {
		logger.Log.Warn("could not notify visitor", "error", err)
	}
}

// toVisitResponse maps a visit. withContact is false wherever the payload could
// reach someone who is not the visitor, the listing owner or an administrator.
func toVisitResponse(v *models.PropertyVisit, withContact bool) dto.VisitResponse {
	resp := dto.VisitResponse{
		ID: v.ID, PropertyID: v.PropertyID,
		Name:        v.Name,
		Message:     v.Message,
		RequestedAt: formatDate(&v.RequestedAt),
		ScheduledAt: formatDate(v.ScheduledAt),
		TimeSlot:    v.TimeSlot,
		Status:      string(v.Status),
		Remarks:     v.Remarks,
		CreatedAt:   formatTime(v.CreatedAt),
	}
	if withContact {
		resp.Mobile = v.Mobile
		resp.Email = v.Email
	}
	if v.Property.ID != 0 {
		resp.Property = &dto.PropertySummary{
			ID: v.Property.ID, Name: v.Property.Name,
			Slug: v.Property.Slug, MinPrice: v.Property.MinPrice,
		}
	}
	return resp
}
