package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PropertyFilter carries every supported search dimension. Zero values mean
// "no constraint", so the same struct serves the public search, the client
// portal and the admin queue.
type PropertyFilter struct {
	// Purpose splits the catalogue into sale and rental listings. An empty
	// value means "either", which the compare and similar lookups rely on.
	Purpose string

	// Rental criteria. They join rental_details, so they are only applied when
	// set — a sale search never pays for the join.
	MinRent          float64
	MaxRent          float64
	Furnishing       string
	TenantPreference string
	PetAllowed       *bool
	AvailableBefore  *time.Time

	// AvailabilityStatuses restricts to listings still on the market. The public
	// search always passes it so rented stock disappears from results.
	AvailabilityStatuses []models.AvailabilityStatus

	CitySlug           string
	LocalitySlug       string
	CityID             uint
	LocalityID         uint
	PropertyType       string
	ConstructionStatus string
	MinPrice           float64
	MaxPrice           float64
	MinArea            float64
	MaxArea            float64
	BHK                int
	AmenityIDs         []uint
	BuilderName        string
	ClientID           uint
	PossessionBefore   *time.Time
	IsFeatured         *bool
	IsRERAVerified     *bool
	Search             string
	Statuses           []models.PropertyStatus // workflow statuses to include
	SortBy             string
	SortOrder          string
	Offset             int
	Limit              int
	ExcludeID          uint
}

// PropertyWrite bundles a property with the child collections written in the
// same transaction. It replaces a growing argument list: rental terms are the
// fifth such collection, and a struct keeps the call sites readable.
//
// A nil RERA or Rental leaves that record untouched on update, which is what
// lets a sale listing and a rental share one write path.
type PropertyWrite struct {
	Property       *models.Property
	Configurations []models.PropertyConfiguration
	AmenityIDs     []uint
	RERA           *models.RERADetails
	Rental         *models.RentalDetails
}

type PropertyRepository interface {
	Create(ctx context.Context, w PropertyWrite) error
	Update(ctx context.Context, w PropertyWrite) error
	GetByID(ctx context.Context, id uint) (*models.Property, error)
	GetBySlug(ctx context.Context, slug string) (*models.Property, error)
	GetForOwner(ctx context.Context, id, clientID uint) (*models.Property, error)
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, f PropertyFilter) ([]models.Property, int64, error)
	ListByIDs(ctx context.Context, ids []uint) ([]models.Property, error)
	SlugExists(ctx context.Context, slug string, excludeID uint) (bool, error)
	ChangeStatus(ctx context.Context, id uint, to models.PropertyStatus, actorID uint, remarks string) error
	CountByStatus(ctx context.Context, status models.PropertyStatus, clientID uint) (int64, error)
	CountAll(ctx context.Context, clientID uint) (int64, error)
	StatusHistory(ctx context.Context, propertyID uint) ([]models.PropertyStatusHistory, error)
	IncrementCounter(ctx context.Context, id uint, column string, delta int64) error
	Similar(ctx context.Context, p *models.Property, limit int) ([]models.Property, error)
	AllPublishedSlugs(ctx context.Context) ([]SlugEntry, error)
	SetFeatured(ctx context.Context, id uint, featured bool) error
	SetRERAVerification(ctx context.Context, propertyID uint, status models.RERAStatus, adminID uint) error
	SetAvailability(ctx context.Context, id uint, status models.AvailabilityStatus) error
	ExtendListing(ctx context.Context, id uint, until time.Time) error
	ExpiringRentals(ctx context.Context, from, to time.Time) ([]models.Property, error)
	SetProvenance(ctx context.Context, id uint, postedBy models.PostedByType, ownerUserID *uint) error
}

// SlugEntry feeds the sitemap generator.
type SlugEntry struct {
	Slug      string
	UpdatedAt time.Time
}

type propertyRepo struct {
	db *gorm.DB
}

func NewPropertyRepository(db *gorm.DB) PropertyRepository { return &propertyRepo{db: db} }

// PublicStatuses are the workflow states visible to anonymous visitors.
var PublicStatuses = []models.PropertyStatus{
	models.PropertyStatusApproved,
	models.PropertyStatusPublished,
}

