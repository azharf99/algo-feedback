// File: pkg/attachment/validate_test.go
package attachment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_RealImageAccepted(t *testing.T) {
	// PNG signature yang valid (8 byte pertama), sisanya tidak relevan untuk sniffing.
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}

	result, err := Validate("evidence.png", png)
	require.NoError(t, err)
	assert.Equal(t, ".png", result.Extension)
	assert.Equal(t, "image/png", result.MimeType)
	assert.True(t, result.IsImage)
}

func TestValidate_JpegDisguisedWithFakeExtension(t *testing.T) {
	// Sengaja beri nama file .pdf padahal isinya JPEG asli — storage extension harus
	// tetap mengikuti isi SEBENARNYA (magic bytes), bukan nama file klaim user.
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0, 0, 0}

	result, err := Validate("not-a-pdf.pdf", jpeg)
	require.NoError(t, err)
	assert.Equal(t, ".jpg", result.Extension, "storage extension harus mengikuti magic bytes, bukan nama file klaim user")
	assert.True(t, result.IsImage)
}

func TestValidate_RejectsHtmlDisguisedAsImage(t *testing.T) {
	// Klasik MIME-sniffing XSS: file HTML/script yang diberi nama "photo.jpg".
	maliciousHTML := []byte("<html><body><script>alert(document.cookie)</script></body></html>")

	_, err := Validate("photo.jpg", maliciousHTML)
	assert.ErrorIs(t, err, ErrTypeNotAllowed)
}

func TestValidate_RejectsSVGEvenThoughItsAnImage(t *testing.T) {
	// SVG bisa memuat <script> — sengaja tidak masuk whitelist meski secara nominal "image".
	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)

	_, err := Validate("logo.svg", svg)
	assert.ErrorIs(t, err, ErrTypeNotAllowed)
}

func TestValidate_RejectsExecutable(t *testing.T) {
	// Windows PE header ("MZ") — tidak boleh pernah lolos meski diberi nama .pdf/.docx.
	exe := []byte{0x4D, 0x5A, 0x90, 0x00, 0x03, 0x00, 0x00, 0x00}

	_, err := Validate("resume.pdf", exe)
	assert.ErrorIs(t, err, ErrTypeNotAllowed)

	_, err = Validate("resume.docx", exe)
	assert.ErrorIs(t, err, ErrTypeNotAllowed, "PE header bukan ZIP valid, docx palsu ini harus ditolak")
}

func TestValidate_AcceptsValidZipAsDocx(t *testing.T) {
	zipHeader := []byte{0x50, 0x4B, 0x03, 0x04, 0, 0, 0, 0}

	result, err := Validate("Laporan.docx", zipHeader)
	require.NoError(t, err)
	assert.Equal(t, ".docx", result.Extension)
	assert.False(t, result.IsImage)
}

func TestValidate_RejectsZipRenamedAsExeOrUnknownExt(t *testing.T) {
	zipHeader := []byte{0x50, 0x4B, 0x03, 0x04, 0, 0, 0, 0}

	// ZIP valid tapi ekstensi klaim bukan salah satu dokumen OOXML yang diizinkan —
	// jangan diam-diam terima sebagai "application/zip" umum.
	_, err := Validate("archive.zip", zipHeader)
	assert.ErrorIs(t, err, ErrTypeNotAllowed)
}

func TestValidate_AcceptsLegacyOfficeOLEFormat(t *testing.T) {
	ole := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1, 0, 0}

	result, err := Validate("old_report.doc", ole)
	require.NoError(t, err)
	assert.Equal(t, ".doc", result.Extension)
}

func TestValidate_PDFAccepted(t *testing.T) {
	pdf := []byte("%PDF-1.4\n%\xe2\xe3\xcf\xd3")

	result, err := Validate("bukti.pdf", pdf)
	require.NoError(t, err)
	assert.Equal(t, ".pdf", result.Extension)
	assert.Equal(t, "application/pdf", result.MimeType)
}

func TestSanitizeDisplayName_StripsPathTraversal(t *testing.T) {
	assert.Equal(t, "passwd", SanitizeDisplayName("../../etc/passwd"))
	assert.Equal(t, "passwd", SanitizeDisplayName("..\\..\\windows\\passwd"))
}

func TestSanitizeDisplayName_StripsControlAndQuoteChars(t *testing.T) {
	name := SanitizeDisplayName("evil\"; rm -rf /\x00.jpg")
	assert.NotContains(t, name, "\"")
	assert.NotContains(t, name, "\x00")
}

func TestSanitizeDisplayName_HandlesEmptyOrDotOnly(t *testing.T) {
	assert.Equal(t, "attachment", SanitizeDisplayName(""))
	assert.Equal(t, "attachment", SanitizeDisplayName("."))
	assert.Equal(t, "attachment", SanitizeDisplayName(".."))
}

func TestSanitizeDisplayName_TruncatesVeryLongNames(t *testing.T) {
	longName := strings.Repeat("a", 300) + ".png"
	result := SanitizeDisplayName(longName)
	assert.LessOrEqual(t, len(result), 150)
	assert.True(t, strings.HasSuffix(result, ".png"))
}

func TestRandomStorageName_IsUnpredictableAndKeepsExtension(t *testing.T) {
	name1, err := RandomStorageName(".png")
	require.NoError(t, err)
	name2, err := RandomStorageName(".png")
	require.NoError(t, err)

	assert.NotEqual(t, name1, name2)
	assert.True(t, strings.HasSuffix(name1, ".png"))
}
