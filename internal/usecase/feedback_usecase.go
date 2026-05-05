// File: internal/usecase/feedback_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/curriculum"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"github.com/azharf99/algo-feedback/pkg/pdfgen"
	"github.com/azharf99/algo-feedback/pkg/taskqueue"
	"github.com/azharf99/algo-feedback/pkg/whatsapp"
)

// Helper function untuk dereference string pointer dengan fallback
func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type feedbackUsecase struct {
	feedRepo    domain.FeedbackRepository
	groupRepo   domain.GroupRepository   // Baru: Menggantikan LessonRepo
	sessionRepo domain.SessionRepository // Baru: Menggantikan LessonRepo
	pdfGen      pdfgen.PDFGenerator
	waService   whatsapp.WhatsappService
	taskPool    taskqueue.WorkerPool // Tambahkan worker pool
}

func NewFeedbackUsecase(
	fr domain.FeedbackRepository,
	gr domain.GroupRepository,
	sr domain.SessionRepository,
	pdf pdfgen.PDFGenerator,
	wa whatsapp.WhatsappService,
	pool taskqueue.WorkerPool, // Tambahkan parameter pool
) domain.FeedbackUsecase {
	return &feedbackUsecase{
		feedRepo:    fr,
		groupRepo:   gr,
		sessionRepo: sr,
		pdfGen:      pdf,
		waService:   wa,
		taskPool:    pool,
	}
}

// -------------------------------------------------------------------------
// 1. GENERATOR DATA FEEDBACK (SEEDER) - DENGAN AUTO ATTENDANCE SCORE!
// -------------------------------------------------------------------------
func (u *feedbackUsecase) GenerateFeedback(ctx context.Context, groupID *uint, all bool) (map[string]int, error) {
	var groups []domain.Group
	var err error

	// 1. Ambil data Group (yang sudah preload Students & Course)
	if all {
		groups, err = u.groupRepo.GetAll(ctx)
	} else if groupID != nil {
		group, errGroup := u.groupRepo.GetByID(ctx, *groupID)
		if errGroup == nil {
			groups = append(groups, *group)
		}
	}

	if err != nil {
		return nil, err
	}

	createdCount := 0
	updatedCount := 0

	for _, group := range groups {
		if group.Course == nil {
			continue // Skip jika grup tidak punya kurikulum
		}

		// 2. Ambil seluruh Sesi absensi untuk grup ini (sudah urut tanggal & preload StudentsAttended)
		sessions, err := u.sessionRepo.GetByGroup(ctx, group.ID)
		if err != nil || len(sessions) == 0 {
			continue
		}

		var monthSessions []domain.Session
		counter := 1

		for _, session := range sessions {
			if session.Lesson == nil {
				continue
			}

			// Reset counter jika modul kembali ke Modul 1 Lesson 1
			if session.Lesson.Level == "M1L1" {
				counter = 1
				monthSessions = nil
			}

			// Kumpulkan sesi untuk dihitung kehadirannya
			monthSessions = append(monthSessions, session)

			// Setiap 4 pertemuan = 1 Bulan / 1 Rapor
			if counter%4 == 0 {
				monthNumber := uint(counter / 4)

				// Pre-calculate attendance score for all students
				studentAttendanceCounts := make(map[uint]int)
				for _, ms := range monthSessions {
					for _, attStudent := range ms.StudentsAttended {
						studentAttendanceCounts[attStudent.ID]++
					}
				}

				for _, student := range group.Students {
					courseName := group.Course.Module
					topic := curriculum.GetTopic(courseName, int(monthNumber))
					result := curriculum.GetResult(courseName, int(monthNumber))
					comp := curriculum.GetCompetency(courseName, int(monthNumber))
					tutorFb := curriculum.GetTutorIntro(student.Fullname)
					level := curriculum.GetCourseLevel(courseName)

					// --- FITUR BARU: AUTO CALCULATE ATTENDANCE SCORE ---
					attendanceScore := studentAttendanceCounts[student.ID]
					// ---------------------------------------------------

					var sessionLessonDate *domain.DateOnly
					if !session.DateStart.Time.IsZero() {
						sessionLessonDate = &domain.DateOnly{Time: session.DateStart.Time}
					}

					var sessionLessonTime *domain.TimeOnly
					if !session.TimeStart.Time.IsZero() {
						sessionLessonTime = &domain.TimeOnly{Time: session.TimeStart.Time}
					}

					feedback := &domain.Feedback{
						StudentID:       &student.ID,
						Number:          monthNumber,
						Course:          &courseName,
						GroupName:       &group.Name,
						Topic:           &topic,
						Result:          &result,
						Competency:      &comp,
						TutorFeedback:   &tutorFb,
						LessonDate:      sessionLessonDate, // Tanggal rapor = tanggal sesi ke-4
						LessonTime:      sessionLessonTime,
						IsSent:          false,
						Level:           &level,
						ProjectLink:     group.RecordingsLink,
						AttendanceScore: domain.AttendanceScore(fmt.Sprintf("%d", attendanceScore)), // Nilai otomatis masuk!
					}

					// Eksekusi Update or Create
					isCreated, err := u.feedRepo.UpsertSeeder(ctx, feedback)
					if err != nil {
						continue
					}

					if isCreated {
						createdCount++
					} else {
						updatedCount++
					}
				}
				// Kosongkan koleksi sesi untuk bulan berikutnya
				monthSessions = nil
			}
			counter++
		}
	}

	return map[string]int{
		"created": createdCount,
		"updated": updatedCount,
	}, nil
}