// Create writes the property together with its configurations, amenities and
// RERA record in a single transaction.
func (r *propertyRepo) Create(ctx context.Context, w PropertyWrite) error {
	p, configs, amenityIDs, rera := w.Property, w.Configurations, w.AmenityIDs, w.RERA
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(p).Error; err != nil {
			return translate(err)
		}
		if err := writeConfigurations(tx, p, configs); err != nil {
			return err
		}
		if err := replaceAmenities(tx, p, amenityIDs); err != nil {
			return err
		}
		if rera != nil {
			rera.PropertyID = p.ID
			if err := tx.Create(rera).Error; err != nil {
				return err
			}
			p.IsRERAVerified = rera.VerificationStatus == models.RERAStatusVerified
		}
		if w.Rental != nil {
			w.Rental.PropertyID = p.ID
			if err := tx.Create(w.Rental).Error; err != nil {
				return err
			}
			p.RentalDetails = w.Rental
		}
		// Pricing summary is derived from configurations, never client input.
		p.UpdatePricingSummary()
		return tx.Model(p).Select("min_price", "max_price", "min_area", "max_area", "is_rera_verified").
			Updates(map[string]any{
				"min_price": p.MinPrice, "max_price": p.MaxPrice,
				"min_area": p.MinArea, "max_area": p.MaxArea,
				"is_rera_verified": p.IsRERAVerified,
			}).Error
	})
}

// Update replaces the property and its child collections atomically.
func (r *propertyRepo) Update(ctx context.Context, w PropertyWrite) error {
	p, configs, amenityIDs, rera := w.Property, w.Configurations, w.AmenityIDs, w.RERA
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if configs != nil {
			if err := tx.Where("property_id = ?", p.ID).Delete(&models.PropertyConfiguration{}).Error; err != nil {
				return err
			}
			if err := writeConfigurations(tx, p, configs); err != nil {
				return err
			}
			p.UpdatePricingSummary()
		}
		if amenityIDs != nil {
			if err := replaceAmenities(tx, p, amenityIDs); err != nil {
				return err
			}
		}
		if rera != nil {
			rera.PropertyID = p.ID
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "property_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"registration_number", "authority", "registration_date", "completion_date", "verification_status", "updated_at"}),
			}).Create(rera).Error; err != nil {
				return err
			}
			p.IsRERAVerified = rera.VerificationStatus == models.RERAStatusVerified
		}
		if w.Rental != nil {
			w.Rental.PropertyID = p.ID
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "property_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"monthly_rent", "security_deposit", "maintenance_amount",
					"maintenance_included", "available_from", "min_lease_months",
					"max_lease_months", "notice_period_days", "furnishing_type",
					"tenant_preference", "pet_allowed", "included_items", "updated_at",
				}),
			}).Create(w.Rental).Error; err != nil {
				return err
			}
			p.RentalDetails = w.Rental
		}
		return tx.Omit("Configurations", "Media", "Documents", "RERADetails", "RentalDetails",
			"Amenities", "StatusHistory", "City", "Locality", "Client").Save(p).Error
	})
}

// writeConfigurations inserts unit configurations, deriving price-per-sqft from
// the carpet area so the value cannot drift from the price.
func writeConfigurations(tx *gorm.DB, p *models.Property, configs []models.PropertyConfiguration) error {
	if len(configs) == 0 {
		p.Configurations = nil
		return nil
	}
	for i := range configs {
		configs[i].ID = 0
		configs[i].PropertyID = p.ID
		if configs[i].CarpetArea > 0 && configs[i].Price > 0 {
			configs[i].PricePerSqft = configs[i].Price / configs[i].CarpetArea
		}
		if configs[i].MinPrice == 0 {
			configs[i].MinPrice = configs[i].Price
		}
		if configs[i].MaxPrice == 0 {
			configs[i].MaxPrice = configs[i].Price
		}
	}
	if err := tx.Create(&configs).Error; err != nil {
		return err
	}
	p.Configurations = configs
	return nil
}

func replaceAmenities(tx *gorm.DB, p *models.Property, amenityIDs []uint) error {
	if amenityIDs == nil {
		return nil
	}
	if len(amenityIDs) == 0 {
		return tx.Model(p).Association("Amenities").Clear()
	}
	var amenities []models.Amenity
	if err := tx.Where("id IN ?", amenityIDs).Find(&amenities).Error; err != nil {
		return err
	}
	return tx.Model(p).Association("Amenities").Replace(amenities)
}

