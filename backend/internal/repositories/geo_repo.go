package repositories

import (
	"context"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// GeoRepository owns the location and amenity master data that powers search
// filters and the homepage. None of it is hard-coded in the application.
type GeoRepository interface {
	// Cities
	CreateCity(ctx context.Context, c *models.City) error
	UpdateCity(ctx context.Context, c *models.City) error
	DeleteCity(ctx context.Context, id uint) error
	GetCity(ctx context.Context, id uint) (*models.City, error)
	GetCityBySlug(ctx context.Context, slug string) (*models.City, error)
	ListCities(ctx context.Context, popularOnly bool) ([]models.City, error)
	PopularCities(ctx context.Context, limit int) ([]CityStat, error)

	// Localities
	CreateLocality(ctx context.Context, l *models.Locality) error
	UpdateLocality(ctx context.Context, l *models.Locality) error
	DeleteLocality(ctx context.Context, id uint) error
	GetLocality(ctx context.Context, id uint) (*models.Locality, error)
	GetLocalityBySlug(ctx context.Context, citySlug, slug string) (*models.Locality, error)
	ListLocalities(ctx context.Context, cityID uint, popularOnly bool) ([]models.Locality, error)
	PopularLocalities(ctx context.Context, limit int) ([]LocalityStat, error)

	// Amenities
	CreateAmenity(ctx context.Context, a *models.Amenity) error
	UpdateAmenity(ctx context.Context, a *models.Amenity) error
	DeleteAmenity(ctx context.Context, id uint) error
	GetAmenity(ctx context.Context, id uint) (*models.Amenity, error)
	ListAmenities(ctx context.Context) ([]models.Amenity, error)
}

// CityStat and LocalityStat carry a live property count for homepage cards.
type CityStat struct {
	ID            uint    `json:"id"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	State         string  `json:"state"`
	PropertyCount int64   `json:"property_count"`
	AvgPrice      float64 `json:"avg_price"`
}

type LocalityStat struct {
	ID            uint    `json:"id"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	CityID        uint    `json:"city_id"`
	CityName      string  `json:"city_name"`
	CitySlug      string  `json:"city_slug"`
	PropertyCount int64   `json:"property_count"`
	AvgPrice      float64 `json:"avg_price"`
}

type geoRepo struct {
	db *gorm.DB
}

func NewGeoRepository(db *gorm.DB) GeoRepository { return &geoRepo{db: db} }

func (r *geoRepo) CreateCity(ctx context.Context, c *models.City) error {
	return translate(r.db.WithContext(ctx).Create(c).Error)
}

func (r *geoRepo) UpdateCity(ctx context.Context, c *models.City) error {
	return r.db.WithContext(ctx).Omit("Localities").Save(c).Error
}

// DeleteCity refuses to remove a city that still has properties, which would
// leave listings pointing at a missing location.
func (r *geoRepo) DeleteCity(ctx context.Context, id uint) error {
	var n int64
	if err := r.db.WithContext(ctx).Model(&models.Property{}).Where("city_id = ?", id).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrInUse
	}
	res := r.db.WithContext(ctx).Delete(&models.City{}, id)
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return res.Error
}

func (r *geoRepo) GetCity(ctx context.Context, id uint) (*models.City, error) {
	var c models.City
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, translate(err)
	}
	return &c, nil
}

func (r *geoRepo) GetCityBySlug(ctx context.Context, slug string) (*models.City, error) {
	var c models.City
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&c).Error; err != nil {
		return nil, translate(err)
	}
	return &c, nil
}

func (r *geoRepo) ListCities(ctx context.Context, popularOnly bool) ([]models.City, error) {
	q := r.db.WithContext(ctx).Model(&models.City{})
	if popularOnly {
		q = q.Where("is_popular = ?", true)
	}
	var cities []models.City
	err := q.Order("display_order ASC, name ASC").Find(&cities).Error
	return cities, err
}

// PopularCities ranks cities by how many published listings they hold.
func (r *geoRepo) PopularCities(ctx context.Context, limit int) ([]CityStat, error) {
	var stats []CityStat
	err := r.db.WithContext(ctx).Model(&models.City{}).
		Select(`cities.id, cities.name, cities.slug, cities.state,
			COUNT(properties.id) AS property_count,
			COALESCE(AVG(NULLIF(properties.min_price, 0)), 0) AS avg_price`).
		Joins(`LEFT JOIN properties ON properties.city_id = cities.id
			AND properties.deleted_at IS NULL
			AND properties.property_status IN ?`, PublicStatuses).
		Group("cities.id, cities.name, cities.slug, cities.state").
		Order("cities.is_popular DESC, property_count DESC, cities.display_order ASC").
		Limit(limit).Scan(&stats).Error
	return stats, err
}

