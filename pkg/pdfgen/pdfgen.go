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
		WithLeftMargin(10). // Margin kiri/kanan sedikit agar tidak terlalu mepet layar
		WithRightMargin(10).
		WithBottomMargin(10).
		Build()

	m := maroto.New(cfg)

	// 1. BANNER (Full 12)
	m.AddRows(
		row.New(50).Add(
			col.New(12).Add(
				image.NewFromFile(filepath.Join(g.assetDir, "header.png"), props.Rect{
					Center:  true,
					Percent: 100,
				}),
			),
		),
	)

	// Spacer (Beri jarak antar baris)
	m.AddRow(10)

	// 2. INFORMASI SISWA & SKOR TOTAL [6 | 6]
	m.AddRows(
		row.New(30).Add(
			// Kiri: Informasi Siswa
			col.New(6).Add(
				text.New("INFORMASI SISWA", props.Text{Style: fontstyle.Bold, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}), // Warna #3F1F75
				text.New(fmt.Sprintf("Nama Siswa: %s", data.StudentName), props.Text{Top: 6, Size: 10}),
				text.New(fmt.Sprintf("Kursus: %s", data.StudentClass), props.Text{Top: 12, Size: 10}),
				text.New(fmt.Sprintf("Lama Pelatihan: Bulan ke-%d", data.StudentMonthCourse), props.Text{Top: 18, Size: 10}),
			),
			// Kanan: Skor Total
			col.New(6).Add(
				text.New("SKOR TOTAL", props.Text{Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(data.StudentLevel, props.Text{Top: 8, Size: 16, Align: align.Center, Style: fontstyle.Bold}),
				text.New("⭐⭐⭐⭐⭐", props.Text{Top: 16, Size: 14, Align: align.Center}),
			),
		),
	)

	m.AddRow(5) // Spacer

	// 3. PROYEK SISWA & FREE LESSON [6 | 6]
	m.AddRows(
		row.New(25).Add(
			// Kiri: Proyek Hasil
			col.New(6).Add(
				text.New("🎓 Proyek hasil Student", props.Text{Style: fontstyle.Bold, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New("Proyek akhir diakses melalui link dibawah ini:", props.Text{Top: 6, Size: 9}),
				text.New("👉 "+data.StudentProjectLink, props.Text{Top: 12, Size: 8, Style: fontstyle.Italic, Color: &props.Color{Red: 91, Green: 136, Blue: 239}}),
			),
			// Kanan: Free Lesson
			col.New(6).Add(
				text.New("💻 Free Lesson", props.Text{Style: fontstyle.Bold, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New("🎁 Mau dapatkan free lesson?", props.Text{Top: 6, Size: 9}),
				text.New("👉 Bagikan link ini: "+data.StudentReferralLink, props.Text{Top: 12, Size: 8, Style: fontstyle.Italic, Color: &props.Color{Red: 91, Green: 136, Blue: 239}}),
			),
		),
	)

	m.AddRow(5) // Spacer

	// 4. TENTANG MODUL & KEAHLIAN [6 | 6]
	m.AddRows(
		row.New(40).Add(
			// Kiri: Tentang Modul
			col.New(6).Add(
				text.New("📚 Tentang Modul Ini", props.Text{Style: fontstyle.Bold, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New("Topik Modul: "+data.ModuleTopic, props.Text{Top: 6, Size: 9}),
				text.New("Hasil: "+data.ModuleResult, props.Text{Top: 12, Size: 9}),
				text.New(fmt.Sprintf("Menyelesaikan bulan ke-%d di level %s/9", data.StudentMonthCourse, data.StudentLevel), props.Text{Top: 20, Size: 8, Style: fontstyle.Italic}),
			),
			// Kanan: Keahlian
			col.New(6).Add(
				text.New("💻 Keahlian yang Didapatkan", props.Text{Style: fontstyle.Bold, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(data.SkillResult, props.Text{Top: 6, Size: 9}),
			),
		),
	)

	m.AddRow(5) // Spacer

	// 5. JALUR PENDIDIKAN & TUTOR FEEDBACK [6 | 6]
	m.AddRows(
		row.New(80).Add(
			// Kiri: Jalur Pendidikan (Image Path)
			col.New(6).Add(
				text.New("Jalur Pendidikan", props.Text{Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				image.NewFromFile(filepath.Join(g.assetDir, "path.png"), props.Rect{
					Top:     6,
					Center:  true,
					Percent: 80,
				}),
				text.New("Lihat Modul Lengkap:", props.Text{Top: 60, Size: 9, Align: align.Center}),
				text.New(data.StudentModuleLink, props.Text{Top: 65, Size: 7, Align: align.Center, Color: &props.Color{Red: 91, Green: 136, Blue: 239}}),
			),
			// Kanan: Tutor's Feedback
			col.New(6).Add(
				text.New("📝 Tutor's Feedback", props.Text{Style: fontstyle.Bold, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(data.TeacherFeedback, props.Text{Top: 6, Size: 9, Align: align.Justify}),
			),
		),
	)

	m.AddRow(10) // Spacer sebelum footer

	// 6. FOOTER (Full 12)
	m.AddRows(
		row.New(10).Add(
			col.New(12).Add(
				text.New("Laporan dibuat oleh: Azhar Faturohman Ahidin", props.Text{
					Size:  9,
					Style: fontstyle.Italic,
					Align: align.Left,
				}),
			),
		),
	)

	// ... (Sisa kode simpan dokumen sama seperti sebelumnya) ...
	doc, err := m.Generate()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(outputPath), os.ModePerm)
	if err != nil {
		return err
	}

	return doc.Save(outputPath)
}
