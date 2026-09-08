package server_test

import (
	"net/http"
	"testing"

	"github.com/rera/property-platform/backend/internal/models"
)

// intentFixture publishes one sale listing and one rental, then returns a
// customer token so the dashboard can be asked what it recommends.
func (env *testEnv) intentFixture(t *testing.T, suffix string) (cityID, localityID uint) {
	t.Helper()
	cityID, localityID = env.seedLocation(t)

	builderToken, _ := env.register(t, "client",
		"intentco"+suffix+"@example.com", "97000021"+suffix, "Intent Co")
	_, _ = env.register(t, "customer",
		"intentadmin"+suffix+"@example.com", "97000022"+suffix, "Admin")
	adminToken := env.promoteToAdmin(t, "intentadmin"+suffix+"@example.com")

	sale := env.do(t, http.MethodPost, "/api/v1/client/properties", builderToken,
		propertyPayload(cityID, localityID, "Intent Sale Tower "+suffix))
	if sale.Status != http.StatusCreated {
		t.Fatalf("create sale listing: status %d (%s)", sale.Status, sale.Message)
	}
	env.publish(t, int(sale.Data["id"].(float64)), builderToken, adminToken)

	rental := env.do(t, http.MethodPost, "/api/v1/client/properties", builderToken,
		rentalPayload(cityID, localityID, "Intent Rental "+suffix, 24000))
	if rental.Status != http.StatusCreated {
		t.Fatalf("create rental: status %d (%s)", rental.Status, rental.Message)
	}
	env.publish(t, int(rental.Data["id"].(float64)), builderToken, adminToken)

	return cityID, localityID
}

// recommendedPurposes reports which catalogues the dashboard recommended from.
func (env *testEnv) recommendedPurposes(t *testing.T, token string) map[string]int {
	t.Helper()
	res := env.do(t, http.MethodGet, "/api/v1/customer/recommendations", token, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("recommendations: status %d (%s)", res.Status, res.Message)
	}
	out := map[string]int{}
	for _, row := range res.List {
		card, _ := row.(map[string]any)
		if card == nil {
			continue
		}
		out[card["purpose"].(string)]++
	}
	return out
}

// The report behind this test: a customer who signed up to rent was being shown
// projects for sale on their dashboard.
func TestARenterIsNotRecommendedFlatsForSale(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "60")

	res := env.registerWith(t, map[string]any{
		"full_name": "Renter Only", "email": "renteronly60@example.com",
		"mobile": "9700002360", "password": "Password@123",
		"profile_type": "RENTER",
	})
	if res.Status != http.StatusCreated {
		t.Fatalf("register renter: status %d (%s)", res.Status, res.Message)
	}
	tokens, _ := res.Data["tokens"].(map[string]any)
	token, _ := tokens["access_token"].(string)

	got := env.recommendedPurposes(t, token)
	if got["BUY"] != 0 {
		t.Errorf("a renter was recommended %d listing(s) for sale", got["BUY"])
	}
	if got["RENT"] == 0 {
		t.Error("a renter was recommended no rentals at all")
	}
}

// The mirror image: a buyer must not be recommended tenancies.
func TestABuyerIsNotRecommendedRentals(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "61")

	token, _ := env.register(t, "customer", "buyeronly61@example.com", "9700002361", "Buyer Only")

	got := env.recommendedPurposes(t, token)
	if got["RENT"] != 0 {
		t.Errorf("a buyer was recommended %d rental(s)", got["RENT"])
	}
}

// Someone doing both should see both, rather than being forced into one.
func TestACustomerWithBothIntentsSeesBoth(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "62")

	res := env.registerWith(t, map[string]any{
		"full_name": "Both Intents", "email": "both62@example.com",
		"mobile": "9700002362", "password": "Password@123",
		"profile_type": "BUYER", "intents": []string{"BUY", "RENT"},
	})
	if res.Status != http.StatusCreated {
		t.Fatalf("register: status %d (%s)", res.Status, res.Message)
	}
	tokens, _ := res.Data["tokens"].(map[string]any)
	token, _ := tokens["access_token"].(string)

	got := env.recommendedPurposes(t, token)
	if got["BUY"] == 0 || got["RENT"] == 0 {
		t.Errorf("a customer wanting both saw %v", got)
	}
}

// Saving the preferences form was overwriting the intent columns, quietly
// turning a renter into an account interested in neither.
func TestSavingPreferencesKeepsTheRentIntent(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "63")

	res := env.registerWith(t, map[string]any{
		"full_name": "Renter Prefs", "email": "renterprefs63@example.com",
		"mobile": "9700002363", "password": "Password@123",
		"profile_type": "RENTER",
	})
	tokens, _ := res.Data["tokens"].(map[string]any)
	token, _ := tokens["access_token"].(string)

	// A form that carries only the buy-side fields, as the existing one does.
	saved := env.do(t, http.MethodPut, "/api/v1/customer/preferences", token, map[string]any{
		"preferred_city": "Pune", "property_type": "FLAT", "bhk": 2,
	})
	if saved.Status != http.StatusOK {
		t.Fatalf("save preferences: status %d (%s)", saved.Status, saved.Message)
	}
	if saved.Data["looking_to_rent"] != true {
		t.Errorf("looking_to_rent = %v after saving preferences, want true",
			saved.Data["looking_to_rent"])
	}

	// The dashboard must still be a renter's dashboard afterwards.
	if got := env.recommendedPurposes(t, token); got["BUY"] != 0 {
		t.Errorf("after saving preferences a renter sees %d sale listing(s)", got["BUY"])
	}
}

