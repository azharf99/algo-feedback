// File: pkg/pdfgen/graduation.go
package pdfgen

import (
	"context"
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

type LessonReport struct {
	LessonNumber string
	Topic        string
	Score        string
	Grade        string
	SessionTitle string
}

type GraduationPDFData struct {
	Lang          string
	StudentName   string
	CourseName    string
	LessonRange   string
	TeacherName   string
	TutorFeedback string
	ReferralLink  string
	Lessons       []LessonReport
}

type GraduationPDFGenerator interface {
	Generate(ctx context.Context, data GraduationPDFData, outputPath string) error
}

type graduationPDFGenerator struct {
	assetDir string
}

func NewGraduationPDFGenerator(assetDir string) GraduationPDFGenerator {
	return &graduationPDFGenerator{assetDir: assetDir}
}

func (g *graduationPDFGenerator) Generate(ctx context.Context, data GraduationPDFData, outputPath string) error {
	lang := data.Lang
	if lang == "" {
		lang = "Indonesia"
	}

	cfg := config.NewBuilder().
		WithPageNumber().
		WithMaxGridSize(12).
		WithTopMargin(10).
		WithLeftMargin(10).
		WithRightMargin(10).
		WithBottomMargin(10).
		Build()

	m := maroto.New(cfg)

	// --- STYLE DEFINITIONS ---
	headerBackgroundColor := &props.Color{Red: 63, Green: 31, Blue: 117}
	whiteColor := &props.Color{Red: 255, Green: 255, Blue: 255}

	infoValueStyle := &props.Cell{
		BackgroundColor: &props.Color{Red: 245, Green: 242, Blue: 250},
		BorderType:      border.Full,
		BorderColor:     &props.Color{Red: 217, Green: 210, Blue: 233},
		BorderThickness: 0.5,
	}

	tableHeaderStyle := &props.Cell{
		BackgroundColor: &props.Color{Red: 63, Green: 31, Blue: 117},
	}

	courseCellStyle := &props.Cell{
		BackgroundColor: &props.Color{Red: 245, Green: 242, Blue: 250},
		BorderType:      border.Full,
		BorderColor:     &props.Color{Red: 217, Green: 210, Blue: 233},
		BorderThickness: 0.3,
	}

	referralBg := &props.Cell{
		BackgroundColor: &props.Color{Red: 255, Green: 242, Blue: 204},
		BorderType:      border.Full,
		BorderColor:     &props.Color{Red: 153, Green: 0, Blue: 255},
		BorderThickness: 0.5,
	}

	// 1. BANNER / LOGO
	m.AddRows(
		row.New(25).Add(
			col.New(12).Add(
				image.NewFromFile(filepath.Join(g.assetDir, "header.png"), props.Rect{
					Center:  false,
					Percent: 100,
				}),
			),
		),
	)

	m.AddRow(2)

	// 2. TITLE
	m.AddRows(
		row.New(10).Add(
			col.New(12).Add(
				text.New(i18n.T(lang, "pdf_grad_report_title"), props.Text{
					Style: fontstyle.Bold,
					Size:  14,
					Align: align.Center,
					Color: headerBackgroundColor,
				}),
			),
		),
	)

	m.AddRow(2)

	// 3. STUDENT INFO
	m.AddRows(
		row.New(6).Add(
			col.New(3).Add(text.New(i18n.T(lang, "pdf_grad_student_name")+" :", props.Text{Style: fontstyle.Bold, Size: 9, Color: headerBackgroundColor})),
			col.New(9).WithStyle(infoValueStyle).Add(text.New(data.StudentName, props.Text{Size: 9, Left: 2, Top: 1})),
		),
		row.New(1),
		row.New(6).Add(
			col.New(3).Add(text.New(i18n.T(lang, "pdf_grad_course_name")+" :", props.Text{Style: fontstyle.Bold, Size: 9, Color: headerBackgroundColor})),
			col.New(9).WithStyle(infoValueStyle).Add(text.New(data.CourseName, props.Text{Size: 9, Left: 2, Top: 1})),
		),
		row.New(1),
		row.New(6).Add(
			col.New(3).Add(text.New(i18n.T(lang, "pdf_grad_lesson")+" :", props.Text{Style: fontstyle.Bold, Size: 9, Color: headerBackgroundColor})),
			col.New(9).WithStyle(infoValueStyle).Add(text.New(data.LessonRange, props.Text{Size: 9, Left: 2, Top: 1})),
		),
		row.New(1),
		row.New(6).Add(
			col.New(3).Add(text.New(i18n.T(lang, "pdf_grad_teacher_name")+" :", props.Text{Style: fontstyle.Bold, Size: 9, Color: headerBackgroundColor})),
			col.New(9).WithStyle(infoValueStyle).Add(text.New(data.TeacherName, props.Text{Size: 9, Left: 2, Top: 1})),
		),
	)

	m.AddRow(4)

	// 4. PENILAIAN KEMAMPUAN TITLE
	m.AddRows(
		row.New(6).Add(
			col.New(12).Add(
				text.New(i18n.T(lang, "pdf_grad_penilaian"), props.Text{
					Style: fontstyle.Bold,
					Size:  11,
					Color: headerBackgroundColor,
				}),
			),
		),
	)

	m.AddRow(2)

	// 5. PENILAIAN TABLE
	m.AddRows(
		row.New(8).Add(
			col.New(1).WithStyle(tableHeaderStyle).Add(text.New("No", props.Text{Align: align.Center, Color: whiteColor, Style: fontstyle.Bold, Size: 8, Top: 2})),
			col.New(3).WithStyle(tableHeaderStyle).Add(text.New("Topic", props.Text{Align: align.Center, Color: whiteColor, Style: fontstyle.Bold, Size: 8, Top: 2})),
			col.New(1).WithStyle(tableHeaderStyle).Add(text.New("Score", props.Text{Align: align.Center, Color: whiteColor, Style: fontstyle.Bold, Size: 8, Top: 2})),
			col.New(1).WithStyle(tableHeaderStyle).Add(text.New("Grade", props.Text{Align: align.Center, Color: whiteColor, Style: fontstyle.Bold, Size: 8, Top: 2})),
			col.New(6).WithStyle(tableHeaderStyle).Add(text.New("Session Title", props.Text{Align: align.Center, Color: whiteColor, Style: fontstyle.Bold, Size: 8, Top: 2})),
		),
	)

	for i, l := range data.Lessons {
		var bg *props.Cell
		if i%2 == 1 {
			bg = &props.Cell{
				BackgroundColor: &props.Color{Red: 245, Green: 242, Blue: 250},
			}
		}

		m.AddRows(
			row.New().Add(
				col.New(1).WithStyle(bg).Add(text.New(l.LessonNumber, props.Text{Align: align.Center, Size: 7.5, Top: 1, Bottom: 1})),
				col.New(3).WithStyle(bg).Add(text.New(l.Topic, props.Text{Align: align.Left, Size: 7.5, Top: 1, Bottom: 1})),
				col.New(1).WithStyle(bg).Add(text.New(l.Score, props.Text{Align: align.Center, Size: 7.5, Top: 1, Bottom: 1})),
				col.New(1).WithStyle(bg).Add(text.New(l.Grade, props.Text{Align: align.Center, Size: 7.5, Top: 1, Bottom: 1})),
				col.New(6).WithStyle(bg).Add(text.New(l.SessionTitle, props.Text{Align: align.Left, Size: 7.5, Top: 1, Bottom: 1})),
			),
		)
	}

	m.AddRow(4)

	// 6. TUTOR FEEDBACK
	m.AddRows(
		row.New(6).Add(
			col.New(12).Add(
				text.New(i18n.T(lang, "pdf_grad_tutor_feedback"), props.Text{
					Style: fontstyle.Bold,
					Size:  11,
					Color: headerBackgroundColor,
				}),
			),
		),
		row.New().Add(
			col.New(12).Add(
				text.New(data.TutorFeedback, props.Text{
					Size:  8.5,
					Align: align.Justify,
				}),
			),
		),
	)

	m.AddRow(4)

	// 7. KURSUS KAMI
	m.AddRows(
		row.New(6).Add(
			col.New(12).Add(
				text.New(i18n.T(lang, "pdf_grad_kursus"), props.Text{
					Style: fontstyle.Bold,
					Size:  11,
					Color: headerBackgroundColor,
				}),
			),
		),
	)

	courses := [][]string{
		{"Coding Knight", "Digital Literacy"},
		{"Game Design", "Visual Programming"},
		{"Artificial Intelligence", "Math"},
		{"Graphic Design Junior", "Graphic Design Senior"},
		{"Python Start (2nd Year)", "Python Pro (1st Year)"},
		{"Python Pro (2nd Year)", "Web Development"},
	}

	for _, pair := range courses {
		m.AddRows(
			row.New(5).Add(
				col.New(6).WithStyle(courseCellStyle).Add(text.New(pair[0], props.Text{Size: 8, Left: 2, Top: 1})),
				col.New(6).WithStyle(courseCellStyle).Add(text.New(pair[1], props.Text{Size: 8, Left: 2, Top: 1})),
			),
		)
	}

	m.AddRow(4)

	// 8. REFERRAL LINK & BENEFITS
	m.AddRows(
		row.New(6).Add(
			col.New(12).Add(
				text.New(i18n.T(lang, "pdf_grad_referral"), props.Text{
					Style: fontstyle.Bold,
					Size:  11,
					Color: headerBackgroundColor,
				}),
			),
		),
		row.New(22).Add(
			col.New(12).WithStyle(referralBg).Add(
				text.New(data.ReferralLink, props.Text{
					Top:   2,
					Left:  4,
					Size:  9,
					Style: fontstyle.BoldItalic,
					Color: &props.Color{Red: 91, Green: 136, Blue: 239},
				}),
				text.New(i18n.T(lang, "pdf_grad_referral_benefit"), props.Text{
					Top:  7,
					Left: 4,
					Size: 8.5,
				}),
				text.New("• "+i18n.T(lang, "pdf_grad_referral_students"), props.Text{
					Top:  12,
					Left: 6,
					Size: 7.5,
				}),
				text.New("• "+i18n.T(lang, "pdf_grad_referral_new_students"), props.Text{
					Top:  16,
					Left: 6,
					Size: 7.5,
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
