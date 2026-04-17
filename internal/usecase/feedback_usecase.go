// File: internal/usecase/feedback_usecase.go
package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/curriculum"
	"github.com/azharf99/algo-feedback/pkg/pdfgen"
	"github.com/azharf99/algo-feedback/pkg/whatsapp"
)

type feedbackUsecase struct {
	feedRepo   domain.FeedbackRepository
	lessonRepo domain.LessonRepository // Kita butuh ini untuk Seeder
	pdfGen     pdfgen.PDFGenerator
	waService  whatsapp.WhatsappService
}

func NewFeedbackUsecase(
	fr domain.FeedbackRepository,
	lr domain.LessonRepository,
	pdf pdfgen.PDFGenerator,
	wa whatsapp.WhatsappService,
) domain.FeedbackUsecase {
	return &feedbackUsecase{
		feedRepo:   fr,
		lessonRepo: lr,
		pdfGen:     pdf,
		waService:  wa,
	}
}

// -------------------------------------------------------------------------
// 1. GENERATOR DATA FEEDBACK (SEEDER)
// Menerjemahkan feedback_seeder.py
// -------------------------------------------------------------------------
func (u *feedbackUsecase) GenerateFeedback(ctx context.Context, groupID *uint, all bool) (map[string]int, error) {
	// Catatan: Di Repository, pastikan GetAll/GetByID melakukan Preload("Group.Students")
	lessons, err := u.lessonRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	createdCount := 0
	updatedCount := 0
	counter := 1

	for _, lesson := range lessons {
		// Filter manual (bisa dioptimasi di tingkat query DB nantinya)
		if !all && groupID != nil && lesson.GroupID != *groupID {
			continue
		}

		if lesson.Level == "M1L1" {
			counter = 1
		}

		if counter%4 == 0 {
			monthNumber := uint(counter / 4)
			for _, student := range lesson.Group.Students {

				// Memanggil kamus dari pkg/curriculum
				topic := curriculum.GetTopic(lesson.Module, int(monthNumber))
				result := curriculum.GetResult(lesson.Module, int(monthNumber))
				comp := curriculum.GetCompetency(lesson.Module, int(monthNumber))
				tutorFb := curriculum.GetTutorFeedback(student.Fullname)
				level := curriculum.GetCourseLevel(lesson.Module)

				feedback := &domain.Feedback{
					StudentID:     &student.ID,
					Number:        monthNumber,
					Course:        &lesson.Module,
					GroupName:     &lesson.Group.Name,
					Topic:         &topic,
					Result:        &result,
					Competency:    &comp,
					TutorFeedback: &tutorFb,
					LessonDate:    lesson.DateStart,
					LessonTime:    lesson.TimeStart,
					IsSent:        false,
					Level:         &level,
					ProjectLink:   lesson.Group.RecordingsLink,
				}

				// Eksekusi Update or Create
				isCreated, err := u.feedRepo.UpsertSeeder(ctx, feedback)
				if err != nil {
					continue // Lanjut ke siswa berikutnya jika error
				}

				if isCreated {
					createdCount++
				} else {
					updatedCount++
				}
			}
		}
		counter++
	}

	return map[string]int{
		"created": createdCount,
		"updated": updatedCount,
	}, nil
}

// -------------------------------------------------------------------------
// 2. GENERATOR PDF (GOROUTINE BACKGROUND TASKS)
// Menerjemahkan generator_pdf.py (Pengganti Celery)
// -------------------------------------------------------------------------
func (u *feedbackUsecase) GeneratePDFAsync(ctx context.Context, studentID *uint, course *string, number *uint, all bool) ([]map[string]interface{}, error) {
	feedbacks, err := u.feedRepo.GetUnsentFeedbacks(ctx, studentID, course, number)
	if err != nil {
		return nil, err
	}

	var response []map[string]interface{}

	for _, f := range feedbacks {
		// Menyiapkan data untuk template HTML
		pdfData := pdfgen.PDFData{
			StudentName:         f.Student.Fullname,
			StudentMonthCourse:  f.Number,
			StudentClass:        *f.Course,
			StudentLevel:        *f.Level,
			StudentProjectLink:  *f.ProjectLink,
			StudentReferralLink: "https://algonova.id/invite?utm_source=refferal...",
			StudentModuleLink:   "https://drive.google.com/...",
			ModuleTopic:         *f.Topic,
			ModuleResult:        *f.Result,
			SkillResult:         *f.Competency,
			// Kita asumsikan ada fungsi curriculum.GetScoredFeedback
			TeacherFeedback: *f.TutorFeedback,
		}

		// Menentukan path output
		fileName := fmt.Sprintf("Rapor %s Bulan ke-%d.pdf", f.Student.Fullname, f.Number)
		outputPath := fmt.Sprintf("mediafiles/%s/%s", *f.GroupName, fileName)

		// ⚡ GOROUTINE ACTION ⚡: Memanggil PDF Generator secara asinkron!
		u.pdfGen.GeneratePDFAsync(ctx, pdfData, "index.html", outputPath)

		response = append(response, map[string]interface{}{
			"student": f.Student.Fullname,
			"status":  "processing in background", // Akan langsung return ke user!
		})
	}

	return response, nil
}

