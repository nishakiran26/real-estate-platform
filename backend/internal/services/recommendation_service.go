package services

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/repositories"
)

// RecommendationService produces personalized property suggestions based on
// user behavior, preferences, and collaborative signals.
type RecommendationService struct {
	engage   repositories.EngagementRepository
	accounts repositories.AccountRepository
	props    repositories.PropertyRepository
}

func NewRecommendationService(
	engage repositories.EngagementRepository,
	accounts repositories.AccountRepository,
	props repositories.PropertyRepository,
) *RecommendationService {
	return &RecommendationService{engage: engage, accounts: accounts, props: props}
}

// ForUser returns properties recommended to a specific user, based on their
// profile, recent views, favorites, and inquiries.
func (s *RecommendationService) ForUser(ctx context.Context, userID uint, limit int) ([]dto.PropertyCard, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}

	// Gather signals: preferences, favorites, recently viewed
	prefs, err := s.accounts.GetCustomerProfile(ctx, userID)
	if err != nil {
		// No profile yet, so no intent to respect: show what is popular across
		// both catalogues rather than guessing.
		return s.popular(ctx, "", limit)
	}

	favIDs, err := s.engage.FavoriteIDs(ctx, userID)
	if err != nil {
		favIDs = nil
	}
	viewedProps, err := s.engage.RecentlyViewed(ctx, &userID, "", 20)
	if err != nil {
		viewedProps = nil
	}
	viewedIDs := make([]uint, 0, len(viewedProps))
	for i := range viewedProps {
		viewedIDs = append(viewedIDs, viewedProps[i].ID)
	}

	exclude := make(map[uint]bool, len(favIDs)+len(viewedIDs))
	for _, id := range favIDs {
		exclude[id] = true
	}
	for _, id := range viewedIDs {
		exclude[id] = true
	}

	// Score and retrieve candidates: filter by preference, then rank by recency
	candidates, err := s.scoreByPreference(ctx, prefs, exclude, limit*3)
	if err != nil {
		// The intent is known even when scoring failed, so it is still honoured.
		return s.popular(ctx, intentPurpose(prefs), limit)
	}

	return s.rankAndTrim(ctx, candidates, viewedIDs, limit)
}

// ForProperty returns "similar" properties to a given one (used on detail page).
func (s *RecommendationService) ForProperty(ctx context.Context, propertyID uint, limit int) ([]dto.PropertyCard, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}

	src, err := s.props.GetByID(ctx, propertyID)
	if err != nil {
		return s.popular(ctx, "", limit)
	}

	exclude := map[uint]bool{propertyID: true}
	return s.similarByMeta(ctx, src, exclude, limit)
}

// popular is the fallback when personalisation has nothing to go on. It still
// takes a purpose, because "we do not know what you like yet" is no reason to
// show a renter a catalogue of flats for sale. An empty purpose means either.
func (s *RecommendationService) popular(ctx context.Context, purpose string, limit int) ([]dto.PropertyCard, error) {
	filter := repositories.PropertyFilter{
		Statuses:  []models.PropertyStatus{models.PropertyStatusPublished},
		Purpose:   purpose,
		SortBy:    "views",
		SortOrder: "DESC",
		Limit:     limit,
	}
	props, _, err := s.props.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return ToPropertyCards(props), nil
}

