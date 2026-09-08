package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rera/property-platform/backend/internal/cache"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/pkg/utils"
)

// metaCacheTTL applies to the location and amenity master data, which changes
// rarely but is requested on nearly every page.
const metaCacheTTL = 10 * time.Minute

// GeoService manages cities, localities and amenities.
type GeoService struct {
	geo   repositories.GeoRepository
	cache *cache.Client
}

func NewGeoService(geo repositories.GeoRepository, c *cache.Client) *GeoService {
	return &GeoService{geo: geo, cache: c}
}

func (s *GeoService) ListCities(ctx context.Context, popularOnly bool) ([]dto.CityResponse, error) {
	key := "meta:cities:" + boolKey(popularOnly)
	var cached []dto.CityResponse
	if err := s.cache.GetJSON(ctx, key, &cached); err == nil {
		return cached, nil
	}
	cities, err := s.geo.ListCities(ctx, popularOnly)
	if err != nil {
		return nil, Internal("load cities", err)
	}
	out := make([]dto.CityResponse, 0, len(cities))
	for i := range cities {
		out = append(out, ToCityResponse(&cities[i]))
	}
	s.cache.SetJSON(ctx, key, out, metaCacheTTL)
	return out, nil
}

func (s *GeoService) GetCityBySlug(ctx context.Context, slug string) (*dto.CityResponse, error) {
	c, err := s.geo.GetCityBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("City not found")
		}
		return nil, Internal("load city", err)
	}
	resp := ToCityResponse(c)
	return &resp, nil
}

func (s *GeoService) CreateCity(ctx context.Context, req dto.CreateCityRequest) (*dto.CityResponse, error) {
	city := &models.City{
		Name: strings.TrimSpace(req.Name), Slug: utils.Slugify(req.Name),
		State: req.State, StateCode: req.StateCode,
		Latitude: req.Latitude, Longitude: req.Longitude,
		IsPopular: req.IsPopular, DisplayOrder: req.DisplayOrder,
	}
	if err := s.geo.CreateCity(ctx, city); err != nil {
		if errors.Is(err, repositories.ErrDuplicate) {
			return nil, Conflict("A city with this name already exists")
		}
		return nil, Internal("create city", err)
	}
	s.invalidate(ctx)
	resp := ToCityResponse(city)
	return &resp, nil
}

func (s *GeoService) UpdateCity(ctx context.Context, id uint, req dto.CreateCityRequest) (*dto.CityResponse, error) {
	city, err := s.geo.GetCity(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("City not found")
		}
		return nil, Internal("load city", err)
	}
	city.Name = strings.TrimSpace(req.Name)
	city.Slug = utils.Slugify(req.Name)
	city.State = req.State
	city.StateCode = req.StateCode
	city.Latitude = req.Latitude
	city.Longitude = req.Longitude
	city.IsPopular = req.IsPopular
	city.DisplayOrder = req.DisplayOrder
	if err := s.geo.UpdateCity(ctx, city); err != nil {
		return nil, Internal("update city", err)
	}
	s.invalidate(ctx)
	resp := ToCityResponse(city)
	return &resp, nil
}

func (s *GeoService) DeleteCity(ctx context.Context, id uint) error {
	if err := s.geo.DeleteCity(ctx, id); err != nil {
		switch {
		case errors.Is(err, repositories.ErrNotFound):
			return NotFound("City not found")
		case errors.Is(err, repositories.ErrInUse):
			return Conflict("This city still has properties and cannot be deleted")
		}
		return Internal("delete city", err)
	}
	s.invalidate(ctx)
	return nil
}

func (s *GeoService) ListLocalities(ctx context.Context, cityID uint, popularOnly bool) ([]dto.LocalityResponse, error) {
	localities, err := s.geo.ListLocalities(ctx, cityID, popularOnly)
	if err != nil {
		return nil, Internal("load localities", err)
	}
	out := make([]dto.LocalityResponse, 0, len(localities))
	for i := range localities {
		out = append(out, ToLocalityResponse(&localities[i]))
	}
	return out, nil
}

func (s *GeoService) GetLocalityBySlug(ctx context.Context, citySlug, slug string) (*dto.LocalityResponse, error) {
	l, err := s.geo.GetLocalityBySlug(ctx, citySlug, slug)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Locality not found")
		}
		return nil, Internal("load locality", err)
	}
	resp := ToLocalityResponse(l)
	return &resp, nil
}

func (s *GeoService) CreateLocality(ctx context.Context, req dto.CreateLocalityRequest) (*dto.LocalityResponse, error) {
	if _, err := s.geo.GetCity(ctx, req.CityID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, Invalid("The selected city does not exist")
		}
		return nil, Internal("load city", err)
	}
	l := &models.Locality{
		CityID: req.CityID, Name: strings.TrimSpace(req.Name), Slug: utils.Slugify(req.Name),
		Pincode: req.Pincode, Latitude: req.Latitude, Longitude: req.Longitude,
		IsPopular: req.IsPopular, DisplayOrder: req.DisplayOrder,
	}
	if err := s.geo.CreateLocality(ctx, l); err != nil {
		return nil, Internal("create locality", err)
	}
	s.invalidate(ctx)
	resp := ToLocalityResponse(l)
	return &resp, nil
}

