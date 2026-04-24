// File: internal/usecase/feedback_usecase.go
package usecase

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/curriculum"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"github.com/azharf99/algo-feedback/pkg/pdfgen"
	"github.com/azharf99/algo-feedback/pkg/taskqueue"
	"github.com/azharf99/algo-feedback/pkg/whatsapp"
)

type feedbackUsecase struct {
	feedRepo   domain.FeedbackRepository
	lessonRepo domain.LessonRepository
	pdfGen     pdfgen.PDFGenerator
	waService  whatsapp.WhatsappService
	taskQueue  taskqueue.WorkerPool
}

func NewFeedbackUsecase(
	fr domain.FeedbackRepository,
	lr domain.LessonRepository,
	pdf pdfgen.PDFGenerator,
	wa whatsapp.WhatsappService,
	tq taskqueue.WorkerPool,
) domain.FeedbackUsecase {
	return &feedbackUsecase{
		feedRepo:   fr,
		lessonRepo: lr,
		pdfGen:     pdf,
		waService:  wa,
		taskQueue:  tq,
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
			StudentReferralLink: "https://algonova.id/invite?utm_source=refferal&utm_medium=employee&utm_campaign=social_network&utm_content=hidin466",
			StudentModuleLink:   "https://drive.google.com/drive/u/0/folders/1lErW_RKjHOkAgqCr9yymELg3yUZzvBEb",
			ModuleTopic:         *f.Topic,
			ModuleResult:        *f.Result,
			SkillResult:         *f.Competency,
				TeacherFeedback: curriculum.GetFeedback(
					f.Student.Fullname,
					f.AttendanceScore,
					f.ActivityScore,
					f.TaskScore,
				),
		}

		// Menentukan path output
		fileName := fmt.Sprintf("Rapor %s Bulan ke-%d.pdf", f.Student.Fullname, f.Number)
		outputPath := fmt.Sprintf("mediafiles/%s/%s", *f.GroupName, fileName)

		// ⚡ TASK QUEUE ACTION ⚡: Memasukkan ke antrian background!
		u.taskQueue.Submit(taskqueue.TaskFunc(func(ctx context.Context) error {
			return u.pdfGen.Generate(ctx, pdfData, outputPath)
		}))

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
	// Ambil data yang belum terkirim
	feedbacks, err := u.feedRepo.GetUnsentFeedbacks(ctx, feedbackID, nil, nil)
	if err != nil {
		return nil, err
	}

	var responseList []map[string]interface{}

	for _, f := range feedbacks {
		// Pastikan file PDF ada
		fileName := fmt.Sprintf("Rapor %s Bulan ke-%d.pdf", f.Student.Fullname, f.Number)
		filePath := fmt.Sprintf("mediafiles/%s/%s", *f.GroupName, fileName)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("[SEND-WA] File not found: %s", filePath)
			continue
		}

		// 1. Upload ke Wablas
		uploadRes, err := u.waService.UploadDocument(*f.GroupName, f.Student.Fullname, *f.Course, f.Number, filePath)
		if err != nil {
			log.Printf("[SEND-WA] Upload failed: %v", err)
			continue
		}

		// Ambil URL dan ID dari response Wablas
		// Struktur: {"data": {"messages": {"url": "...", "id": "..."}}}
		data, ok := uploadRes["data"].(map[string]interface{})
		if !ok {
			log.Printf("[SEND-WA] Invalid response format (no data)")
			continue
		}
		messages, ok := data["messages"].(map[string]interface{})
		if !ok {
			log.Printf("[SEND-WA] Invalid response format (no messages)")
			continue
		}

		pdfURL, _ := messages["url"].(string)
		wablasMsgID, _ := messages["id"].(string)

		if pdfURL == "" {
			log.Printf("[SEND-WA] PDF URL is empty in response")
			continue
		}

		// 2. Siapkan Jadwal (Schedule) - Tambah 2 jam + detik acak
		randSeconds := time.Duration(randomInt(1, 59)) * time.Second
		// Gunakan LessonDate dan LessonTime untuk menghitung waktu kirim
		// Di model, LessonTime adalah time.Time yang jam/menitnya berisi waktu mulai
		scheduledTime := f.LessonDate.Add(time.Duration(f.LessonTime.Hour())*time.Hour +
			time.Duration(f.LessonTime.Minute())*time.Minute +
			2*time.Hour + randSeconds)

		scheduleStr := scheduledTime.Format("2006-01-02 15:04:05")

		scheduleData := []map[string]interface{}{
			{
				"category":     "document",
				"phone":        *f.Student.ParentContact,
				"scheduled_at": scheduleStr,
				"url":          pdfURL,
				"text":         curriculum.GetTutorIntro(f.Student.Fullname), // Menggunakan Intro template
			},
		}

		// 3. Eksekusi Create Schedule
		_, err = u.waService.CreateSchedule(scheduleData)
		if err != nil {
			log.Printf("[SEND-WA] Create schedule failed: %v", err)
			continue
		}

		// 4. Update Database
		f.IsSent = true
		f.URLPDF = &pdfURL
		f.ScheduleID = &wablasMsgID

		if err := u.feedRepo.Update(ctx, &f); err != nil {
			log.Printf("[SEND-WA] Failed to update feedback record: %v", err)
		}

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

// GetPaginated mengambil data feedback dengan pagination
func (u *feedbackUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.Feedback], error) {
	params = pagination.Normalize(params)
	feedbacks, total, err := u.feedRepo.GetPaginated(ctx, params)
	if err != nil {
		return nil, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return &domain.PaginatedResult[domain.Feedback]{
		Data:       feedbacks,
		Page:       params.Page,
		Limit:      params.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
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
