package server_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
)

// A single "properties" figure hides which half of the business is growing, so
// both dashboards split the catalogue by purpose.
func TestDashboardsSplitTheCatalogueByPurpose(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)
	adminToken := env.promoteToAdmin(t, "rentadmin@example.com")

	admin := env.do(t, http.MethodGet, "/api/v1/admin/dashboard", adminToken, nil)
	if admin.Status != http.StatusOK {
		t.Fatalf("admin dashboard: status %d (%s)", admin.Status, admin.Message)
	}
	stats, _ := admin.Data["stats"].(map[string]any)
	if stats == nil {
		t.Fatal("the admin dashboard carries no stats block")
	}
	if stats["sale_properties"] != float64(1) {
		t.Errorf("sale_properties = %v, want 1", stats["sale_properties"])
	}
	if stats["rental_properties"] != float64(3) {
		t.Errorf("rental_properties = %v, want 3", stats["rental_properties"])
	}
	// The buy figures are the ones that existed before rentals arrived; they
	// must still be right with rentals in the same table.
	if stats["total_properties"] != float64(4) {
		t.Errorf("total_properties = %v, want 4", stats["total_properties"])
	}
}

// An enquiry about a flat to let is a different kind of lead from an enquiry
// about a flat to buy, and the owner dashboard has to keep them apart.
func TestClientDashboardSplitsLeadsByType(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "60")

	res := env.do(t, http.MethodPost, "/api/v1/inquiries", "", map[string]any{
		"property_id": propertyID, "name": "Meera Joshi", "mobile": "9812345601",
		"email": "meera@example.com", "message": "Is it still available?",
		"consent_contact": true,
	})
	if res.Status != http.StatusCreated {
		t.Fatalf("inquiry: status %d (%s)", res.Status, res.Message)
	}

	dash := env.do(t, http.MethodGet, "/api/v1/client/dashboard", ownerToken, nil)
	if dash.Status != http.StatusOK {
		t.Fatalf("client dashboard: status %d (%s)", dash.Status, dash.Message)
	}
	stats, _ := dash.Data["stats"].(map[string]any)
	if stats == nil {
		t.Fatal("the client dashboard carries no stats block")
	}
	if stats["rent_leads"] != float64(1) {
		t.Errorf("rent_leads = %v, want 1", stats["rent_leads"])
	}
	if stats["buy_leads"] != float64(0) {
		t.Errorf("buy_leads = %v, want 0", stats["buy_leads"])
	}
	if stats["rental_properties"] != float64(1) {
		t.Errorf("rental_properties = %v, want 1", stats["rental_properties"])
	}
}

// The owner dashboard has to show what needs doing: viewings waiting on a
// reply, and listings about to lapse.
func TestClientDashboardCountsPendingVisitsAndExpiries(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "61")

	env.do(t, http.MethodPost, fmt.Sprintf("/api/v1/properties/%d/visits", propertyID), "",
		visitPayload(tomorrow()))
	env.db.Exec("UPDATE properties SET listing_expiry_date = ? WHERE id = ?",
		time.Now().AddDate(0, 0, 2), propertyID)

	dash := env.do(t, http.MethodGet, "/api/v1/client/dashboard", ownerToken, nil)
	stats, _ := dash.Data["stats"].(map[string]any)
	if stats == nil {
		t.Fatal("the client dashboard carries no stats block")
	}
	if stats["pending_visits"] != float64(1) {
		t.Errorf("pending_visits = %v, want 1", stats["pending_visits"])
	}
	if stats["expiring_soon"] != float64(1) {
		t.Errorf("expiring_soon = %v, want 1", stats["expiring_soon"])
	}
}

// The rent guide is derived from live listings, and says how many are behind
// the figures so a thin sample is visible as one.
func TestRentGuideSummarisesLiveListingsOnly(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.rentalFixture(t)

	guide := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/rent-guide?city_id=%d&locality_id=%d", cityID, localityID), "", nil)
	if guide.Status != http.StatusOK {
		t.Fatalf("rent guide: status %d (%s)", guide.Status, guide.Message)
	}
	// The fixture publishes rentals at 18000, 28000 and 55000.
	if guide.Data["listings"] != float64(3) {
		t.Fatalf("listings = %v, want 3", guide.Data["listings"])
	}
	if guide.Data["min_rent"] != float64(18000) {
		t.Errorf("min_rent = %v, want 18000", guide.Data["min_rent"])
	}
	if guide.Data["max_rent"] != float64(55000) {
		t.Errorf("max_rent = %v, want 55000", guide.Data["max_rent"])
	}
	if guide.Data["average_rent"] != float64(33667) {
		t.Errorf("average_rent = %v, want 33667", guide.Data["average_rent"])
	}

	// A let flat is no longer a guide to what the place goes for.
	env.db.Exec("UPDATE properties SET availability_status = 'RENTED' WHERE name = ?", "Premium Rental")
	after := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/rent-guide?city_id=%d&locality_id=%d", cityID, localityID), "", nil)
	if after.Data["listings"] != float64(2) {
		t.Errorf("listings after letting one = %v, want 2", after.Data["listings"])
	}
	if after.Data["max_rent"] != float64(28000) {
		t.Errorf("max_rent after letting the dearest = %v, want 28000", after.Data["max_rent"])
	}
}

