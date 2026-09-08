package server_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
)

// publishRental creates a published rental listing directly. Phase 03 wires the
// search dimension only — the owner-facing posting flow arrives in phase 04, so
// the rental row is written through the model here.
func (env *testEnv) publishRental(t *testing.T, clientID, userID, cityID, localityID uint,
	name string, rent float64, furnishing models.FurnishingType, availableIn int) uint {
	t.Helper()

	property := models.Property{
		ClientID: clientID, UserID: userID,
		Name: name, Slug: slugFor(name),
		PropertyType:       models.PropertyTypeFlat,
		PropertyStatus:     models.PropertyStatusPublished,
		ConstructionStatus: models.ConstructionReadyToMove,
		Purpose:            models.PurposeRent,
		PostedByType:       models.PostedByOwner,
		AvailabilityStatus: models.AvailabilityAvailable,
		CityID:             cityID, LocalityID: localityID,
		Address: "Test Road", Description: "A rental used by the purpose tests.",
	}
	if err := env.db.Create(&property).Error; err != nil {
		t.Fatalf("create rental property: %v", err)
	}

	config := models.PropertyConfiguration{
		PropertyID: property.ID, Name: "2 BHK", Bedrooms: 2, Bathrooms: 2,
		CarpetArea: 850, Price: rent,
	}
	if err := env.db.Create(&config).Error; err != nil {
		t.Fatalf("create configuration: %v", err)
	}

	availableFrom := time.Now().AddDate(0, 0, availableIn)
	rental := models.RentalDetails{
		PropertyID: property.ID, MonthlyRent: rent, SecurityDeposit: rent * 3,
		AvailableFrom: &availableFrom, FurnishingType: furnishing,
		TenantPreference: models.TenantAnyone, MinLeaseMonths: 11, NoticePeriodDays: 30,
	}
	if err := env.db.Create(&rental).Error; err != nil {
		t.Fatalf("create rental details: %v", err)
	}

	// A cover image keeps the card payload realistic.
	media := models.PropertyMedia{
		PropertyID: property.ID, MediaType: models.MediaPropertyImage,
		FileURL: "https://example.com/rental.jpg", IsCoverImage: true,
	}
	if err := env.db.Create(&media).Error; err != nil {
		t.Fatalf("create media: %v", err)
	}
	return property.ID
}

func slugFor(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+32)
		case r == ' ':
			out = append(out, '-')
		}
	}
	return string(out)
}

// rentalFixture publishes one sale listing and three rentals at different rents,
// furnishing levels and availability dates.
func (env *testEnv) rentalFixture(t *testing.T) (cityID, localityID uint) {
	t.Helper()
	cityID, localityID = env.seedLocation(t)

	builderToken, _ := env.register(t, "client", "rentfix@example.com", "9800000301", "RentFix")
	_, _ = env.register(t, "customer", "rentadmin@example.com", "9800000302", "Admin")
	adminToken := env.promoteToAdmin(t, "rentadmin@example.com")

	// One ordinary sale listing, published the normal way.
	created := env.do(t, http.MethodPost, "/api/v1/client/properties", builderToken,
		propertyPayload(cityID, localityID, "Sale Residency"))
	env.publish(t, int(created.Data["id"].(float64)), builderToken, adminToken)

	var client models.Client
	if err := env.db.Joins("JOIN users ON users.id = clients.user_id").
		Where("users.email = ?", "rentfix@example.com").First(&client).Error; err != nil {
		t.Fatalf("load listing account: %v", err)
	}

	env.publishRental(t, client.ID, client.UserID, cityID, localityID,
		"Budget Rental", 18000, models.FurnishingUnfurnished, 0)
	env.publishRental(t, client.ID, client.UserID, cityID, localityID,
		"Mid Rental", 28000, models.FurnishingSemiFurnished, 5)
	env.publishRental(t, client.ID, client.UserID, cityID, localityID,
		"Premium Rental", 55000, models.FurnishingFullyFurnished, 45)

	return cityID, localityID
}

