// File: internal/usecase/session_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/csvutil"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"github.com/azharf99/algo-feedback/pkg/whatsapp"
)

type sessionUsecase struct {
	repo       domain.SessionRepository
	waService  whatsapp.WhatsappService
	userRepo   domain.UserRepository
	groupRepo  domain.GroupRepository
	lessonRepo domain.LessonRepository
}

func NewSessionUsecase(repo domain.SessionRepository, waService whatsapp.WhatsappService, userRepo domain.UserRepository, groupRepo domain.GroupRepository, lessonRepo domain.LessonRepository) domain.SessionUsecase {
	return &sessionUsecase{
		repo:       repo,
		waService:  waService,
		userRepo:   userRepo,
		groupRepo:  groupRepo,
		lessonRepo: lessonRepo,
	}
}

// checkGroupOwnership memastikan groupID yang dikirim client benar-benar milik user yang
// login. groupRepo.GetByID sudah menerapkan scopeByUser, jadi GroupID milik tenant lain
// akan gagal (not found) di sini alih-alih dipakai begitu saja lewat FK yang tidak divalidasi.
func (u *sessionUsecase) checkGroupOwnership(ctx context.Context, groupID uint) error {
	lang := ctxutil.GetLanguage(ctx)
	if _, err := u.groupRepo.GetByID(ctx, groupID); err != nil {
		return errors.New(i18n.T(lang, "error_group_not_found"))
	}
	return nil
}

// checkLessonOwnership sama seperti checkGroupOwnership, tapi untuk LessonID.
func (u *sessionUsecase) checkLessonOwnership(ctx context.Context, lessonID uint) error {
	lang := ctxutil.GetLanguage(ctx)
	if _, err := u.lessonRepo.GetByID(ctx, lessonID); err != nil {
		return errors.New(i18n.T(lang, "error_lesson_not_found"))
	}
	return nil
}

func (u *sessionUsecase) getLanguage(session *domain.Session) string {
	if session.Group != nil && session.Group.Language != "" {
		return session.Group.Language
	}
	return "Indonesia"
}

func (u *sessionUsecase) Create(ctx context.Context, session *domain.Session) error {
	if err := u.checkGroupOwnership(ctx, session.GroupID); err != nil {
		return err
	}
	if err := u.checkLessonOwnership(ctx, session.LessonID); err != nil {
		return err
	}
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
	lang := ctxutil.GetLanguage(ctx)
	existing, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New(i18n.T(lang, "error_session_not_found"))
	}

	lang = u.getLanguage(existing)

	if existing.Status == "Cancelled" && req.IsDone {
		return errors.New(i18n.T(lang, "error_cancelled_session_done"))
	}

	var daysShift int
	oldDate := existing.DateStart.Time

	if req.ShiftSubsequent && !req.DateStart.Time.IsZero() && !oldDate.IsZero() {
		daysShift = int(req.DateStart.Time.Sub(oldDate).Hours() / 24)
	}

	if req.GroupID != 0 && req.GroupID != existing.GroupID {
		if err := u.checkGroupOwnership(ctx, req.GroupID); err != nil {
			return err
		}
		existing.GroupID = req.GroupID
	}
	if req.LessonID != 0 && req.LessonID != existing.LessonID {
		if err := u.checkLessonOwnership(ctx, req.LessonID); err != nil {
			return err
		}
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

	if req.Status != existing.Status {
		existing.Status = req.Status
	}

	wasDone := existing.IsDone

	if req.Status == "Cancelled" {
		existing.IsDone = false
	} else {
		existing.IsDone = req.IsDone
	}

	err = u.repo.Update(ctx, existing)
	if err != nil {
		return err
	}

	// Shift subsequent sessions if requested and date changed
	if daysShift != 0 {
		sessions, err := u.repo.GetByGroup(ctx, existing.GroupID)
		if err == nil {
			for _, s := range sessions {
				if s.ID != existing.ID && s.DateStart.Time.After(oldDate) {
					s.DateStart.Time = s.DateStart.Time.AddDate(0, 0, daysShift)
					err := u.repo.Update(ctx, &s)
					if err != nil {
						log.Printf("Gagal mengupdate tanggal sesi %d saat shifting: %v", s.ID, err)
					}
				}
			}
		}
	}

	if req.IsDone && !wasDone {
		u.TriggerAfterSessionFeedback(ctx, existing)
	}
	return nil
}

