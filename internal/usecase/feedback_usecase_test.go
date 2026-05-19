package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pdfgen"
	"github.com/azharf99/algo-feedback/pkg/taskqueue"
	"github.com/stretchr/testify/assert"
)

// Mock repositories and services

type mockFeedbackRepository struct {
	feedbacks []domain.Feedback
	err       error
	getByIDFn func(ctx context.Context, id uint) (*domain.Feedback, error)
	updateFn  func(ctx context.Context, f *domain.Feedback) error
}

func (m *mockFeedbackRepository) Create(ctx context.Context, f *domain.Feedback) error { return nil }
func (m *mockFeedbackRepository) GetByID(ctx context.Context, id uint) (*domain.Feedback, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockFeedbackRepository) GetAll(ctx context.Context) ([]domain.Feedback, error) {
	return nil, nil
}
func (m *mockFeedbackRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Feedback, int64, error) {
	return nil, 0, nil
}
func (m *mockFeedbackRepository) Update(ctx context.Context, f *domain.Feedback) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, f)
	}
	return nil
}
func (m *mockFeedbackRepository) Delete(ctx context.Context, id uint) error        { return nil }
func (m *mockFeedbackRepository) BulkDelete(ctx context.Context, ids []uint) error { return nil }
func (m *mockFeedbackRepository) GetStats(ctx context.Context) (domain.FeedbackStats, error) {
	return domain.FeedbackStats{}, nil
}
func (m *mockFeedbackRepository) UpsertSeeder(ctx context.Context, f *domain.Feedback) (bool, error) {
	return false, nil
}
func (m *mockFeedbackRepository) GetUnsentFeedbacks(ctx context.Context, studentID *uint, course *string, number *uint) ([]domain.Feedback, error) {
	return nil, nil
}
func (m *mockFeedbackRepository) GetFeedbacks(ctx context.Context, studentID *uint, course *string, number *uint, onlyUnsent bool) ([]domain.Feedback, error) {
	return m.feedbacks, m.err
}
func (m *mockFeedbackRepository) GetPendingPDFFeedbacks(ctx context.Context) ([]domain.Feedback, error) {
	return nil, nil
}
func (m *mockFeedbackRepository) GetByDateRange(ctx context.Context, start, end time.Time) ([]domain.Feedback, error) {
	return nil, nil
}

type mockGraduationFeedbackRepository struct {
	created []*domain.GraduationFeedback
}

func (m *mockGraduationFeedbackRepository) Create(ctx context.Context, gf *domain.GraduationFeedback) error {
	m.created = append(m.created, gf)
	return nil
}
func (m *mockGraduationFeedbackRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.GraduationFeedback, int64, error) {
	return nil, 0, nil
}
func (m *mockGraduationFeedbackRepository) GetByID(ctx context.Context, id uint) (*domain.GraduationFeedback, error) {
	return nil, nil
}
func (m *mockGraduationFeedbackRepository) Update(ctx context.Context, gf *domain.GraduationFeedback) error {
	return nil
}
func (m *mockGraduationFeedbackRepository) Delete(ctx context.Context, id uint) error {
	return nil
}

type mockGroupRepository struct {
	groups []domain.Group
	err    error
}

func (m *mockGroupRepository) Create(ctx context.Context, g *domain.Group, sIDs []uint) error {
	return nil
}
func (m *mockGroupRepository) GetByID(ctx context.Context, id uint) (*domain.Group, error) {
	return nil, nil
}
func (m *mockGroupRepository) GetAll(ctx context.Context) ([]domain.Group, error) {
	return m.groups, m.err
}
func (m *mockGroupRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Group, int64, error) {
	return nil, 0, nil
}
func (m *mockGroupRepository) Update(ctx context.Context, g *domain.Group, sIDs []uint) error {
	return nil
}
func (m *mockGroupRepository) Delete(ctx context.Context, id uint) error        { return nil }
func (m *mockGroupRepository) BulkDelete(ctx context.Context, ids []uint) error { return nil }
func (m *mockGroupRepository) Upsert(ctx context.Context, g *domain.Group, sIDs []uint) (bool, error) {
	return false, nil
}

type mockSessionRepository struct {
	sessions []domain.Session
	err      error
}