// The gate for this phase: the Buy search must behave exactly as before, with
// rentals invisible to it.
func TestBuySearchIsUnaffectedByRentals(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)

	res := env.do(t, http.MethodGet, "/api/v1/properties", "", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("search: status %d (%s)", res.Status, res.Message)
	}
	if len(res.List) != 1 {
		t.Fatalf("default search returned %d listings, want only the 1 sale listing", len(res.List))
	}

	first, _ := res.List[0].(map[string]any)
	if first["purpose"] != "BUY" {
		t.Errorf("purpose = %v, want BUY", first["purpose"])
	}
	if first["name"] != "Sale Residency" {
		t.Errorf("name = %v, want the sale listing", first["name"])
	}
}

func TestRentSearchReturnsOnlyRentals(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)

	res := env.do(t, http.MethodGet, "/api/v1/properties?purpose=rent", "", nil)
	if len(res.List) != 3 {
		t.Fatalf("rent search returned %d listings, want 3", len(res.List))
	}
	for _, item := range res.List {
		card, _ := item.(map[string]any)
		if card["purpose"] != "RENT" {
			t.Errorf("a %v listing appeared in the rent search", card["purpose"])
		}
		if card["monthly_rent"] == nil {
			t.Errorf("%v has no monthly rent on its card", card["name"])
		}
	}
}

func TestPurposeAllReturnsBothCatalogues(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)

	res := env.do(t, http.MethodGet, "/api/v1/properties?purpose=all", "", nil)
	if len(res.List) != 4 {
		t.Fatalf("purpose=all returned %d listings, want 4", len(res.List))
	}
}

func TestRentRangeFilter(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)

	cases := []struct {
		query string
		want  int
		note  string
	}{
		{"purpose=rent&max_rent=20000", 1, "only the budget rental"},
		{"purpose=rent&min_rent=20000&max_rent=30000", 1, "only the mid rental"},
		{"purpose=rent&min_rent=20000", 2, "mid and premium"},
		{"purpose=rent&min_rent=100000", 0, "nothing that expensive"},
	}

	for _, tc := range cases {
		res := env.do(t, http.MethodGet, "/api/v1/properties?"+tc.query, "", nil)
		if len(res.List) != tc.want {
			t.Errorf("%s: got %d listings, want %d (%s)", tc.query, len(res.List), tc.want, tc.note)
		}
	}
}

func TestFurnishingFilter(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)

	for furnishing, want := range map[string]int{
		"UNFURNISHED":     1,
		"SEMI_FURNISHED":  1,
		"FULLY_FURNISHED": 1,
	} {
		res := env.do(t, http.MethodGet,
			"/api/v1/properties?purpose=rent&furnishing="+furnishing, "", nil)
		if len(res.List) != want {
			t.Errorf("furnishing=%s: got %d, want %d", furnishing, len(res.List), want)
		}
	}
}

// "Available within 7 days" must include a flat already available, and exclude
// one that frees up in six weeks.
func TestAvailabilityFilter(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)

	within7 := env.do(t, http.MethodGet,
		"/api/v1/properties?purpose=rent&available_from=7d", "", nil)
	if len(within7.List) != 2 {
		t.Errorf("available within 7 days: got %d, want 2", len(within7.List))
	}

	within30 := env.do(t, http.MethodGet,
		"/api/v1/properties?purpose=rent&available_from=30d", "", nil)
	if len(within30.List) != 2 {
		t.Errorf("available within 30 days: got %d, want 2", len(within30.List))
	}
}

