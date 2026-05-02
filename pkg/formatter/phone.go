package formatter

import (
	"regexp"
	"strings"
)

// NormalizePhoneNumber membersihkan nomor telepon ke format standar (62...)
func NormalizePhoneNumber(phone string) string {
	// 1. Hapus semua karakter non-digit
	re := regexp.MustCompile(`\D`)
	cleaned := re.ReplaceAllString(phone, "")

	// 2. Jika kosong, kembalikan kosong
	if cleaned == "" {
		return ""
	}

	// 3. Tangani awalan '0' menjadi '62'
	if strings.HasPrefix(cleaned, "0") {
		cleaned = "62" + cleaned[1:]
	}

	// 4. Pastikan diawali dengan 62 (misal inputnya hanya 812...)
	// Tapi biasanya di Indonesia formatnya 08 atau 628.
	// Jika ada yang input "812..." tanpa 0, kita asumsikan 628.
	if !strings.HasPrefix(cleaned, "62") {
		cleaned = "62" + cleaned
	}

	return cleaned
}
