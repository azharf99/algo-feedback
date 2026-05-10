package formatter

import (
	"testing"
)

func TestNormalizePhoneNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"6285347029992", "6285347029992"},
		{"085347029992", "6285347029992"},
		{"+62 812-3456-789", "628123456789"},
		{"8123456789", "628123456789"}, // ID fallback
		{"79991234567", "79991234567"}, // RU
		{"89991234567", "79991234567"}, // RU with local 8 prefix
		{"+7 (999) 123-45-67", "79991234567"}, // RU with plus
		{"12125550123", "12125550123"}, // US
		{"+1 212-555-0123", "12125550123"}, // US with plus
		{"447911123456", "447911123456"}, // UK
		{"+44 7911 123456", "447911123456"}, // UK with plus
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizePhoneNumber(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizePhoneNumber(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
