// File: pkg/pdfgen/pdfgen.go
package pdfgen

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)


// PDFData berisi semua parameter yang dibutuhkan oleh file index.html
type PDFData struct {
	StudentName         string
	StudentMonthCourse  uint
	StudentClass        string
	StudentLevel        string
	StudentProjectLink  string
	StudentReferralLink string
	StudentModuleLink   string
	ModuleTopic         string
	ModuleResult        string
	SkillResult         string
	TeacherFeedback     string
}

type PDFGenerator interface {
	GeneratePDFAsync(ctx context.Context, data PDFData, templateName string, outputPath string)
	GeneratePDFSync(ctx context.Context, data PDFData, templateName string, outputPath string) error
}

type pdfGenerator struct {
	templateDir string
}

func NewPDFGenerator(templateDir string) PDFGenerator {
	return &pdfGenerator{
		templateDir: templateDir,
	}
}

func (g *pdfGenerator) GeneratePDFAsync(ctx context.Context, data PDFData, templateName string, outputPath string) {
	// Menjalankan proses di background (Goroutine)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PDF-ERROR] Goroutine panic saat membuat PDF %s: %v\n", outputPath, r)
			}
		}()

		err := g.GeneratePDFSync(context.Background(), data, templateName, outputPath)
		if err != nil {
			log.Printf("[PDF-FAILED] Gagal membuat PDF untuk %s: %v\n", data.StudentName, err)
			return
		}
		log.Printf("[PDF-SUCCESS] Berhasil membuat PDF: %s\n", outputPath)
	}()
}

// GeneratePDFSync adalah mesin utama pembuat PDF menggunakan chromedp
func (g *pdfGenerator) GeneratePDFSync(ctx context.Context, data PDFData, templateName string, outputPath string) error {
	// 1. Parsing file HTML Template
	templatePath := filepath.Join(g.templateDir, templateName)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("gagal memuat template HTML: %w", err)
	}

	// 2. Merender HTML dengan data struct ke dalam memory (buffer)
	var renderedHTML bytes.Buffer
	if err := tmpl.Execute(&renderedHTML, data); err != nil {
		return fmt.Errorf("gagal merender data ke template HTML: %w", err)
	}

	// 3. Membuat file HTML sementara agar bisa dibuka oleh Chrome
	// (Menggunakan file sementara jauh lebih stabil untuk chromedp dibanding injeksi string langsung)
	tmpFile, err := os.CreateTemp("", "pdfgen-*.html")
	if err != nil {
		return fmt.Errorf("gagal membuat file temporary: %w", err)
	}
	// Pastikan file sementara dihapus setelah fungsi selesai
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(renderedHTML.Bytes()); err != nil {
		return fmt.Errorf("gagal menulis ke file temporary: %w", err)
	}
	tmpFile.Close() // Tutup file agar bisa diakses oleh Chrome

	// 4. Konfigurasi chromedp (Headless Chrome)
	// Gunakan allocCtx jika ingin mengatur parameter khusus (seperti mematikan sandbox di server Linux)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true), // Sangat penting jika dijalankan di dalam Docker/Linux
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	chromeCtx, cancelChrome := chromedp.NewContext(allocCtx)
	defer cancelChrome()

	// 5. Eksekusi Chrome untuk mencetak PDF
	var pdfBuffer []byte
	fileURL := "file://" + filepath.ToSlash(tmpFile.Name())

	err = chromedp.Run(chromeCtx,
		chromedp.Navigate(fileURL),
		// Tunggu hingga tag <body> muncul untuk memastikan render selesai
		chromedp.WaitVisible("body", chromedp.ByQuery),
		// Action khusus untuk generate PDF
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Atur ukuran kertas (A4), margin, dan background
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).   // Lebar A4 dalam Inchi
				WithPaperHeight(11.69). // Tinggi A4 dalam Inchi
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuffer = buf
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("chromedp gagal mencetak PDF: %w", err)
	}

	// 6. Menyimpan hasil PDF ke folder tujuan
	// Pastikan folder tujuan sudah ada, jika belum buatkan
	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return fmt.Errorf("gagal membuat direktori tujuan: %w", err)
	}

	if err := os.WriteFile(outputPath, pdfBuffer, 0644); err != nil {
		return fmt.Errorf("gagal menyimpan file PDF: %w", err)
	}

	return nil
}
