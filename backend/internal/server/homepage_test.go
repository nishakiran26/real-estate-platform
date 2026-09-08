package server_test

import (
	"fmt"
	"net/http"
	"testing"
)

// homepageSection returns the purposes present in one homepage strip.
func homepageSection(t *testing.T, data map[string]any, key string) map[string]int {
	t.Helper()
	rows, _ := data[key].([]any)
	out := map[string]int{}
	for _, row := range rows {
		card, _ := row.(map[string]any)
		if card == nil {
			continue
		}
		purpose, _ := card["purpose"].(string)
		out[purpose]++
	}
	return out
}

// A monthly rent and a sale price are not comparable figures. Shown in one row,
// a flat listed at ₹17,000 a month reads as a flat costing ₹17,000 — so no sale
// strip on the homepage may carry a rental.
func TestHomepageNeverMixesRentWithSalePrices(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "70")

	res := env.do(t, http.MethodGet, "/api/v1/homepage", "", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("homepage: status %d (%s)", res.Status, res.Message)
	}

	for _, section := range []string{"featured", "new_launches", "ready_to_move", "upcoming"} {
		got := homepageSection(t, res.Data, section)
		if got["RENT"] != 0 {
			t.Errorf("%q carries %d rental(s) among the sale listings", section, got["RENT"])
		}
	}

	// The sale strips still work: the fix must not empty the homepage.
	if homepageSection(t, res.Data, "new_launches")["BUY"] == 0 {
		t.Error("new_launches lost its sale listings")
	}
}

// Rentals are not dropped from the homepage, only moved somewhere they read
// correctly.
func TestHomepageCarriesRentalsInTheirOwnSection(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "71")

	res := env.do(t, http.MethodGet, "/api/v1/homepage", "", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("homepage: status %d (%s)", res.Status, res.Message)
	}

	got := homepageSection(t, res.Data, "rentals")
	if got["RENT"] == 0 {
		t.Error("the rentals section is empty although a rental is published")
	}
	if got["BUY"] != 0 {
		t.Errorf("the rentals section carries %d sale listing(s)", got["BUY"])
	}
}

// A let flat on the homepage wastes the visitor's time.
func TestHomepageRentalsExcludeLetFlats(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "72")

	before := env.do(t, http.MethodGet, "/api/v1/homepage", "", nil)
	if homepageSection(t, before.Data, "rentals")["RENT"] == 0 {
		t.Fatal("no rental to take off the market")
	}

	env.db.Exec("UPDATE properties SET availability_status = 'RENTED' WHERE purpose = 'RENT'")

	after := env.do(t, http.MethodGet, "/api/v1/homepage", "", nil)
	if got := homepageSection(t, after.Data, "rentals")["RENT"]; got != 0 {
		t.Errorf("the homepage still advertises %d let flat(s)", got)
	}
}

// The same confusion in a different place: a comparison table puts the figures
// side by side, so a monthly rent beside a sale price reads as one flat being
// hundreds of times cheaper than the other.
func TestCompareRefusesToMixRentWithSale(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "73")

	home := env.do(t, http.MethodGet, "/api/v1/homepage", "", nil)
	saleRows, _ := home.Data["new_launches"].([]any)
	rentRows, _ := home.Data["rentals"].([]any)
	if len(saleRows) == 0 || len(rentRows) == 0 {
		t.Fatalf("fixture is missing a listing: %d sale, %d rent", len(saleRows), len(rentRows))
	}
	saleID := int(saleRows[0].(map[string]any)["id"].(float64))
	rentID := int(rentRows[0].(map[string]any)["id"].(float64))

	mixed := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/properties/compare?ids=%d,%d", saleID, rentID), "", nil)
	if mixed.Status != http.StatusBadRequest {
		t.Errorf("comparing a rental with a sale listing: status %d (%s), want 400",
			mixed.Status, mixed.Message)
	}
}
