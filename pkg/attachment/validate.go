// File: pkg/attachment/validate.go
// Package attachment memvalidasi file yang diunggah user (mis. lampiran chat Help Center)
// menggunakan magic bytes/isi file sungguhan — TIDAK PERNAH mempercayai Content-Type header
// atau ekstensi nama file yang dikirim client, karena keduanya trivial dipalsukan.
package attachment

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

// MaxSize adalah batas ukuran maksimum satu attachment. Sengaja dibuat kecil untuk
// mencegah penyalahgunaan storage server lewat upload file besar berulang kali
// (denial-of-service via disk exhaustion).
const MaxSize = 10 << 20 // 10 MB

var (
	ErrTypeNotAllowed = errors.New("tipe file tidak diizinkan")
)

// sniffOnlyTypes memetakan mime type hasil http.DetectContentType (setelah parameter
// seperti "; charset=..." dibuang) ke ekstensi penyimpanan. Jenis-jenis ini punya magic
// bytes yang kuat dan didukung penuh oleh Go stdlib, sehingga sangat sulit dipalsukan.
// SVG sengaja TIDAK dimasukkan meski technically image — SVG bisa memuat <script> dan
// menimbulkan stored-XSS jika pernah dirender langsung oleh browser.
var sniffOnlyTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
	"text/plain":      ".txt",
}

// zipBasedExt: format dokumen modern (OOXML) yang sebenarnya adalah arsip ZIP. Go stdlib
// tidak punya signature spesifik untuk membedakan docx/xlsx/pptx dari file zip biasa, jadi
// kita hanya memvalidasi bahwa isinya memang arsip ZIP yang valid, berdasarkan ekstensi
// yang diklaim user di nama file aslinya.
var zipBasedExt = map[string]bool{".docx": true, ".xlsx": true, ".pptx": true}

// oleBasedExt: format dokumen lama (OLE Compound File) — Word/Excel/PowerPoint versi lama.
var oleBasedExt = map[string]bool{".doc": true, ".xls": true, ".ppt": true}

var (
	zipSignature = []byte{0x50, 0x4B, 0x03, 0x04}
	oleSignature = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
)

var extensionsForCSV = map[string]bool{".csv": true}

// Result adalah hasil validasi sebuah attachment yang aman disimpan.
type Result struct {
	// Extension adalah ekstensi penyimpanan yang SUDAH tervalidasi lewat magic bytes
	// (bukan sekadar diambil dari nama file klaim user), mis. ".png".
	Extension string
	// MimeType dipakai sebagai header Content-Type saat file disajikan kembali —
	// jangan pernah pakai Content-Type yang dikirim client saat upload.
	MimeType string
	IsImage  bool
}

// Validate memeriksa isi (magic bytes) sebuah file terhadap whitelist tipe yang
// diizinkan. header harus berisi beberapa ratus byte pertama file (lihat
// http.DetectContentType — 512 byte sudah cukup).
func Validate(originalFilename string, header []byte) (*Result, error) {
	claimedExt := strings.ToLower(filepath.Ext(originalFilename))

	sniffed := http.DetectContentType(header)
	mimeType := sniffed
	if parsed, _, err := mime.ParseMediaType(sniffed); err == nil {
		mimeType = parsed
	}

	if ext, ok := sniffOnlyTypes[mimeType]; ok {
		// CSV umumnya juga terdeteksi sebagai text/plain karena tidak punya magic bytes
		// sendiri — pertahankan ekstensi .csv aslinya untuk tampilan jika itu yang diklaim,
		// selama isinya memang tervalidasi sebagai teks polos.
		if mimeType == "text/plain" && extensionsForCSV[claimedExt] {
			return &Result{Extension: ".csv", MimeType: "text/csv", IsImage: false}, nil
		}
		return &Result{Extension: ext, MimeType: mimeType, IsImage: strings.HasPrefix(mimeType, "image/")}, nil
	}

	if zipBasedExt[claimedExt] && bytes.HasPrefix(header, zipSignature) {
		return &Result{Extension: claimedExt, MimeType: officeMimeFor(claimedExt), IsImage: false}, nil
	}

	if oleBasedExt[claimedExt] && bytes.HasPrefix(header, oleSignature) {
		return &Result{Extension: claimedExt, MimeType: officeMimeFor(claimedExt), IsImage: false}, nil
	}

	return nil, ErrTypeNotAllowed
}

func officeMimeFor(ext string) string {
	switch ext {
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".doc":
		return "application/msword"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	}
	return "application/octet-stream"
}

// unsafeFilenameChars mencakup karakter kontrol, separator path, dan karakter yang bisa
// dipakai untuk header injection (mis. lewat Content-Disposition) atau path traversal.
var unsafeFilenameChars = regexp.MustCompile(`[\x00-\x1f\x7f/\\"]`)

// SanitizeDisplayName membersihkan nama file asli dari user untuk keperluan TAMPILAN
// dan header Content-Disposition saja. Nama ini TIDAK PERNAH dipakai sebagai nama file
// di disk — nama file di disk selalu dibuat server lewat RandomStorageName, sehingga
// path traversal (mis. "../../etc/passwd") atau penimpaan file lain sama sekali tidak
// mungkin terjadi lewat input ini.
func SanitizeDisplayName(original string) string {
	name := filepath.Base(strings.TrimSpace(original))
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	if name == "" || name == "." || name == ".." {
		name = "attachment"
	}
	const maxLen = 150
	if len(name) > maxLen {
		ext := filepath.Ext(name)
		name = name[:maxLen-len(ext)] + ext
	}
	return name
}

// RandomStorageName menghasilkan nama file acak (tidak bisa ditebak) untuk disimpan di
// disk, sehingga nama file asli dari user tidak pernah dipakai untuk operasi filesystem.
func RandomStorageName(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ext, nil
}