func TestRentSorting(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)

	asc := env.do(t, http.MethodGet, "/api/v1/properties?purpose=rent&sort=rent_asc", "", nil)
	if len(asc.List) != 3 {
		t.Fatalf("got %d listings, want 3", len(asc.List))
	}
	first, _ := asc.List[0].(map[string]any)
	last, _ := asc.List[2].(map[string]any)
	if fmt.Sprint(first["monthly_rent"]) != "18000" {
		t.Errorf("cheapest first = %v, want 18000", first["monthly_rent"])
	}
	if fmt.Sprint(last["monthly_rent"]) != "55000" {
		t.Errorf("most expensive last = %v, want 55000", last["monthly_rent"])
	}

	desc := env.do(t, http.MethodGet, "/api/v1/properties?purpose=rent&sort=rent_desc", "", nil)
	top, _ := desc.List[0].(map[string]any)
	if fmt.Sprint(top["monthly_rent"]) != "55000" {
		t.Errorf("most expensive first = %v, want 55000", top["monthly_rent"])
	}
}

// A rented flat must leave the search the moment it is marked so, whatever its
// approval status.
func TestRentedListingsLeaveTheSearch(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.rentalFixture(t)
	_ = localityID

	before := env.do(t, http.MethodGet, "/api/v1/properties?purpose=rent", "", nil)
	if len(before.List) != 3 {
		t.Fatalf("got %d rentals before, want 3", len(before.List))
	}

	if err := env.db.Model(&models.Property{}).
		Where("purpose = ? AND name = ?", models.PurposeRent, "Mid Rental").
		Update("availability_status", models.AvailabilityRented).Error; err != nil {
		t.Fatalf("mark rented: %v", err)
	}

	after := env.do(t, http.MethodGet, "/api/v1/properties?purpose=rent", "", nil)
	if len(after.List) != 2 {
		t.Errorf("got %d rentals after marking one rented, want 2", len(after.List))
	}
	for _, item := range after.List {
		card, _ := item.(map[string]any)
		if card["name"] == "Mid Rental" {
			t.Error("a rented listing is still in the search results")
		}
	}
	_ = cityID
}

// The rental terms must reach the detail payload, and stay absent from a sale
// listing rather than appearing empty.
func TestRentalDetailsOnDetailPayload(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)

	rental := env.do(t, http.MethodGet, "/api/v1/properties/mid-rental", "", nil)
	if rental.Status != http.StatusOK {
		t.Fatalf("rental detail: status %d (%s)", rental.Status, rental.Message)
	}
	details, ok := rental.Data["rental_details"].(map[string]any)
	if !ok {
		t.Fatal("rental_details is missing from a rental payload")
	}
	if fmt.Sprint(details["monthly_rent"]) != "28000" {
		t.Errorf("monthly_rent = %v, want 28000", details["monthly_rent"])
	}
	if details["furnishing_type"] != "SEMI_FURNISHED" {
		t.Errorf("furnishing_type = %v", details["furnishing_type"])
	}

	// Sale listings get their slug from the service, which appends the locality
	// and city: "sale-residency-baner-pune".
	var saleProperty models.Property
	if err := env.db.Where("name = ?", "Sale Residency").First(&saleProperty).Error; err != nil {
		t.Fatalf("load sale listing: %v", err)
	}

	sale := env.do(t, http.MethodGet, "/api/v1/properties/"+saleProperty.Slug, "", nil)
	if sale.Status != http.StatusOK {
		t.Fatalf("sale detail: status %d (%s)", sale.Status, sale.Message)
	}
	if _, present := sale.Data["rental_details"]; present {
		t.Error("a sale listing carries a rental_details block")
	}
	if sale.Data["purpose"] != "BUY" {
		t.Errorf("sale purpose = %v, want BUY", sale.Data["purpose"])
	}
}

// Rental criteria must not be applied to a sale search: doing so would join
// rental_details and silently return nothing.
func TestRentalFiltersIgnoredOnBuySearch(t *testing.T) {
	env := setup(t)
	env.rentalFixture(t)

	res := env.do(t, http.MethodGet,
		"/api/v1/properties?purpose=buy&min_rent=1000&max_rent=5000&furnishing=UNFURNISHED", "", nil)
	if len(res.List) != 1 {
		t.Errorf("buy search with stray rental filters returned %d, want the 1 sale listing",
			len(res.List))
	}
}
