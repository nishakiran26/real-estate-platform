package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/notification"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/pkg/logger"
)

// maxAlertsPerUser stops a saved search becoming a way to have every new
// listing emailed. Someone with fifty alerts is subscribing to the catalogue,
// not watching for a flat.
const maxAlertsPerUser = 20

// AlertService owns saved rental searches and the notices they produce.
//
// The criteria are stored as the search query string rather than as a private
// format, so reopening an alert reruns exactly the search that created it. The
// duplicated columns beside it exist only so a newly published rental can be
// matched with one indexed query.
type AlertService struct {
	alerts   repositories.AlertRepository
	accounts repositories.AccountRepository
	geo      repositories.GeoRepository
	notifier *notification.Dispatcher
}

func NewAlertService(
	alerts repositories.AlertRepository,
	accounts repositories.AccountRepository,
	geo repositories.GeoRepository,
	notifier *notification.Dispatcher,
) *AlertService {
	return &AlertService{alerts: alerts, accounts: accounts, geo: geo, notifier: notifier}
}

// Create saves a search for the signed-in customer.
func (s *AlertService) Create(ctx context.Context, userID uint, req dto.CreateRentalAlertRequest) (*dto.RentalAlertResponse, error) {
	if req.MaxRent > 0 && req.MinRent > req.MaxRent {
		return nil, Invalid("The minimum rent cannot be more than the maximum")
	}

	count, err := s.alerts.CountForUser(ctx, userID)
	if err != nil {
		return nil, Internal("count alerts", err)
	}
	if count >= maxAlertsPerUser {
		return nil, Invalid(fmt.Sprintf(
			"You can keep up to %d saved searches. Delete one to add another.", maxAlertsPerUser))
	}

	criteria, err := json.Marshal(req)
	if err != nil {
		return nil, Internal("encode alert criteria", err)
	}

	alert := &models.RentalAlert{
		UserID:     userID,
		Name:       s.alertName(ctx, req),
		Criteria:   string(criteria),
		CityID:     req.CityID,
		LocalityID: req.LocalityID,
		BHK:        req.BHK,
		MinRent:    req.MinRent,
		MaxRent:    req.MaxRent,
		// Instant is the only delivery this platform implements: there is no
		// scheduler to build a digest on.
		Frequency: models.AlertInstant,
		// Set explicitly rather than left to the column default: GORM omits a
		// false bool from the INSERT and the default would win, which is how a
		// deliberate value gets lost.
		IsActive: true,
	}
	if err := s.alerts.Create(ctx, alert); err != nil {
		return nil, Internal("save alert", err)
	}

	resp := s.toResponse(alert)
	return &resp, nil
}

// List returns the customer's saved searches.
func (s *AlertService) List(ctx context.Context, userID uint) ([]dto.RentalAlertResponse, error) {
	alerts, err := s.alerts.ListForUser(ctx, userID)
	if err != nil {
		return nil, Internal("load alerts", err)
	}
	out := make([]dto.RentalAlertResponse, 0, len(alerts))
	for i := range alerts {
		out = append(out, s.toResponse(&alerts[i]))
	}
	return out, nil
}

// Delete removes a saved search the customer owns.
func (s *AlertService) Delete(ctx context.Context, id, userID uint) error {
	if err := s.alerts.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Saved search not found")
		}
		return Internal("delete alert", err)
	}
	return nil
}

// SetActive pauses or resumes a saved search. Pausing is offered because the
// alternative — deleting it — loses a search the renter spent time refining.
func (s *AlertService) SetActive(ctx context.Context, id, userID uint, active bool) error {
	if err := s.alerts.SetActive(ctx, id, userID, active); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Saved search not found")
		}
		return Internal("update alert", err)
	}
	return nil
}

// NotifyForProperty tells the matching renters that a flat has been published.
//
// It is best-effort and never fails the publish: an alert that could not be
// delivered must not stop a listing going live. Failures are logged rather than
// swallowed, because a silent notification path is how a feature comes to look
// like it works while doing nothing.
func (s *AlertService) NotifyForProperty(ctx context.Context, p *models.Property) {
	if p == nil || p.Purpose != models.PurposeRent || p.RentalDetails == nil {
		return
	}

	match := repositories.AlertMatch{
		CityID:     p.CityID,
		LocalityID: p.LocalityID,
		Bedrooms:   bedroomsOf(p),
		Rent:       p.RentalDetails.MonthlyRent,
	}
	alerts, err := s.alerts.Matching(ctx, match)
	if err != nil {
		logger.Log.Warn("could not match rental alerts", "property_id", p.ID, "error", err)
		return
	}
	if len(alerts) == 0 {
		return
	}

	title := "A flat matching your saved search is available"
	body := fmt.Sprintf("%s in %s is now listed at %s a month.",
		p.Name, localityLabel(p), formatRupees(p.RentalDetails.MonthlyRent))
	link := "/property/" + p.Slug

	for i := range alerts {
		alert := &alerts[i]
		if err := s.accounts.CreateNotification(ctx, &models.Notification{
			UserID: alert.UserID, Type: models.NotificationTypeInApp,
			Title: title, Message: body, Link: link,
		}); err != nil {
			logger.Log.Warn("could not create alert notification",
				"alert_id", alert.ID, "error", err)
			continue
		}
		if err := s.alerts.RecordMatch(ctx, alert.ID); err != nil {
			logger.Log.Warn("could not record alert match", "alert_id", alert.ID, "error", err)
		}
	}
}

