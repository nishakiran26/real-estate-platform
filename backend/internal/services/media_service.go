package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/internal/storage"
)

// Upload is a provider-neutral description of an incoming file. Handlers turn a
// multipart part into this struct so the service never depends on net/http.
type Upload struct {
	Filename string
	MimeType string
	Size     int64
	Reader   io.Reader
}

// MediaService owns property galleries and downloadable documents. Every write
// is preceded by an ownership check so one builder can never touch another
// builder's listing.
type MediaService struct {
	media   repositories.MediaRepository
	props   repositories.PropertyRepository
	store   storage.Storage
	propSvc *PropertyService
}

func NewMediaService(
	media repositories.MediaRepository,
	props repositories.PropertyRepository,
	store storage.Storage,
	propSvc *PropertyService,
) *MediaService {
	return &MediaService{media: media, props: props, store: store, propSvc: propSvc}
}

// imageMediaTypes are the gallery buckets a builder may upload into.
var imageMediaTypes = map[string]models.MediaType{
	"COVER_IMAGE":        models.MediaCoverImage,
	"PROPERTY_IMAGE":     models.MediaPropertyImage,
	"SOCIETY_IMAGE":      models.MediaSocietyImage,
	"FLOOR_PLAN":         models.MediaFloorPlan,
	"MASTER_PLAN":        models.MediaMasterPlan,
	"LOCATION_MAP":       models.MediaLocationMap,
	"CONSTRUCTION_IMAGE": models.MediaConstructionImage,
	"RERA_CERTIFICATE":   models.MediaRERACertificate,

	// Rentals are photographed room by room rather than as a project, so the
	// gallery accepts those buckets too.
	"LIVING_ROOM":       models.MediaLivingRoom,
	"BEDROOM":           models.MediaBedroom,
	"KITCHEN":           models.MediaKitchen,
	"BATHROOM":          models.MediaBathroom,
	"BALCONY":           models.MediaBalcony,
	"BUILDING_EXTERIOR": models.MediaBuildingExterior,
	"PARKING":           models.MediaParking,
	"AMENITIES":         models.MediaAmenities,
}

var docTypes = map[string]models.DocType{
	"BROCHURE":         models.DocBrochure,
	"RERA_CERTIFICATE": models.DocRERACertificate,
	"OTHER":            models.DocOther,
}

// UploadMedia stores an image against a property gallery.
func (s *MediaService) UploadMedia(ctx context.Context, propertyID, clientID uint, mediaType string, up Upload) (*dto.MediaResponse, error) {
	if err := s.assertOwner(ctx, propertyID, clientID); err != nil {
		return nil, err
	}
	mt, ok := imageMediaTypes[strings.ToUpper(strings.TrimSpace(mediaType))]
	if !ok {
		return nil, Invalid("Unsupported media type")
	}
	if _, err := storage.ValidateUpload(up.MimeType, up.Size, storage.MaxImageSize, storage.AllowedImageTypes); err != nil {
		return nil, uploadError(err)
	}
	obj, err := s.store.Save(ctx, fmt.Sprintf("properties/%d/images", propertyID), up.Filename, up.Reader, up.Size, up.MimeType)
	if err != nil {
		return nil, Internal("Could not store the uploaded file", err)
	}
	m := &models.PropertyMedia{
		PropertyID:   propertyID,
		MediaType:    mt,
		FileURL:      obj.URL,
		StorageKey:   obj.Key,
		IsCoverImage: mt == models.MediaCoverImage,
	}
	if err := s.media.AddMedia(ctx, m); err != nil {
		_ = s.store.Delete(ctx, obj.Key) // do not leave an orphaned file behind
		return nil, Internal("Could not save the media record", err)
	}
	s.propSvc.InvalidateSearchCache(ctx)
	resp := toMediaResponse(m)
	return &resp, nil
}

// UploadDocument stores a PDF (brochure, RERA certificate) against a property.
func (s *MediaService) UploadDocument(ctx context.Context, propertyID, clientID uint, docType, name string, up Upload) (*dto.DocumentResponse, error) {
	if err := s.assertOwner(ctx, propertyID, clientID); err != nil {
		return nil, err
	}
	dt, ok := docTypes[strings.ToUpper(strings.TrimSpace(docType))]
	if !ok {
		return nil, Invalid("Unsupported document type")
	}
	if _, err := storage.ValidateUpload(up.MimeType, up.Size, storage.MaxDocSize, storage.AllowedDocTypes); err != nil {
		return nil, uploadError(err)
	}
	obj, err := s.store.Save(ctx, fmt.Sprintf("properties/%d/documents", propertyID), up.Filename, up.Reader, up.Size, up.MimeType)
	if err != nil {
		return nil, Internal("Could not store the uploaded file", err)
	}
	if strings.TrimSpace(name) == "" {
		name = up.Filename
	}
	d := &models.PropertyDocument{
		PropertyID: propertyID,
		DocType:    dt,
		Name:       name,
		FileURL:    obj.URL,
		StorageKey: obj.Key,
		FileSize:   obj.Size,
		MimeType:   up.MimeType,
	}
	if err := s.media.AddDocument(ctx, d); err != nil {
		_ = s.store.Delete(ctx, obj.Key)
		return nil, Internal("Could not save the document record", err)
	}
	return &dto.DocumentResponse{
		ID: d.ID, DocType: string(d.DocType), Name: d.Name,
		FileURL: d.FileURL, FileSize: d.FileSize,
	}, nil
}

