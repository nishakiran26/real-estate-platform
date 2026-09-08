package repositories

import (
	"context"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// AlertMatch describes the property a newly published rental represents, in the
// terms an alert was saved in. It exists so matching is one indexed query
// rather than decoding every stored search.
type AlertMatch struct {
	CityID     uint
	LocalityID uint
	Bedrooms   int
	Rent       float64
}

// AlertRepository owns saved rental searches.
type AlertRepository interface {
	Create(ctx context.Context, a *models.RentalAlert) error
	Get(ctx context.Context, id uint) (*models.RentalAlert, error)
	ListForUser(ctx context.Context, userID uint) ([]models.RentalAlert, error)
	Delete(ctx context.Context, id, userID uint) error
	SetActive(ctx context.Context, id, userID uint, active bool) error
	// Matching returns the active alerts a property satisfies.
	Matching(ctx context.Context, m AlertMatch) ([]models.RentalAlert, error)
	// RecordMatch counts a notified alert, which is what stops the same alert
	// being reported as never having matched anything.
	RecordMatch(ctx context.Context, id uint) error
	CountForUser(ctx context.Context, userID uint) (int64, error)
}

type alertRepo struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) AlertRepository { return &alertRepo{db: db} }

func (r *alertRepo) Create(ctx context.Context, a *models.RentalAlert) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *alertRepo) Get(ctx context.Context, id uint) (*models.RentalAlert, error) {
	var a models.RentalAlert
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, translate(err)
	}
	return &a, nil
}

func (r *alertRepo) ListForUser(ctx context.Context, userID uint) ([]models.RentalAlert, error) {
	var out []models.RentalAlert
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Find(&out).Error
	return out, err
}

func (r *alertRepo) Delete(ctx context.Context, id, userID uint) error {
	res := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.RentalAlert{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *alertRepo) SetActive(ctx context.Context, id, userID uint, active bool) error {
	res := r.db.WithContext(ctx).Model(&models.RentalAlert{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_active", active)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Matching finds the alerts a property satisfies. Every criterion is optional,
// and a zero value means "no constraint" — the same convention PropertyFilter
// uses — so an alert saved for "anything in Pune" matches on city alone.
func (r *alertRepo) Matching(ctx context.Context, m AlertMatch) ([]models.RentalAlert, error) {
	var out []models.RentalAlert
	q := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Where("city_id = 0 OR city_id = ?", m.CityID).
		Where("locality_id = 0 OR locality_id = ?", m.LocalityID).
		Where("min_rent = 0 OR min_rent <= ?", m.Rent).
		Where("max_rent = 0 OR max_rent >= ?", m.Rent)
	// A property with no configuration recorded has no BHK to match on, so
	// alerts that ask for one are not satisfied by it.
	if m.Bedrooms > 0 {
		q = q.Where("bhk = 0 OR bhk = ?", m.Bedrooms)
	} else {
		q = q.Where("bhk = 0")
	}
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *alertRepo) RecordMatch(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.RentalAlert{}).Where("id = ?", id).
		Updates(map[string]any{
			"match_count":      gorm.Expr("match_count + 1"),
			"last_notified_at": now,
		}).Error
}

func (r *alertRepo) CountForUser(ctx context.Context, userID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.RentalAlert{}).
		Where("user_id = ?", userID).Count(&n).Error
	return n, err
}