func (u *sessionUsecase) Delete(ctx context.Context, id uint) error {
	return u.repo.Delete(ctx, id)
}

func (u *sessionUsecase) BulkDelete(ctx context.Context, ids []uint) error {
	return u.repo.BulkDelete(ctx, ids)
}

func (u *sessionUsecase) UpdateAttendance(ctx context.Context, sessionID uint, studentIDs []uint) error {
	lang := ctxutil.GetLanguage(ctx)
	existing, err := u.repo.GetByID(ctx, sessionID)
	if err != nil {
		return errors.New(i18n.T(lang, "error_session_not_found"))
	}

	lang = u.getLanguage(existing)

	if existing.Status == "Cancelled" {

		return errors.New(i18n.T(lang, "error_cancelled_session_att"))
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

func (u *sessionUsecase) generateFeedbackMessage(_ context.Context, session *domain.Session, userName string) string {
	// Language check
	language := u.getLanguage(session)

	// Format Tanggal dan Waktu
	sessionDate := session.DateStart.Time
	var dateStr string
	switch language {
	case "Russian":
		dateStr = formatRussianDate(sessionDate)
	case "English":
		dateStr = formatEnglishDate(sessionDate)
	default:
		dateStr = formatIndonesianDate(sessionDate)
	}
	timeStr := session.TimeStart.Time.Format("15.04")

	// Surnames
	var surnames []string
	for _, student := range session.StudentsAttended {
		surnames = append(surnames, strings.TrimSpace(student.Surname))
	}
	surnamesStr := strings.Join(surnames, ", ")
	if len(surnames) == 0 {
		switch language {
		case "Russian":
			surnamesStr = "Ученики"
		case "English":
			surnamesStr = "Students"
		default:
			surnamesStr = "Siswa"
		}
	}

	// Lesson dan Course
	lessonName := "-"
	courseName := "-"
	recording_link := "-"
	if session.Lesson != nil {
		lessonName = session.Lesson.Title
		courseName = session.Lesson.Module
	}

	if session.Group != nil {
		if session.Group.RecordingsLink != nil {
			recording_link = *session.Group.RecordingsLink
		}
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

	var template string
	switch language {
	case "Russian":
		template = `Здравствуйте!

Сегодня, %s в %s по МСК, %s посетил(а) урок %s по курсу %s. Они изучили:
%s

Чтобы продолжать обучение и практику, родители могут предложить детям зайти на онлайн-платформу Algonova и выполнить задания. Если у вас есть вопросы, вы можете связаться со мной в любое время.

Запись урока доступна по следующей ссылке:
%s

Спасибо и до встречи!
%s – Algonova International`
	case "English":
		template = `Hello, parents!

Today, %s at %s WIB, %s attended the %s lesson in the %s course. They learned:
%s

To keep learning while practicing, parents can encourage their children to access the Algonova Indonesia online platform and complete their assignments. If you have any questions or need consultation, feel free to contact me anytime.

The lesson recording can be accessed through the following link:
%s

Thank you and see you!
%s – Algonova Indonesia`
	default: // Indonesia
		template = `Halo, Ayah/Bunda!

Hari ini, %s pukul %s WIB %s telah mengikuti pelajaran %s di kursus %s. Mereka telah belajar:
%s

Untuk tetap belajar sambil berlatih, ayah/bunda bisa mengajak putra-putri membuka platform daring Algonova Indonesia dan menyelesaikan tugas-tugas mereka. Jika ada yang ingin dikonsultasikan, ayah/bunda bisa menghubungi saya kapan saja.

Rekaman pelajaran bisa diakses melalui tautan berikut: 
%s

Terima Kasih dan Sampai jumpa!
%s – Algonova Indonesia`
	}

	return fmt.Sprintf(template, dateStr, timeStr, surnamesStr, lessonName, courseName, competenciesStr, recording_link, userName)
}

func formatIndonesianDate(t time.Time) string {
	days := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

	dayName := days[t.Weekday()]
	monthName := months[t.Month()]
	return fmt.Sprintf("%s, %d %s %d", dayName, t.Day(), monthName, t.Year())
}

func formatEnglishDate(t time.Time) string {
	return t.Format("Monday, 2 January 2006")
}

func formatRussianDate(t time.Time) string {
	days := []string{"Воскресенье", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота"}
	months := []string{"", "Января", "Февраля", "Марта", "Апреля", "Мая", "Июня", "Июля", "Августа", "Сентября", "Октября", "Ноября", "Декабря"}

	dayName := days[t.Weekday()]
	monthName := months[t.Month()]
	return fmt.Sprintf("%s, %d %s %d", dayName, t.Day(), monthName, t.Year())
}

func (u *sessionUsecase) TriggerAfterSessionFeedback(ctx context.Context, session *domain.Session) {
	if session.Status == "Cancelled" {
		return
	}

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
	// ID grup WA pasti nomornya lebih banyak (> 14 karakter)
	isGroup := len(groupPhone) > 14

	// Dapatkan credentials WhatsApp dari User
	var apiKey, deviceID string
	userName := "Tutor"
	if user, err := u.userRepo.GetByID(ctx, session.UserID); err == nil {
		apiKey = user.WhatsappAPIKey
		deviceID = user.WhatsappDeviceID
		userName = user.Name
	} else {
		// Fallback ke context jika session.UserID gagal atau bernilai 0
		if userID, ok := ctx.Value("user_id").(float64); ok {
			if u, err := u.userRepo.GetByID(ctx, uint(userID)); err == nil {
				userName = u.Name
			}
		} else if userID, ok := ctx.Value("user_id").(uint); ok {
			if u, err := u.userRepo.GetByID(ctx, userID); err == nil {
				userName = u.Name
			}
		}
	}

	// Generate Pesan
	msg := u.generateFeedbackMessage(ctx, session, userName)

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
			isGroup,
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
			isGroup,
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

func (u *sessionUsecase) MarkDoneUpToDate(ctx context.Context, groupID uint, date time.Time) error {
	return u.repo.MarkDoneUpToDate(ctx, groupID, date)
}

func (u *sessionUsecase) AutoFillAttendance(ctx context.Context, groupID uint, date time.Time) error {
	return u.repo.AutoFillAttendance(ctx, groupID, date)
}

func (u *sessionUsecase) GetWeeklySummary(ctx context.Context) (map[string][]domain.Session, error) {
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

	lastWeekSessions, err := u.repo.GetByDateRange(ctx, lastMonday, lastSunday)
	if err != nil {
		return nil, err
	}

	thisWeekSessions, err := u.repo.GetByDateRange(ctx, thisMonday, thisSunday)
	if err != nil {
		return nil, err
	}

	nextWeekSessions, err := u.repo.GetByDateRange(ctx, nextMonday, nextSunday)
	if err != nil {
		return nil, err
	}

	return map[string][]domain.Session{
		"last_week": lastWeekSessions,
		"this_week": thisWeekSessions,
		"next_week": nextWeekSessions,
	}, nil
}

func (u *sessionUsecase) MarkCancelled(ctx context.Context, groupID uint, fromDate, beforeDate time.Time) error {
	sessions, err := u.repo.GetByGroup(ctx, groupID)
	if err != nil {
		return err
	}

	for _, s := range sessions {
		if (s.DateStart.Time.After(fromDate) || s.DateStart.Time.Equal(fromDate)) &&
			(s.DateStart.Time.Before(beforeDate) || s.DateStart.Time.Equal(beforeDate)) {
			if s.ScheduledMessageID != nil {
				var apiKey string
				if user, err := u.userRepo.GetByID(ctx, s.UserID); err == nil {
					apiKey = user.WhatsappAPIKey
				}
				err := u.waService.DeleteSchedule(apiKey, int(*s.ScheduledMessageID))
				if err != nil {
					log.Printf("Gagal menghapus jadwal WhatsApp %d: %v", *s.ScheduledMessageID, err)
				}
			}
		}
	}

	return u.repo.MarkCancelled(ctx, groupID, fromDate, beforeDate)
}

func (u *sessionUsecase) StartSessionBot(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		defer ticker.Stop()
		log.Println("[SESSION-BOT] Bot started, running every 1 hour")

		// Run once on startup
		if err := u.processSessionBot(ctx); err != nil {
			log.Printf("[SESSION-BOT] Error: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := u.processSessionBot(ctx); err != nil {
					log.Printf("[SESSION-BOT] Error: %v", err)
				}
			case <-ctx.Done():
				log.Println("[SESSION-BOT] Bot stopped")
				return
			}
		}
	}()
}

func (u *sessionUsecase) processSessionBot(ctx context.Context) error {
	log.Println("[SESSION-BOT] Checking sessions...")

	// Create admin context to bypass user scoping
	botCtx := ctxutil.WithRole(ctx, "Admin")
	botCtx = ctxutil.WithUserID(botCtx, 1) // Fallback admin user ID

	now := time.Now()
	sessions, err := u.repo.GetSessionsToAutoComplete(botCtx, now)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("Indonesia", "error_fetch_sessions"), err)
	}

	log.Printf("[SESSION-BOT] Found %d sessions to process", len(sessions))

	for _, s := range sessions {
		var studentIDs []uint
		if s.Group != nil {
			for _, student := range s.Group.Students {
				studentIDs = append(studentIDs, student.ID)
			}
		}

		log.Printf("[SESSION-BOT] Processing session %d", s.ID)

		// Re-use UpdateAttendance which marks done and triggers feedback
		err := u.UpdateAttendance(botCtx, s.ID, studentIDs)
		if err != nil {
			log.Printf("[SESSION-BOT] Failed to process session %d: %v", s.ID, err)
		} else {
			// Kirim notifikasi project jika pelajaran memiliki project
			u.sendProjectNotification(botCtx, &s)
		}
	}

	return nil
}

func (u *sessionUsecase) sendProjectNotification(ctx context.Context, session *domain.Session) {
	if session.Lesson == nil || !session.Lesson.IsProjectLesson {
		return
	}

	user, err := u.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		log.Printf("[SESSION-BOT] Failed to get user for session %d: %v", session.ID, err)
		return
	}

	if user.PhoneNumber == "" {
		log.Printf("[SESSION-BOT] Skipping project notification for session %d: user has no phone number", session.ID)
		return
	}

	if user.WhatsappAPIKey == "" || user.WhatsappDeviceID == "" {
		log.Printf("[SESSION-BOT] Skipping project notification for session %d: user has no whatsapp credentials", session.ID)
		return
	}

	lessonTitle := session.Lesson.Title
	lessonNumber := session.Lesson.Number
	courseName := "-"
	if session.Lesson != nil {
		courseName = session.Lesson.Module
	}

	msg := fmt.Sprintf(
		"Halo %s, sesi Anda \"%d %s\" (kursus: %s) memiliki Project yang harus diselesaikan. Harap segera menyelesaikannya. Terima kasih!",
		user.Name, lessonNumber, lessonTitle, courseName,
	)

	// Send message directly
	err = u.waService.SendMessage(user.WhatsappAPIKey, user.WhatsappDeviceID, user.PhoneNumber, msg, false)
	if err != nil {
		log.Printf("[SESSION-BOT] Failed to send project notification for session %d: %v", session.ID, err)
	} else {
		log.Printf("[SESSION-BOT] Project notification sent for session %d to %s", session.ID, user.PhoneNumber)
	}
}