// -------------------------------------------------------------------------
// 2. GENERATOR PDF (GOROUTINE BACKGROUND TASKS)
// -------------------------------------------------------------------------
func (u *feedbackUsecase) GeneratePDFAsync(ctx context.Context, studentID *uint, course *string, number *uint, all bool) ([]map[string]interface{}, error) {
	// Menggunakan GetFeedbacks (bukan GetUnsentFeedbacks) agar bisa me-regenerate PDF
	// yang sudah pernah dibuat/dikirim jika filter student/course/number diberikan.
	// Jika filter kosong (all=true), kita tetap ambil semua sesuai filter yang ada.
	feedbacks, err := u.feedRepo.GetFeedbacks(ctx, studentID, course, number, false)
	if err != nil {
		return nil, err
	}

	var response []map[string]interface{}

	for _, f := range feedbacks {
		// Safety check: Skip jika data student tidak ada (akibat data korup atau null)
		if f.Student == nil {
			continue
		}

		// Menggunakan GetFeedback dari curriculum untuk merangkai paragraf
		teacherFeedback := curriculum.GetFeedback(
			f.Student.Fullname,
			f.AttendanceScore,
			f.ActivityScore,
			f.TaskScore,
		)

		pdfData := pdfgen.PDFData{
			StudentName:         f.Student.Fullname,
			StudentMonthCourse:  f.Number,
			StudentClass:        strVal(f.Course),
			StudentLevel:        strVal(f.Level),
			StudentProjectLink:  strVal(f.ProjectLink),
			StudentReferralLink: "https://s.id/ar4C9",
			StudentModuleLink:   "https://s.id/ytNGs",
			ModuleTopic:         strVal(f.Topic),
			ModuleResult:        strVal(f.Result),
			SkillResult:         strVal(f.Competency),
			TeacherFeedback:     teacherFeedback,
		}

		fileName := fmt.Sprintf("Rapor %s Bulan ke-%d.pdf", f.Student.Fullname, f.Number)
		groupName := strVal(f.GroupName)
		if groupName == "" {
			groupName = "UnknownGroup"
		}
		outputPath := fmt.Sprintf("mediafiles/%d/%s/%s", f.UserID, groupName, fileName)

		// ⚡ GOROUTINE ACTION (Background Task) ⚡
		// Kita kirim ke Worker Pool agar tidak blocking request HTTP
		fID := f.ID // Capture ID untuk closure
		u.taskPool.Submit(taskqueue.TaskFunc(func(taskCtx context.Context) error {
			// 1. Generate PDF
			err := u.pdfGen.Generate(taskCtx, pdfData, outputPath)
			if err != nil {
				return fmt.Errorf("gagal generate PDF untuk student %s: %w", pdfData.StudentName, err)
			}

			// 2. Update URL PDF di Database
			// Kita ambil data terbaru dulu agar tidak menimpa data lain (pattern Fetch-then-Update)
			existing, err := u.feedRepo.GetByID(taskCtx, fID)
			if err != nil {
				return err
			}

			existing.URLPDF = &outputPath
			return u.feedRepo.Update(taskCtx, existing)
		}))

		response = append(response, map[string]interface{}{
			"student": f.Student.Fullname,
			"status":  "processing in background",
		})
	}

	return response, nil
}

