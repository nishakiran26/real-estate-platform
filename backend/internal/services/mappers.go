package services

import (
	"encoding/json"
	"time"

	"github.com/rera/property-platform/backend/internal/dto"
	"github.com/rera/property-platform/backend/internal/models"
)

// formatDate renders a nullable timestamp as an ISO date, or an empty string.
func formatDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ToPropertyResponse builds the full detail payload for a property.
func ToPropertyResponse(p *models.Property) dto.PropertyResponse {
	resp := dto.PropertyResponse{
		ID: p.ID, Name: p.Name, Slug: p.Slug,
		ProjectName: p.ProjectName, BuilderName: p.BuilderName,
		PropertyType:       string(p.PropertyType),
		PropertyStatus:     string(p.PropertyStatus),
		ConstructionStatus: string(p.ConstructionStatus),
		Address:            p.Address, Pincode: p.Pincode,
		Latitude: p.Latitude, Longitude: p.Longitude,
		Description: p.Description,
		MinPrice:    p.MinPrice, MaxPrice: p.MaxPrice,
		MinArea: p.MinArea, MaxArea: p.MaxArea,
		IsFeatured: p.IsFeatured, IsRERAVerified: p.IsRERAVerified,
		ViewCount: p.ViewCount, InquiryCount: p.InquiryCount, FavoriteCount: p.FavoriteCount,
		RejectionReason: p.RejectionReason,
		MetaTitle:       p.MetaTitle, MetaDescription: p.MetaDescription,
		CreatedAt: formatTime(p.CreatedAt), UpdatedAt: formatTime(p.UpdatedAt),
		Purpose:            string(p.Purpose),
		AvailabilityStatus: string(p.AvailabilityStatus),
		PostedByType:       string(p.PostedByType),
		RentalDetails:      toRentalResponse(p.RentalDetails),
	}
	if d := formatDate(p.ListingExpiryDate); d != "" {
		resp.ListingExpiryDate = &d
	}
	if d := formatDate(p.PossessionDate); d != "" {
		resp.PossessionDate = &d
	}
	if p.City.ID != 0 {
		resp.City = &dto.NameIDResponse{ID: p.City.ID, Name: p.City.Name, Slug: p.City.Slug}
	}
	if p.Locality.ID != 0 {
		resp.Locality = &dto.NameIDResponse{ID: p.Locality.ID, Name: p.Locality.Name, Slug: p.Locality.Slug}
	}
	if p.Client.ID != 0 {
		resp.Builder = &dto.BuilderResponse{
			ID: p.Client.ID, CompanyName: p.Client.CompanyName,
			LogoURL: p.Client.CompanyLogoURL, Description: p.Client.CompanyDesc,
			Website: p.Client.Website, RERARegNo: p.Client.RERARegNo,
			TotalProperties:   p.Client.TotalProperties,
			ProfileType:       string(p.Client.ProfileType),
			VerificationLevel: string(p.Client.VerificationLevel),
		}
	}
	for _, c := range p.Configurations {
		resp.Configurations = append(resp.Configurations, dto.ConfigurationResponse{
			ID: c.ID, Name: c.Name,
			CarpetArea: c.CarpetArea, BuiltUpArea: c.BuiltUpArea, SuperBuiltUpArea: c.SuperBuiltUpArea,
			Bedrooms: c.Bedrooms, Bathrooms: c.Bathrooms, Balconies: c.Balconies,
			Price: c.Price, MinPrice: c.MinPrice, MaxPrice: c.MaxPrice,
			PricePerSqft: c.PricePerSqft, Availability: c.Availability,
			NumberOfUnits: c.NumberOfUnits,
		})
	}
	for _, m := range p.Media {
		resp.Media = append(resp.Media, dto.MediaResponse{
			ID: m.ID, MediaType: string(m.MediaType), FileURL: m.FileURL,
			ThumbnailURL: m.ThumbnailURL, DisplayOrder: m.DisplayOrder,
			IsCoverImage: m.IsCoverImage,
		})
		if m.IsCoverImage && resp.CoverImage == "" {
			resp.CoverImage = m.FileURL
		}
	}
	if resp.CoverImage == "" && len(resp.Media) > 0 {
		resp.CoverImage = resp.Media[0].FileURL
	}
	for _, d := range p.Documents {
		resp.Documents = append(resp.Documents, dto.DocumentResponse{
			ID: d.ID, DocType: string(d.DocType), Name: d.Name,
			FileURL: d.FileURL, FileSize: d.FileSize, DownloadCount: d.DownloadCount,
		})
	}
	if p.RERADetails.ID != 0 {
		resp.RERADetails = &dto.RERAResponse{
			ID:                 p.RERADetails.ID,
			RegistrationNumber: p.RERADetails.RegistrationNumber,
			Authority:          p.RERADetails.Authority,
			RegistrationDate:   formatDate(p.RERADetails.RegistrationDate),
			CompletionDate:     formatDate(p.RERADetails.CompletionDate),
			VerificationStatus: string(p.RERADetails.VerificationStatus),
			VerifiedAt:         formatDate(p.RERADetails.VerifiedAt),
		}
	}
	for _, a := range p.Amenities {
		resp.Amenities = append(resp.Amenities, dto.AmenityResponse{
			ID: a.ID, Name: a.Name, Slug: a.Slug, Icon: a.Icon, Category: a.Category,
		})
	}
	return resp
}

