// File: internal/usecase/session_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"github.com/azharf99/algo-feedback/pkg/whatsapp"
)

type sessionUsecase struct {
	repo      domain.SessionRepository
	waService whatsapp.WhatsappService
	userRepo  domain.UserRepository
}

func NewSessionUsecase(repo domain.SessionRepository, waService whatsapp.WhatsappService, userRepo domain.UserRepository) domain.SessionUsecase {
	return &sessionUsecase{
		repo:      repo,
		waService: waService,
		userRepo:  userRepo,
	}
}

func (u *sessionUsecase) Create(ctx context.Context, session *domain.Session) error {
	return u.repo.Create(ctx, session)
}

func (u *sessionUsecase) GetByID(ctx context.Context, id uint) (*domain.Session, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *sessionUsecase) GetByGroup(ctx context.Context, groupID uint) ([]domain.Session, error) {
	return u.repo.GetByGroup(ctx, groupID)
}

func (u *sessionUsecase) GetByLesson(ctx context.Context, lessonID uint) ([]domain.Session, error) {
	return u.repo.GetByLesson(ctx, lessonID)
}

func (u *sessionUsecase) GetAll(ctx context.Context) ([]domain.Session, error) {
	return u.repo.GetAll(ctx)
}

func (u *sessionUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.Session], error) {
	params = pagination.Normalize(params)
	sessions, total, err := u.repo.GetPaginated(ctx, params)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return &domain.PaginatedResult[domain.Session]{
		Data:       sessions,
		Total:      total,
		TotalPages: totalPages,
		Page:       params.Page,
		Limit:      params.Limit,
	}, nil
}

func (u *sessionUsecase) Update(ctx context.Context, id uint, req *domain.Session) error {
	existing, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New("sesi tidak ditemukan")
	}

	if req.GroupID != 0 {
		existing.GroupID = req.GroupID
	}
	if req.LessonID != 0 {
		existing.LessonID = req.LessonID
	}
	if !req.DateStart.Time.IsZero() {
		existing.DateStart = req.DateStart
	}
	if !req.TimeStart.Time.IsZero() {
		existing.TimeStart = req.TimeStart
	}
	if req.AfterSessionFeedback != nil {
		existing.AfterSessionFeedback = req.AfterSessionFeedback
	}

	// Untuk boolean, kita asumsikan jika dikirim dalam JSON akan ter-bind.
	// Namun Gin ShouldBindJSON akan selalu set false jika tidak ada.
	// Untuk keamanan, kita hanya update jika ada perubahan nilai dari existing.
	if req.IsDone && !existing.IsDone {
		// It will trigger in the end after update to ensure DB is saved, wait, if we trigger it needs to save ScheduledMessageID.
		// Actually, let's just trigger it before, but we must update the DB. triggerAfterSessionFeedback will save it.
		// So we update existing.IsDone first.
	}
	wasDone := existing.IsDone
	existing.IsDone = req.IsDone

	err = u.repo.Update(ctx, existing)
	if err != nil {
		return err
	}

	if req.IsDone && !wasDone {
		u.TriggerAfterSessionFeedback(ctx, existing)
	}
	return nil
}

func (u *sessionUsecase) Delete(ctx context.Context, id uint) error {
	return u.repo.Delete(ctx, id)
}

func (u *sessionUsecase) UpdateAttendance(ctx context.Context, sessionID uint, studentIDs []uint) error {
	_, err := u.repo.GetByID(ctx, sessionID)
	if err != nil {
		return errors.New("sesi tidak ditemukan")
	}

	// Menyiapkan struct Session dengan IsDone otomatis True saat absen dikirim
	session := &domain.Session{
		ID:     sessionID,
		IsDone: true,
	}

	// Lempar ke repository untuk melakukan update dasar dan mereplace relasi Many-to-Many
	err = u.repo.UpsertAttendance(ctx, session, studentIDs)
	if err != nil {
		return err
	}

	updatedSession, err := u.repo.GetByID(ctx, sessionID)
	if err == nil {
		u.TriggerAfterSessionFeedback(ctx, updatedSession)
	}

	return nil
}