// scoreByPreference finds published properties matching user budget/type/BHK
// excluding already-seen IDs, then ranks them by view count + recency.
func (s *RecommendationService) scoreByPreference(ctx context.Context, p *models.CustomerProfile, exclude map[uint]bool, limit int) ([]models.Property, error) {
	if p == nil {
		return nil, nil
	}

	filter := repositories.PropertyFilter{
		Statuses:     []models.PropertyStatus{models.PropertyStatusPublished},
		PropertyType: p.PropertyType,
		SortBy:       "views",
		SortOrder:    "DESC",
		Limit:        limit,
		Purpose:      intentPurpose(p),
	}
	if p.BHK > 0 {
		filter.BHK = p.BHK
	}

	// A renter's budget is a monthly rent, a buyer's is a sale price. They are
	// different columns, and applying the wrong one is not a near miss: a sale
	// listing has no rent, and a rental's sale price is zero, so either mistake
	// filters out everything the customer actually wants.
	switch filter.Purpose {
	case string(models.PurposeRent):
		filter.MinRent, filter.MaxRent = p.MonthlyRentMin, p.MonthlyRentMax
		if p.PreferredFurnishing != "" {
			filter.Furnishing = string(p.PreferredFurnishing)
		}
	case string(models.PurposeBuy):
		filter.MinPrice, filter.MaxPrice = p.MinBudget, p.MaxBudget
	default:
		// Interested in both. Neither budget can be applied across both
		// catalogues, so the recommendation falls back to type and BHK rather
		// than silently excluding one half.
	}

	props, _, err := s.props.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter out excluded in Go (repository could also do NOT IN, but
	// the repo interface does not expose ExcludeIDs).
	filtered := props[:0]
	for _, prop := range props {
		if !exclude[prop.ID] {
			filtered = append(filtered, prop)
		}
	}
	return filtered, nil
}

// similarByMeta uses city + property type + price band to find matches.
func (s *RecommendationService) similarByMeta(ctx context.Context, src *models.Property, exclude map[uint]bool, limit int) ([]dto.PropertyCard, error) {
	// "Similar" must stay in the same catalogue. A flat to let is not similar to
	// a flat for sale, however alike they look.
	purpose := string(src.Purpose)
	if src.CityID == 0 {
		return s.popular(ctx, purpose, limit)
	}

	filter := repositories.PropertyFilter{
		Statuses:     []models.PropertyStatus{models.PropertyStatusPublished},
		CityID:       src.CityID,
		PropertyType: string(src.PropertyType),
		Purpose:      purpose,
		SortBy:       "views",
		SortOrder:    "DESC",
		Limit:        limit * 2,
	}

	// The band is built from whichever figure this listing actually carries:
	// a rental's comparable is its rent, and its sale price is zero.
	if src.Purpose == models.PurposeRent {
		if src.RentalDetails != nil && src.RentalDetails.MonthlyRent > 0 {
			rent := src.RentalDetails.MonthlyRent
			filter.MinRent, filter.MaxRent = rent*0.6, rent*1.4
		}
	} else if midpoint := src.MinPrice + (src.MaxPrice-src.MinPrice)/2; midpoint > 0 {
		filter.MinPrice = math.Max(midpoint*0.5, 1)
		filter.MaxPrice = midpoint * 1.5
	}
	props, _, err := s.props.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	out := make([]dto.PropertyCard, 0, limit)
	for _, p := range props {
		if !exclude[p.ID] {
			out = append(out, ToPropertyCard(&p))
			if len(out) == limit {
				break
			}
		}
	}
	return out, nil
}

// rankAndTrim applies a simple weighted blend of popularity and freshness.
func (s *RecommendationService) rankAndTrim(ctx context.Context, props []models.Property, viewedIDs []uint, limit int) ([]dto.PropertyCard, error) {
	viewedSet := make(map[uint]bool, len(viewedIDs))
	for _, id := range viewedIDs {
		viewedSet[id] = true
	}

	// Score: log(views+1) + 10 if recently viewed sibling + freshness bonus
	type scored struct {
		card  dto.PropertyCard
		score float64
	}
	items := make([]scored, 0, len(props))
	oneWeekAgo := time.Now().AddDate(0, 0, -7)

	for i := range props {
		views := float64(props[i].ViewCount + 1)
		score := math.Log(views)

		// Small boost if it's similar to what they already viewed (same city/locality)
		if viewedSet[props[i].ID] {
			score += 0.5 // they viewed it but it was already excluded upstream
		}

		// Freshness bonus for new listings
		if props[i].CreatedAt.After(oneWeekAgo) {
			score += 1.0
		}

		items = append(items, scored{card: ToPropertyCard(&props[i]), score: score})
	}

	// Partial sort: find top `limit` by score using insertion (limit is small)
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].score > items[j-1].score; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}

	if len(items) > limit {
		items = items[:limit]
	}
	cards := make([]dto.PropertyCard, len(items))
	for i := range items {
		cards[i] = items[i].card
	}
	return cards, nil
}

// intentPurpose reads which catalogue a customer is actually shopping in.
//
// A renter must not be recommended flats for sale, and a buyer must not be
// recommended tenancies — a recommendation for the wrong catalogue is not a
// weak suggestion, it is an irrelevant one. An empty result means "both", which
// is the honest answer for someone who wants to buy and to rent, and also the
// safe answer for an older profile that predates the intent columns.
func intentPurpose(p *models.CustomerProfile) string {
	if p == nil {
		return ""
	}
	switch {
	case p.LookingToRent && !p.LookingToBuy:
		return string(models.PurposeRent)
	case p.LookingToBuy && !p.LookingToRent:
		return string(models.PurposeBuy)
	case p.LookingToBuy && p.LookingToRent:
		return ""
	}

	// Neither flag is set. Profiles created before the rental work have the
	// older single-valued Purpose instead, so fall back to it rather than
	// treating an unmigrated account as interested in nothing.
	if strings.EqualFold(p.Purpose, string(models.PurposeRent)) {
		return string(models.PurposeRent)
	}
	return string(models.PurposeBuy)
}