// -------------------------------------------------------------------------
// 3. PENGIRIMAN WHATSAPP & UPDATE STATUS
// -------------------------------------------------------------------------
func (u *feedbackUsecase) SendFeedbackPDF(ctx context.Context, studentID *uint) ([]map[string]interface{}, error) {
	feedbacks, err := u.feedRepo.GetUnsentFeedbacks(ctx, studentID, nil, nil)
	if err != nil {
		return nil, err
	}

	var responseList []map[string]interface{}

	for _, f := range feedbacks {
		// Safety check: Skip jika data student tidak ada
		if f.Student == nil {
			continue
		}

		fileName := fmt.Sprintf("Rapor %s Bulan ke-%d.pdf", f.Student.Fullname, f.Number)
		groupName := strVal(f.GroupName)
		if groupName == "" {
			groupName = "UnknownGroup"
		}
		filePath := fmt.Sprintf("mediafiles/%d/%s/%s", f.UserID, groupName, fileName)

		// Persiapkan data kirim
		to := strVal(f.Student.ParentContact)
		if to == "" {
			// Fallback jika tidak ada nomor HP
			continue
		}
		// Pastikan format nomor WhatsApp (misal tambahkan @s.whatsapp.net jika belum ada)
		if !strings.Contains(to, "@") {
			to = to + "@s.whatsapp.net"
		}

		var parentName string
		if strVal(f.Student.ParentName) == "" {
			parentName = "{nama}" // {nama} ini nanti akan otomatis diganti dengan nama kontak yang terdaftar di Backend Whatsapp
		} else {
			parentName = *f.Student.ParentName
		}

		caption := fmt.Sprintf("Halo %s. Semoga %s sehat selalu, berikut adalah laporan perkembangan belajar Ananda %s untuk %s bulan ke-%d.",
			parentName, parentName, f.Student.Fullname, strVal(f.Course), f.Number)

		// Tentukan waktu kirim (misal 5 menit dari sekarang)
		runAt := time.Date(f.LessonDate.Year(),
			f.LessonDate.Month(),
			f.LessonDate.Day(),
			f.LessonTime.Hour(),
			f.LessonTime.Minute(),
			0, 0,
			time.Local).Add(5 * time.Minute).Format("2006-01-02 15:04:05")

		// Panggil Gateway baru: ScheduleMedia
		scheduleID, err := u.waService.ScheduleMedia(to, caption, filePath, runAt)
		if err != nil {
			continue
		}

		// Update schedule_id di Database
		scheduleIDStr := fmt.Sprintf("%d", scheduleID)
		f.ScheduleID = &scheduleIDStr
		f.IsSent = true
		_ = u.feedRepo.Update(ctx, &f)

		responseList = append(responseList, map[string]interface{}{
			"student":     f.Student.Fullname,
			"schedule_id": scheduleID,
			"status":      "scheduled",
		})
	}

	return responseList, nil
}