// An area with nothing in it must report no data rather than a made-up figure.
func TestRentGuideReportsAnEmptyAreaHonestly(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)

	guide := env.do(t, http.MethodGet, "/api/v1/rent-guide?locality_id=9999", "", nil)
	if guide.Status != http.StatusOK {
		t.Fatalf("rent guide: status %d (%s)", guide.Status, guide.Message)
	}
	if guide.Data["listings"] != float64(0) {
		t.Errorf("listings = %v, want 0", guide.Data["listings"])
	}
	if guide.Data["average_rent"] != float64(0) {
		t.Errorf("average_rent = %v, want 0", guide.Data["average_rent"])
	}
}

// Retiring a stale listing is reversible and never destructive: the approval
// state, the leads and the analytics all survive, and Renew brings it back.
func TestExpirySweepRetiresStaleRentalsWithoutLosingThem(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "62")
	adminToken := env.promoteToAdmin(t, "visitadmin62@example.com")

	env.db.Exec("UPDATE properties SET listing_expiry_date = ? WHERE id = ?",
		time.Now().AddDate(0, 0, -1), propertyID)

	res := env.do(t, http.MethodPost, "/api/v1/admin/listings/sweep-expiry", adminToken, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("sweep: status %d (%s)", res.Status, res.Message)
	}
	if res.Data["expired"] != float64(1) {
		t.Errorf("expired = %v, want 1", res.Data["expired"])
	}

	if got := env.countRentResults(t); got != 0 {
		t.Errorf("a lapsed listing is still searchable (%d results)", got)
	}

	// The listing is still the owner's, still published, and still theirs to renew.
	owned := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/client/properties/%d", propertyID), ownerToken, nil)
	if owned.Data["property_status"] != "PUBLISHED" {
		t.Errorf("property_status = %v, want PUBLISHED", owned.Data["property_status"])
	}
	if owned.Data["availability_status"] != "INACTIVE" {
		t.Errorf("availability_status = %v, want INACTIVE", owned.Data["availability_status"])
	}

	// The owner was told, rather than the listing silently vanishing.
	var client models.Client
	env.db.Where("id = ?", 1).First(&client)
	var notes int64
	env.db.Model(&models.Notification{}).
		Where("title LIKE ?", "Your listing has expired%").Count(&notes)
	if notes != 1 {
		t.Errorf("expiry notices sent = %d, want 1", notes)
	}

	renewed := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/renew", propertyID), ownerToken, nil)
	if renewed.Status != http.StatusOK {
		t.Fatalf("renew after expiry: status %d (%s)", renewed.Status, renewed.Message)
	}
	if got := env.countRentResults(t); got != 1 {
		t.Errorf("a renewed listing did not come back (%d results)", got)
	}
}

// Running the sweep twice must not retire or warn about the same listing again.
func TestExpirySweepIsSafeToRepeat(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "63")
	adminToken := env.promoteToAdmin(t, "visitadmin63@example.com")

	env.db.Exec("UPDATE properties SET listing_expiry_date = ? WHERE id = ?",
		time.Now().AddDate(0, 0, -1), propertyID)

	first := env.do(t, http.MethodPost, "/api/v1/admin/listings/sweep-expiry", adminToken, nil)
	if first.Data["expired"] != float64(1) {
		t.Fatalf("first sweep expired = %v, want 1", first.Data["expired"])
	}
	second := env.do(t, http.MethodPost, "/api/v1/admin/listings/sweep-expiry", adminToken, nil)
	if second.Data["expired"] != float64(0) {
		t.Errorf("second sweep expired = %v, want 0", second.Data["expired"])
	}

	var notes int64
	env.db.Model(&models.Notification{}).
		Where("title LIKE ?", "Your listing has expired%").Count(&notes)
	if notes != 1 {
		t.Errorf("expiry notices after two sweeps = %d, want 1", notes)
	}
}

// A sale listing has no expiry, so the sweep must never touch one.
func TestExpirySweepIgnoresSaleListings(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedSale(t, "64")
	adminToken := env.promoteToAdmin(t, "saleadmin64@example.com")

	res := env.do(t, http.MethodPost, "/api/v1/admin/listings/sweep-expiry", adminToken, nil)
	if res.Data["expired"] != float64(0) {
		t.Errorf("expired = %v, want 0", res.Data["expired"])
	}

	var property models.Property
	env.db.First(&property, propertyID)
	if property.AvailabilityStatus != models.AvailabilityAvailable {
		t.Errorf("a sale listing was retired: availability = %s", property.AvailabilityStatus)
	}
	if property.ListingExpiryDate != nil {
		t.Errorf("a sale listing has an expiry date: %v", property.ListingExpiryDate)
	}
}
