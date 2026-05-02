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
		{"62 (812) 216-13-004", "6281221613004"},
		{"62(812) 116-1911", "628121161911"},
		{"62 (816) 407-986", "62816407986"},
		{"085347029992", "6285347029992"},
		{"+62 812-3456-789", "628123456789"},
		{"8123456789", "628123456789"}, // Handle missing prefix
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