// ToPropertyCard builds the compact payload used by listings, search results and
// homepage carousels. It deliberately omits descriptions and documents.
func ToPropertyCard(p *models.Property) dto.PropertyCard {
	card := dto.PropertyCard{
		ID: p.ID, Name: p.Name, Slug: p.Slug,
		BuilderName:        p.BuilderName,
		PropertyType:       string(p.PropertyType),
		ConstructionStatus: string(p.ConstructionStatus),
		PropertyStatus:     string(p.PropertyStatus),
		MinPrice:           p.MinPrice, MaxPrice: p.MaxPrice,
		MinArea: p.MinArea, MaxArea: p.MaxArea,
		IsFeatured: p.IsFeatured, IsRERAVerified: p.IsRERAVerified,
		ViewCount: p.ViewCount, FavoriteCount: p.FavoriteCount,
		CreatedAt:          formatTime(p.CreatedAt),
		Purpose:            string(p.Purpose),
		AvailabilityStatus: string(p.AvailabilityStatus),
	}
	// Rentals lead with the monthly rent and furnishing, which is what a renter
	// scans a card for; a sale card is unchanged.
	if p.RentalDetails != nil {
		card.MonthlyRent = p.RentalDetails.MonthlyRent
		card.SecurityDeposit = p.RentalDetails.SecurityDeposit
		card.FurnishingType = string(p.RentalDetails.FurnishingType)
		card.AvailableFrom = formatDate(p.RentalDetails.AvailableFrom)
	}
	if d := formatDate(p.PossessionDate); d != "" {
		card.PossessionDate = d
	}
	if p.City.ID != 0 {
		card.City = p.City.Name
		card.CitySlug = p.City.Slug
	}
	if p.Locality.ID != 0 {
		card.Locality = p.Locality.Name
		card.LocalitySlug = p.Locality.Slug
	}
	if p.RERADetails.ID != 0 {
		card.RERANumber = p.RERADetails.RegistrationNumber
		card.RERAStatus = string(p.RERADetails.VerificationStatus)
	}
	for _, m := range p.Media {
		if m.IsCoverImage {
			card.CoverImage = m.FileURL
			break
		}
	}
	if card.CoverImage == "" && len(p.Media) > 0 {
		card.CoverImage = p.Media[0].FileURL
	}
	// Summarise the available unit types, e.g. "2, 3 BHK".
	seen := map[int]bool{}
	for _, c := range p.Configurations {
		card.Configurations = append(card.Configurations, dto.ConfigSummary{
			Name: c.Name, Bedrooms: c.Bedrooms, CarpetArea: c.CarpetArea, Price: c.Price,
		})
		if c.Bedrooms > 0 && !seen[c.Bedrooms] {
			seen[c.Bedrooms] = true
			card.BHKOptions = append(card.BHKOptions, c.Bedrooms)
		}
	}
	sortInts(card.BHKOptions)
	return card
}

// ToPropertyCards maps a slice in one pass.
func ToPropertyCards(props []models.Property) []dto.PropertyCard {
	cards := make([]dto.PropertyCard, 0, len(props))
	for i := range props {
		cards = append(cards, ToPropertyCard(&props[i]))
	}
	return cards
}

// sortInts is an insertion sort; BHK lists are tiny, so this avoids pulling in
// the sort package for a handful of elements.
func sortInts(xs []int) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
}

// ToInquiryResponse maps a lead. Contact details are included because only the
// owning client or an administrator can reach this mapping.
func ToInquiryResponse(in *models.Inquiry) dto.InquiryResponse {
	resp := dto.InquiryResponse{
		ID: in.ID, PropertyID: in.PropertyID, ClientID: in.ClientID,
		Name: in.Name, Mobile: in.Mobile, Email: in.Email,
		Message: in.Message, Status: in.Status, Source: in.Source,
		Budget: in.Budget, PreferredTime: in.PreferredTime,
		CreatedAt: formatTime(in.CreatedAt),
	}
	if in.Property.ID != 0 {
		resp.Property = &dto.PropertySummary{
			ID: in.Property.ID, Name: in.Property.Name,
			Slug: in.Property.Slug, MinPrice: in.Property.MinPrice,
		}
	}
	if in.Configuration != nil {
		resp.Configuration = in.Configuration.Name
	}
	return resp
}