func (r *geoRepo) CreateLocality(ctx context.Context, l *models.Locality) error {
	return translate(r.db.WithContext(ctx).Create(l).Error)
}

func (r *geoRepo) UpdateLocality(ctx context.Context, l *models.Locality) error {
	return r.db.WithContext(ctx).Omit("City").Save(l).Error
}

func (r *geoRepo) DeleteLocality(ctx context.Context, id uint) error {
	var n int64
	if err := r.db.WithContext(ctx).Model(&models.Property{}).Where("locality_id = ?", id).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrInUse
	}
	res := r.db.WithContext(ctx).Delete(&models.Locality{}, id)
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return res.Error
}

func (r *geoRepo) GetLocality(ctx context.Context, id uint) (*models.Locality, error) {
	var l models.Locality
	if err := r.db.WithContext(ctx).Preload("City").First(&l, id).Error; err != nil {
		return nil, translate(err)
	}
	return &l, nil
}

func (r *geoRepo) GetLocalityBySlug(ctx context.Context, citySlug, slug string) (*models.Locality, error) {
	var l models.Locality
	q := r.db.WithContext(ctx).Preload("City").
		Joins("JOIN cities ON cities.id = localities.city_id").
		Where("localities.slug = ?", slug)
	if citySlug != "" {
		q = q.Where("cities.slug = ?", citySlug)
	}
	if err := q.First(&l).Error; err != nil {
		return nil, translate(err)
	}
	return &l, nil
}

func (r *geoRepo) ListLocalities(ctx context.Context, cityID uint, popularOnly bool) ([]models.Locality, error) {
	q := r.db.WithContext(ctx).Model(&models.Locality{})
	if cityID > 0 {
		q = q.Where("city_id = ?", cityID)
	}
	if popularOnly {
		q = q.Where("is_popular = ?", true)
	}
	var list []models.Locality
	err := q.Preload("City").Order("display_order ASC, name ASC").Find(&list).Error
	return list, err
}

func (r *geoRepo) PopularLocalities(ctx context.Context, limit int) ([]LocalityStat, error) {
	var stats []LocalityStat
	err := r.db.WithContext(ctx).Model(&models.Locality{}).
		Select(`localities.id, localities.name, localities.slug, localities.city_id,
			cities.name AS city_name, cities.slug AS city_slug,
			COUNT(properties.id) AS property_count,
			COALESCE(AVG(NULLIF(properties.min_price, 0)), 0) AS avg_price`).
		Joins("JOIN cities ON cities.id = localities.city_id").
		Joins(`LEFT JOIN properties ON properties.locality_id = localities.id
			AND properties.deleted_at IS NULL
			AND properties.property_status IN ?`, PublicStatuses).
		Group("localities.id, localities.name, localities.slug, localities.city_id, cities.name, cities.slug").
		Order("localities.is_popular DESC, property_count DESC").
		Limit(limit).Scan(&stats).Error
	return stats, err
}

func (r *geoRepo) CreateAmenity(ctx context.Context, a *models.Amenity) error {
	return translate(r.db.WithContext(ctx).Create(a).Error)
}

func (r *geoRepo) UpdateAmenity(ctx context.Context, a *models.Amenity) error {
	return r.db.WithContext(ctx).Save(a).Error
}

func (r *geoRepo) DeleteAmenity(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM property_amenities WHERE amenity_id = ?", id).Error; err != nil {
			return err
		}
		res := tx.Delete(&models.Amenity{}, id)
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return res.Error
	})
}

func (r *geoRepo) GetAmenity(ctx context.Context, id uint) (*models.Amenity, error) {
	var a models.Amenity
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, translate(err)
	}
	return &a, nil
}

func (r *geoRepo) ListAmenities(ctx context.Context) ([]models.Amenity, error) {
	var list []models.Amenity
	err := r.db.WithContext(ctx).Order("category ASC, display_order ASC, name ASC").Find(&list).Error
	return list, err
}