func (m *mockSessionRepository) Create(ctx context.Context, s *domain.Session) error { return nil }
func (m *mockSessionRepository) GetByID(ctx context.Context, id uint) (*domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepository) GetByGroup(ctx context.Context, groupID uint) ([]domain.Session, error) {
	return m.sessions, m.err
}
func (m *mockSessionRepository) GetByLesson(ctx context.Context, lessonID uint) ([]domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepository) GetAll(ctx context.Context) ([]domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Session, int64, error) {
	return nil, 0, nil
}
func (m *mockSessionRepository) Update(ctx context.Context, s *domain.Session) error { return nil }
func (m *mockSessionRepository) Delete(ctx context.Context, id uint) error           { return nil }
func (m *mockSessionRepository) BulkDelete(ctx context.Context, ids []uint) error    { return nil }
func (m *mockSessionRepository) Upsert(ctx context.Context, s *domain.Session) (bool, error) {
	return false, nil
}
func (m *mockSessionRepository) UpsertAttendance(ctx context.Context, s *domain.Session, studentIDs []uint) error {
	return nil
}
func (m *mockSessionRepository) MarkDoneUpToDate(ctx context.Context, groupID uint, date time.Time) error {
	return nil
}
func (m *mockSessionRepository) AutoFillAttendance(ctx context.Context, groupID uint, date time.Time) error {
	return nil
}
func (m *mockSessionRepository) GetByDateRange(ctx context.Context, start, end time.Time) ([]domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepository) MarkCancelled(ctx context.Context, groupID uint, start, end time.Time) error {
	return nil
}
func (m *mockSessionRepository) GetSessionsToAutoComplete(ctx context.Context, now time.Time) ([]domain.Session, error) {
	return nil, nil
}

type mockStudentRepository struct {
	student *domain.Student
	err     error
}

func (m *mockStudentRepository) Create(ctx context.Context, s *domain.Student) error { return nil }
func (m *mockStudentRepository) GetByID(ctx context.Context, id uint) (*domain.Student, error) {
	return m.student, m.err
}
func (m *mockStudentRepository) GetAll(ctx context.Context) ([]domain.Student, error) {
	return nil, nil
}
func (m *mockStudentRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.Student, int64, error) {
	return nil, 0, nil
}
func (m *mockStudentRepository) Update(ctx context.Context, s *domain.Student) error { return nil }
func (m *mockStudentRepository) Delete(ctx context.Context, id uint) error           { return nil }
func (m *mockStudentRepository) BulkDelete(ctx context.Context, ids []uint) error    { return nil }
func (m *mockStudentRepository) Upsert(ctx context.Context, s *domain.Student) (bool, error) {
	return false, nil
}

type mockUserRepository struct{}

