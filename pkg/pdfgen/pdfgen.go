// File: pkg/pdfgen/pdfgen.go
package pdfgen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type PDFData struct {
	Lang                string
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
	lang := data.Lang
	if lang == "" {
		lang = "Indonesia"
	}

	cfg := config.NewBuilder().
		WithPageNumber().
		WithMaxGridSize(15).
		WithTopMargin(10).
		WithLeftMargin(10).
		WithRightMargin(10).
		WithBottomMargin(10).
		Build()

	m := maroto.New(cfg)

	// --- STYLE DEFINITIONS ---
	purpleBorder := &props.Cell{
		BorderType:      border.Full,
		BorderColor:     &props.Color{Red: 153, Green: 0, Blue: 255},
		BorderThickness: 0.5,
	}

	infoBackgroundColor := &props.Cell{
		BackgroundColor: &props.Color{Red: 217, Green: 210, Blue: 233},
	}

	linkBackgroundColor := &props.Cell{
		BackgroundColor: &props.Color{Red: 255, Green: 242, Blue: 204},
		BorderType:      border.Full,
		BorderColor:     &props.Color{Red: 153, Green: 0, Blue: 255},
		BorderThickness: 0.5,
	}

	// 1. BANNER
	m.AddRows(
		row.New(40).Add(
			col.New(15).Add(
				image.NewFromFile(filepath.Join(g.assetDir, "header.png"), props.Rect{
					Center:  true,
					Percent: 100,
				}),
			),
		),
	)

	m.AddRow(3)

	// 2. INFORMASI SISWA & SKOR TOTAL
	m.AddRows(
		row.New(30).Add(
			col.New(7).WithStyle(infoBackgroundColor).Add(
				text.New(i18n.T(lang, "pdf_student_info"), props.Text{Top: 2, Left: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(fmt.Sprintf("%s		: %s", i18n.T(lang, "pdf_student_name"), data.StudentName), props.Text{Top: 8, Left: 2, Size: 9}),
				text.New(fmt.Sprintf("%s	: %s", i18n.T(lang, "pdf_course_name"), data.StudentClass), props.Text{Top: 14, Left: 2, Size: 9}),
				text.New(fmt.Sprintf("%s	: %s", i18n.T(lang, "pdf_training_duration"), i18n.Tf(lang, "pdf_month_count", data.StudentMonthCourse)), props.Text{Top: 20, Left: 2, Size: 9}),
			),
			col.New(1),
			col.New(7).WithStyle(infoBackgroundColor).Add(
				text.New(i18n.T(lang, "pdf_total_score"), props.Text{Top: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(data.StudentLevel, props.Text{Top: 10, Size: 12, Align: align.Center, Style: fontstyle.Bold}),
				image.NewFromFile(filepath.Join(g.assetDir, "star.png"), props.Rect{Top: 16, Left: 20, Percent: 30}),
				image.NewFromFile(filepath.Join(g.assetDir, "star.png"), props.Rect{Top: 16, Left: 30, Percent: 30}),
				image.NewFromFile(filepath.Join(g.assetDir, "star.png"), props.Rect{Top: 16, Left: 40, Percent: 30}),
				image.NewFromFile(filepath.Join(g.assetDir, "star.png"), props.Rect{Top: 16, Left: 50, Percent: 30}),
				image.NewFromFile(filepath.Join(g.assetDir, "star.png"), props.Rect{Top: 16, Left: 60, Percent: 30}),
			),
		),
	)

	m.AddRow(3)

	// 3. PROYEK SISWA & FREE LESSON
	m.AddRows(
		row.New(20).Add(
			col.New(7).WithStyle(linkBackgroundColor).Add(
				image.NewFromFile(filepath.Join(g.assetDir, "present.png"), props.Rect{Top: 2, Left: 16, Percent: 25}),
				text.New(i18n.T(lang, "pdf_student_project"), props.Text{Top: 2.5, Left: 6, Style: fontstyle.Bold, Align: align.Center, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(i18n.T(lang, "pdf_project_link_desc"), props.Text{Top: 8, Left: 2, Size: 9, Align: align.Center}),
				text.New(data.StudentProjectLink, props.Text{Top: 14, Left: 2, Size: 5, Style: fontstyle.BoldItalic, Align: align.Center, Color: &props.Color{Red: 91, Green: 136, Blue: 239}}),
			),
			col.New(1),
			col.New(7).WithStyle(linkBackgroundColor).Add(
				image.NewFromFile(filepath.Join(g.assetDir, "computer.png"), props.Rect{Top: 2, Left: 22, Percent: 30}),
				text.New(i18n.T(lang, "pdf_free_lesson"), props.Text{Top: 2.5, Left: 8, Style: fontstyle.Bold, Align: align.Center, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(i18n.T(lang, "pdf_free_lesson_desc"), props.Text{Top: 8, Left: 2, Size: 9, Align: align.Center}),
				text.New(i18n.T(lang, "pdf_share_link")+" "+data.StudentReferralLink, props.Text{Top: 14, Left: 2, Size: 10, Style: fontstyle.BoldItalic, Align: align.Center, Color: &props.Color{Red: 91, Green: 136, Blue: 239}}),
			),
		),
	)

	m.AddRow(3)

	// 4. TENTANG MODUL & KEAHLIAN
	m.AddRows(
		row.New(80).Add(
			col.New(7).WithStyle(purpleBorder).Add(
				image.NewFromFile(filepath.Join(g.assetDir, "notebook.png"), props.Rect{Top: 2, Left: 18, Percent: 8}),
				text.New(i18n.T(lang, "pdf_about_module"), props.Text{Top: 2.5, Left: 4, Style: fontstyle.Bold, Align: align.Center, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(i18n.T(lang, "pdf_module_topic")+" "+data.ModuleTopic, props.Text{Top: 10, Left: 4, Right: 4, Bottom: 4, Style: fontstyle.Bold, Size: 9}),
				text.New(i18n.T(lang, "pdf_module_result")+" "+data.ModuleResult, props.Text{Top: 20, Left: 4, Right: 4, Bottom: 4, Size: 9}),
				text.New(i18n.Tf(lang, "pdf_completion_footer", data.StudentMonthCourse, data.StudentLevel), props.Text{Top: 75, Left: 4, Right: 4, Size: 8, Style: fontstyle.Italic}),
			),
			col.New(1),
			col.New(7).WithStyle(purpleBorder).Add(
				image.NewFromFile(filepath.Join(g.assetDir, "computer.png"), props.Rect{Top: 2, Left: 16, Percent: 8}),
				text.New(i18n.T(lang, "pdf_skills_acquired"), props.Text{Top: 2.5, Left: 4, Style: fontstyle.Bold, Align: align.Center, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(data.SkillResult, props.Text{Top: 8, Left: 4, Right: 4, Bottom: 4, Size: 9, Align: align.Justify}),
			),
		),
	)

	m.AddRow(3)

	// 5. JALUR PENDIDIKAN & TUTOR FEEDBACK
	m.AddRows(
		row.New(75).Add(
			col.New(7).WithStyle(linkBackgroundColor).Add(
				text.New(i18n.T(lang, "pdf_education_path"), props.Text{Top: 2, Style: fontstyle.Bold, Size: 11, Align: align.Center, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				image.NewFromFile(filepath.Join(g.assetDir, "path.png"), props.Rect{
					Top:     8,
					Center:  true,
					Percent: 80,
				}),
				text.New(i18n.T(lang, "pdf_see_full_module"), props.Text{Top: 65, Left: 4, Right: 4, Size: 9, Align: align.Center}),
				text.New(data.StudentModuleLink, props.Text{Top: 70, Left: 4, Right: 4, Size: 9, Style: fontstyle.Bold, Align: align.Center, Color: &props.Color{Red: 91, Green: 136, Blue: 239}}),
			),
			col.New(1),
			col.New(7).WithStyle(purpleBorder).Add(
				image.NewFromFile(filepath.Join(g.assetDir, "checklist.png"), props.Rect{Top: 2, Left: 16, Percent: 8}),
				text.New(i18n.T(lang, "pdf_tutor_feedback"), props.Text{Top: 2.5, Left: 4, Style: fontstyle.Bold, Align: align.Center, Size: 11, Color: &props.Color{Red: 63, Green: 31, Blue: 117}}),
				text.New(data.TeacherFeedback, props.Text{Top: 8, Left: 4, Right: 4, Bottom: 4, Size: 9, Align: align.Justify}),
			),
		),
	)

	m.AddRow(3)

	// 6. FOOTER
	m.AddRows(
		row.New(8).Add(
			col.New(15).Add(
				text.New(i18n.T(lang, "pdf_report_created_by")+" Azhar Faturohman Ahidin", props.Text{
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
