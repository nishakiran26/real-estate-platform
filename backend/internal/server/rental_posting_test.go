package server_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rera/property-platform/backend/internal/models"
)

// rentalPayload is the owner-facing shape for a flat offered to let. It has no
// RERA block and no sale price: the rent lives on the rental block instead.
func rentalPayload(cityID, localityID uint, name string, rent float64) map[string]any {
	return map[string]any{
		"name":                name,
		"purpose":             "RENT",
		"property_type":       "FLAT",
		"construction_status": "READY_TO_MOVE",
		"city_id":             cityID,
		"locality_id":         localityID,
		"address":             "Datta Mandir Road, Wakad",
		"pincode":             "411057",
		"description":         "A bright 2 BHK with covered parking, close to the Hinjawadi bridge.",
		"configurations": []map[string]any{
			{"name": "2 BHK", "carpet_area": 850, "bedrooms": 2, "bathrooms": 2, "balconies": 1},
		},
		"rental": map[string]any{
			"monthly_rent":       rent,
			"security_deposit":   rent * 3,
			"maintenance_amount": 1500,
			"available_from":     time.Now().AddDate(0, 0, 10).Format("2006-01-02"),
			"furnishing_type":    "SEMI_FURNISHED",
			"tenant_preference":  "FAMILY",
			"pet_allowed":        true,
			"included_items":     []string{"Geyser", "Wardrobe", "Modular kitchen"},
		},
	}
}

// registerOwner creates an owner account and returns its token.
func (env *testEnv) registerOwner(t *testing.T, email, mobile, name string) string {
	t.Helper()
	res := env.registerWith(t, map[string]any{
		"full_name": name, "email": email, "mobile": mobile,
		"password": "Password@123", "profile_type": "OWNER",
	})
	if res.Status != http.StatusCreated {
		t.Fatalf("register owner: status %d (%s)", res.Status, res.Message)
	}
	tokens, _ := res.Data["tokens"].(map[string]any)
	token, _ := tokens["access_token"].(string)
	return token
}

// uploadRoomPhoto attaches a room photo, which is how a flat is illustrated.
func (env *testEnv) uploadRoomPhoto(t *testing.T, propertyID int, token string) {
	t.Helper()
	env.uploadMediaAs(t, propertyID, token, "LIVING_ROOM")
}

func TestOwnerCanPostARental(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	token := env.registerOwner(t, "poster@example.com", "9800000401", "Rahul Sharma")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", token,
		rentalPayload(cityID, localityID, "Rahul 2 BHK Wakad", 28000))
	if created.Status != http.StatusCreated {
		t.Fatalf("post rental: status %d (%s)", created.Status, created.Message)
	}

	if created.Data["purpose"] != "RENT" {
		t.Errorf("purpose = %v, want RENT", created.Data["purpose"])
	}
	// The listing account is an owner, so the listing records that provenance.
	if created.Data["posted_by_type"] != "OWNER" {
		t.Errorf("posted_by_type = %v, want OWNER", created.Data["posted_by_type"])
	}
	if created.Data["availability_status"] != "AVAILABLE" {
		t.Errorf("availability_status = %v, want AVAILABLE", created.Data["availability_status"])
	}

	rental, ok := created.Data["rental_details"].(map[string]any)
	if !ok {
		t.Fatal("the response carries no rental_details block")
	}
	if fmt.Sprint(rental["monthly_rent"]) != "28000" {
		t.Errorf("monthly_rent = %v, want 28000", rental["monthly_rent"])
	}
	if fmt.Sprint(rental["security_deposit"]) != "84000" {
		t.Errorf("security_deposit = %v, want 84000", rental["security_deposit"])
	}
	if rental["furnishing_type"] != "SEMI_FURNISHED" {
		t.Errorf("furnishing_type = %v", rental["furnishing_type"])
	}
	if rental["pet_allowed"] != true {
		t.Error("pet_allowed was not stored")
	}

	items, _ := rental["included_items"].([]any)
	if len(items) != 3 {
		t.Errorf("included_items = %v, want 3 entries", rental["included_items"])
	}

	// Defaults an owner should not have to think about.
	if fmt.Sprint(rental["min_lease_months"]) != "11" {
		t.Errorf("min_lease_months = %v, want the 11-month default", rental["min_lease_months"])
	}
	if fmt.Sprint(rental["notice_period_days"]) != "30" {
		t.Errorf("notice_period_days = %v, want the 30-day default", rental["notice_period_days"])
	}

	// A rental expires; a sale listing does not.
	var property models.Property
	if err := env.db.First(&property, uint(created.Data["id"].(float64))).Error; err != nil {
		t.Fatalf("reload property: %v", err)
	}
	if property.ListingExpiryDate == nil {
		t.Error("a rental was created without a listing expiry date")
	}
}

