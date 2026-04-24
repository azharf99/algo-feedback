// File: pkg/pdfgen/pdfgen.go
package pdfgen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

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
	Generate(ctx context.Context, data PDFData, outputPath string) error
}

type pdfGenerator struct {
	assetDir string // Folder untuk menyimpan header.png dan path.png
}

func NewPDFGenerator(assetDir string) PDFGenerator {
	return &pdfGenerator{assetDir: assetDir}
}

func (g *pdfGenerator) Generate(ctx context.Context, data PDFData, outputPath string) error {
	cfg := config.NewBuilder().
		WithPageNumber().
		WithTopMargin(0).
		WithLeftMargin(0).
		WithRightMargin(0).
		WithBottomMargin(10).
		Build()

	m := maroto.New(cfg)

	// 1. HEADER (Full Width Banner)
	m.AddRows(
		row.New(60).Add(
			col.New(12).Add(
				image.NewFromFile(filepath.Join(g.assetDir, "header.png"), props.Rect{
					Center:  true,
					Percent: 100,
				}),
			),
		),
	)

	// Margin Wrapper untuk konten utama
	// Karena maroto v2 menggunakan model komponen, kita bisa menambahkan padding di baris
	
	// 2. JUDUL RAPOR
	m.AddRows(
		row.New(20).Add(
			col.New(12).Add(
				text.New("LAPORAN PERKEMBANGAN BELAJAR", props.Text{
					Top:   5,
					Size:  16,
					Style: fontstyle.Bold,
					Align: align.Center,
					Color: &props.Color{Red: 20, Green: 20, Blue: 80},
				}),
				text.New(fmt.Sprintf("Siswa: %s | Level: %s", data.StudentName, data.StudentLevel), props.Text{
					Top:   12,
					Size:  10,
					Align: align.Center,
					Style: fontstyle.Italic,
				}),
			),
		),
	)

	// 3. INFORMASI UTAMA (Topik & Hasil)
	m.AddRows(
		row.New(10).Add(col.New(12)), // Spacer
		row.New(70).Add(
			// Kolom Kiri: Kurikulum
			col.New(7).Add(
				text.New("📚 TOPIK PEMBELAJARAN", props.Text{Style: fontstyle.Bold, Size: 11, Color: &props.Color{Red: 200, Green: 0, Blue: 0}}),
				text.New(data.ModuleTopic, props.Text{Top: 6, Size: 10, Left: 2}),

				text.New("✅ HASIL PEMBELAJARAN", props.Text{Top: 20, Style: fontstyle.Bold, Size: 11, Color: &props.Color{Red: 0, Green: 120, Blue: 0}}),
				text.New(data.ModuleResult, props.Text{Top: 26, Size: 10, Left: 2}),
				
				text.New("🛠️ KEAHLIAN YANG DIKUASAI", props.Text{Top: 40, Style: fontstyle.Bold, Size: 11, Color: &props.Color{Red: 0, Green: 0, Blue: 200}}),
				text.New(data.SkillResult, props.Text{Top: 46, Size: 10, Left: 2}),
			),
			// Kolom Kanan: Visual Path
			col.New(5).Add(
				text.New("🛤️ JALUR PENDIDIKAN", props.Text{Style: fontstyle.Bold, Size: 10, Align: align.Center}),
				image.NewFromFile(filepath.Join(g.assetDir, "path.png"), props.Rect{
					Top:     5,
					Center:  true,
					Percent: 90,
				}),
				text.New("Lihat Modul Lengkap di Sini:", props.Text{Top: 55, Size: 8, Align: align.Center}),
				text.New(data.StudentModuleLink, props.Text{Top: 60, Size: 7, Align: align.Center, Style: fontstyle.Italic, Color: &props.Color{Red: 0, Green: 0, Blue: 255}}),
			),
		),
	)

	// 4. TUTOR FEEDBACK (Area Luas)
	m.AddRows(
		row.New(10).Add(col.New(12)), // Spacer
		row.New(80).Add(
			col.New(12).Add(
				text.New("💬 CATATAN TUTOR (FEEDBACK)", props.Text{Style: fontstyle.Bold, Size: 11}),
				text.New(data.TeacherFeedback, props.Text{
					Top:  6,
					Size: 10,
					Align: align.Justify,
				}),
			),
		),
	)

	// 5. FOOTER / LINKS
	m.AddRows(
		row.New(30).Add(
			col.New(6).Add(
				text.New("🔗 Link Project Siswa:", props.Text{Style: fontstyle.Bold, Size: 9}),
				text.New(data.StudentProjectLink, props.Text{Top: 5, Size: 8, Color: &props.Color{Blue: 255}}),
			),
			col.New(6).Add(
				text.New("🎁 Program Referral:", props.Text{Style: fontstyle.Bold, Size: 9, Align: align.Right}),
				text.New("Dapatkan diskon dengan mengajak teman!", props.Text{Top: 5, Size: 8, Align: align.Right}),
				text.New(data.StudentReferralLink, props.Text{Top: 10, Size: 7, Align: align.Right, Color: &props.Color{Blue: 255}}),
			),
		),
	)

	// Generate PDF ke Memory
	doc, err := m.Generate()
	if err != nil {
		return err
	}

	// Pastikan folder tersedia
	err = os.MkdirAll(filepath.Dir(outputPath), os.ModePerm)
	if err != nil {
		return err
	}

	// Simpan ke File
	return doc.Save(outputPath)
}
