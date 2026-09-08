package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rera/property-platform/backend/internal/config"
	"github.com/rera/property-platform/backend/internal/repositories"
)

// SitemapURL is one entry of the generated XML sitemap.
type SitemapURL struct {
	Loc        string
	LastMod    string
	ChangeFreq string
	Priority   string
}

// SEOService generates the crawler-facing artefacts: the sitemap and robots.txt.
// URLs point at the public SPA, not the API, because that is what search
// engines index.
type SEOService struct {
	props repositories.PropertyRepository
	geo   repositories.GeoRepository
	cms   repositories.CMSRepository
	cfg   *config.Config
}

func NewSEOService(
	props repositories.PropertyRepository,
	geo repositories.GeoRepository,
	cms repositories.CMSRepository,
	cfg *config.Config,
) *SEOService {
	return &SEOService{props: props, geo: geo, cms: cms, cfg: cfg}
}

// Sitemap lists the homepage, city and locality landing pages, static pages and
// every published property.
func (s *SEOService) Sitemap(ctx context.Context) ([]SitemapURL, error) {
	base := strings.TrimRight(s.cfg.App.FrontendURL, "/")
	today := time.Now().Format("2006-01-02")

	urls := []SitemapURL{
		{Loc: base + "/", LastMod: today, ChangeFreq: "daily", Priority: "1.0"},
		{Loc: base + "/properties", LastMod: today, ChangeFreq: "hourly", Priority: "0.9"},
	}

	cities, err := s.geo.ListCities(ctx, false)
	if err != nil {
		return nil, Internal("Could not build the sitemap", err)
	}
	for _, city := range cities {
		urls = append(urls, SitemapURL{
			Loc: fmt.Sprintf("%s/city/%s", base, city.Slug), LastMod: today,
			ChangeFreq: "daily", Priority: "0.8",
		})
		localities, err := s.geo.ListLocalities(ctx, city.ID, false)
		if err != nil {
			continue
		}
		for _, loc := range localities {
			urls = append(urls, SitemapURL{
				Loc:        fmt.Sprintf("%s/city/%s/%s", base, city.Slug, loc.Slug),
				LastMod:    today,
				ChangeFreq: "daily", Priority: "0.7",
			})
		}
	}

	pages, err := s.cms.ListPages(ctx, true)
	if err == nil {
		for _, p := range pages {
			urls = append(urls, SitemapURL{
				Loc: fmt.Sprintf("%s/page/%s", base, p.Slug), LastMod: p.UpdatedAt.Format("2006-01-02"),
				ChangeFreq: "monthly", Priority: "0.4",
			})
		}
	}

	slugs, err := s.props.AllPublishedSlugs(ctx)
	if err != nil {
		return nil, Internal("Could not build the sitemap", err)
	}
	for _, entry := range slugs {
		urls = append(urls, SitemapURL{
			Loc:        fmt.Sprintf("%s/properties/%s", base, entry.Slug),
			LastMod:    entry.UpdatedAt.Format("2006-01-02"),
			ChangeFreq: "weekly", Priority: "0.9",
		})
	}
	return urls, nil
}

// Robots returns the robots.txt body. Account areas are disallowed so private
// pages never enter a search index.
func (s *SEOService) Robots() string {
	base := strings.TrimRight(s.cfg.App.URL, "/")
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	b.WriteString("Allow: /\n")
	for _, path := range []string{"/admin", "/client", "/customer", "/login", "/register", "/api/"} {
		b.WriteString("Disallow: " + path + "\n")
	}
	b.WriteString("\nSitemap: " + base + "/sitemap.xml\n")
	return b.String()
}