// alertName gives an unnamed search a label a renter will recognise in a list,
// built from what they actually searched for.
func (s *AlertService) alertName(ctx context.Context, req dto.CreateRentalAlertRequest) string {
	if name := strings.TrimSpace(req.Name); name != "" {
		return name
	}

	parts := make([]string, 0, 3)
	if req.BHK > 0 {
		parts = append(parts, fmt.Sprintf("%d BHK", req.BHK))
	} else {
		parts = append(parts, "Rentals")
	}
	if req.LocalityID > 0 {
		if locality, err := s.geo.GetLocality(ctx, req.LocalityID); err == nil {
			parts = append(parts, "in "+locality.Name)
		}
	} else if req.CityID > 0 {
		if city, err := s.geo.GetCity(ctx, req.CityID); err == nil {
			parts = append(parts, "in "+city.Name)
		}
	}
	if req.MaxRent > 0 {
		parts = append(parts, "under "+formatRupees(req.MaxRent))
	}
	return strings.Join(parts, " ")
}

// toResponse renders an alert, rebuilding the query string that reruns it.
func (s *AlertService) toResponse(a *models.RentalAlert) dto.RentalAlertResponse {
	resp := dto.RentalAlertResponse{
		ID: a.ID, Name: a.Name,
		CityID: a.CityID, LocalityID: a.LocalityID, BHK: a.BHK,
		MinRent: a.MinRent, MaxRent: a.MaxRent,
		Frequency: string(a.Frequency), IsActive: a.IsActive,
		MatchCount:  a.MatchCount,
		SearchQuery: alertSearchQuery(a),
		CreatedAt:   formatTime(a.CreatedAt),
	}
	if d := formatDate(a.LastNotifiedAt); d != "" {
		resp.LastNotifiedAt = d
	}
	return resp
}

// alertSearchQuery rebuilds the rent search this alert was saved from, so
// "view matches" lands on the real results rather than an approximation.
func alertSearchQuery(a *models.RentalAlert) string {
	q := url.Values{}
	q.Set("purpose", "rent")
	if a.CityID > 0 {
		q.Set("city_id", strconv.FormatUint(uint64(a.CityID), 10))
	}
	if a.LocalityID > 0 {
		q.Set("locality_id", strconv.FormatUint(uint64(a.LocalityID), 10))
	}
	if a.BHK > 0 {
		q.Set("bhk", strconv.Itoa(a.BHK))
	}
	if a.MinRent > 0 {
		q.Set("min_rent", strconv.FormatFloat(a.MinRent, 'f', -1, 64))
	}
	if a.MaxRent > 0 {
		q.Set("max_rent", strconv.FormatFloat(a.MaxRent, 'f', -1, 64))
	}

	// Furnishing and tenant preference are not matched on, but they are part of
	// the search the renter saved, so the link must carry them back.
	var criteria dto.CreateRentalAlertRequest
	if err := json.Unmarshal([]byte(a.Criteria), &criteria); err == nil {
		if criteria.Furnishing != "" {
			q.Set("furnishing", criteria.Furnishing)
		}
		if criteria.TenantPreference != "" {
			q.Set("tenant_preference", criteria.TenantPreference)
		}
	}
	return q.Encode()
}

// bedroomsOf reports the BHK a listing offers, taking the smallest
// configuration so a "2 BHK" alert still matches a building offering 2 and 3.
func bedroomsOf(p *models.Property) int {
	smallest := 0
	for _, c := range p.Configurations {
		if c.Bedrooms <= 0 {
			continue
		}
		if smallest == 0 || c.Bedrooms < smallest {
			smallest = c.Bedrooms
		}
	}
	return smallest
}

// localityLabel names where a property is, falling back sensibly when the
// association was not preloaded.
func localityLabel(p *models.Property) string {
	switch {
	case p.Locality.Name != "" && p.City.Name != "":
		return p.Locality.Name + ", " + p.City.Name
	case p.Locality.Name != "":
		return p.Locality.Name
	case p.City.Name != "":
		return p.City.Name
	default:
		return "your search area"
	}
}
