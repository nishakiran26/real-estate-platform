package repositories

import (
	"context"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InquiryFilter scopes lead listings. ClientID is set for the builder portal so
// a client can never read another client's leads.
type InquiryFilter struct {
	ClientID   uint
	UserID     uint
	PropertyID uint
	Status     string
	Search     string
	From       *time.Time
	To         *time.Time
	Offset     int
	Limit      int
}

// EngagementRepository owns leads, favourites and the raw event stream that
// feeds analytics and recommendations.
type EngagementRepository interface {
	// Inquiries
	CreateInquiry(ctx context.Context, in *models.Inquiry) error
	GetInquiry(ctx context.Context, id uint) (*models.Inquiry, error)
	ListInquiries(ctx context.Context, f InquiryFilter) ([]models.Inquiry, int64, error)
	UpdateInquiryStatus(ctx context.Context, id uint, status, remarks string, actorID uint) error
	CountInquiries(ctx context.Context, clientID uint, status string) (int64, error)
	RecentInquiries(ctx context.Context, clientID uint, limit int) ([]models.Inquiry, error)

	// Favourites
	AddFavorite(ctx context.Context, userID, propertyID uint) (bool, error)
	RemoveFavorite(ctx context.Context, userID, propertyID uint) (bool, error)
	ListFavorites(ctx context.Context, userID uint, offset, limit int) ([]models.Property, int64, error)
	FavoriteIDs(ctx context.Context, userID uint) ([]uint, error)
	IsFavorite(ctx context.Context, userID, propertyID uint) (bool, error)

	// Events and views
	RecordEvent(ctx context.Context, e *models.UserPropertyEvent) error
	RecordView(ctx context.Context, v *models.PropertyView) error
	ViewedRecently(ctx context.Context, propertyID uint, userID *uint, sessionID string, within time.Duration) (bool, error)
	RecentlyViewed(ctx context.Context, userID *uint, sessionID string, limit int) ([]models.Property, error)
	EventCount(ctx context.Context, propertyID uint, eventType models.EventType) (int64, error)
}

type engagementRepo struct {
	db *gorm.DB
}

func NewEngagementRepository(db *gorm.DB) EngagementRepository { return &engagementRepo{db: db} }

// CreateInquiry stores the lead and bumps the property counter together so the
// dashboard total can never drift from the lead table.
func (r *engagementRepo) CreateInquiry(ctx context.Context, in *models.Inquiry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(in).Error; err != nil {
			return err
		}
		return tx.Model(&models.Property{}).Where("id = ?", in.PropertyID).
			UpdateColumn("inquiry_count", gorm.Expr("inquiry_count + 1")).Error
	})
}

func (r *engagementRepo) GetInquiry(ctx context.Context, id uint) (*models.Inquiry, error) {
	var in models.Inquiry
	err := r.db.WithContext(ctx).Preload("Property").Preload("Configuration").First(&in, id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &in, nil
}

func (r *engagementRepo) ListInquiries(ctx context.Context, f InquiryFilter) ([]models.Inquiry, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Inquiry{})
	if f.ClientID > 0 {
		q = q.Where("inquiries.client_id = ?", f.ClientID)
	}
	if f.UserID > 0 {
		q = q.Where("inquiries.user_id = ?", f.UserID)
	}
	if f.PropertyID > 0 {
		q = q.Where("inquiries.property_id = ?", f.PropertyID)
	}
	if f.Status != "" {
		q = q.Where("inquiries.status = ?", f.Status)
	}
	if f.From != nil {
		q = q.Where("inquiries.created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("inquiries.created_at <= ?", *f.To)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("inquiries.name LIKE ? OR inquiries.mobile LIKE ? OR inquiries.email LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.Inquiry
	err := q.Preload("Property").Preload("Configuration").
		Order("inquiries.created_at DESC").
		Offset(f.Offset).Limit(f.Limit).Find(&list).Error
	return list, total, err
}

func (r *engagementRepo) UpdateInquiryStatus(ctx context.Context, id uint, status, remarks string, actorID uint) error {
	updates := map[string]any{"status": status}
	if status != "NEW" {
		updates["followed_up_at"] = time.Now()
		updates["followed_by"] = actorID
	}
	if remarks != "" {
		updates["message"] = gorm.Expr("CONCAT(message, ?)", "\n---\n"+remarks)
	}
	res := r.db.WithContext(ctx).Model(&models.Inquiry{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *engagementRepo) CountInquiries(ctx context.Context, clientID uint, status string) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Inquiry{})
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *engagementRepo) RecentInquiries(ctx context.Context, clientID uint, limit int) ([]models.Inquiry, error) {
	q := r.db.WithContext(ctx).Model(&models.Inquiry{})
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	var list []models.Inquiry
	err := q.Preload("Property").Order("created_at DESC").Limit(limit).Find(&list).Error
	return list, err
}

// AddFavorite is idempotent; the bool reports whether a new row was inserted so
// the counter is only incremented once per user.
func (r *engagementRepo) AddFavorite(ctx context.Context, userID, propertyID uint) (bool, error) {
	var added bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fav := models.Favorite{UserID: userID, PropertyID: propertyID}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&fav)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		added = true
		return tx.Model(&models.Property{}).Where("id = ?", propertyID).
			UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1")).Error
	})
	return added, err
}