// The customer decides later that they want to buy as well. Turning the intent
// on must open the sale catalogue without closing the rental one.
func TestACustomerCanTakeUpBuyingLater(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "64")

	res := env.registerWith(t, map[string]any{
		"full_name": "Later Buyer", "email": "later64@example.com",
		"mobile": "9700002364", "password": "Password@123",
		"profile_type": "RENTER",
	})
	tokens, _ := res.Data["tokens"].(map[string]any)
	token, _ := tokens["access_token"].(string)

	before := env.recommendedPurposes(t, token)
	if before["BUY"] != 0 {
		t.Fatalf("renter already sees sale listings: %v", before)
	}

	saved := env.do(t, http.MethodPut, "/api/v1/customer/preferences", token, map[string]any{
		"looking_to_buy": true, "looking_to_rent": true,
	})
	if saved.Status != http.StatusOK {
		t.Fatalf("enable buying: status %d (%s)", saved.Status, saved.Message)
	}

	after := env.recommendedPurposes(t, token)
	if after["BUY"] == 0 {
		t.Error("after taking up buying, no sale listings are recommended")
	}
	if after["RENT"] == 0 {
		t.Error("taking up buying lost the rental recommendations")
	}
}

// Switching both intents off would leave an account recommended nothing, so the
// stored purpose decides which one stays on.
func TestAnAccountIsNeverLeftWantingNeither(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "65")

	res := env.registerWith(t, map[string]any{
		"full_name": "Neither", "email": "neither65@example.com",
		"mobile": "9700002365", "password": "Password@123",
		"profile_type": "RENTER",
	})
	tokens, _ := res.Data["tokens"].(map[string]any)
	token, _ := tokens["access_token"].(string)

	saved := env.do(t, http.MethodPut, "/api/v1/customer/preferences", token, map[string]any{
		"looking_to_buy": false, "looking_to_rent": false, "purpose": "RENT",
	})
	if saved.Status != http.StatusOK {
		t.Fatalf("save: status %d (%s)", saved.Status, saved.Message)
	}
	if saved.Data["looking_to_rent"] != true {
		t.Errorf("both intents were switched off, leaving the account with nothing: %v", saved.Data)
	}
}

// Rental preferences must survive the same form, for the same reason.
func TestSavingPreferencesKeepsRentalCriteria(t *testing.T) {
	env := setup(t)
	env.intentFixture(t, "66")

	res := env.registerWith(t, map[string]any{
		"full_name": "Rent Criteria", "email": "criteria66@example.com",
		"mobile": "9700002366", "password": "Password@123",
		"profile_type": "RENTER",
	})
	tokens, _ := res.Data["tokens"].(map[string]any)
	token, _ := tokens["access_token"].(string)

	first := env.do(t, http.MethodPut, "/api/v1/customer/preferences", token, map[string]any{
		"monthly_rent_min": 15000, "monthly_rent_max": 30000,
		"preferred_furnishing": "SEMI_FURNISHED",
	})
	if first.Status != http.StatusOK {
		t.Fatalf("save rental criteria: status %d (%s)", first.Status, first.Message)
	}
	if first.Data["monthly_rent_max"].(float64) != 30000 {
		t.Fatalf("monthly_rent_max = %v, want 30000", first.Data["monthly_rent_max"])
	}

	var profile models.CustomerProfile
	if err := env.db.Joins("JOIN users ON users.id = customer_profiles.user_id").
		Where("users.email = ?", "criteria66@example.com").First(&profile).Error; err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if profile.PreferredFurnishing != models.FurnishingSemiFurnished {
		t.Errorf("preferred_furnishing = %q, want SEMI_FURNISHED", profile.PreferredFurnishing)
	}
	if !profile.LookingToRent {
		t.Error("looking_to_rent was lost")
	}
}

// A minimum rent above the maximum is a preference that can match nothing.
func TestAnImpossibleRentPreferenceIsRejected(t *testing.T) {
	env := setup(t)
	token, _ := env.register(t, "customer", "badrent67@example.com", "9700002367", "Bad Rent")

	res := env.do(t, http.MethodPut, "/api/v1/customer/preferences", token, map[string]any{
		"monthly_rent_min": 50000, "monthly_rent_max": 20000,
	})
	if res.Status != http.StatusBadRequest {
		t.Errorf("status %d (%s), want 400", res.Status, res.Message)
	}
}