// -------------------------------------------------------------------------
// 3. PENGIRIMAN WHATSAPP & UPDATE STATUS
// Menerjemahkan views.py -> send_feedback_pdf
// -------------------------------------------------------------------------
func (u *feedbackUsecase) SendFeedbackPDF(ctx context.Context, feedbackID *uint) ([]map[string]interface{}, error) {
	// Ambil data yang belum terkirim (Bisa 1 ID atau semuanya)
	feedbacks, err := u.feedRepo.GetUnsentFeedbacks(ctx, nil, nil, nil) // Disederhanakan untuk contoh
	if err != nil {
		return nil, err
	}

	var responseList []map[string]interface{}

	for _, f := range feedbacks {
		// Path file PDF yang sudah digenerate sebelumnya
		fileName := fmt.Sprintf("Rapor %s Bulan ke-%d.pdf", f.Student.Fullname, f.Number)
		filePath := fmt.Sprintf("mediafiles/%s/%s", *f.GroupName, fileName)

		// 1. Upload ke Wablas
		uploadRes, err := u.waService.UploadDocument(*f.GroupName, f.Student.Fullname, *f.Course, f.Number, filePath)
		if err != nil || uploadRes == nil {
			continue // Skip jika gagal upload
		}

		// Ambil URL dari response Wablas
		// Asumsi struktur JSON Wablas: data -> messages -> url
		dataMap := uploadRes["data"].(map[string]interface{})
		msgMap := dataMap["messages"].(map[string]interface{})
		pdfURL := msgMap["url"].(string)

		// 2. Siapkan Jadwal (Schedule) - Tambah 2 jam + detik acak dari waktu lesson
		randSeconds := time.Duration(randomInt(1, 59)) * time.Second
		scheduledTime := f.LessonDate.Add(2 * time.Hour).Add(randSeconds)
		scheduleStr := scheduledTime.Format("2006-01-02 15:04:05")

		scheduleData := []map[string]interface{}{
			{
				"category":     "document",
				"phone":        *f.Student.ParentContact,
				"scheduled_at": scheduleStr,
				"url":          pdfURL,
				"text":         *f.TutorFeedback,
			},
		}

		// 3. Eksekusi Create Schedule
		u.waService.CreateSchedule(scheduleData)

		// 4. Update Database
		f.IsSent = true
		f.URLPDF = &pdfURL
		// Asumsikan kita dapat ID dari scheduleRes
		scheduleID := "SCH-WABLAS-123"
		f.ScheduleID = &scheduleID

		u.feedRepo.Update(ctx, &f)

		responseList = append(responseList, scheduleData[0])
	}

	return responseList, nil
}

// Fungsi bantuan untuk angka acak
func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

// Create menyimpan data feedback baru secara manual (bukan dari Seeder)
func (u *feedbackUsecase) Create(ctx context.Context, feedback *domain.Feedback) error {
	return u.feedRepo.Create(ctx, feedback)
}

// GetByID mengambil satu data feedback beserta relasi siswanya berdasarkan ID
func (u *feedbackUsecase) GetByID(ctx context.Context, id uint) (*domain.Feedback, error) {
	return u.feedRepo.GetByID(ctx, id)
}

// GetAll mengambil seluruh data feedback
func (u *feedbackUsecase) GetAll(ctx context.Context) ([]domain.Feedback, error) {
	return u.feedRepo.GetAll(ctx)
}

// Update memperbarui data feedback yang sudah ada (misalnya jika Tutor ingin mengedit teks)
func (u *feedbackUsecase) Update(ctx context.Context, id uint, req *domain.Feedback) error {
	// Pastikan ID yang diubah sesuai dengan parameter URL
	req.ID = id
	return u.feedRepo.Update(ctx, req)
}

// Delete menghapus data feedback dari database
func (u *feedbackUsecase) Delete(ctx context.Context, id uint) error {
	return u.feedRepo.Delete(ctx, id)
}
