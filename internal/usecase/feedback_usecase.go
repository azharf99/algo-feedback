// File: internal/usecase/feedback_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/curriculum"
	"github.com/azharf99/algo-feedback/pkg/i18n"
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

// sanitizeFilename membersihkan nama file dari karakter berbahaya
func sanitizeFilename(s string) string {
	// Ganti karakter non-alphanumeric dengan underscore
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	return reg.ReplaceAllString(s, "_")
}

type feedbackUsecase struct {
	feedRepo    domain.FeedbackRepository
	groupRepo   domain.GroupRepository   // Baru: Menggantikan LessonRepo
	sessionRepo domain.SessionRepository // Baru: Menggantikan LessonRepo
	studentRepo domain.StudentRepository
	pdfGen      pdfgen.PDFGenerator
	gradPdfGen  pdfgen.GraduationPDFGenerator
	waService   whatsapp.WhatsappService
	userRepo    domain.UserRepository // Tambahkan user repo
	taskPool    taskqueue.WorkerPool  // Tambahkan worker pool
}

func NewFeedbackUsecase(
	fr domain.FeedbackRepository,
	gr domain.GroupRepository,
	sr domain.SessionRepository,
	str domain.StudentRepository,
	pdf pdfgen.PDFGenerator,
	gradPdf pdfgen.GraduationPDFGenerator,
	wa whatsapp.WhatsappService,
	ur domain.UserRepository, // Tambahkan parameter user repo
	pool taskqueue.WorkerPool, // Tambahkan parameter pool
) domain.FeedbackUsecase {
	return &feedbackUsecase{
		feedRepo:    fr,
		groupRepo:   gr,
		sessionRepo: sr,
		studentRepo: str,
		pdfGen:      pdf,
		gradPdfGen:  gradPdf,
		waService:   wa,
		userRepo:    ur,
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

		lang := group.Language
		if lang == "" {
			lang = "Indonesia"
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
					courseName := session.Lesson.Module
					topic := curriculum.GetTopic(courseName, int(monthNumber), lang)
					result := curriculum.GetResult(courseName, int(monthNumber), lang)
					comp := curriculum.GetCompetency(courseName, int(monthNumber), lang)
					tutorFb := curriculum.GetTutorIntro(lang, student.Fullname)
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
						Language:        lang,
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

	return u.processPDFTasks(ctx, feedbacks), nil
}

func (u *feedbackUsecase) GeneratePendingPDFs(ctx context.Context) ([]map[string]interface{}, error) {
	feedbacks, err := u.feedRepo.GetPendingPDFFeedbacks(ctx)
	if err != nil {
		return nil, err
	}

	return u.processPDFTasks(ctx, feedbacks), nil
}

func (u *feedbackUsecase) processPDFTasks(ctx context.Context, feedbacks []domain.Feedback) []map[string]interface{} {
	var response []map[string]interface{}

	for _, f := range feedbacks {
		// Safety check: Skip jika data student tidak ada (akibat data korup atau null)
		if f.Student == nil {
			continue
		}

		lang := f.Language
		if lang == "" {
			lang = "Indonesia"
		}

		// Menggunakan GetFeedback dari curriculum untuk merangkai paragraf
		teacherFeedback := curriculum.GetFeedback(
			lang,
			f.Student.Fullname,
			f.AttendanceScore,
			f.ActivityScore,
			f.TaskScore,
		)

		pdfData := pdfgen.PDFData{
			Lang:                lang,
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

		courseName := sanitizeFilename(strVal(f.Course))
		if courseName == "" {
			courseName = "UnknownCourse"
		}

		fileName := fmt.Sprintf("Rapor %s - %s Bulan ke-%d.pdf", sanitizeFilename(f.Student.Fullname), courseName, f.Number)
		groupName := sanitizeFilename(strVal(f.GroupName))
		if groupName == "" {
			groupName = "UnknownGroup"
		}
		outputPath := filepath.Join("mediafiles", fmt.Sprintf("%d", f.UserID), groupName, courseName, fileName)

		// ⚡ GOROUTINE ACTION (Background Task) ⚡
		// Kita kirim ke Worker Pool agar tidak blocking request HTTP
		fID := f.ID         // Capture ID untuk closure
		fUserID := f.UserID // Capture UserID pemilik data
		isAdmin := ctxutil.IsAdmin(ctx)

		u.taskPool.Submit(taskqueue.TaskFunc(func(taskCtx context.Context) error {
			// Rekonstruksi context agar layer repository bisa melewati filter scopeByUser
			bgCtx := ctxutil.WithUserID(context.Background(), fUserID)
			if isAdmin {
				bgCtx = ctxutil.WithRole(bgCtx, "Admin")
			}

			// 1. Generate PDF
			err := u.pdfGen.Generate(bgCtx, pdfData, outputPath)
			if err != nil {
				return fmt.Errorf("%s %s: %w", i18n.T(lang, "error_pdf_gen"), pdfData.StudentName, err)
			}

			// 2. Update URL PDF di Database menggunakan sparse struct
			updateFeedback := &domain.Feedback{
				ID:     fID,
				URLPDF: &outputPath,
			}
			return u.feedRepo.Update(bgCtx, updateFeedback)
		}))

		response = append(response, map[string]interface{}{
			"student": f.Student.Fullname,
			"status":  "processing in background",
		})
	}

	return response
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

		lang := f.Language
		if lang == "" {
			lang = "Indonesia"
		}

		courseName := sanitizeFilename(strVal(f.Course))
		if courseName == "" {
			courseName = "UnknownCourse"
		}

		fileName := fmt.Sprintf("Rapor %s - %s Bulan ke-%d.pdf", sanitizeFilename(f.Student.Fullname), courseName, f.Number)
		groupName := sanitizeFilename(strVal(f.GroupName))
		if groupName == "" {
			groupName = "UnknownGroup"
		}
		filePath := filepath.Join("mediafiles", fmt.Sprintf("%d", f.UserID), groupName, courseName, fileName)

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

		var caption string
		switch lang {
		case "Russian":
			caption = fmt.Sprintf("Здравствуйте, %s. Надеемся, у вас все хорошо. Вот отчет о прогрессе обучения %s по курсу %s, месяц %d.",
				parentName, f.Student.Fullname, strVal(f.Course), f.Number)
		case "English":
			caption = fmt.Sprintf("Hello %s. We hope you are doing well. Here is the progress report for %s in the %s course, month %d.",
				parentName, f.Student.Fullname, strVal(f.Course), f.Number)
		default:
			caption = fmt.Sprintf("Halo %s. Semoga %s sehat selalu, berikut adalah laporan perkembangan belajar Ananda %s untuk %s bulan ke-%d.",
				parentName, parentName, f.Student.Fullname, strVal(f.Course), f.Number)
		}

		// Tentukan waktu kirim (misal 5 menit dari sekarang)
		runAt := time.Date(f.LessonDate.Year(),
			f.LessonDate.Month(),
			f.LessonDate.Day(),
			f.LessonTime.Hour(),
			f.LessonTime.Minute(),
			0, 0,
			time.Local).Add(5 * time.Minute).Format("2006-01-02 15:04:05")

		// Dapatkan credentials WhatsApp dari User
		var apiKey, deviceID string
		if user, err := u.userRepo.GetByID(ctx, f.UserID); err == nil {
			apiKey = user.WhatsappAPIKey
			deviceID = user.WhatsappDeviceID
		}

		// Panggil Gateway baru: ScheduleMedia
		scheduleID, err := u.waService.ScheduleMedia(apiKey, deviceID, to, caption, filePath, runAt, false)
		if err != nil {
			continue
		}

		// Update schedule_id di Database menggunakan sparse struct
		scheduleIDStr := fmt.Sprintf("%d", scheduleID)
		updateFeedback := &domain.Feedback{
			ID:         f.ID,
			ScheduleID: &scheduleIDStr,
			IsSent:     true,
		}
		_ = u.feedRepo.Update(ctx, updateFeedback)

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
func (u *feedbackUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.Feedback], *domain.FeedbackStats, error) {
	params = pagination.Normalize(params)
	feedbacks, totalRows, err := u.feedRepo.GetPaginated(ctx, params)
	if err != nil {
		return nil, nil, err
	}

	stats, err := u.feedRepo.GetStats(ctx)
	if err != nil {
		// Log error tapi jangan gagalkan request utama
		log.Printf("Gagal mengambil statistik feedback: %v", err)
	}

	totalPages := int(math.Ceil(float64(totalRows) / float64(params.Limit)))
	return &domain.PaginatedResult[domain.Feedback]{
		Data: feedbacks, Total: totalRows, TotalPages: totalPages, Page: params.Page, Limit: params.Limit,
	}, &stats, nil
}
func (u *feedbackUsecase) Update(ctx context.Context, id uint, req *domain.Feedback) error {
	lang := ctxutil.GetLanguage(ctx)
	// 1. Ambil data feedback yang sudah ada
	existing, err := u.feedRepo.GetByID(ctx, id)
	if err != nil {
		return errors.New(i18n.T(lang, "error_feedback_not_found"))
	}

	// 2. Update hanya field yang diizinkan untuk diubah manual
	updateFeedback := &domain.Feedback{ID: id}

	if req.AttendanceScore != "" {
		existing.AttendanceScore = req.AttendanceScore
		updateFeedback.AttendanceScore = req.AttendanceScore
	}
	if req.ActivityScore != "" {
		existing.ActivityScore = req.ActivityScore
		updateFeedback.ActivityScore = req.ActivityScore
	}
	if req.TaskScore != "" {
		existing.TaskScore = req.TaskScore
		updateFeedback.TaskScore = req.TaskScore
	}
	if req.TutorFeedback != nil {
		existing.TutorFeedback = req.TutorFeedback
		updateFeedback.TutorFeedback = req.TutorFeedback
	}
	if req.Result != nil {
		existing.Result = req.Result
		updateFeedback.Result = req.Result
	}
	if req.ProjectLink != nil {
		existing.ProjectLink = req.ProjectLink
		updateFeedback.ProjectLink = req.ProjectLink
	}

	if req.LessonDate != nil {
		existing.LessonDate = req.LessonDate
		updateFeedback.LessonDate = req.LessonDate
	}
	if req.LessonTime != nil {
		existing.LessonTime = req.LessonTime
		updateFeedback.LessonTime = req.LessonTime
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

				// Dapatkan credentials WhatsApp dari User
				var apiKey, deviceID string
				if user, err := u.userRepo.GetByID(ctx, existing.UserID); err == nil {
					apiKey = user.WhatsappAPIKey
					deviceID = user.WhatsappDeviceID
				}

				_ = u.waService.UpdateSchedule(apiKey, deviceID, scheduleIDInt, to, caption, newRunAt, false)
				existing.IsSent = true
				updateFeedback.IsSent = true
			}
		}
	}

	return u.feedRepo.Update(ctx, updateFeedback)
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

func (u *feedbackUsecase) BulkDelete(ctx context.Context, ids []uint) error {
	// Kita hapus file fisiknya satu-satu (optional, tapi bagus untuk kebersihan storage)
	for _, id := range ids {
		existing, err := u.feedRepo.GetByID(ctx, id)
		if err == nil && existing.URLPDF != nil && *existing.URLPDF != "" {
			_ = os.Remove(*existing.URLPDF)
		}
	}
	return u.feedRepo.BulkDelete(ctx, ids)
}

func (u *feedbackUsecase) GetWeeklySummary(ctx context.Context) (map[string][]domain.Feedback, error) {
	now := time.Now()

	// Get current Monday
	offset := int(now.Weekday()) - 1
	if offset < 0 {
		offset = 6 // Sunday
	}
	thisMonday := now.AddDate(0, 0, -offset)
	thisMonday = time.Date(thisMonday.Year(), thisMonday.Month(), thisMonday.Day(), 0, 0, 0, 0, thisMonday.Location())

	lastMonday := thisMonday.AddDate(0, 0, -7)
	nextMonday := thisMonday.AddDate(0, 0, 7)

	lastSunday := thisMonday.AddDate(0, 0, -1)
	thisSunday := thisMonday.AddDate(0, 0, 6)
	nextSunday := thisMonday.AddDate(0, 0, 13)

	lastWeekFeedbacks, err := u.feedRepo.GetByDateRange(ctx, lastMonday, lastSunday)
	if err != nil {
		return nil, err
	}

	thisWeekFeedbacks, err := u.feedRepo.GetByDateRange(ctx, thisMonday, thisSunday)
	if err != nil {
		return nil, err
	}

	nextWeekFeedbacks, err := u.feedRepo.GetByDateRange(ctx, nextMonday, nextSunday)
	if err != nil {
		return nil, err
	}

	return map[string][]domain.Feedback{
		"last_week": lastWeekFeedbacks,
		"this_week": thisWeekFeedbacks,
		"next_week": nextWeekFeedbacks,
	}, nil
}

func parseScore(s string, def int) int {
	if s == "" {
		return def
	}
	val := 0
	_, err := fmt.Sscanf(s, "%d", &val)
	if err != nil {
		return def
	}
	return val
}

func getGrade(pct float64) string {
	if pct >= 80 {
		return "A"
	} else if pct >= 75 {
		return "B+"
	} else if pct >= 70 {
		return "B"
	} else if pct >= 60 {
		return "C"
	} else {
		return "D"
	}
}

func (u *feedbackUsecase) GenerateGraduationPDFAsync(ctx context.Context, studentID *uint, course *string) ([]map[string]interface{}, error) {
	if studentID == nil || course == nil || *course == "" {
		return nil, errors.New("student_id and course (module name) are required")
	}

	// 1. Fetch student
	student, err := u.studentRepo.GetByID(ctx, *studentID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch all groups, find the one containing our student and matching course name
	groups, err := u.groupRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var matchedGroup *domain.Group
	for _, g := range groups {
		hasStudent := false
		for _, s := range g.Students {
			if s.ID == *studentID {
				hasStudent = true
				break
			}
		}
		if !hasStudent {
			continue
		}

		if g.Course != nil && (g.Course.Title == *course || g.Course.Module == *course) {
			matchedGroup = &g
			break
		}
	}

	// Fallback to check sessions module if not matched directly
	if matchedGroup == nil {
		for _, g := range groups {
			hasStudent := false
			for _, s := range g.Students {
				if s.ID == *studentID {
					hasStudent = true
					break
				}
			}
			if !hasStudent {
				continue
			}

			sessions, err := u.sessionRepo.GetByGroup(ctx, g.ID)
			if err == nil {
				for _, sess := range sessions {
					if sess.Lesson != nil && sess.Lesson.Module == *course {
						matchedGroup = &g
						break
					}
				}
			}
			if matchedGroup != nil {
				break
			}
		}
	}

	if matchedGroup == nil {
		return nil, errors.New("no group found for this student and course")
	}

	// 3. Fetch sessions for the group
	sessions, err := u.sessionRepo.GetByGroup(ctx, matchedGroup.ID)
	if err != nil {
		return nil, err
	}

	// 4. Filter sessions for the module/course
	var moduleSessions []domain.Session
	for _, sess := range sessions {
		if sess.Lesson != nil && sess.Lesson.Module == *course {
			moduleSessions = append(moduleSessions, sess)
		}
	}

	if len(moduleSessions) == 0 {
		return nil, errors.New("no sessions found for this course module")
	}

	// Sort sessions by lesson number ASC
	sort.Slice(moduleSessions, func(i, j int) bool {
		return moduleSessions[i].Lesson.Number < moduleSessions[j].Lesson.Number
	})

	// 5. Retrieve monthly feedbacks
	feedbacks, err := u.feedRepo.GetFeedbacks(ctx, studentID, course, nil, false)
	if err != nil {
		return nil, err
	}

	feedbackMap := make(map[uint]domain.Feedback)
	var tutorFeedbacks []string
	for _, f := range feedbacks {
		feedbackMap[f.Number] = f
		if f.TutorFeedback != nil && *f.TutorFeedback != "" {
			tFb := *f.TutorFeedback
			tutorFeedbacks = append(tutorFeedbacks, tFb)
		}
	}

	// Aggregate tutor feedback
	aggregatedTutorFeedback := ""
	if len(tutorFeedbacks) > 0 {
		aggregatedTutorFeedback = strings.Join(tutorFeedbacks, "\n\n")
	} else {
		// Fallback/Default intro
		aggregatedTutorFeedback = curriculum.GetTutorIntro(matchedGroup.Language, student.Fullname)
	}

	teacherName := "Algorithmics Teacher"
	if matchedGroup.UserID != 0 {
		if uVal, err := u.userRepo.GetByID(ctx, matchedGroup.UserID); err == nil && uVal != nil {
			teacherName = uVal.Name
		}
	}

	// 6. Build the lesson reports and construct range
	var lessonReports []pdfgen.LessonReport
	for _, sess := range moduleSessions {
		lesson := sess.Lesson
		if lesson == nil {
			continue
		}

		// Calculate Attendance score: Present = 4, Absent = 0
		isAttended := false
		for _, s := range sess.StudentsAttended {
			if s.ID == *studentID {
				isAttended = true
				break
			}
		}
		var attendanceVal int
		if isAttended {
			attendanceVal = 4
		} else {
			attendanceVal = 0
		}

		// Get monthly feedback number
		monthNum := uint(math.Ceil(float64(lesson.Number) / 4.0))

		activityVal := 3 // default
		taskVal := 2     // default
		if f, exists := feedbackMap[monthNum]; exists {
			activityVal = parseScore(string(f.ActivityScore), 3)
			taskVal = parseScore(string(f.TaskScore), 2)
		}

		totalScore := attendanceVal + activityVal + taskVal
		pct := (float64(totalScore) / 9.0) * 100.0

		lessonReports = append(lessonReports, pdfgen.LessonReport{
			LessonNumber: fmt.Sprintf("%d", lesson.Number),
			Topic:        strVal(lesson.Category),
			Score:        fmt.Sprintf("%.0f%%", pct),
			Grade:        getGrade(pct),
			SessionTitle: lesson.Title,
		})
	}

	// Lesson Range
	lessonRange := ""
	if len(moduleSessions) > 0 {
		firstLesson := moduleSessions[0].Lesson
		lastLesson := moduleSessions[len(moduleSessions)-1].Lesson
		if firstLesson != nil && lastLesson != nil {
			lessonRange = fmt.Sprintf("%s - %s", firstLesson.Level, lastLesson.Level)
		}
	}

	pdfData := pdfgen.GraduationPDFData{
		Lang:          matchedGroup.Language,
		StudentName:   student.Fullname,
		CourseName:    *course,
		LessonRange:   lessonRange,
		TeacherName:   teacherName,
		TutorFeedback: aggregatedTutorFeedback,
		ReferralLink:  "https://url.azharfa.cloud/pL3LHq",
		Lessons:       lessonReports,
	}

	courseName := sanitizeFilename(*course)
	fileName := fmt.Sprintf("Graduation %s - %s.pdf", sanitizeFilename(student.Fullname), courseName)
	groupName := sanitizeFilename(matchedGroup.Name)
	outputPath := filepath.Join("mediafiles", fmt.Sprintf("%d", student.UserID), groupName, courseName, fileName)

	fUserID := student.UserID
	isAdmin := ctxutil.IsAdmin(ctx)

	u.taskPool.Submit(taskqueue.TaskFunc(func(taskCtx context.Context) error {
		bgCtx := ctxutil.WithUserID(context.Background(), fUserID)
		if isAdmin {
			bgCtx = ctxutil.WithRole(bgCtx, "Admin")
		}

		// Ensure directories exist
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			log.Printf("Gagal membuat direktori untuk %s: %v", outputPath, err)
			return err
		}

		err := u.gradPdfGen.Generate(bgCtx, pdfData, outputPath)
		if err != nil {
			log.Printf("Gagal generate graduation PDF untuk %s: %v", student.Fullname, err)
			return err
		}
		return nil
	}))

	var response []map[string]interface{}
	response = append(response, map[string]interface{}{
		"student": student.Fullname,
		"course":  *course,
		"status":  "processing in background",
		"path":    outputPath,
	})

	return response, nil
}
