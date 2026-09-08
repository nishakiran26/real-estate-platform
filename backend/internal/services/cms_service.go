package services

import (
	"context"
	"errors"
	"strings"

	"github.com/rera/property-platform/backend/internal/cache"
	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
	"github.com/rera/property-platform/backend/internal/repositories"
	"github.com/rera/property-platform/backend/pkg/utils"
)

// CMSService manages editable homepage content and static pages.
type CMSService struct {
	cms   repositories.CMSRepository
	geo   repositories.GeoRepository
	props *PropertyService
	cache *cache.Client
}

func NewCMSService(cms repositories.CMSRepository, geo repositories.GeoRepository, props *PropertyService, c *cache.Client) *CMSService {
	return &CMSService{cms: cms, geo: geo, props: props, cache: c}
}

// Homepage assembles every section the landing page renders, in one round trip.
func (s *CMSService) Homepage(ctx context.Context) (*dto.HomepageResponse, error) {
	var cached dto.HomepageResponse
	if err := s.cache.GetJSON(ctx, "homepage", &cached); err == nil {
		return &cached, nil
	}

	banners, err := s.ListBanners(ctx, true)
	if err != nil {
		return nil, err
	}
	featured, err := s.props.Featured(ctx, 8)
	if err != nil {
		return nil, err
	}
	newLaunches, err := s.props.NewLaunches(ctx, 8)
	if err != nil {
		return nil, err
	}
	ready, err := s.props.ByConstructionStatus(ctx, string(models.ConstructionReadyToMove), 8)
	if err != nil {
		return nil, err
	}
	upcoming, err := s.props.ByConstructionStatus(ctx, string(models.ConstructionUpcoming), 8)
	if err != nil {
		return nil, err
	}
	rentals, err := s.props.LatestRentals(ctx, 8)
	if err != nil {
		return nil, err
	}
	cities, err := s.geo.PopularCities(ctx, 8)
	if err != nil {
		return nil, Internal("load popular cities", err)
	}
	localities, err := s.geo.PopularLocalities(ctx, 10)
	if err != nil {
		return nil, Internal("load popular localities", err)
	}

	resp := &dto.HomepageResponse{
		Banners: banners, Featured: featured, NewLaunches: newLaunches,
		ReadyToMove: ready, Upcoming: upcoming, Rentals: rentals,
	}
	for _, c := range cities {
		resp.PopularCities = append(resp.PopularCities,
			dto.LabelCountDTO{Label: c.Name, Slug: c.Slug, Count: c.PropertyCount})
	}
	for _, l := range localities {
		resp.PopularLocalities = append(resp.PopularLocalities,
			dto.LabelCountDTO{Label: l.Name, Slug: l.Slug, Count: l.PropertyCount})
	}
	s.cache.SetJSON(ctx, "homepage", resp, searchCacheTTL)
	return resp, nil
}

func (s *CMSService) ListBanners(ctx context.Context, activeOnly bool) ([]dto.BannerResponse, error) {
	list, err := s.cms.ListBanners(ctx, activeOnly)
	if err != nil {
		return nil, Internal("load banners", err)
	}
	out := make([]dto.BannerResponse, 0, len(list))
	for i := range list {
		out = append(out, toBannerResponse(&list[i]))
	}
	return out, nil
}

func (s *CMSService) CreateBanner(ctx context.Context, req dto.CreateBannerRequest) (*dto.BannerResponse, error) {
	b := &models.HomeBanner{
		Title: req.Title, Subtitle: req.Subtitle, ImageURL: req.ImageURL,
		LinkURL: req.LinkURL, CTAText: req.CTAText, CTALink: req.CTALink,
		Position: req.Position, IsActive: req.IsActive,
	}
	var err error
	if b.StartDate, err = parseDate(req.StartDate); err != nil {
		return nil, err
	}
	if b.EndDate, err = parseDate(req.EndDate); err != nil {
		return nil, err
	}
	if err := s.cms.CreateBanner(ctx, b); err != nil {
		return nil, Internal("create banner", err)
	}
	s.invalidate(ctx)
	resp := toBannerResponse(b)
	return &resp, nil
}

func (s *CMSService) UpdateBanner(ctx context.Context, id uint, req dto.CreateBannerRequest) (*dto.BannerResponse, error) {
	b, err := s.cms.GetBanner(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Banner not found")
		}
		return nil, Internal("load banner", err)
	}
	b.Title, b.Subtitle, b.ImageURL = req.Title, req.Subtitle, req.ImageURL
	b.LinkURL, b.CTAText, b.CTALink = req.LinkURL, req.CTAText, req.CTALink
	b.Position, b.IsActive = req.Position, req.IsActive
	if b.StartDate, err = parseDate(req.StartDate); err != nil {
		return nil, err
	}
	if b.EndDate, err = parseDate(req.EndDate); err != nil {
		return nil, err
	}
	if err := s.cms.UpdateBanner(ctx, b); err != nil {
		return nil, Internal("update banner", err)
	}
	s.invalidate(ctx)
	resp := toBannerResponse(b)
	return &resp, nil
}