// fullPreload loads everything the property detail page renders.
func fullPreload(db *gorm.DB) *gorm.DB {
	return db.Preload("City").Preload("Locality").
		Preload("Client").
		Preload("Configurations", func(d *gorm.DB) *gorm.DB { return d.Order("price ASC") }).
		Preload("Media", func(d *gorm.DB) *gorm.DB { return d.Order("display_order ASC, id ASC") }).
		Preload("Documents").
		Preload("RERADetails").
		Preload("RentalDetails").
		Preload("Amenities")
}

func (r *propertyRepo) GetByID(ctx context.Context, id uint) (*models.Property, error) {
	var p models.Property
	if err := fullPreload(r.db.WithContext(ctx)).First(&p, id).Error; err != nil {
		return nil, translate(err)
	}
	return &p, nil
}

func (r *propertyRepo) GetBySlug(ctx context.Context, slug string) (*models.Property, error) {
	var p models.Property
	if err := fullPreload(r.db.WithContext(ctx)).Where("slug = ?", slug).First(&p).Error; err != nil {
		return nil, translate(err)
	}
	return &p, nil
}

// GetForOwner scopes the lookup to a single builder account. It is the guard
// that stops one client reading or editing another client's listing.
func (r *propertyRepo) GetForOwner(ctx context.Context, id, clientID uint) (*models.Property, error) {
	var p models.Property
	err := fullPreload(r.db.WithContext(ctx)).
		Where("id = ? AND client_id = ?", id, clientID).First(&p).Error
	if err != nil {
		return nil, translate(err)
	}
	return &p, nil
}