// -------------------------------------------------------------------------
// 4. CRUD STANDAR
// -------------------------------------------------------------------------
func (u *feedbackUsecase) Create(ctx context.Context, feedback *domain.Feedback) error {
	return u.feedRepo.Create(ctx, feedback)
}
func (u *feedbackUsecase) GetByID(ctx context.Context, id uint) (*domain.Feedback, error) {
	return u.feedRepo.GetByID(ctx, id)
}
func (u *feedbackUsecase) GetAll(ctx context.Context) ([]domain.Feedback, error) {
	return u.feedRepo.GetAll(ctx)
}
func (u *feedbackUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.Feedback], error) {
	params = pagination.Normalize(params)
	feedbacks, totalRows, err := u.feedRepo.GetPaginated(ctx, params)
	if err != nil {
		return nil, err
	}
	totalPages := int(math.Ceil(float64(totalRows) / float64(params.Limit)))
	return &domain.PaginatedResult[domain.Feedback]{
		Data: feedbacks, Total: totalRows, TotalPages: totalPages, Page: params.Page, Limit: params.Limit,
	}, nil
}
func (u *feedbackUsecase) Update(ctx context.Context, id uint, req *domain.Feedback) error {
	// 1. Ambil data feedback yang sudah ada
	existing, err := u.feedRepo.GetByID(ctx, id)
	if err != nil {
		return errors.New("feedback tidak ditemukan")
	}

	// 2. Update hanya field yang diizinkan untuk diubah manual
	if req.AttendanceScore != "" {
		existing.AttendanceScore = req.AttendanceScore
	}
	if req.ActivityScore != "" {
		existing.ActivityScore = req.ActivityScore
	}
	if req.TaskScore != "" {
		existing.TaskScore = req.TaskScore
	}
	if req.TutorFeedback != nil {
		existing.TutorFeedback = req.TutorFeedback
	}
	if req.Result != nil {
		existing.Result = req.Result
	}
	if req.ProjectLink != nil {
		existing.ProjectLink = req.ProjectLink
	}

	if req.LessonDate != nil {
		existing.LessonDate = req.LessonDate
	}
	if req.LessonTime != nil {
		existing.LessonTime = req.LessonTime
	}

	// 3. Sinkronisasi dengan WhatsApp Gateway jika ada schedule_id
	if existing.ScheduleID != nil && *existing.ScheduleID != "" {
		scheduleIDInt, _ := strconv.Atoi(*existing.ScheduleID)
		if scheduleIDInt > 0 && existing.Student != nil {
			to := strVal(existing.Student.ParentContact)
			if to != "" {
				if !strings.Contains(to, "@") {
					to = to + "@s.whatsapp.net"
				}

				// Format waktu baru (LessonDate + LessonTime)
				// Kita ambil jam dari LessonTime dan tanggal dari LessonDate
				var newRunAt string
				if existing.LessonDate != nil && existing.LessonTime != nil {
					newRunAt = time.Date(
						existing.LessonDate.Year(), existing.LessonDate.Month(), existing.LessonDate.Day(),
						existing.LessonTime.Hour(), existing.LessonTime.Minute(), existing.LessonTime.Second(),
						0, time.Local,
					).Format("2006-01-02 15:04:05")
				} else {
					// Fallback jika salah satu null
					newRunAt = time.Now().Add(5 * time.Minute).Format("2006-01-02 15:04:05")
				}

				caption := fmt.Sprintf("Halo %s. Semoga %s sehat selalu, berikut adalah laporan perkembangan belajar Ananda %s untuk %s bulan ke-%d.",
					*existing.Student.ParentName, *existing.Student.ParentName, existing.Student.Fullname, strVal(existing.Course), existing.Number)

				_ = u.waService.UpdateSchedule(scheduleIDInt, to, caption, newRunAt)
				existing.IsSent = true
			}
		}
	}

	return u.feedRepo.Update(ctx, existing)
}
func (u *feedbackUsecase) Delete(ctx context.Context, id uint) error {
	// 1. Ambil data feedback untuk cek URL PDF
	existing, err := u.feedRepo.GetByID(ctx, id)
	if err == nil && existing.URLPDF != nil && *existing.URLPDF != "" {
		// 2. Hapus file fisik jika ada
		_ = os.Remove(*existing.URLPDF)
	}

	return u.feedRepo.Delete(ctx, id)
}
