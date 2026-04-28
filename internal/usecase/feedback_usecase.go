// File: internal/usecase/feedback_usecase.go
package usecase

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/curriculum"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"github.com/azharf99/algo-feedback/pkg/pdfgen"
	"github.com/azharf99/algo-feedback/pkg/whatsapp"
)

type feedbackUsecase struct {
	feedRepo    domain.FeedbackRepository
	groupRepo   domain.GroupRepository   // Baru: Menggantikan LessonRepo
	sessionRepo domain.SessionRepository // Baru: Menggantikan LessonRepo
	pdfGen      pdfgen.PDFGenerator
	waService   whatsapp.WhatsappService
}

func NewFeedbackUsecase(
	fr domain.FeedbackRepository,
	gr domain.GroupRepository,
	sr domain.SessionRepository,
	pdf pdfgen.PDFGenerator,
	wa whatsapp.WhatsappService,
) domain.FeedbackUsecase {
	return &feedbackUsecase{
		feedRepo:    fr,
		groupRepo:   gr,
		sessionRepo: sr,
		pdfGen:      pdf,
		waService:   wa,
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

				for _, student := range group.Students {
					courseName := group.Course.Module
					topic := curriculum.GetTopic(courseName, int(monthNumber))
					result := curriculum.GetResult(courseName, int(monthNumber))
					comp := curriculum.GetCompetency(courseName, int(monthNumber))
					tutorFb := curriculum.GetTutorIntro(student.Fullname)
					level := curriculum.GetCourseLevel(courseName)

					// --- FITUR BARU: AUTO CALCULATE ATTENDANCE SCORE ---
					attendanceScore := 0
					for _, ms := range monthSessions {
						for _, attStudent := range ms.StudentsAttended {
							if attStudent.ID == student.ID {
								attendanceScore++
								break
							}
						}
					}
					// ---------------------------------------------------

					feedback := &domain.Feedback{
						StudentID:       &student.ID,
						Number:          monthNumber,
						Course:          &courseName,
						GroupName:       &group.Name,
						Topic:           &topic,
						Result:          &result,
						Competency:      &comp,
						TutorFeedback:   &tutorFb,
						LessonDate:      session.DateStart, // Tanggal rapor = tanggal sesi ke-4
						LessonTime:      session.TimeStart,
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
	feedbacks, err := u.feedRepo.GetUnsentFeedbacks(ctx, studentID, course, number)
	if err != nil {
		return nil, err
	}

	var response []map[string]interface{}

	for _, f := range feedbacks {
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
			StudentClass:        *f.Course,
			StudentLevel:        *f.Level,
			StudentProjectLink:  *f.ProjectLink,
			StudentReferralLink: "https://algonova.id/invite?utm_source=refferal&utm_medium=employee&utm_campaign=social_network&utm_content=hidin466",
			StudentModuleLink:   "https://drive.google.com/drive/u/0/folders/1lErW_RKjHOkAgqCr9yymELg3yUZzvBEb",
			ModuleTopic:         *f.Topic,
			ModuleResult:        *f.Result,
			SkillResult:         *f.Competency,
			TeacherFeedback:     teacherFeedback,
		}

		fileName := fmt.Sprintf("Rapor %s Bulan ke-%d.pdf", f.Student.Fullname, f.Number)
		outputPath := fmt.Sprintf("mediafiles/%s/%s", *f.GroupName, fileName)

		// ⚡ GOROUTINE ACTION ⚡
		u.pdfGen.Generate(ctx, pdfData, outputPath)

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
func (u *feedbackUsecase) SendFeedbackPDF(ctx context.Context, feedbackID *uint) ([]map[string]interface{}, error) {
	feedbacks, err := u.feedRepo.GetUnsentFeedbacks(ctx, feedbackID, nil, nil)
	if err != nil {
		return nil, err
	}

	var responseList []map[string]interface{}

	for _, f := range feedbacks {
		fileName := fmt.Sprintf("Rapor %s Bulan ke-%d.pdf", f.Student.Fullname, f.Number)
		filePath := fmt.Sprintf("mediafiles/%s/%s", *f.GroupName, fileName)

		// Upload Document ke Wablas
		uploadRes, err := u.waService.UploadDocument(*f.GroupName, f.Student.Fullname, *f.Course, f.Number, filePath)
		if err != nil || uploadRes == nil {
			continue
		}

		dataMap := uploadRes["data"].(map[string]interface{})
		msgMap := dataMap["messages"].(map[string]interface{})
		pdfURL := msgMap["url"].(string)

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

		u.waService.CreateSchedule(scheduleData)
		// scheduleRes, _ := u.waService.CreateSchedule(scheduleData)

		f.IsSent = true
		f.URLPDF = &pdfURL
		// Simulasi ID Jadwal (Di production ambil dari scheduleRes)
		scheduleID := "SCH-WABLAS-123"
		f.ScheduleID = &scheduleID

		u.feedRepo.Update(ctx, &f)

		responseList = append(responseList, scheduleData[0])
	}

	return responseList, nil
}

func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
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
	req.ID = id
	return u.feedRepo.Update(ctx, req)
}
func (u *feedbackUsecase) Delete(ctx context.Context, id uint) error {
	return u.feedRepo.Delete(ctx, id)
}
