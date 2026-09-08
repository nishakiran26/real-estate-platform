package repositories

import (
	"context"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// RentSummary is what the market can tell a renter or an owner about a place:
// what a flat there actually goes for. It is computed from live listings only,
// because an asking price from a listing let six months ago is not a guide to
// anything.
type RentSummary struct {
	Listings    int64   `gorm:"column:listings"`
	AverageRent float64 `gorm:"column:average_rent"`
	MinRent     float64 `gorm:"column:min_rent"`
	MaxRent     float64 `gorm:"column:max_rent"`
}

// RentStatsFilter narrows the summary. Zero values widen it, so an empty filter
// summarises the whole live rental market.
type RentStatsFilter struct {
	CityID     uint
	LocalityID uint
	BHK        int
}

// StatsRepository answers the aggregate questions the dashboards and the rent
// guide ask. It is separate from PropertyRepository because none of it returns
// listings: these are counts and averages, and mixing them into the listing
// repository would blur what that interface promises.
type StatsRepository interface {
	// CountByPurpose splits the catalogue. clientID of 0 counts every account.
	CountByPurpose(ctx context.Context, purpose models.PropertyPurpose, clientID uint) (int64, error)
	// CountByAvailability answers "how many of my flats are let".
	CountByAvailability(ctx context.Context, status models.AvailabilityStatus, clientID uint) (int64, error)
	// CountLeadsByType splits enquiries into buying and renting.
	CountLeadsByType(ctx context.Context, leadType models.LeadType, clientID uint) (int64, error)
	// CountVisitsByStatus backs the "viewings waiting on you" figure.
	CountVisitsByStatus(ctx context.Context, status models.VisitStatus, clientID uint) (int64, error)
	// RentSummary is the rent guide for a city, locality or unit size.
	RentSummary(ctx context.Context, f RentStatsFilter) (*RentSummary, error)
}

type statsRepo struct {
	db *gorm.DB
}

func NewStatsRepository(db *gorm.DB) StatsRepository { return &statsRepo{db: db} }

func (r *statsRepo) CountByPurpose(ctx context.Context, purpose models.PropertyPurpose, clientID uint) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Property{}).Where("purpose = ?", purpose)
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *statsRepo) CountByAvailability(ctx context.Context, status models.AvailabilityStatus, clientID uint) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Property{}).
		Where("purpose = ? AND availability_status = ?", models.PurposeRent, status)
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *statsRepo) CountLeadsByType(ctx context.Context, leadType models.LeadType, clientID uint) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Inquiry{}).Where("lead_type = ?", leadType)
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *statsRepo) CountVisitsByStatus(ctx context.Context, status models.VisitStatus, clientID uint) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.PropertyVisit{}).Where("status = ?", status)
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *statsRepo) RentSummary(ctx context.Context, f RentStatsFilter) (*RentSummary, error) {
	q := r.db.WithContext(ctx).Model(&models.RentalDetails{}).
		Joins("JOIN properties ON properties.id = rental_details.property_id").
		// Live listings only: a rent asked six months ago on a flat since let
		// is not a guide to what the place goes for now.
		Where("properties.deleted_at IS NULL").
		Where("properties.property_status = ?", models.PropertyStatusPublished).
		Where("properties.availability_status = ?", models.AvailabilityAvailable).
		Where("rental_details.monthly_rent > 0")

	if f.CityID > 0 {
		q = q.Where("properties.city_id = ?", f.CityID)
	}
	if f.LocalityID > 0 {
		q = q.Where("properties.locality_id = ?", f.LocalityID)
	}
	if f.BHK > 0 {
		q = q.Where("EXISTS (SELECT 1 FROM property_configurations pc "+
			"WHERE pc.property_id = properties.id AND pc.deleted_at IS NULL AND pc.bedrooms = ?)", f.BHK)
	}

	var out RentSummary
	err := q.Select(
		"COUNT(*) AS listings",
		"COALESCE(AVG(rental_details.monthly_rent), 0) AS average_rent",
		"COALESCE(MIN(rental_details.monthly_rent), 0) AS min_rent",
		"COALESCE(MAX(rental_details.monthly_rent), 0) AS max_rent",
	).Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}
