package repositories

import (
	"context"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
)

// MediaRepository owns property images and downloadable documents.
type MediaRepository interface {
	AddMedia(ctx context.Context, m *models.PropertyMedia) error
	GetMedia(ctx context.Context, id uint) (*models.PropertyMedia, error)
	ListMedia(ctx context.Context, propertyID uint) ([]models.PropertyMedia, error)
	DeleteMedia(ctx context.Context, id uint) error
	SetCoverImage(ctx context.Context, propertyID, mediaID uint) error
	ReorderMedia(ctx context.Context, propertyID uint, orderedIDs []uint) error

	AddDocument(ctx context.Context, d *models.PropertyDocument) error
	GetDocument(ctx context.Context, id uint) (*models.PropertyDocument, error)
	ListDocuments(ctx context.Context, propertyID uint) ([]models.PropertyDocument, error)
	DeleteDocument(ctx context.Context, id uint) error
	IncrementDownloads(ctx context.Context, id uint) error
}

type mediaRepo struct {
	db *gorm.DB
}

func NewMediaRepository(db *gorm.DB) MediaRepository { return &mediaRepo{db: db} }

// AddMedia stores an image. The first image uploaded for a property becomes its
// cover automatically so listings always have a thumbnail.
func (r *mediaRepo) AddMedia(ctx context.Context, m *models.PropertyMedia) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&models.PropertyMedia{}).
			Where("property_id = ?", m.PropertyID).Count(&existing).Error; err != nil {
			return err
		}
		if existing == 0 && isImage(m.MediaType) {
			m.IsCoverImage = true
		}
		if m.IsCoverImage {
			if err := tx.Model(&models.PropertyMedia{}).
				Where("property_id = ?", m.PropertyID).
				Update("is_cover_image", false).Error; err != nil {
				return err
			}
		}
		if m.DisplayOrder == 0 {
			m.DisplayOrder = int(existing) + 1
		}
		return tx.Create(m).Error
	})
}

// isImage reports whether a media type can serve as the listing thumbnail.
// Rental galleries are shot room by room, so those buckets qualify: otherwise a
// flat photographed only as "living room" would never get a cover image and its
// card would render blank.
func isImage(t models.MediaType) bool {
	switch t {
	case models.MediaCoverImage, models.MediaPropertyImage, models.MediaSocietyImage,
		models.MediaLivingRoom, models.MediaBedroom, models.MediaKitchen,
		models.MediaBathroom, models.MediaBalcony, models.MediaBuildingExterior:
		return true
	}
	return false
}

func (r *mediaRepo) GetMedia(ctx context.Context, id uint) (*models.PropertyMedia, error) {
	var m models.PropertyMedia
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, translate(err)
	}
	return &m, nil
}

func (r *mediaRepo) ListMedia(ctx context.Context, propertyID uint) ([]models.PropertyMedia, error) {
	var list []models.PropertyMedia
	err := r.db.WithContext(ctx).Where("property_id = ?", propertyID).
		Order("is_cover_image DESC, display_order ASC, id ASC").Find(&list).Error
	return list, err
}

// DeleteMedia removes an image and promotes another one to cover if the deleted
// image was the cover.
func (r *mediaRepo) DeleteMedia(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var m models.PropertyMedia
		if err := tx.First(&m, id).Error; err != nil {
			return translate(err)
		}
		if err := tx.Delete(&m).Error; err != nil {
			return err
		}
		if !m.IsCoverImage {
			return nil
		}
		var next models.PropertyMedia
		err := tx.Where("property_id = ? AND media_type IN ?", m.PropertyID, coverCandidates).
			Order("display_order ASC, id ASC").First(&next).Error
		if err != nil {
			return nil // nothing left to promote
		}
		return tx.Model(&next).Update("is_cover_image", true).Error
	})
}

func (r *mediaRepo) SetCoverImage(ctx context.Context, propertyID, mediaID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var m models.PropertyMedia
		if err := tx.Where("id = ? AND property_id = ?", mediaID, propertyID).First(&m).Error; err != nil {
			return translate(err)
		}
		if err := tx.Model(&models.PropertyMedia{}).Where("property_id = ?", propertyID).
			Update("is_cover_image", false).Error; err != nil {
			return err
		}
		return tx.Model(&m).Update("is_cover_image", true).Error
	})
}

func (r *mediaRepo) ReorderMedia(ctx context.Context, propertyID uint, orderedIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range orderedIDs {
			if err := tx.Model(&models.PropertyMedia{}).
				Where("id = ? AND property_id = ?", id, propertyID).
				Update("display_order", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *mediaRepo) AddDocument(ctx context.Context, d *models.PropertyDocument) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *mediaRepo) GetDocument(ctx context.Context, id uint) (*models.PropertyDocument, error) {
	var d models.PropertyDocument
	if err := r.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, translate(err)
	}
	return &d, nil
}

func (r *mediaRepo) ListDocuments(ctx context.Context, propertyID uint) ([]models.PropertyDocument, error) {
	var list []models.PropertyDocument
	err := r.db.WithContext(ctx).Where("property_id = ?", propertyID).
		Order("id ASC").Find(&list).Error
	return list, err
}

func (r *mediaRepo) DeleteDocument(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(&models.PropertyDocument{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *mediaRepo) IncrementDownloads(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.PropertyDocument{}).Where("id = ?", id).
		UpdateColumn("download_count", gorm.Expr("download_count + 1")).Error
}

// coverCandidates are the media types that may be promoted to cover image when
// the current cover is deleted. It mirrors isImage.
var coverCandidates = []models.MediaType{
	models.MediaCoverImage, models.MediaPropertyImage, models.MediaSocietyImage,
	models.MediaLivingRoom, models.MediaBedroom, models.MediaKitchen,
	models.MediaBathroom, models.MediaBalcony, models.MediaBuildingExterior,
}