func (m *mockUserRepository) Create(ctx context.Context, u *domain.User) error { return nil }
func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepository) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	return &domain.User{ID: id, Name: "Mock Tutor"}, nil
}
func (m *mockUserRepository) GetByResetToken(ctx context.Context, token string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepository) GetAll(ctx context.Context) ([]domain.User, error) { return nil, nil }
func (m *mockUserRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (m *mockUserRepository) Update(ctx context.Context, u *domain.User) error { return nil }
func (m *mockUserRepository) Delete(ctx context.Context, id uint) error        { return nil }
func (m *mockUserRepository) BulkDelete(ctx context.Context, ids []uint) error { return nil }

type mockPDFGenerator struct {
	calledWith []pdfgen.PDFData
}

func (m *mockPDFGenerator) Generate(ctx context.Context, data pdfgen.PDFData, outputPath string) error {
	m.calledWith = append(m.calledWith, data)
	return nil
}

type mockGraduationPDFGenerator struct {
	calledWith pdfgen.GraduationPDFData
}

func (m *mockGraduationPDFGenerator) Generate(ctx context.Context, data pdfgen.GraduationPDFData, outputPath string) error {
	m.calledWith = data
	return nil
}

type mockWhatsappService struct{}

func (m *mockWhatsappService) ScheduleMedia(apiKey, deviceID, to, caption, filePath, runAt string, isGroup bool) (int, error) {
	return 0, nil
}
func (m *mockWhatsappService) ScheduleMessage(apiKey, deviceID, to, message, runAt string, isGroup bool) (int, error) {
	return 0, nil
}
func (m *mockWhatsappService) UpdateSchedule(apiKey, deviceID string, id int, to, message, runAt string, isGroup bool) error {
	return nil
}
func (m *mockWhatsappService) DeleteSchedule(apiKey string, id int) error {
	return nil
}

type mockWorkerPool struct{}

func (m *mockWorkerPool) Submit(task taskqueue.Task) {
	// Execute immediately in test for simplicity
	_ = task.Execute(context.Background())
}
func (m *mockWorkerPool) Start() {}
func (m *mockWorkerPool) Stop()  {}

func TestParseScore(t *testing.T) {
	assert.Equal(t, 3, parseScore("3", 2))
	assert.Equal(t, 2, parseScore("", 2))
	assert.Equal(t, 2, parseScore("invalid", 2))
}

func TestGetGrade(t *testing.T) {
	assert.Equal(t, "A", getGrade(85.0))
	assert.Equal(t, "A", getGrade(80.0))
	assert.Equal(t, "B+", getGrade(78.0))
	assert.Equal(t, "B+", getGrade(75.0))
	assert.Equal(t, "B", getGrade(72.0))
	assert.Equal(t, "B", getGrade(70.0))
	assert.Equal(t, "C", getGrade(65.0))
	assert.Equal(t, "C", getGrade(60.0))
	assert.Equal(t, "D", getGrade(59.0))
}

func TestGenerateGraduationPDFAsync(t *testing.T) {
	student := &domain.Student{
		ID:       12,
		Fullname: "Andi Wijaya",
	}

	course := &domain.Course{
		ID:     3,
		Title:  "Python Start 1st year",
		Module: "Python Start 1st year",
	}

	group := domain.Group{
		ID:       10,
		UserID:   5,
		Name:     "Python-A",
		Language: "Indonesia",
		Students: []domain.Student{*student},
		Course:   course,
	}

	sessions := []domain.Session{
		{
			ID:       101,
			GroupID:  10,
			LessonID: 1,
			Lesson: &domain.Lesson{
				ID:       1,
				Number:   1,
				Module:   "Python Start 1st year",
				Level:    "M1L1",
				Title:    "Linear Algorithm Intro",
				Category: pointerToString("Introduction"),
			},
			StudentsAttended: []domain.Student{*student},
		},
		{
			ID:       102,
			GroupID:  10,
			LessonID: 2,
			Lesson: &domain.Lesson{
				ID:       2,
				Number:   2,
				Module:   "Python Start 1st year",
				Level:    "M1L2",
				Title:    "Variables and Types",
				Category: pointerToString("Data Types"),
			},
			StudentsAttended: []domain.Student{}, // Absent!
		},
	}

	feedbacks := []domain.Feedback{
		{
			StudentID:       pointerToUint(12),
			Number:          1,
			Course:          pointerToString("Python Start 1st year"),
			ActivityScore:   "3",
			TaskScore:       "2",
			TutorFeedback:   pointerToString("Great job in month 1!"),
			AttendanceScore: "3",
		},
	}

	feedRepo := &mockFeedbackRepository{feedbacks: feedbacks}
	gradFeedRepo := &mockGraduationFeedbackRepository{}
	groupRepo := &mockGroupRepository{groups: []domain.Group{group}}
	sessionRepo := &mockSessionRepository{sessions: sessions}
	studentRepo := &mockStudentRepository{student: student}
	pdfGen := &mockPDFGenerator{}
	gradPdfGen := &mockGraduationPDFGenerator{}
	waService := &mockWhatsappService{}
	userRepo := &mockUserRepository{}
	pool := &mockWorkerPool{}

	u := NewFeedbackUsecase(feedRepo, gradFeedRepo, groupRepo, sessionRepo, studentRepo, pdfGen, gradPdfGen, waService, userRepo, pool)

	ctx := context.Background()
	studentID := uint(12)
	courseName := "Python Start 1st year"

	resp, err := u.GenerateGraduationPDFAsync(ctx, &studentID, &courseName)
	assert.NoError(t, err)
	assert.Len(t, resp, 1)
	assert.Equal(t, "Andi Wijaya", resp[0]["student"])
	assert.Equal(t, "processing in background", resp[0]["status"])

	// Verify graduation PDF data passed to generator
	calledData := gradPdfGen.calledWith
	assert.Equal(t, "Andi Wijaya", calledData.StudentName)
	assert.Equal(t, "Python Start 1st year", calledData.CourseName)
	assert.Equal(t, "M1L1 - M1L2", calledData.LessonRange)
	assert.Equal(t, "Mock Tutor", calledData.TeacherName)
	assert.Equal(t, "Great job in month 1!", calledData.TutorFeedback)
	assert.Len(t, calledData.Lessons, 2)

	// Lesson 1: Present (Attendance=4, Activity=3, Task=2) => Total=9 => 100% => A
	assert.Equal(t, "1", calledData.Lessons[0].LessonNumber)
	assert.Equal(t, "100%", calledData.Lessons[0].Score)
	assert.Equal(t, "A", calledData.Lessons[0].Grade)

	// Lesson 2: Absent (Attendance=0, Activity=3, Task=2) => Total=5 => 56% => D
	assert.Equal(t, "2", calledData.Lessons[1].LessonNumber)
	assert.Equal(t, "56%", calledData.Lessons[1].Score)
	assert.Equal(t, "D", calledData.Lessons[1].Grade)

	// Verify database record was created
	assert.Len(t, gradFeedRepo.created, 1)
	assert.Equal(t, uint(12), gradFeedRepo.created[0].StudentID)
	assert.Equal(t, "Python Start 1st year", gradFeedRepo.created[0].Course)
	assert.Equal(t, "B+", gradFeedRepo.created[0].Grade)
	assert.Equal(t, "Great job in month 1!", gradFeedRepo.created[0].TutorFeedback)
	assert.NotNil(t, gradFeedRepo.created[0].URLPDF)
}

func pointerToString(s string) *string {
	return &s
}

func pointerToUint(u uint) *uint {
	return &u
}

func TestUpdateFeedback_RegeneratesPDF(t *testing.T) {
	student := &domain.Student{
		ID:       12,
		Fullname: "Andi Wijaya",
	}

	existingFeedback := &domain.Feedback{
		ID:              1,
		StudentID:       pointerToUint(12),
		Student:         student,
		Number:          1,
		Course:          pointerToString("Python Start 1st year"),
		GroupName:       pointerToString("Python-A"),
		Level:           pointerToString("M1"),
		AttendanceScore: "4",
		ActivityScore:   "3",
		TaskScore:       "2",
		Language:        "Indonesia",
		UserID:          100,
	}

	var updates []*domain.Feedback
	feedRepo := &mockFeedbackRepository{
		getByIDFn: func(ctx context.Context, id uint) (*domain.Feedback, error) {
			return existingFeedback, nil
		},
		updateFn: func(ctx context.Context, f *domain.Feedback) error {
			updates = append(updates, f)
			return nil
		},
	}

	pdfGen := &mockPDFGenerator{}
	gradPdfGen := &mockGraduationPDFGenerator{}
	groupRepo := &mockGroupRepository{}
	sessionRepo := &mockSessionRepository{}
	studentRepo := &mockStudentRepository{}
	waService := &mockWhatsappService{}
	userRepo := &mockUserRepository{}
	pool := &mockWorkerPool{}

	u := NewFeedbackUsecase(feedRepo, &mockGraduationFeedbackRepository{}, groupRepo, sessionRepo, studentRepo, pdfGen, gradPdfGen, waService, userRepo, pool)

	req := &domain.Feedback{
		AttendanceScore: "3",
		ActivityScore:   "2",
		TaskScore:       "1",
		TutorFeedback:   pointerToString("Updated tutor feedback"),
	}

	ctx := context.Background()
	err := u.Update(ctx, 1, req)
	assert.NoError(t, err)

	assert.GreaterOrEqual(t, len(updates), 1)
	firstUpdate := updates[0]
	assert.Equal(t, domain.AttendanceScore("3"), firstUpdate.AttendanceScore)
	assert.Equal(t, domain.ActivityScore("2"), firstUpdate.ActivityScore)
	assert.Equal(t, domain.TaskScore("1"), firstUpdate.TaskScore)
	assert.Equal(t, "Updated tutor feedback", *firstUpdate.TutorFeedback)

	assert.Len(t, pdfGen.calledWith, 1)
	calledData := pdfGen.calledWith[0]
	assert.Equal(t, "Andi Wijaya", calledData.StudentName)
	assert.Equal(t, uint(1), calledData.StudentMonthCourse)
	assert.Equal(t, "Python Start 1st year", calledData.StudentClass)

	if len(updates) > 1 {
		secondUpdate := updates[1]
		assert.NotNil(t, secondUpdate.URLPDF)
		assert.Contains(t, *secondUpdate.URLPDF, "Andi_Wijaya")
	}
}