// A rental has no project registration, so demanding one would make it
// impossible to submit. Rent is required instead.
func TestRentalSubmissionDoesNotRequireRERA(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	token := env.registerOwner(t, "submitter@example.com", "9800000402", "Submitter")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", token,
		rentalPayload(cityID, localityID, "Submitter Flat", 22000))
	propertyID := int(created.Data["id"].(float64))

	// Without a photo the submission is refused, exactly as for a sale listing.
	tooEarly := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/submit", propertyID), token, nil)
	if tooEarly.Status == http.StatusOK {
		t.Fatal("a rental with no photograph was accepted for review")
	}

	env.uploadRoomPhoto(t, propertyID, token)

	submitted := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/submit", propertyID), token, nil)
	if submitted.Status != http.StatusOK {
		t.Fatalf("submit rental: status %d (%s) — RERA should not be required",
			submitted.Status, submitted.Message)
	}
	if submitted.Data["property_status"] != "SUBMITTED" {
		t.Errorf("status = %v, want SUBMITTED", submitted.Data["property_status"])
	}
}

// A sale listing still needs its RERA number: the rental exemption must not
// weaken the existing rule.
func TestSaleSubmissionStillRequiresRERA(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	builderToken, _ := env.register(t, "client", "reraneed@example.com", "9800000403", "RERA Needed")

	payload := propertyPayload(cityID, localityID, "No RERA Towers")
	delete(payload, "rera")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", builderToken, payload)
	propertyID := int(created.Data["id"].(float64))
	env.uploadImage(t, propertyID, builderToken)

	res := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/client/properties/%d/submit", propertyID), builderToken, nil)
	if res.Status == http.StatusOK {
		t.Error("a sale listing without a RERA number was accepted for review")
	}
}

func TestRentalRequiresRent(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	token := env.registerOwner(t, "norent@example.com", "9800000404", "No Rent")

	payload := rentalPayload(cityID, localityID, "No Rent Flat", 0)
	delete(payload, "rental")

	res := env.do(t, http.MethodPost, "/api/v1/client/properties", token, payload)
	if res.Status == http.StatusCreated {
		t.Error("a rental was created with no rental terms at all")
	}

	withZero := rentalPayload(cityID, localityID, "Zero Rent Flat", 0)
	if res := env.do(t, http.MethodPost, "/api/v1/client/properties", token, withZero); res.Status == http.StatusCreated {
		t.Error("a rental was created with a zero monthly rent")
	}
}

// Rental terms on a sale listing would leave it in an inconsistent state.
func TestRentalTermsRejectedOnSaleListing(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	builderToken, _ := env.register(t, "client", "mixed@example.com", "9800000405", "Mixed")

	payload := propertyPayload(cityID, localityID, "Mixed Signals")
	payload["rental"] = map[string]any{"monthly_rent": 30000}

	res := env.do(t, http.MethodPost, "/api/v1/client/properties", builderToken, payload)
	if res.Status == http.StatusCreated {
		t.Error("a sale listing accepted rental terms")
	}
}

// A sale configuration still needs a price. The validation moved from the struct
// tag into the service, so this guards against it being lost in the move.
func TestSaleConfigurationStillRequiresAPrice(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	builderToken, _ := env.register(t, "client", "freeflat@example.com", "9800000406", "Free Flat")

	payload := propertyPayload(cityID, localityID, "Priceless Towers")
	payload["configurations"] = []map[string]any{
		{"name": "2 BHK", "carpet_area": 850, "bedrooms": 2, "price": 0},
	}

	res := env.do(t, http.MethodPost, "/api/v1/client/properties", builderToken, payload)
	if res.Status == http.StatusCreated {
		t.Error("a sale listing was created with a zero-priced configuration")
	}
}

