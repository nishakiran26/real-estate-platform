package services

import (
	"strings"
	"testing"

	"github.com/rera/property-platform/backend/internal/models"
)

// The saved search is only as good as the link back to it: if the rebuilt query
// does not reproduce the search, "view matches" shows the renter something
// other than what they saved.
func TestAlertSearchQueryReproducesTheSearch(t *testing.T) {
	alert := &models.RentalAlert{
		CityID: 1, LocalityID: 4, BHK: 2, MinRent: 15000, MaxRent: 30000,
		Criteria: `{"furnishing":"SEMI_FURNISHED","tenant_preference":"FAMILY"}`,
	}

	query := alertSearchQuery(alert)
	for _, want := range []string{
		"purpose=rent",
		"city_id=1",
		"locality_id=4",
		"bhk=2",
		"min_rent=15000",
		"max_rent=30000",
		// Not matched on, but part of the search the renter saved.
		"furnishing=SEMI_FURNISHED",
		"tenant_preference=FAMILY",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("query %q is missing %q", query, want)
		}
	}
}

// A zero criterion means "no constraint", so it must not appear in the link as
// a filter of zero — "max_rent=0" would return nothing.
func TestAlertSearchQueryOmitsUnsetCriteria(t *testing.T) {
	alert := &models.RentalAlert{CityID: 1, Criteria: "{}"}

	query := alertSearchQuery(alert)
	for _, unwanted := range []string{"bhk=", "min_rent=", "max_rent=", "locality_id="} {
		if strings.Contains(query, unwanted) {
			t.Errorf("query %q should not carry %q for an unset criterion", query, unwanted)
		}
	}
	if !strings.Contains(query, "city_id=1") {
		t.Errorf("query %q lost the criterion that was set", query)
	}
}

// Criteria that cannot be decoded must not take the whole link down with them.
func TestAlertSearchQuerySurvivesUnreadableCriteria(t *testing.T) {
	alert := &models.RentalAlert{CityID: 1, BHK: 3, Criteria: "not json"}

	query := alertSearchQuery(alert)
	if !strings.Contains(query, "bhk=3") || !strings.Contains(query, "purpose=rent") {
		t.Errorf("query %q lost the indexed criteria", query)
	}
}

func TestBedroomsOfTakesTheSmallestConfiguration(t *testing.T) {
	tests := []struct {
		name string
		beds []int
		want int
	}{
		// A building offering 2 and 3 BHK satisfies someone watching for a 2 BHK,
		// so the smallest is what an alert matches against.
		{"several configurations", []int{3, 2, 4}, 2},
		{"one configuration", []int{2}, 2},
		{"no configurations", nil, 0},
		// A plot or a shop carries no bedrooms; it must not read as a 0 BHK match.
		{"configurations without bedrooms", []int{0, 0}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &models.Property{}
			for _, b := range tc.beds {
				p.Configurations = append(p.Configurations, models.PropertyConfiguration{Bedrooms: b})
			}
			if got := bedroomsOf(p); got != tc.want {
				t.Errorf("bedroomsOf = %d, want %d", got, tc.want)
			}
		})
	}
}

// The notice names where the flat is. The associations are not always loaded,
// and "in " followed by nothing reads as a bug to the renter receiving it.
func TestLocalityLabelDegradesSensibly(t *testing.T) {
	withNames := func(locality, city string) *models.Property {
		p := &models.Property{}
		p.Locality.Name = locality
		p.City.Name = city
		return p
	}

	tests := []struct {
		name     string
		property *models.Property
		want     string
	}{
		{"both", withNames("Baner", "Pune"), "Baner, Pune"},
		{"locality only", withNames("Baner", ""), "Baner"},
		{"city only", withNames("", "Pune"), "Pune"},
		{"neither", withNames("", ""), "your search area"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := localityLabel(tc.property); got != tc.want {
				t.Errorf("localityLabel = %q, want %q", got, tc.want)
			}
		})
	}
}

// The rent band a renter types is the one they see back, in Indian digit
// grouping. This pins the naming the saved-search list depends on.
func TestFormatRupeesUsesIndianGrouping(t *testing.T) {
	tests := []struct {
		amount float64
		want   string
	}{
		{30000, "₹30,000"},
		{2800000, "₹28.00 L"},
		{15000, "₹15,000"},
		{0, "On request"},
	}
	for _, tc := range tests {
		if got := formatRupees(tc.amount); got != tc.want {
			t.Errorf("formatRupees(%v) = %q, want %q", tc.amount, got, tc.want)
		}
	}
}