// UploadImageAsset stores a standalone image (avatar, company logo, banner
// artwork) and returns its public URL. It is not attached to a property.
func (s *MediaService) UploadImageAsset(ctx context.Context, folder string, up Upload) (*dto.UploadedFileResponse, error) {
	if _, err := storage.ValidateUpload(up.MimeType, up.Size, storage.MaxImageSize, storage.AllowedImageTypes); err != nil {
		return nil, uploadError(err)
	}
	obj, err := s.store.Save(ctx, folder, up.Filename, up.Reader, up.Size, up.MimeType)
	if err != nil {
		return nil, Internal("Could not store the uploaded file", err)
	}
	return &dto.UploadedFileResponse{Key: obj.Key, URL: obj.URL, Size: obj.Size, MimeType: obj.MimeType}, nil
}

func (s *MediaService) ListMedia(ctx context.Context, propertyID, clientID uint) ([]dto.MediaResponse, error) {
	if err := s.assertOwner(ctx, propertyID, clientID); err != nil {
		return nil, err
	}
	list, err := s.media.ListMedia(ctx, propertyID)
	if err != nil {
		return nil, Internal("Could not load media", err)
	}
	out := make([]dto.MediaResponse, 0, len(list))
	for i := range list {
		out = append(out, toMediaResponse(&list[i]))
	}
	return out, nil
}

func (s *MediaService) DeleteMedia(ctx context.Context, propertyID, mediaID, clientID uint) error {
	if err := s.assertOwner(ctx, propertyID, clientID); err != nil {
		return err
	}
	m, err := s.media.GetMedia(ctx, mediaID)
	if err != nil {
		return notFoundOr(err, "Media not found")
	}
	if m.PropertyID != propertyID {
		return NotFound("Media not found")
	}
	if err := s.media.DeleteMedia(ctx, mediaID); err != nil {
		return notFoundOr(err, "Media not found")
	}
	if m.StorageKey != "" {
		_ = s.store.Delete(ctx, m.StorageKey)
	}
	s.propSvc.InvalidateSearchCache(ctx)
	return nil
}

func (s *MediaService) SetCover(ctx context.Context, propertyID, mediaID, clientID uint) error {
	if err := s.assertOwner(ctx, propertyID, clientID); err != nil {
		return err
	}
	if err := s.media.SetCoverImage(ctx, propertyID, mediaID); err != nil {
		return notFoundOr(err, "Media not found")
	}
	s.propSvc.InvalidateSearchCache(ctx)
	return nil
}

func (s *MediaService) Reorder(ctx context.Context, propertyID, clientID uint, ids []uint) error {
	if err := s.assertOwner(ctx, propertyID, clientID); err != nil {
		return err
	}
	if err := s.media.ReorderMedia(ctx, propertyID, ids); err != nil {
		return Internal("Could not reorder media", err)
	}
	return nil
}

func (s *MediaService) ListDocuments(ctx context.Context, propertyID, clientID uint) ([]dto.DocumentResponse, error) {
	if err := s.assertOwner(ctx, propertyID, clientID); err != nil {
		return nil, err
	}
	list, err := s.media.ListDocuments(ctx, propertyID)
	if err != nil {
		return nil, Internal("Could not load documents", err)
	}
	out := make([]dto.DocumentResponse, 0, len(list))
	for i := range list {
		out = append(out, dto.DocumentResponse{
			ID: list[i].ID, DocType: string(list[i].DocType), Name: list[i].Name,
			FileURL: list[i].FileURL, FileSize: list[i].FileSize,
			DownloadCount: list[i].DownloadCount,
		})
	}
	return out, nil
}

func (s *MediaService) DeleteDocument(ctx context.Context, propertyID, docID, clientID uint) error {
	if err := s.assertOwner(ctx, propertyID, clientID); err != nil {
		return err
	}
	d, err := s.media.GetDocument(ctx, docID)
	if err != nil {
		return notFoundOr(err, "Document not found")
	}
	if d.PropertyID != propertyID {
		return NotFound("Document not found")
	}
	if err := s.media.DeleteDocument(ctx, docID); err != nil {
		return notFoundOr(err, "Document not found")
	}
	if d.StorageKey != "" {
		_ = s.store.Delete(ctx, d.StorageKey)
	}
	return nil
}

// assertOwner allows administrators (clientID == 0) through and otherwise
// requires the property to belong to the calling builder.
func (s *MediaService) assertOwner(ctx context.Context, propertyID, clientID uint) error {
	if clientID == 0 {
		if _, err := s.props.GetByID(ctx, propertyID); err != nil {
			return notFoundOr(err, "Property not found")
		}
		return nil
	}
	return s.propSvc.OwnsProperty(ctx, propertyID, clientID)
}

func toMediaResponse(m *models.PropertyMedia) dto.MediaResponse {
	return dto.MediaResponse{
		ID:           m.ID,
		MediaType:    string(m.MediaType),
		FileURL:      m.FileURL,
		ThumbnailURL: m.ThumbnailURL,
		DisplayOrder: m.DisplayOrder,
		IsCoverImage: m.IsCoverImage,
	}
}

func uploadError(err error) error {
	switch {
	case errors.Is(err, storage.ErrFileTooLarge):
		return Invalid("The file is larger than the allowed limit")
	case errors.Is(err, storage.ErrTypeNotAllowed):
		return Invalid("Only JPG, PNG and WEBP images and PDF documents are accepted")
	default:
		return Invalid(err.Error())
	}
}

// notFoundOr maps a repository miss onto a 404 and passes existing service
// errors through untouched.
func notFoundOr(err error, msg string) error {
	if errors.Is(err, repositories.ErrNotFound) {
		return NotFound(msg)
	}
	var svcErr *Error
	if errors.As(err, &svcErr) {
		return err
	}
	return Internal(msg, err)
}