func TestOwnerCanUpdateRentalTerms(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	token := env.registerOwner(t, "updater@example.com", "9800000407", "Updater")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", token,
		rentalPayload(cityID, localityID, "Updater Flat", 25000))
	propertyID := int(created.Data["id"].(float64))

	updated := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/client/properties/%d", propertyID), token, map[string]any{
			"rental": map[string]any{
				"monthly_rent":      31000,
				"security_deposit":  93000,
				"furnishing_type":   "FULLY_FURNISHED",
				"tenant_preference": "ANYONE",
			},
		})
	if updated.Status != http.StatusOK {
		t.Fatalf("update rental: status %d (%s)", updated.Status, updated.Message)
	}

	rental, ok := updated.Data["rental_details"].(map[string]any)
	if !ok {
		t.Fatal("rental_details missing after update")
	}
	if fmt.Sprint(rental["monthly_rent"]) != "31000" {
		t.Errorf("monthly_rent = %v, want 31000", rental["monthly_rent"])
	}
	if rental["furnishing_type"] != "FULLY_FURNISHED" {
		t.Errorf("furnishing_type = %v, want FULLY_FURNISHED", rental["furnishing_type"])
	}

	// The update replaced the row rather than inserting a second one.
	var count int64
	env.db.Model(&models.RentalDetails{}).Where("property_id = ?", propertyID).Count(&count)
	if count != 1 {
		t.Errorf("rental_details rows = %d, want exactly 1", count)
	}
}

// Ownership still holds for rentals: the rules are enforced on the property, not
// on its purpose.
func TestOwnerCannotTouchAnotherOwnersRental(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)

	first := env.registerOwner(t, "owner.a@example.com", "9800000408", "Owner A")
	second := env.registerOwner(t, "owner.b@example.com", "9800000409", "Owner B")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", first,
		rentalPayload(cityID, localityID, "Owner A Flat", 24000))
	propertyID := int(created.Data["id"].(float64))

	read := env.do(t, http.MethodGet,
		fmt.Sprintf("/api/v1/client/properties/%d", propertyID), second, nil)
	if read.Status != http.StatusNotFound && read.Status != http.StatusForbidden {
		t.Errorf("another owner read the listing: status %d", read.Status)
	}

	edit := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/client/properties/%d", propertyID), second,
		map[string]any{"rental": map[string]any{"monthly_rent": 1}})
	if edit.Status != http.StatusNotFound && edit.Status != http.StatusForbidden {
		t.Errorf("another owner edited the listing: status %d", edit.Status)
	}
}

// A customer holds no listing account, so the posting endpoints stay closed.
func TestCustomerCannotPostARental(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)
	token, _ := env.register(t, "customer", "nolisting@example.com", "9800000410", "No Listing")

	res := env.do(t, http.MethodPost, "/api/v1/client/properties", token,
		rentalPayload(cityID, localityID, "Should Not Exist", 20000))
	if res.Status != http.StatusForbidden {
		t.Errorf("a customer posted a rental: status %d", res.Status)
	}
}

// The full owner journey, ending with the flat visible on the rent page and
// absent from the buy page.
func TestOwnerRentalReachesTheRentSearch(t *testing.T) {
	env := setup(t)
	cityID, localityID := env.seedLocation(t)

	token := env.registerOwner(t, "journey@example.com", "9800000411", "Journey Owner")
	_, _ = env.register(t, "customer", "journeyadmin@example.com", "9800000412", "Admin")
	adminToken := env.promoteToAdmin(t, "journeyadmin@example.com")

	created := env.do(t, http.MethodPost, "/api/v1/client/properties", token,
		rentalPayload(cityID, localityID, "Journey 2 BHK", 27000))
	propertyID := int(created.Data["id"].(float64))

	// Not public while it is still a draft.
	if before := env.do(t, http.MethodGet, "/api/v1/properties?purpose=rent", "", nil); len(before.List) != 0 {
		t.Fatalf("a draft rental was publicly visible: %d results", len(before.List))
	}

	env.uploadRoomPhoto(t, propertyID, token)
	env.do(t, http.MethodPost, fmt.Sprintf("/api/v1/client/properties/%d/submit", propertyID), token, nil)

	approve := env.do(t, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/properties/%d/approve", propertyID), adminToken,
		map[string]any{"publish": true})
	if approve.Status != http.StatusOK {
		t.Fatalf("approve rental: status %d (%s)", approve.Status, approve.Message)
	}

	rent := env.do(t, http.MethodGet, "/api/v1/properties?purpose=rent", "", nil)
	if len(rent.List) != 1 {
		t.Fatalf("rent search returned %d listings, want 1", len(rent.List))
	}
	card, _ := rent.List[0].(map[string]any)
	if fmt.Sprint(card["monthly_rent"]) != "27000" {
		t.Errorf("card monthly_rent = %v, want 27000", card["monthly_rent"])
	}
	if card["furnishing_type"] != "SEMI_FURNISHED" {
		t.Errorf("card furnishing_type = %v", card["furnishing_type"])
	}

	// And it stays out of the Buy page.
	if buy := env.do(t, http.MethodGet, "/api/v1/properties", "", nil); len(buy.List) != 0 {
		t.Errorf("a rental appeared in the buy search: %d results", len(buy.List))
	}
}