// ToCityResponse and ToLocalityResponse map location master data.
func ToCityResponse(c *models.City) dto.CityResponse {
	return dto.CityResponse{
		ID: c.ID, Name: c.Name, Slug: c.Slug, State: c.State,
		StateCode: c.StateCode, Latitude: c.Latitude, Longitude: c.Longitude,
		IsPopular: c.IsPopular, DisplayOrder: c.DisplayOrder,
	}
}

func ToLocalityResponse(l *models.Locality) dto.LocalityResponse {
	resp := dto.LocalityResponse{
		ID: l.ID, CityID: l.CityID, Name: l.Name, Slug: l.Slug,
		Pincode: l.Pincode, Latitude: l.Latitude, Longitude: l.Longitude,
		AvgPricePerSqft: l.AvgPricePerSqft, IsPopular: l.IsPopular,
		DisplayOrder: l.DisplayOrder,
	}
	if l.City.ID != 0 {
		resp.CityName = l.City.Name
		resp.CitySlug = l.City.Slug
	}
	return resp
}

func ToAmenityResponse(a *models.Amenity) dto.AmenityResponse {
	return dto.AmenityResponse{
		ID: a.ID, Name: a.Name, Slug: a.Slug, Icon: a.Icon,
		Category: a.Category, DisplayOrder: a.DisplayOrder,
	}
}

// ToClientResponse maps a builder profile for the portal and public pages.
func ToClientResponse(c *models.Client) dto.ClientResponse {
	resp := dto.ClientResponse{
		ID: c.ID, UserID: c.UserID, CompanyName: c.CompanyName,
		CompanyLogoURL: c.CompanyLogoURL, CompanyDesc: c.CompanyDesc,
		Website: c.Website, ContactPhone: c.ContactPhone, ContactEmail: c.ContactEmail,
		Address: c.Address, City: c.City, State: c.State, Pincode: c.Pincode,
		GSTIN: c.GSTIN, RERARegNo: c.RERARegNo, TotalProperties: c.TotalProperties,
		ProfileType:       string(c.ProfileType),
		VerificationLevel: string(c.VerificationLevel),
		CreatedAt:         formatTime(c.CreatedAt),
	}
	if c.User.ID != 0 {
		resp.OwnerName = c.User.FullName
		resp.OwnerEmail = c.User.Email
		resp.Status = string(c.User.Status)
	}
	return resp
}

func ToNotificationResponse(n *models.Notification) dto.NotificationResponse {
	resp := dto.NotificationResponse{
		ID: n.ID, Type: string(n.Type), Title: n.Title,
		Message: n.Message, Link: n.Link, CreatedAt: formatTime(n.CreatedAt),
	}
	if n.ReadAt != nil {
		resp.ReadAt = formatTime(*n.ReadAt)
	}
	return resp
}

func ToLoginHistoryResponse(h *models.LoginHistory) dto.LoginHistoryResponse {
	resp := dto.LoginHistoryResponse{
		ID: h.ID, Method: string(h.Method), IPAddress: h.IPAddress,
		Device: h.DeviceName, LoginAt: formatTime(h.LoginAt),
	}
	if h.LogoutAt != nil {
		resp.LogoutAt = formatTime(*h.LogoutAt)
	}
	return resp
}

// toRentalResponse renders the rental block. It returns nil for a sale listing,
// so the field is simply absent from the payload rather than present and empty.
func toRentalResponse(r *models.RentalDetails) *dto.RentalResponse {
	if r == nil {
		return nil
	}
	resp := &dto.RentalResponse{
		MonthlyRent:         r.MonthlyRent,
		SecurityDeposit:     r.SecurityDeposit,
		MaintenanceAmount:   r.MaintenanceAmount,
		MaintenanceIncluded: r.MaintenanceIncluded,
		AvailableFrom:       formatDate(r.AvailableFrom),
		MinLeaseMonths:      r.MinLeaseMonths,
		MaxLeaseMonths:      r.MaxLeaseMonths,
		NoticePeriodDays:    r.NoticePeriodDays,
		FurnishingType:      string(r.FurnishingType),
		TenantPreference:    string(r.TenantPreference),
		PetAllowed:          r.PetAllowed,
	}
	if r.IncludedItems != "" {
		// A malformed list must not break the page; an absent list is fine.
		var items []string
		if err := json.Unmarshal([]byte(r.IncludedItems), &items); err == nil {
			resp.IncludedItems = items
		}
	}
	return resp
}
