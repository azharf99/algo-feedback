package formatter

import (
	"regexp"
	"strings"
)

// NormalizePhoneNumber cleans a phone number and converts it to a standard format (digits only, with country code).
// It prioritizes Indonesian numbers but handles RU (+7), US (+1), and UK (+44) based on common prefixes and lengths.
func NormalizePhoneNumber(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" {
		return ""
	}

	// 1. Check if original input has '+' prefix
	hasPlus := strings.HasPrefix(trimmed, "+")

	// 2. Remove all non-digits
	re := regexp.MustCompile(`\D`)
	cleaned := re.ReplaceAllString(trimmed, "")

	if cleaned == "" {
		return ""
	}

	// 3. If it had a '+', trust it as a complete international number
	if hasPlus {
		return cleaned
	}

	// 4. Handle Indonesian '0' prefix (08... -> 628...)
	if strings.HasPrefix(cleaned, "0") {
		return "62" + cleaned[1:]
	}

	// 5. Heuristics for common countries if no '+' and no '0'
	length := len(cleaned)

	// Russian local prefix handling (89xx... -> 79xx...)
	// In RU, mobile numbers start with 9. 8 is the trunk prefix.
	// In ID, mobile numbers starting with 9 are not standard (usually 08...).
	if length == 11 && strings.HasPrefix(cleaned, "89") {
		return "7" + cleaned[1:]
	}

	// Russian (7 + 10 digits) or US (1 + 10 digits)
	if length == 11 {
		if strings.HasPrefix(cleaned, "7") || strings.HasPrefix(cleaned, "1") {
			return cleaned
		}
	}

	// UK (44 + 10-12 digits)
	if (length == 12 || length == 13) && strings.HasPrefix(cleaned, "44") {
		return cleaned
	}

	// 6. Default fallback to Indonesia for local-style inputs (e.g., 812... -> 62812...)
	// Usually ID mobile numbers are 9-13 digits after 62.
	if length >= 9 && length <= 12 && !strings.HasPrefix(cleaned, "62") {
		return "62" + cleaned
	}

	return cleaned
}
