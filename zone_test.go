package main_test

import (
	"testing"

	"github.com/linode/linodego"
)

func TestFetchZone_ExactMatch(t *testing.T) {
	zones := []linodego.Domain{
		{ID: 1, Domain: "example.com"},
		{ID: 2, Domain: "sub.example.com"},
	}

	result := findMatchingZone(zones, "example.com")
	if result == nil {
		t.Fatal("Expected to find zone example.com")
	}
	if result.Domain != "example.com" {
		t.Errorf("Expected zone example.com, got %s", result.Domain)
	}
}

func TestFetchZone_LongestSuffixMatch(t *testing.T) {
	zones := []linodego.Domain{
		{ID: 1, Domain: "example.com"},
		{ID: 2, Domain: "zone.example.com"},
		{ID: 3, Domain: "other.com"},
	}

	// Should match zone.example.com (longest suffix)
	result := findMatchingZone(zones, "sub.zone.example.com")
	if result == nil {
		t.Fatal("Expected to find zone for sub.zone.example.com")
	}
	if result.Domain != "zone.example.com" {
		t.Errorf("Expected zone zone.example.com, got %s", result.Domain)
	}
}

func TestFetchZone_NoMatch(t *testing.T) {
	zones := []linodego.Domain{
		{ID: 1, Domain: "example.com"},
		{ID: 2, Domain: "other.com"},
	}

	result := findMatchingZone(zones, "notfound.net")
	if result != nil {
		t.Errorf("Expected no match for notfound.net, got %s", result.Domain)
	}
}

func TestFetchZone_CrossZoneDomains(t *testing.T) {
	// Real-world scenario: delegated subdomain zones
	zones := []linodego.Domain{
		{ID: 1, Domain: "example.com"},
		{ID: 2, Domain: "to.example.com"},
		{ID: 3, Domain: "team.example.com"},
	}

	tests := []struct {
		domain       string
		expectedZone string
	}{
		{"www.example.com", "example.com"},
		{"to.example.com", "to.example.com"},
		{"www.to.example.com", "to.example.com"},
		{"team.example.com", "team.example.com"},
		{"www.team.example.com", "team.example.com"},
		{"to.team.example.com", "team.example.com"},     // Cross-zone: in team zone
		{"www.to.team.example.com", "team.example.com"}, // Cross-zone: in team zone
		{"team.to.example.com", "to.example.com"},       // Cross-zone: in to zone
		{"www.team.to.example.com", "to.example.com"},   // Cross-zone: in to zone
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := findMatchingZone(zones, tt.domain)
			if result == nil {
				t.Fatalf("Expected to find zone for %s", tt.domain)
			}
			if result.Domain != tt.expectedZone {
				t.Errorf("For domain %s: expected zone %s, got %s",
					tt.domain, tt.expectedZone, result.Domain)
			}
		})
	}
}

func TestEntryRecalculation(t *testing.T) {
	tests := []struct {
		name          string
		domain        string
		zoneDomain    string
		entry         string
		expectedEntry string
	}{
		{
			name:          "exact match - no recalculation needed",
			domain:        "example.com",
			zoneDomain:    "example.com",
			entry:         "_acme-challenge",
			expectedEntry: "_acme-challenge",
		},
		{
			name:          "one level subdomain",
			domain:        "sub.example.com",
			zoneDomain:    "example.com",
			entry:         "_acme-challenge",
			expectedEntry: "_acme-challenge.sub",
		},
		{
			name:          "two level subdomain",
			domain:        "www.sub.example.com",
			zoneDomain:    "example.com",
			entry:         "_acme-challenge",
			expectedEntry: "_acme-challenge.www.sub",
		},
		{
			name:          "cross-zone domain - team.to.example.com",
			domain:        "team.to.example.com",
			zoneDomain:    "to.example.com",
			entry:         "_acme-challenge",
			expectedEntry: "_acme-challenge.team",
		},
		{
			name:          "cross-zone domain - www.team.to.example.com",
			domain:        "team.to.example.com",
			zoneDomain:    "to.example.com",
			entry:         "_acme-challenge.www",
			expectedEntry: "_acme-challenge.www.team",
		},
		{
			name:          "cross-zone domain - to.team.example.com",
			domain:        "team.example.com",
			zoneDomain:    "team.example.com",
			entry:         "_acme-challenge.to",
			expectedEntry: "_acme-challenge.to",
		},
		{
			name:          "cross-zone domain - www.to.team.example.com",
			domain:        "to.team.example.com",
			zoneDomain:    "team.example.com",
			entry:         "_acme-challenge.www",
			expectedEntry: "_acme-challenge.www.to",
		},
		{
			name:          "empty entry",
			domain:        "sub.example.com",
			zoneDomain:    "example.com",
			entry:         "",
			expectedEntry: "sub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualEntry := calculateActualEntry(tt.domain, tt.zoneDomain, tt.entry)
			if actualEntry != tt.expectedEntry {
				t.Errorf("For domain=%s, zone=%s, entry=%s:\n  expected: %s\n  got: %s",
					tt.domain, tt.zoneDomain, tt.entry, tt.expectedEntry, actualEntry)
			}
		})
	}
}

// Helper function extracted from fetchZone logic for testing
func findMatchingZone(zones []linodego.Domain, domain string) *linodego.Domain {
	var bestMatch *linodego.Domain
	var bestMatchLen int

	for i := range zones {
		zone := &zones[i]
		// Check if domain equals zone or is a subdomain of zone
		if zone.Domain == domain || hasSuffix(domain, "."+zone.Domain) {
			// Keep track of longest matching zone
			if len(zone.Domain) > bestMatchLen {
				bestMatch = zone
				bestMatchLen = len(zone.Domain)
			}
		}
	}

	return bestMatch
}

// Helper function extracted from fetchZoneAndRecord logic for testing
func calculateActualEntry(domain, zoneDomain, entry string) string {
	actualEntry := entry
	if zoneDomain != domain {
		// Zone is a parent of domain, need to recalculate entry from FQDN
		// Reconstruct FQDN from entry + domain
		fqdn := entry
		if entry != "" && domain != "" {
			fqdn = entry + "." + domain
		} else if domain != "" {
			fqdn = domain
		}

		// Now calculate entry in the actual zone
		actualEntry = trimSuffix(fqdn, "."+zoneDomain)
	}
	return actualEntry
}

// Helper functions to avoid importing strings in test
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func trimSuffix(s, suffix string) string {
	if hasSuffix(s, suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}