func (s *GeoService) UpdateLocality(ctx context.Context, id uint, req dto.CreateLocalityRequest) (*dto.LocalityResponse, error) {
	l, err := s.geo.GetLocality(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Locality not found")
		}
		return nil, Internal("load locality", err)
	}
	l.CityID = req.CityID
	l.Name = strings.TrimSpace(req.Name)
	l.Slug = utils.Slugify(req.Name)
	l.Pincode = req.Pincode
	l.Latitude = req.Latitude
	l.Longitude = req.Longitude
	l.IsPopular = req.IsPopular
	l.DisplayOrder = req.DisplayOrder
	if err := s.geo.UpdateLocality(ctx, l); err != nil {
		return nil, Internal("update locality", err)
	}
	s.invalidate(ctx)
	resp := ToLocalityResponse(l)
	return &resp, nil
}

func (s *GeoService) DeleteLocality(ctx context.Context, id uint) error {
	if err := s.geo.DeleteLocality(ctx, id); err != nil {
		switch {
		case errors.Is(err, repositories.ErrNotFound):
			return NotFound("Locality not found")
		case errors.Is(err, repositories.ErrInUse):
			return Conflict("This locality still has properties and cannot be deleted")
		}
		return Internal("delete locality", err)
	}
	s.invalidate(ctx)
	return nil
}

func (s *GeoService) ListAmenities(ctx context.Context) ([]dto.AmenityResponse, error) {
	var cached []dto.AmenityResponse
	if err := s.cache.GetJSON(ctx, "meta:amenities", &cached); err == nil {
		return cached, nil
	}
	list, err := s.geo.ListAmenities(ctx)
	if err != nil {
		return nil, Internal("load amenities", err)
	}
	out := make([]dto.AmenityResponse, 0, len(list))
	for i := range list {
		out = append(out, ToAmenityResponse(&list[i]))
	}
	s.cache.SetJSON(ctx, "meta:amenities", out, metaCacheTTL)
	return out, nil
}

func (s *GeoService) CreateAmenity(ctx context.Context, req dto.CreateAmenityRequest) (*dto.AmenityResponse, error) {
	a := &models.Amenity{
		Name: strings.TrimSpace(req.Name), Slug: utils.Slugify(req.Name),
		Icon: req.Icon, Category: req.Category, DisplayOrder: req.DisplayOrder,
	}
	if err := s.geo.CreateAmenity(ctx, a); err != nil {
		if errors.Is(err, repositories.ErrDuplicate) {
			return nil, Conflict("An amenity with this name already exists")
		}
		return nil, Internal("create amenity", err)
	}
	s.invalidate(ctx)
	resp := ToAmenityResponse(a)
	return &resp, nil
}

func (s *GeoService) UpdateAmenity(ctx context.Context, id uint, req dto.CreateAmenityRequest) (*dto.AmenityResponse, error) {
	a, err := s.geo.GetAmenity(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Amenity not found")
		}
		return nil, Internal("load amenity", err)
	}
	a.Name = strings.TrimSpace(req.Name)
	a.Slug = utils.Slugify(req.Name)
	a.Icon = req.Icon
	a.Category = req.Category
	a.DisplayOrder = req.DisplayOrder
	if err := s.geo.UpdateAmenity(ctx, a); err != nil {
		return nil, Internal("update amenity", err)
	}
	s.invalidate(ctx)
	resp := ToAmenityResponse(a)
	return &resp, nil
}

func (s *GeoService) DeleteAmenity(ctx context.Context, id uint) error {
	if err := s.geo.DeleteAmenity(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Amenity not found")
		}
		return Internal("delete amenity", err)
	}
	s.invalidate(ctx)
	return nil
}

// PopularCities and PopularLocalities feed the homepage discovery sections.
func (s *GeoService) PopularCities(ctx context.Context, limit int) ([]repositories.CityStat, error) {
	stats, err := s.geo.PopularCities(ctx, limit)
	if err != nil {
		return nil, Internal("load popular cities", err)
	}
	return stats, nil
}

func (s *GeoService) PopularLocalities(ctx context.Context, limit int) ([]repositories.LocalityStat, error) {
	stats, err := s.geo.PopularLocalities(ctx, limit)
	if err != nil {
		return nil, Internal("load popular localities", err)
	}
	return stats, nil
}

func (s *GeoService) invalidate(ctx context.Context) {
	_ = s.cache.DelByPrefix(ctx, "meta:")
	_ = s.cache.DelByPrefix(ctx, "search:")
}

func boolKey(b bool) string {
	if b {
		return "popular"
	}
	return "all"
}
