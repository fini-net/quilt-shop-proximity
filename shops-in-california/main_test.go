package main

import "testing"

func TestIsPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Valid phone numbers
		{"714-995-3178", true},
		{"(714) 995-3178", true},
		{"714.995.3178", true},
		{"7149953178", true},
		{"+1 714-995-3178", true},

		// Invalid - addresses that look like phone numbers
		{"3430 W Ball Rd, Anaheim, CA 92804", false},
		{"1189 N Euclid St, Anaheim, CA 92801", false},

		// Invalid - too short
		{"123-4567", false},

		// Invalid - contains letters
		{"714-ABC-3178", false},
		{"Call 714-995-3178", false},

		// Edge cases
		{"", false},
		{"123", false},
	}

	for _, tt := range tests {
		result := isPhone(tt.input)
		if result != tt.expected {
			t.Errorf("isPhone(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseShopFromPre(t *testing.T) {
	tests := []struct {
		name         string
		preText      string
		shopName     string
		city         string
		wantAddress  string
		wantPhone    string
		wantEmail    string
	}{
		{
			name: "Complete shop with all fields",
			preText: `Mel's Sewing & Fabric Center
1189 N Euclid St, Anaheim, CA 92801
714-774-3460
info@melssewing.com`,
			shopName:    "Mel's Sewing & Fabric Center",
			city:        "anaheim",
			wantAddress: "1189 N Euclid St, Anaheim, CA 92801",
			wantPhone:   "714-774-3460",
			wantEmail:   "info@melssewing.com",
		},
		{
			name: "Shop without email",
			preText: `M & L Fabrics Discount Store
3430 W Ball Rd, Anaheim, CA 92804
714-995-3178`,
			shopName:    "M & L Fabrics Discount Store",
			city:        "anaheim",
			wantAddress: "3430 W Ball Rd, Anaheim, CA 92804",
			wantPhone:   "714-995-3178",
			wantEmail:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shop := parseShopFromPre(tt.preText, tt.shopName, tt.city)

			if shop.Address != tt.wantAddress {
				t.Errorf("Address = %q, want %q", shop.Address, tt.wantAddress)
			}
			if shop.Phone != tt.wantPhone {
				t.Errorf("Phone = %q, want %q", shop.Phone, tt.wantPhone)
			}
			if shop.Email != tt.wantEmail {
				t.Errorf("Email = %q, want %q", shop.Email, tt.wantEmail)
			}
		})
	}
}