func (r *engagementRepo) RemoveFavorite(ctx context.Context, userID, propertyID uint) (bool, error) {
	var removed bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Unscoped().Where("user_id = ? AND property_id = ?", userID, propertyID).
			Delete(&models.Favorite{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		removed = true
		return tx.Model(&models.Property{}).
			Where("id = ? AND favorite_count > 0", propertyID).
			UpdateColumn("favorite_count", gorm.Expr("favorite_count - 1")).Error
	})
	return removed, err
}

func (r *engagementRepo) ListFavorites(ctx context.Context, userID uint, offset, limit int) ([]models.Property, int64, error) {
	base := r.db.WithContext(ctx).Model(&models.Property{}).
		Joins("JOIN favorites ON favorites.property_id = properties.id AND favorites.deleted_at IS NULL").
		Where("favorites.user_id = ?", userID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var props []models.Property
	err := r.db.WithContext(ctx).Model(&models.Property{}).
		Joins("JOIN favorites ON favorites.property_id = properties.id AND favorites.deleted_at IS NULL").
		Where("favorites.user_id = ?", userID).
		Preload("City").Preload("Locality").Preload("Configurations").
		Preload("Media", func(d *gorm.DB) *gorm.DB { return d.Order("is_cover_image DESC, display_order ASC") }).
		Preload("RERADetails").
		Order("favorites.created_at DESC").
		Offset(offset).Limit(limit).Find(&props).Error
	return props, total, err
}

func (r *engagementRepo) FavoriteIDs(ctx context.Context, userID uint) ([]uint, error) {
	var ids []uint
	err := r.db.WithContext(ctx).Model(&models.Favorite{}).
		Where("user_id = ?", userID).Pluck("property_id", &ids).Error
	return ids, err
}

func (r *engagementRepo) IsFavorite(ctx context.Context, userID, propertyID uint) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Favorite{}).
		Where("user_id = ? AND property_id = ?", userID, propertyID).Count(&n).Error
	return n > 0, err
}

func (r *engagementRepo) RecordEvent(ctx context.Context, e *models.UserPropertyEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *engagementRepo) RecordView(ctx context.Context, v *models.PropertyView) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// ViewedRecently supports view de-duplication: a refresh within the window does
// not inflate the counter.
func (r *engagementRepo) ViewedRecently(ctx context.Context, propertyID uint, userID *uint, sessionID string, within time.Duration) (bool, error) {
	q := r.db.WithContext(ctx).Model(&models.PropertyView{}).
		Where("property_id = ? AND created_at > ?", propertyID, time.Now().Add(-within))
	switch {
	case userID != nil:
		q = q.Where("user_id = ?", *userID)
	case sessionID != "":
		q = q.Where("session_id = ?", sessionID)
	default:
		return false, nil
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

// RecentlyViewed returns distinct properties in most-recent-first order for a
// logged-in user or an anonymous session.
func (r *engagementRepo) RecentlyViewed(ctx context.Context, userID *uint, sessionID string, limit int) ([]models.Property, error) {
	sub := r.db.WithContext(ctx).Model(&models.PropertyView{}).
		Select("property_id, MAX(created_at) AS last_seen").
		Group("property_id").Limit(limit)
	switch {
	case userID != nil:
		sub = sub.Where("user_id = ?", *userID)
	case sessionID != "":
		sub = sub.Where("session_id = ?", sessionID)
	default:
		return nil, nil
	}
	sub = sub.Order("last_seen DESC")

	var props []models.Property
	err := r.db.WithContext(ctx).Model(&models.Property{}).
		Joins("JOIN (?) v ON v.property_id = properties.id", sub).
		Where("properties.property_status IN ?", PublicStatuses).
		Preload("City").Preload("Locality").Preload("Configurations").
		Preload("Media", func(d *gorm.DB) *gorm.DB { return d.Order("is_cover_image DESC, display_order ASC") }).
		Preload("RERADetails").
		Order("v.last_seen DESC").
		Limit(limit).Find(&props).Error
	return props, err
}

func (r *engagementRepo) EventCount(ctx context.Context, propertyID uint, eventType models.EventType) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.UserPropertyEvent{}).
		Where("property_id = ? AND event_type = ?", propertyID, eventType).Count(&n).Error
	return n, err
}
