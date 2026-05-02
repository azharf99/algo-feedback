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
	"github.com/johnfercher/maroto/v2/pkg/consts/border" // IMPORT BARU UNTUK BORDER
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
		WithMaxGridSize(13).
		WithTopMargin(10).
		WithLeftMargin(10).
		WithRightMargin(10).
		WithBottomMargin(10).
		Build()

	m := maroto.New(cfg)

	// --- STYLE DEFINITIONS ---
	// 1. Mendefinisikan border ungu (tetap kita simpan untuk section di bawahnya)
	purpleBorder := &props.Cell{
		BorderType:      border.Full,                                 // Kotak penuh di 4 sisi
		BorderColor:     &props.Color{Red: 153, Green: 0, Blue: 255}, // Hex #9900FF
		BorderThickness: 0.5,                                         // Ketebalan garis
	}

	// 2. [BARU] Mendefinisikan warna background #D9D2E9 untuk Informasi Siswa
	infoBackgroundColor := &props.Cell{
		BackgroundColor: &props.Color{Red: 217, Green: 210, Blue: 233},
	}

	// 3. [BARU] Mendefinisikan warna background #D9D2E9 untuk Link Container
	linkBackgroundColor := &props.Cell{
		BackgroundColor: &props.Color{Red: 255, Green: 242, Blue: 204},
		BorderType:      border.Full,                                 // Kotak penuh di 4 sisi
		BorderColor:     &props.Color{Red: 153, Green: 0, Blue: 255}, // Hex #9900FF
		BorderThickness: 0.5,
	}

	// 1. BANNER (Full 14) - Tanpa Border
	m.AddRows(
		row.New(40).Add(
			col.New(13).Add(
				image.NewFromFile(filepath.Join(g.assetDir, "header.png"), props.Rect{
					Center:  true,
					Percent: 100,
				}),
			),
		),
	)

	m.AddRow(3) // Spacer

	// 2. INFORMASI SISWA & SKOR TOTAL [6 | 6]
	m.AddRows(
		row.New(30).Add(
			// Kiri: Informasi Siswa
			col.New(6).WithStyle(infoBackgroundColor).Add(
				text.New("INFORMASI SISWA", props.Text{Top: 2, Left: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(fmt.Sprintf("Nama Siswa		: %s", data.StudentName), props.Text{Top: 8, Left: 2, Size: 10}),
				text.New(fmt.Sprintf("Nama Kursus	: %s", data.StudentClass), props.Text{Top: 14, Left: 2, Size: 10}),
				text.New(fmt.Sprintf("Lama Pelatihan	: Bulan ke-%d", data.StudentMonthCourse), props.Text{Top: 20, Left: 2, Size: 10}),
			),
			// Ini adalah GAP / Spacer
			col.New(1),
			// Kanan: Skor Total
			col.New(6).WithStyle(infoBackgroundColor).Add(
				text.New("SKOR TOTAL", props.Text{Top: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(data.StudentLevel, props.Text{Top: 10, Size: 12, Align: align.Center, Style: fontstyle.Bold}),
				text.New("*****", props.Text{Top: 18, Size: 30, Align: align.Center}),
			),
		),
	)

	m.AddRow(3) // Spacer antar baris

	// 3. PROYEK SISWA & FREE LESSON [6 | 6]
	m.AddRows(
		row.New(20).Add(
			// Kiri: Proyek Hasil
			col.New(6).WithStyle(linkBackgroundColor).Add(
				text.New("Proyek Hasil Student", props.Text{Top: 2, Left: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New("Proyek akhir diakses melalui link dibawah ini:", props.Text{Top: 8, Left: 2, Size: 9, Align: align.Center}),
				text.New(data.StudentProjectLink, props.Text{Top: 14, Left: 2, Size: 5, Style: fontstyle.BoldItalic, Align: align.Center, Color: &props.Color{Red: 91, Green: 136, Blue: 239}}),
			),
			// Ini adalah GAP / Spacer
			col.New(1),
			// Kanan: Free Lesson
			col.New(6).WithStyle(linkBackgroundColor).Add(
				text.New("Free Lesson", props.Text{Top: 2, Left: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New("Mau dapatkan free lesson?", props.Text{Top: 8, Left: 2, Size: 9, Align: align.Center}),
				text.New("Bagikan link ini: "+data.StudentReferralLink, props.Text{Top: 14, Left: 2, Size: 10, Style: fontstyle.BoldItalic, Align: align.Center, Color: &props.Color{Red: 91, Green: 136, Blue: 239}}),
			),
		),
	)

	m.AddRow(3)

	// 4. TENTANG MODUL & KEAHLIAN [6 | 6]
	m.AddRows(
		row.New(80).Add(
			// Kiri: Tentang Modul
			col.New(6).WithStyle(purpleBorder).Add(
				text.New("Tentang Modul Ini", props.Text{Top: 2, Left: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New("Topik Modul: "+data.ModuleTopic, props.Text{Top: 10, Left: 4, Right: 4, Bottom: 4, Style: fontstyle.Bold, Size: 9}),
				text.New("Hasil: "+data.ModuleResult, props.Text{Top: 20, Left: 4, Right: 4, Bottom: 4, Size: 9}),
				text.New(fmt.Sprintf("Menyelesaikan bulan ke-%d di level %s/9", data.StudentMonthCourse, data.StudentLevel), props.Text{Top: 75, Left: 4, Right: 4, Size: 8, Style: fontstyle.Italic}),
			),
			// Ini adalah GAP / Spacer
			col.New(1),
			// Kanan: Keahlian
			col.New(6).WithStyle(purpleBorder).Add(
				text.New("Keahlian yang Didapatkan", props.Text{Top: 2, Left: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(data.SkillResult, props.Text{Top: 8, Left: 4, Right: 4, Bottom: 4, Size: 9, Align: align.Justify}),
			),
		),
	)

	m.AddRow(3)

	// 5. JALUR PENDIDIKAN & TUTOR FEEDBACK [6 | 6]
	m.AddRows(
		row.New(75).Add(
			// Kiri: Jalur Pendidikan (Image Path)
			col.New(6).WithStyle(linkBackgroundColor).Add(
				text.New("Jalur Pendidikan", props.Text{Top: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				image.NewFromFile(filepath.Join(g.assetDir, "path.png"), props.Rect{
					Top:     8,
					Center:  true,
					Percent: 80,
				}),
				text.New("Lihat Modul Lengkap:", props.Text{Top: 65, Left: 4, Right: 4, Size: 9, Align: align.Center}),
				text.New(data.StudentModuleLink, props.Text{Top: 70, Left: 4, Right: 4, Size: 9, Style: fontstyle.Bold, Align: align.Center, Color: &props.Color{Red: 91, Green: 136, Blue: 239}}),
			),
			// Ini adalah GAP / Spacer
			col.New(1),
			// Kanan: Tutor's Feedback
			col.New(6).WithStyle(purpleBorder).Add(
				text.New("Tutor's Feedback", props.Text{Top: 2, Left: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(data.TeacherFeedback, props.Text{Top: 8, Left: 4, Right: 4, Bottom: 4, Size: 9, Align: align.Justify}),
			),
		),
	)

	m.AddRow(3) // Spacer sebelum footer

	// 6. FOOTER (Full 12) - Tanpa Border
	m.AddRows(
		row.New(8).Add(
			col.New(13).Add(
				text.New("Laporan dibuat oleh: Azhar Faturohman Ahidin", props.Text{
					Size:  9,
					Style: fontstyle.Italic,
					Align: align.Left,
				}),
			),
		),
	)

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
