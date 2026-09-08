package server_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// countRentResults reports how many rentals the public search returns, which is
// the only way to prove that a let flat really left the listings.
func (env *testEnv) countRentResults(t *testing.T) int {
	t.Helper()
	res := env.do(t, http.MethodGet, "/api/v1/properties?purpose=rent", "", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("rent search: status %d (%s)", res.Status, res.Message)
	}
	return len(res.List)
}

// publishedSale gives the availability tests an ordinary sale listing to prove
// the rental-only rules against.
func (env *testEnv) publishedSale(t *testing.T, suffix string) (int, string) {
	t.Helper()
	cityID, localityID := env.seedLocation(t)
	builderToken, _ := env.register(t, "client",
		"sale"+suffix+"@example.com", "97000088"+suffix, "Sale Co")
	_, _ = env.register(t, "customer",
		"saleadmin"+suffix+"@example.com", "97000089"+suffix, "Admin")
	adminToken := env.promoteToAdmin(t, "saleadmin"+suffix+"@example.com")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", builderToken,
		propertyPayload(cityID, localityID, "Sale Tower "+suffix))
	if created.Status != http.StatusCreated {
		t.Fatalf("create sale listing: status %d (%s)", created.Status, created.Message)
	}
	id := int(created.Data["id"].(float64))
	env.publish(t, id, builderToken, adminToken)
	return id, builderToken
}

// A let flat must leave the search results without the owner deleting the
// listing, and must come back in one click when the tenant moves out.
func TestOwnerMarksARentalRentedAndRelists(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "30")

	if got := env.countRentResults(t); got != 1 {
		t.Fatalf("published rentals in search = %d, want 1", got)
	}

	res := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/client/properties/%d/availability", propertyID), ownerToken,
		map[string]any{"availability_status": "RENTED"})
	if res.Status != http.StatusOK {
		t.Fatalf("mark rented: status %d (%s)", res.Status, res.Message)
	}
	if res.Data["availability_status"] != "RENTED" {
		t.Errorf("availability_status = %v, want RENTED", res.Data["availability_status"])
	}
	// The approval state is untouched: the owner keeps the listing's history,
	// its leads and its analytics.
	if res.Data["property_status"] != "PUBLISHED" {
		t.Errorf("property_status = %v, want PUBLISHED", res.Data["property_status"])
	}
	if got := env.countRentResults(t); got != 0 {
		t.Errorf("a rented flat is still in the search results (%d)", got)
	}

	// Relisting puts it straight back, with a fresh availability date.
	back := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/client/properties/%d/availability", propertyID), ownerToken,
		map[string]any{"availability_status": "AVAILABLE", "available_from": nextWeek()})
	if back.Status != http.StatusOK {
		t.Fatalf("relist: status %d (%s)", back.Status, back.Message)
	}
	if got := env.countRentResults(t); got != 1 {
		t.Errorf("relisted rentals in search = %d, want 1", got)
	}
	rental, _ := back.Data["rental_details"].(map[string]any)
	if rental == nil || rental["available_from"] == nil {
		t.Fatalf("rental details missing after relisting: %v", back.Data["rental_details"])
	}
	if got := fmt.Sprint(rental["available_from"]); len(got) < 10 || got[:10] != nextWeek() {
		t.Errorf("available_from = %s, want %s", got, nextWeek())
	}
}

// Availability is a rental concept. A flat for sale cannot be "rented".
func TestAvailabilityIsRejectedOnASaleListing(t *testing.T) {
	env := setup(t)
	propertyID, token := env.publishedSale(t, "40")

	res := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/client/properties/%d/availability", propertyID), token,
		map[string]any{"availability_status": "RENTED"})
	if res.Status != http.StatusBadRequest {
		t.Fatalf("status %d (%s), want 400", res.Status, res.Message)
	}
}

// Renewing is not an edit: confirming a flat is still free must not push the
// listing back through approval.
func TestRenewingExtendsExpiryWithoutReview(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "31")

	before := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/client/properties/%d", propertyID), ownerToken, nil)
	firstExpiry := fmt.Sprint(before.Data["listing_expiry_date"])

	// Move the expiry into the past, as it would be for a stale listing.
	env.db.Exec("UPDATE properties SET listing_expiry_date = ?, availability_status = 'INACTIVE' WHERE id = ?",
		time.Now().AddDate(0, 0, -1), propertyID)

	res := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/renew", propertyID), ownerToken, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("renew: status %d (%s)", res.Status, res.Message)
	}
	if res.Data["property_status"] != "PUBLISHED" {
		t.Errorf("property_status = %v, want PUBLISHED — renewing is not an edit", res.Data["property_status"])
	}
	// An expired listing was taken off the market; renewing puts it back.
	if res.Data["availability_status"] != "AVAILABLE" {
		t.Errorf("availability_status = %v, want AVAILABLE", res.Data["availability_status"])
	}
	// Renewing runs from today, not from the old expiry, so a listing renewed
	// late still gets a full period rather than one already half spent. The
	// listing above was pushed into the past, so the renewed date must be ahead
	// of today whatever the original was.
	renewed := fmt.Sprint(res.Data["listing_expiry_date"])
	if renewed == "<nil>" {
		t.Fatalf("the response carries no expiry date (original was %s)", firstExpiry)
	}
	parsed, err := time.Parse("2006-01-02", renewed)
	if err != nil {
		t.Fatalf("listing_expiry_date %q is not a date: %v", renewed, err)
	}
	if !parsed.After(time.Now()) {
		t.Errorf("renewed expiry %s is not in the future", renewed)
	}
}

// One owner must not be able to relist or renew another owner's flat.
func TestAvailabilityAndRenewalAreOwnerScoped(t *testing.T) {
	env := setup(t)
	propertyID, _ := env.publishedRental(t, "32")
	intruder := env.registerOwner(t, "intruder32@example.com", "9700007320", "Intruder")

	res := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/client/properties/%d/availability", propertyID), intruder,
		map[string]any{"availability_status": "RENTED"})
	if res.Status != http.StatusNotFound {
		t.Errorf("availability: status %d (%s), want 404", res.Status, res.Message)
	}

	renew := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/renew", propertyID), intruder, nil)
	if renew.Status != http.StatusNotFound {
		t.Errorf("renew: status %d (%s), want 404", renew.Status, renew.Message)
	}
}

// The owner dashboard warns about listings that lapse soon.
func TestExpiringListingsPanelListsWhatLapsesSoon(t *testing.T) {
	env := setup(t)
	propertyID, ownerToken := env.publishedRental(t, "33")

	empty := env.do(t, http.MethodGet,
		"/api/v1/client/expiring-listings?within_days=3", ownerToken, nil)
	if len(empty.List) != 0 {
		t.Fatalf("a listing with a full period left is already flagged: %v", empty.List)
	}

	env.db.Exec("UPDATE properties SET listing_expiry_date = ? WHERE id = ?",
		time.Now().AddDate(0, 0, 2), propertyID)

	soon := env.do(t, http.MethodGet,
		"/api/v1/client/expiring-listings?within_days=3", ownerToken, nil)
	if len(soon.List) != 1 {
		t.Fatalf("expiring listings = %d, want 1", len(soon.List))
	}
	row, _ := soon.List[0].(map[string]any)
	if row == nil || row["id"].(float64) != float64(propertyID) {
		t.Errorf("wrong listing flagged: %v", soon.List[0])
	}
}