func (s *CMSService) DeleteBanner(ctx context.Context, id uint) error {
	if err := s.cms.DeleteBanner(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Banner not found")
		}
		return Internal("delete banner", err)
	}
	s.invalidate(ctx)
	return nil
}

func (s *CMSService) ListFeatured(ctx context.Context, activeOnly bool, limit int) ([]dto.FeaturedPropertyResponse, error) {
	list, err := s.cms.ListFeatured(ctx, activeOnly, limit)
	if err != nil {
		return nil, Internal("load featured properties", err)
	}
	out := make([]dto.FeaturedPropertyResponse, 0, len(list))
	for i := range list {
		item := dto.FeaturedPropertyResponse{
			ID: list[i].ID, Position: list[i].Position, IsActive: list[i].IsActive,
			StartDate: formatDate(list[i].StartDate), EndDate: formatDate(list[i].EndDate),
		}
		if list[i].Property.ID != 0 {
			card := ToPropertyCard(&list[i].Property)
			item.Property = &card
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *CMSService) AddFeatured(ctx context.Context, req dto.CreateFeaturedPropertyRequest) (*dto.FeaturedPropertyResponse, error) {
	f := &models.FeaturedProperty{PropertyID: req.PropertyID, Position: req.Position, IsActive: true}
	var err error
	if f.StartDate, err = parseDate(req.StartDate); err != nil {
		return nil, err
	}
	if f.EndDate, err = parseDate(req.EndDate); err != nil {
		return nil, err
	}
	if err := s.cms.AddFeatured(ctx, f); err != nil {
		return nil, Internal("feature property", err)
	}
	s.invalidate(ctx)
	return &dto.FeaturedPropertyResponse{
		ID: f.ID, Position: f.Position, IsActive: f.IsActive,
		StartDate: formatDate(f.StartDate), EndDate: formatDate(f.EndDate),
	}, nil
}

func (s *CMSService) RemoveFeatured(ctx context.Context, id uint) error {
	if err := s.cms.RemoveFeatured(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Featured entry not found")
		}
		return Internal("remove featured property", err)
	}
	s.invalidate(ctx)
	return nil
}

func (s *CMSService) ListPages(ctx context.Context, publishedOnly bool) ([]dto.StaticPageResponse, error) {
	list, err := s.cms.ListPages(ctx, publishedOnly)
	if err != nil {
		return nil, Internal("load pages", err)
	}
	out := make([]dto.StaticPageResponse, 0, len(list))
	for i := range list {
		out = append(out, toPageResponse(&list[i]))
	}
	return out, nil
}

func (s *CMSService) GetPage(ctx context.Context, slug string) (*dto.StaticPageResponse, error) {
	p, err := s.cms.GetPage(ctx, slug)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, NotFound("Page not found")
		}
		return nil, Internal("load page", err)
	}
	resp := toPageResponse(p)
	return &resp, nil
}

func (s *CMSService) SavePage(ctx context.Context, req dto.StaticPageRequest) (*dto.StaticPageResponse, error) {
	p := &models.StaticPage{
		Title: strings.TrimSpace(req.Title), Slug: utils.Slugify(req.Slug),
		Content: req.Content, MetaTitle: req.MetaTitle,
		MetaDescription: req.MetaDescription, IsPublished: req.IsPublished,
	}
	if err := s.cms.UpsertPage(ctx, p); err != nil {
		return nil, Internal("save page", err)
	}
	resp := toPageResponse(p)
	return &resp, nil
}

func (s *CMSService) DeletePage(ctx context.Context, id uint) error {
	if err := s.cms.DeletePage(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return NotFound("Page not found")
		}
		return Internal("delete page", err)
	}
	return nil
}

func (s *CMSService) invalidate(ctx context.Context) {
	_ = s.cache.Del(ctx, "homepage")
	_ = s.cache.DelByPrefix(ctx, "search:")
}

func toBannerResponse(b *models.HomeBanner) dto.BannerResponse {
	return dto.BannerResponse{
		ID: b.ID, Title: b.Title, Subtitle: b.Subtitle, ImageURL: b.ImageURL,
		LinkURL: b.LinkURL, CTAText: b.CTAText, CTALink: b.CTALink,
		Position: b.Position, IsActive: b.IsActive,
		StartDate: formatDate(b.StartDate), EndDate: formatDate(b.EndDate),
	}
}

func toPageResponse(p *models.StaticPage) dto.StaticPageResponse {
	return dto.StaticPageResponse{
		ID: p.ID, Title: p.Title, Slug: p.Slug, Content: p.Content,
		MetaTitle: p.MetaTitle, MetaDescription: p.MetaDescription,
		IsPublished: p.IsPublished, UpdatedAt: formatTime(p.UpdatedAt),
	}
}
