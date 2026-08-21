// File: pkg/csvutil/csvutil.go
// Package csvutil menyediakan helper generik untuk menghasilkan file CSV
// dari data tabular (headers + rows), dipakai oleh fitur Export Data.
package csvutil

import (
	"bytes"
	"encoding/csv"
)

// utf8BOM ditambahkan di awal file agar Microsoft Excel membaca karakter
// non-ASCII (mis. nama dengan huruf beraksen) dengan benar.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Generate membangun file CSV (sebagai []byte) dari header kolom dan baris data.
// Setiap elemen di rows harus memiliki panjang yang sama dengan headers.
func Generate(headers []string, rows [][]string) ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.Write(utf8BOM)

	writer := csv.NewWriter(buf)

	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