func (u *sessionUsecase) generateFeedbackMessage(ctx context.Context, session *domain.Session) string {
	// Dapatkan nama user dari context
	userName := "Tutor"
	if userID, ok := ctx.Value("user_id").(float64); ok { // JWT decode numeric as float64
		if user, err := u.userRepo.GetByID(ctx, uint(userID)); err == nil {
			userName = user.Name
		}
	} else if userID, ok := ctx.Value("user_id").(uint); ok {
		if user, err := u.userRepo.GetByID(ctx, userID); err == nil {
			userName = user.Name
		}
	}

	// Format Tanggal dan Waktu
	sessionDate := session.DateStart.Time
	dateStr := formatIndonesianDate(sessionDate)
	timeStr := session.TimeStart.Time.Format("15.04")

	// Surnames
	var surnames []string
	for _, student := range session.StudentsAttended {
		surnames = append(surnames, student.Surname)
	}
	surnamesStr := strings.Join(surnames, ", ")
	if len(surnames) == 0 {
		surnamesStr = "Siswa"
	}

	// Lesson dan Course
	lessonName := "-"
	if session.Lesson != nil {
		lessonName = session.Lesson.Title
	}
	courseName := "-"
	if session.Group != nil && session.Group.Course != nil {
		courseName = session.Group.Course.Title
	}

	// Competency
	var competencies []string
	if session.Lesson != nil && session.Lesson.Competency != "" {
		comps := strings.Split(session.Lesson.Competency, ";")
		for _, c := range comps {
			trimmed := strings.TrimSpace(c)
			if trimmed != "" {
				competencies = append(competencies, "• "+trimmed+";")
			}
		}
	}
	competenciesStr := strings.Join(competencies, "\n")

	template := `Halo, Parents!

Hari ini %s pukul %s %s telah mengikuti pelajaran %s di kursus %s. Mereka telah belajar:
%s

Untuk tetap belajar sambil berlatih, Bapak/Ibu bisa mengajak anak-anak membuka platform daring Algonova Indonesia dan menyelesaikan tugas-tugas mereka. Jika ada yang ingin dikonsultasikan, Ayah/Bunda bisa hubungi saya kapan saja.

Terima Kasih dan Sampai jumpa!
%s – Algonova Indonesia`

	return fmt.Sprintf(template, dateStr, timeStr, surnamesStr, lessonName, courseName, competenciesStr, userName)
}

func formatIndonesianDate(t time.Time) string {
	days := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

	dayName := days[t.Weekday()]
	monthName := months[t.Month()]
	return fmt.Sprintf("%s, %d %s %d", dayName, t.Day(), monthName, t.Year())
}

func (u *sessionUsecase) TriggerAfterSessionFeedback(ctx context.Context, session *domain.Session) {
	sessionDate := session.DateStart.Time
	sessionTime := session.TimeStart.Time

	runAtTime := time.Date(
		sessionDate.Year(), sessionDate.Month(), sessionDate.Day(),
		sessionTime.Hour(), sessionTime.Minute(), sessionTime.Second(),
		0, sessionDate.Location(),
	).Add(120 * time.Minute)

	if !runAtTime.After(time.Now()) {
		return
	}

	if session.Group == nil || session.Group.GroupPhone == nil || *session.Group.GroupPhone == "" {
		return
	}

	groupPhone := *session.Group.GroupPhone
	if len(groupPhone) > 14 {
		groupPhone += "@g.us"
	} else {
		groupPhone += "@s.whatsapp.net"
	}

	// Dapatkan credentials WhatsApp dari User
	var apiKey, deviceID string
	if user, err := u.userRepo.GetByID(ctx, session.UserID); err == nil {
		apiKey = user.WhatsappAPIKey
		deviceID = user.WhatsappDeviceID
	}

	// Generate Pesan
	msg := u.generateFeedbackMessage(ctx, session)

	// Simpan pesan ke AfterSessionFeedback agar terlihat di DB
	session.AfterSessionFeedback = &msg

	if session.ScheduledMessageID != nil {
		// Update existing schedule
		err := u.waService.UpdateSchedule(
			apiKey,
			deviceID,
			int(*session.ScheduledMessageID),
			groupPhone,
			msg,
			runAtTime.Format("2006-01-02 15:04:05"),
		)
		if err != nil {
			log.Printf("Gagal mengupdate jadwal WhatsApp after_session_feedback: %v", err)
		}
	} else {
		// Create new schedule
		id, err := u.waService.ScheduleMessage(
			apiKey,
			deviceID,
			groupPhone,
			msg,
			runAtTime.Format("2006-01-02 15:04:05"),
		)
		if err != nil {
			log.Printf("Gagal mendaftarkan jadwal WhatsApp after_session_feedback: %v", err)
		} else {
			uid := uint(id)
			session.ScheduledMessageID = &uid
		}
	}

	// Update DB (menyimpan ScheduledMessageID dan AfterSessionFeedback)
	_ = u.repo.Update(ctx, session)
}
