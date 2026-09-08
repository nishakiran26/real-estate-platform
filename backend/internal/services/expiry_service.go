package services

import (
	"context"
	"fmt"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/pkg/logger"
)

// expiryWarningDays is how far ahead an owner is warned. Long enough to act on,
// short enough that the warning still means something when it arrives.
const expiryWarningDays = 3

// ExpiryService retires rentals that have gone stale. Rental stock turns over in
// weeks, and a listing nobody has confirmed in two months is the single fastest
// way for a rental site to lose a renter's trust — they call, the flat is gone,
// and they stop believing the rest of the results too.
//
// Retiring is deliberately reversible and never destructive: the listing keeps
// its approval state, its leads and its analytics, and one click on Renew brings
// it straight back. See PropertyService.RenewListing.
type ExpiryService struct {
	props    repositories.PropertyRepository
	accounts repositories.AccountRepository
}

func NewExpiryService(props repositories.PropertyRepository, accounts repositories.AccountRepository) *ExpiryService {
	return &ExpiryService{props: props, accounts: accounts}
}

// SweepResult reports what a run did, so an administrator triggering it sees an
// outcome rather than a silent success.
type SweepResult struct {
	Warned  int `json:"warned"`
	Expired int `json:"expired"`
}

// Sweep warns owners whose listings lapse shortly and retires the ones that
// already have. It is safe to run repeatedly: a listing already INACTIVE is no
// longer returned as expiring, so it is neither retired nor warned about twice.
func (s *ExpiryService) Sweep(ctx context.Context) (*SweepResult, error) {
	now := time.Now()
	out := &SweepResult{}

	// Already lapsed: take them off the market.
	expired, err := s.props.ExpiringRentals(ctx, time.Time{}, now)
	if err != nil {
		return nil, Internal("load expired listings", err)
	}
	for i := range expired {
		p := &expired[i]
		if err := s.props.SetAvailability(ctx, p.ID, models.AvailabilityInactive); err != nil {
			logger.Log.Error("could not retire expired listing", "property_id", p.ID, "error", err)
			continue
		}
		out.Expired++
		s.notifyOwner(ctx, p,
			"Your listing has expired: "+p.Name,
			fmt.Sprintf("%s is no longer shown to renters. If it is still available, "+
				"renew it from your listings and it goes straight back up.", p.Name))
	}

	// Lapsing shortly: warn while there is still time to renew.
	soon, err := s.props.ExpiringRentals(ctx, now, now.AddDate(0, 0, expiryWarningDays))
	if err != nil {
		return nil, Internal("load expiring listings", err)
	}
	for i := range soon {
		p := &soon[i]
		days := 0
		if p.ListingExpiryDate != nil {
			days = daysUntil(*p.ListingExpiryDate)
		}
		out.Warned++
		s.notifyOwner(ctx, p,
			"Your listing expires soon: "+p.Name,
			fmt.Sprintf("%s stops being shown to renters in %d day(s). "+
				"Renew it if the flat is still available.", p.Name, days))
	}

	return out, nil
}

// notifyOwner is best effort. A listing that was retired but not announced is
// still correctly retired, so a notification failure must not abort the sweep
// and leave the rest of the stale listings live.
func (s *ExpiryService) notifyOwner(ctx context.Context, p *models.Property, title, body string) {
	client, err := s.accounts.GetClient(ctx, p.ClientID)
	if err != nil {
		logger.Log.Warn("could not load account for expiry notice",
			"client_id", p.ClientID, "error", err)
		return
	}
	if err := s.accounts.CreateNotification(ctx, &models.Notification{
		UserID: client.UserID, Type: models.NotificationTypeInApp,
		Title: title, Message: body, Link: "/client/properties",
	}); err != nil {
		logger.Log.Warn("could not create expiry notice", "property_id", p.ID, "error", err)
	}
}
