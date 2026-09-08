package repositories

import (
	"context"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// CMSRepository owns editable homepage content and static pages so none of it
// has to be hard-coded in the frontend.
type CMSRepository interface {
	// Banners
	CreateBanner(ctx context.Context, b *models.HomeBanner) error
	UpdateBanner(ctx context.Context, b *models.HomeBanner) error
	DeleteBanner(ctx context.Context, id uint) error
	GetBanner(ctx context.Context, id uint) (*models.HomeBanner, error)
	ListBanners(ctx context.Context, activeOnly bool) ([]models.HomeBanner, error)

	// Featured properties
	AddFeatured(ctx context.Context, f *models.FeaturedProperty) error
	RemoveFeatured(ctx context.Context, id uint) error
	ListFeatured(ctx context.Context, activeOnly bool, limit int) ([]models.FeaturedProperty, error)

	// Static pages
	UpsertPage(ctx context.Context, p *models.StaticPage) error
	GetPage(ctx context.Context, slug string) (*models.StaticPage, error)
	ListPages(ctx context.Context, publishedOnly bool) ([]models.StaticPage, error)
	DeletePage(ctx context.Context, id uint) error
}

type cmsRepo struct {
	db *gorm.DB
}

func NewCMSRepository(db *gorm.DB) CMSRepository { return &cmsRepo{db: db} }

func (r *cmsRepo) CreateBanner(ctx context.Context, b *models.HomeBanner) error {
	return r.db.WithContext(ctx).Create(b).Error
}

func (r *cmsRepo) UpdateBanner(ctx context.Context, b *models.HomeBanner) error {
	return r.db.WithContext(ctx).Save(b).Error
}

func (r *cmsRepo) DeleteBanner(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(&models.HomeBanner{}, id)
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return res.Error
}

func (r *cmsRepo) GetBanner(ctx context.Context, id uint) (*models.HomeBanner, error) {
	var b models.HomeBanner
	if err := r.db.WithContext(ctx).First(&b, id).Error; err != nil {
		return nil, translate(err)
	}
	return &b, nil
}

// ListBanners honours the optional scheduling window when activeOnly is set.
func (r *cmsRepo) ListBanners(ctx context.Context, activeOnly bool) ([]models.HomeBanner, error) {
	q := r.db.WithContext(ctx).Model(&models.HomeBanner{})
	if activeOnly {
		now := time.Now()
		q = q.Where("is_active = ?", true).
			Where("start_date IS NULL OR start_date <= ?", now).
			Where("end_date IS NULL OR end_date >= ?", now)
	}
	var list []models.HomeBanner
	err := q.Order("position ASC, id ASC").Find(&list).Error
	return list, err
}

// AddFeatured pins a property to the homepage and mirrors the flag onto the
// property row so search can filter on it cheaply.
func (r *cmsRepo) AddFeatured(ctx context.Context, f *models.FeaturedProperty) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.FeaturedProperty
		err := tx.Where("property_id = ?", f.PropertyID).First(&existing).Error
		switch {
		case err == nil:
			existing.Position = f.Position
			existing.StartDate = f.StartDate
			existing.EndDate = f.EndDate
			existing.IsActive = true
			if err := tx.Omit("Property").Save(&existing).Error; err != nil {
				return err
			}
			*f = existing
		case err == gorm.ErrRecordNotFound:
			if err := tx.Omit("Property").Create(f).Error; err != nil {
				return err
			}
		default:
			return err
		}
		return tx.Model(&models.Property{}).Where("id = ?", f.PropertyID).
			Update("is_featured", true).Error
	})
}

func (r *cmsRepo) RemoveFeatured(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var f models.FeaturedProperty
		if err := tx.First(&f, id).Error; err != nil {
			return translate(err)
		}
		if err := tx.Delete(&f).Error; err != nil {
			return err
		}
		return tx.Model(&models.Property{}).Where("id = ?", f.PropertyID).
			Update("is_featured", false).Error
	})
}

func (r *cmsRepo) ListFeatured(ctx context.Context, activeOnly bool, limit int) ([]models.FeaturedProperty, error) {
	q := r.db.WithContext(ctx).Model(&models.FeaturedProperty{})
	if activeOnly {
		now := time.Now()
		q = q.Where("is_active = ?", true).
			Where("start_date IS NULL OR start_date <= ?", now).
			Where("end_date IS NULL OR end_date >= ?", now)
	}
	var list []models.FeaturedProperty
	err := q.Preload("Property.City").Preload("Property.Locality").
		Preload("Property.Configurations").
		Preload("Property.RERADetails").
		Preload("Property.Media", func(d *gorm.DB) *gorm.DB {
			return d.Order("is_cover_image DESC, display_order ASC")
		}).
		Order("position ASC, id ASC").Limit(limit).Find(&list).Error
	return list, err
}

// UpsertPage writes a static page, matching on slug so admins can edit a page
// without knowing its ID.
func (r *cmsRepo) UpsertPage(ctx context.Context, p *models.StaticPage) error {
	var existing models.StaticPage
	err := r.db.WithContext(ctx).Where("slug = ?", p.Slug).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.WithContext(ctx).Create(p).Error
	}
	if err != nil {
		return err
	}
	p.ID = existing.ID
	p.CreatedAt = existing.CreatedAt
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *cmsRepo) GetPage(ctx context.Context, slug string) (*models.StaticPage, error) {
	var p models.StaticPage
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&p).Error; err != nil {
		return nil, translate(err)
	}
	return &p, nil
}

func (r *cmsRepo) ListPages(ctx context.Context, publishedOnly bool) ([]models.StaticPage, error) {
	q := r.db.WithContext(ctx).Model(&models.StaticPage{})
	if publishedOnly {
		q = q.Where("is_published = ?", true)
	}
	var list []models.StaticPage
	err := q.Order("title ASC").Find(&list).Error
	return list, err
}

func (r *cmsRepo) DeletePage(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(&models.StaticPage{}, id)
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return res.Error
}