func (r *propertyRepo) Delete(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(&models.Property{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// sortColumns whitelists ORDER BY targets so a query parameter can never reach
// the SQL string unchecked.
var sortColumns = map[string]string{
	"price":      "properties.min_price",
	"price_asc":  "properties.min_price",
	"price_desc": "properties.min_price",
	"newest":     "properties.created_at",
	"created_at": "properties.created_at",
	"popular":    "properties.inquiry_count",
	"views":      "properties.view_count",
	"area":       "properties.min_area",
	"name":       "properties.name",

	// Rent sorting reads the joined rental_details row; needsRentalJoin keeps
	// the join in place whenever one of these is selected.
	"rent":      "rd.monthly_rent",
	"rent_asc":  "rd.monthly_rent",
	"rent_desc": "rd.monthly_rent",
}

func (r *propertyRepo) List(ctx context.Context, f PropertyFilter) ([]models.Property, int64, error) {
	q := r.applyFilter(r.db.WithContext(ctx).Model(&models.Property{}), f)

	var total int64
	if err := q.Distinct("properties.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Re-apply filters for the page query: Count above consumed the Distinct.
	q = r.applyFilter(r.db.WithContext(ctx).Model(&models.Property{}), f).
		Group("properties.id")

	var props []models.Property
	err := q.Preload("City").Preload("Locality").
		Preload("Configurations", func(d *gorm.DB) *gorm.DB { return d.Order("price ASC") }).
		Preload("Media", func(d *gorm.DB) *gorm.DB {
			return d.Where("is_cover_image = ? OR media_type = ?", true, models.MediaPropertyImage).
				Order("is_cover_image DESC, display_order ASC")
		}).
		Preload("RERADetails").
		Preload("RentalDetails").
		Order(orderClause(f)).
		Offset(f.Offset).Limit(f.Limit).
		Find(&props).Error
	return props, total, err
}

func orderClause(f PropertyFilter) string {
	col, ok := sortColumns[strings.ToLower(f.SortBy)]
	if !ok {
		// Relevance default: featured first, then freshest.
		return "properties.is_featured DESC, properties.created_at DESC"
	}
	dir := "DESC"
	switch {
	case f.SortBy == "price_desc" || f.SortBy == "rent_desc":
		dir = "DESC"
	case strings.EqualFold(f.SortOrder, "asc"),
		f.SortBy == "price_asc", f.SortBy == "rent_asc",
		f.SortBy == "name", f.SortBy == "area":
		dir = "ASC"
	}
	return col + " " + dir
}

func (r *propertyRepo) applyFilter(q *gorm.DB, f PropertyFilter) *gorm.DB {
	statuses := f.Statuses
	if len(statuses) == 0 {
		statuses = PublicStatuses
	}
	q = q.Where("properties.property_status IN ?", statuses)

	if f.Purpose != "" {
		q = q.Where("properties.purpose = ?", strings.ToUpper(f.Purpose))
	}
	if len(f.AvailabilityStatuses) > 0 {
		q = q.Where("properties.availability_status IN ?", f.AvailabilityStatuses)
	}

	// Rental criteria live on rental_details, so the join is added once and only
	// when one of them is actually in play.
	if f.needsRentalJoin() {
		q = q.Joins("JOIN rental_details rd ON rd.property_id = properties.id AND rd.deleted_at IS NULL")

		if f.MinRent > 0 {
			q = q.Where("rd.monthly_rent >= ?", f.MinRent)
		}
		if f.MaxRent > 0 {
			q = q.Where("rd.monthly_rent <= ? AND rd.monthly_rent > 0", f.MaxRent)
		}
		if f.Furnishing != "" {
			q = q.Where("rd.furnishing_type = ?", strings.ToUpper(f.Furnishing))
		}
		if f.TenantPreference != "" {
			// A listing open to anyone matches every tenant-type search.
			q = q.Where("rd.tenant_preference IN ?",
				[]string{strings.ToUpper(f.TenantPreference), string(models.TenantAnyone)})
		}
		if f.PetAllowed != nil {
			q = q.Where("rd.pet_allowed = ?", *f.PetAllowed)
		}
		if f.AvailableBefore != nil {
			// A listing with no stated date is available now, so it qualifies.
			q = q.Where("rd.available_from IS NULL OR rd.available_from <= ?", *f.AvailableBefore)
		}
	}

	if f.ClientID > 0 {
		q = q.Where("properties.client_id = ?", f.ClientID)
	}
	if f.CityID > 0 {
		q = q.Where("properties.city_id = ?", f.CityID)
	}
	if f.LocalityID > 0 {
		q = q.Where("properties.locality_id = ?", f.LocalityID)
	}
	if f.CitySlug != "" {
		q = q.Joins("JOIN cities ON cities.id = properties.city_id").Where("cities.slug = ?", f.CitySlug)
	}
	if f.LocalitySlug != "" {
		q = q.Joins("JOIN localities ON localities.id = properties.locality_id").
			Where("localities.slug = ?", f.LocalitySlug)
	}
	if f.PropertyType != "" {
		q = q.Where("properties.property_type = ?", strings.ToUpper(f.PropertyType))
	}
	if f.ConstructionStatus != "" {
		q = q.Where("properties.construction_status = ?", strings.ToUpper(f.ConstructionStatus))
	}
	if f.MinPrice > 0 {
		q = q.Where("properties.max_price >= ?", f.MinPrice)
	}
	if f.MaxPrice > 0 {
		q = q.Where("properties.min_price <= ? AND properties.min_price > 0", f.MaxPrice)
	}
	if f.MinArea > 0 {
		q = q.Where("properties.max_area >= ?", f.MinArea)
	}
	if f.MaxArea > 0 {
		q = q.Where("properties.min_area <= ? AND properties.min_area > 0", f.MaxArea)
	}
	if f.BHK > 0 {
		q = q.Where("EXISTS (SELECT 1 FROM property_configurations pc WHERE pc.property_id = properties.id AND pc.bedrooms = ? AND pc.deleted_at IS NULL)", f.BHK)
	}
	if len(f.AmenityIDs) > 0 {
		// Require every requested amenity, not just any of them.
		q = q.Where(`(SELECT COUNT(DISTINCT pa.amenity_id) FROM property_amenities pa
			WHERE pa.property_id = properties.id AND pa.amenity_id IN ?) = ?`,
			f.AmenityIDs, len(f.AmenityIDs))
	}
	if f.BuilderName != "" {
		q = q.Where("properties.builder_name LIKE ?", "%"+f.BuilderName+"%")
	}
	if f.PossessionBefore != nil {
		q = q.Where("properties.possession_date IS NOT NULL AND properties.possession_date <= ?", *f.PossessionBefore)
	}
	if f.IsFeatured != nil {
		q = q.Where("properties.is_featured = ?", *f.IsFeatured)
	}
	if f.IsRERAVerified != nil {
		q = q.Where("properties.is_rera_verified = ?", *f.IsRERAVerified)
	}
	if f.ExcludeID > 0 {
		q = q.Where("properties.id <> ?", f.ExcludeID)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where(`properties.name LIKE ? OR properties.project_name LIKE ?
			OR properties.builder_name LIKE ? OR properties.address LIKE ?`,
			like, like, like, like)
	}
	return q
}

func (r *propertyRepo) ListByIDs(ctx context.Context, ids []uint) ([]models.Property, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var props []models.Property
	err := fullPreload(r.db.WithContext(ctx)).Where("id IN ?", ids).Find(&props).Error
	return props, err
}

func (r *propertyRepo) SlugExists(ctx context.Context, slug string, excludeID uint) (bool, error) {
	q := r.db.WithContext(ctx).Model(&models.Property{}).Where("slug = ?", slug)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

// ChangeStatus moves a property through the approval workflow and appends the
// transition to its history in the same transaction.
func (r *propertyRepo) ChangeStatus(ctx context.Context, id uint, to models.PropertyStatus, actorID uint, remarks string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p models.Property
		if err := tx.First(&p, id).Error; err != nil {
			return translate(err)
		}
		from := p.PropertyStatus
		now := time.Now()

		updates := map[string]any{"property_status": to}
		switch to {
		case models.PropertyStatusSubmitted:
			updates["submitted_at"] = now
			updates["rejection_reason"] = ""
		case models.PropertyStatusApproved, models.PropertyStatusPublished:
			updates["approved_at"] = now
			updates["approved_by"] = actorID
			updates["rejection_reason"] = ""
		case models.PropertyStatusRejected:
			updates["rejected_at"] = now
			updates["rejected_by"] = actorID
			updates["rejection_reason"] = remarks
		}
		if err := tx.Model(&models.Property{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Create(&models.PropertyStatusHistory{
			PropertyID: id, FromStatus: from, ToStatus: to,
			ActorID: actorID, Remarks: remarks,
		}).Error
	})
}

func (r *propertyRepo) CountByStatus(ctx context.Context, status models.PropertyStatus, clientID uint) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Property{}).Where("property_status = ?", status)
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *propertyRepo) CountAll(ctx context.Context, clientID uint) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Property{})
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *propertyRepo) StatusHistory(ctx context.Context, propertyID uint) ([]models.PropertyStatusHistory, error) {
	var list []models.PropertyStatusHistory
	err := r.db.WithContext(ctx).Where("property_id = ?", propertyID).
		Order("created_at DESC").Find(&list).Error
	return list, err
}

// IncrementCounter bumps a denormalised engagement counter atomically.
func (r *propertyRepo) IncrementCounter(ctx context.Context, id uint, column string, delta int64) error {
	allowed := map[string]bool{
		"view_count": true, "inquiry_count": true,
		"whatsapp_click_count": true, "favorite_count": true,
	}
	if !allowed[column] {
		return fmt.Errorf("unsupported counter column %q", column)
	}
	return r.db.WithContext(ctx).Model(&models.Property{}).Where("id = ?", id).
		UpdateColumn(column, gorm.Expr(column+" + ?", delta)).Error
}

// Similar finds comparable published listings: same locality first, falling back
// to the same city within a similar price band.
func (r *propertyRepo) Similar(ctx context.Context, p *models.Property, limit int) ([]models.Property, error) {
	lower, upper := p.MinPrice*0.7, p.MaxPrice*1.3
	if upper == 0 {
		upper = 1 << 62
	}
	var props []models.Property
	err := r.db.WithContext(ctx).Model(&models.Property{}).
		Preload("City").Preload("Locality").Preload("RERADetails").
		Preload("Configurations").
		Preload("Media", func(d *gorm.DB) *gorm.DB {
			return d.Order("is_cover_image DESC, display_order ASC")
		}).
		Where("properties.id <> ? AND properties.property_status IN ?", p.ID, PublicStatuses).
		Where("properties.city_id = ?", p.CityID).
		Where("(properties.locality_id = ? OR (properties.min_price BETWEEN ? AND ?))",
			p.LocalityID, lower, upper).
		Order(gorm.Expr("properties.locality_id = ? DESC, properties.view_count DESC", p.LocalityID)).
		Limit(limit).Find(&props).Error
	return props, err
}

func (r *propertyRepo) AllPublishedSlugs(ctx context.Context) ([]SlugEntry, error) {
	var entries []SlugEntry
	err := r.db.WithContext(ctx).Model(&models.Property{}).
		Where("property_status IN ?", PublicStatuses).
		Order("updated_at DESC").
		Select("slug, updated_at").Scan(&entries).Error
	return entries, err
}

func (r *propertyRepo) SetFeatured(ctx context.Context, id uint, featured bool) error {
	res := r.db.WithContext(ctx).Model(&models.Property{}).Where("id = ?", id).
		Update("is_featured", featured)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// SetRERAVerification records an admin decision on a RERA registration and keeps
// the denormalised flag on the property in step.
func (r *propertyRepo) SetRERAVerification(ctx context.Context, propertyID uint, status models.RERAStatus, adminID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		res := tx.Model(&models.RERADetails{}).Where("property_id = ?", propertyID).
			Updates(map[string]any{
				"verification_status": status,
				"verified_by":         adminID,
				"verified_at":         now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return tx.Model(&models.Property{}).Where("id = ?", propertyID).
			Update("is_rera_verified", status == models.RERAStatusVerified).Error
	})
}

// needsRentalJoin reports whether any rental criterion is in play. Sorting by
// rent counts, because the order clause references the joined table.
func (f PropertyFilter) needsRentalJoin() bool {
	return f.MinRent > 0 || f.MaxRent > 0 ||
		f.Furnishing != "" || f.TenantPreference != "" ||
		f.PetAllowed != nil || f.AvailableBefore != nil ||
		strings.HasPrefix(strings.ToLower(f.SortBy), "rent")
}

// SetAvailability records whether a rental is still on the market. It is kept
// apart from ChangeStatus because approval state and availability are
// independent facts: a listing stays PUBLISHED after it is marked RENTED, and
// collapsing the two would make "rented" indistinguishable from "rejected".
func (r *propertyRepo) SetAvailability(ctx context.Context, id uint, status models.AvailabilityStatus) error {
	res := r.db.WithContext(ctx).Model(&models.Property{}).Where("id = ?", id).
		Update("availability_status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ExtendListing pushes a rental's expiry date out. Renewing is how an owner
// says the flat is still free without editing and resubmitting the listing.
func (r *propertyRepo) ExtendListing(ctx context.Context, id uint, until time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.Property{}).Where("id = ?", id).
		Update("listing_expiry_date", until)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ExpiringRentals returns live rentals whose expiry falls in a window, which is
// what both the reminder notice and the expiry sweep are built on. Passing a
// zero `from` selects everything already past `to`.
func (r *propertyRepo) ExpiringRentals(ctx context.Context, from, to time.Time) ([]models.Property, error) {
	var out []models.Property
	q := r.db.WithContext(ctx).
		Where("purpose = ?", models.PurposeRent).
		Where("property_status = ?", models.PropertyStatusPublished).
		Where("availability_status = ?", models.AvailabilityAvailable).
		Where("listing_expiry_date IS NOT NULL AND listing_expiry_date <= ?", to)
	if !from.IsZero() {
		q = q.Where("listing_expiry_date > ?", from)
	}
	if err := q.Order("listing_expiry_date ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// SetProvenance records who really posted a listing. It exists for the case
// where an administrator posts on behalf of an owner who is not at a keyboard:
// the listing belongs to the owner's account, but the trust badge must not
// claim the owner wrote it.
func (r *propertyRepo) SetProvenance(ctx context.Context, id uint, postedBy models.PostedByType, ownerUserID *uint) error {
	res := r.db.WithContext(ctx).Model(&models.Property{}).Where("id = ?", id).
		Updates(map[string]any{"posted_by_type": postedBy, "owner_user_id": ownerUserID})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
