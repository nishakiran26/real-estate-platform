package server_test

import (
	"net/http"
	"testing"

	"github.com/rera/property-platform/backend/internal/database"
	"github.com/rera/property-platform/backend/internal/models"
)

// A fresh installation must not open the Rent page on an empty catalogue. The
// rest of the suite runs with demo data off, so this test turns it on for its
// own schema and checks what the seeder actually wrote.
func TestDemoSeedPublishesBrowsableRentals(t *testing.T) {
	env := setup(t)

	cfg := *env.cfg
	cfg.Seed.DemoData = true
	if err := database.Seed(env.db, &cfg); err != nil {
		t.Fatalf("seed demo data: %v", err)
	}

	res := env.do(t, http.MethodGet, "/api/v1/properties?purpose=rent", "", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("rent search: status %d (%s)", res.Status, res.Message)
	}
	if len(res.List) < 2 {
		t.Fatalf("seeded rentals visible in search = %d, want at least 2", len(res.List))
	}

	// The Buy catalogue is seeded alongside and must not have absorbed them.
	buy := env.do(t, http.MethodGet, "/api/v1/properties", "", nil)
	for _, row := range buy.List {
		card, _ := row.(map[string]any)
		if card != nil && card["purpose"] == "RENT" {
			t.Errorf("a rental is in the buy results: %v", card["name"])
		}
	}

	// Every seeded rental is a complete, honest record: rent and deposit set,
	// posted by an owner, and carrying no invented RERA registration, because
	// RERA governs projects for sale.
	var rentals []models.Property
	if err := env.db.Preload("RentalDetails").
		Where("purpose = ?", models.PurposeRent).Find(&rentals).Error; err != nil {
		t.Fatalf("load seeded rentals: %v", err)
	}
	if len(rentals) == 0 {
		t.Fatal("the seeder wrote no rentals")
	}
	for _, p := range rentals {
		if p.RentalDetails == nil {
			t.Errorf("%s has no rental terms", p.Name)
			continue
		}
		if p.RentalDetails.MonthlyRent <= 0 || p.RentalDetails.SecurityDeposit <= 0 {
			t.Errorf("%s: rent %v deposit %v", p.Name,
				p.RentalDetails.MonthlyRent, p.RentalDetails.SecurityDeposit)
		}
		if p.PostedByType != models.PostedByOwner {
			t.Errorf("%s posted_by_type = %s, want OWNER", p.Name, p.PostedByType)
		}
		if p.ListingExpiryDate == nil {
			t.Errorf("%s has no expiry date", p.Name)
		}
		if p.IsRERAVerified {
			t.Errorf("%s claims RERA verification", p.Name)
		}
		var rera int64
		env.db.Model(&models.RERADetails{}).Where("property_id = ?", p.ID).Count(&rera)
		if rera != 0 {
			t.Errorf("%s carries an invented RERA registration", p.Name)
		}
	}

	// The owner account is a listing account with an OWNER profile, not a
	// fourth role.
	var owner models.Client
	if err := env.db.Where("profile_type = ?", models.ProfileOwner).First(&owner).Error; err != nil {
		t.Fatalf("no owner account was seeded: %v", err)
	}
	if owner.VerificationLevel != models.VerificationUnverified {
		t.Errorf("a seeded account claims verification: %s", owner.VerificationLevel)
	}
}

// Seeding twice must not duplicate the rentals: a restart re-runs it.
func TestDemoSeedIsIdempotentForRentals(t *testing.T) {
	env := setup(t)

	cfg := *env.cfg
	cfg.Seed.DemoData = true
	for i := 0; i < 2; i++ {
		if err := database.Seed(env.db, &cfg); err != nil {
			t.Fatalf("seed run %d: %v", i+1, err)
		}
	}

	var count int64
	env.db.Model(&models.Property{}).Where("purpose = ?", models.PurposeRent).Count(&count)
	var rentalRows int64
	env.db.Model(&models.RentalDetails{}).Count(&rentalRows)
	if count != rentalRows {
		t.Errorf("%d rentals but %d rental_details rows", count, rentalRows)
	}

	var owners int64
	env.db.Model(&models.Client{}).Where("profile_type = ?", models.ProfileOwner).Count(&owners)
	if owners != 1 {
		t.Errorf("owner accounts after two seed runs = %d, want 1", owners)
	}
}