// ExportCSV mengekspor seluruh data sesi pembelajaran ke dalam format CSV.
func (u *sessionUsecase) ExportCSV(ctx context.Context) ([]byte, error) {
	sessions, err := u.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	headers := []string{
		"id", "group_id", "group_name", "lesson_id", "lesson_title",
		"date_start", "time_start", "is_done", "status",
		"after_session_feedback", "students_attended", "created_at", "updated_at",
	}

	rows := make([][]string, 0, len(sessions))
	for _, s := range sessions {
		groupName := ""
		if s.Group != nil {
			groupName = s.Group.Name
		}

		lessonTitle := ""
		if s.Lesson != nil {
			lessonTitle = s.Lesson.Title
		}

		dateStart := ""
		if !s.DateStart.Time.IsZero() {
			dateStart = s.DateStart.Time.Format("2006-01-02")
		}

		timeStart := ""
		if !s.TimeStart.Time.IsZero() {
			timeStart = s.TimeStart.Time.Format("15:04")
		}

		studentNames := make([]string, 0, len(s.StudentsAttended))
		for _, st := range s.StudentsAttended {
			studentNames = append(studentNames, st.Fullname)
		}

		rows = append(rows, []string{
			strconv.FormatUint(uint64(s.ID), 10),
			strconv.FormatUint(uint64(s.GroupID), 10),
			groupName,
			strconv.FormatUint(uint64(s.LessonID), 10),
			lessonTitle,
			dateStart,
			timeStart,
			strconv.FormatBool(s.IsDone),
			s.Status,
			strPtr(s.AfterSessionFeedback),
			strings.Join(studentNames, ", "),
			s.CreatedAt.Format("2006-01-02 15:04:05"),
			s.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return csvutil.Generate(headers, rows)
}
