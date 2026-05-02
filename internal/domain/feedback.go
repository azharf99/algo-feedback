// File: internal/domain/feedback.go
package domain

import (
	"context"
	"time"
)

// Membuat Custom Type untuk Type Safety
type AttendanceScore string
type ActivityScore string
type TaskScore string

// Definisi Konstanta dengan Custom Type
const (
	// Attendance Score
	AttendanceScoreNone    AttendanceScore = "0" // Tidak Hadir
	AttendanceScore1xMonth AttendanceScore = "1" // Hadir 1x Sebulan
	AttendanceScore2xMonth AttendanceScore = "2" // Hadir 2x Sebulan
	AttendanceScore3xMonth AttendanceScore = "3" // Hadir 3x Sebulan
	AttendanceScoreAlways  AttendanceScore = "4" // Selalu Hadir

	// Activity Score
	ActivityScoreInactive     ActivityScore = "0" // Tidak Aktif
	ActivityScoreLessActive   ActivityScore = "1" // Kurang Aktif
	ActivityScoreActiveEnough ActivityScore = "2" // Cukup Aktif
	ActivityScoreActive       ActivityScore = "3" // Aktif

	// Task Score
	TaskScoreNone    TaskScore = "0" // Tidak Mengerjakan Tugas
	TaskScorePartial TaskScore = "1" // Mengerjakan Sebagian Tugas
	TaskScoreAll     TaskScore = "2" // Mengerjakan Semua
)

// Feedback merepresentasikan tabel feedbacks di database.
type Feedback struct {
	ID            uint    `json:"id" gorm:"primaryKey"`
	Number        uint    `json:"number"`
	GroupName     *string `json:"group_name" gorm:"type:varchar(100)"`
	Topic         *string `json:"topic" gorm:"type:varchar(200)"`
	Result        *string `json:"result" gorm:"type:text"`
	Level         *string `json:"level" gorm:"type:varchar(50)"`
	Course        *string `json:"course" gorm:"type:varchar(100)"`
	ProjectLink   *string `json:"project_link" gorm:"type:text"`
	Competency    *string `json:"competency" gorm:"type:text"`
	TutorFeedback *string `json:"tutor_feedback" gorm:"type:text"`

	// Menggunakan Custom Type di Struct
	AttendanceScore AttendanceScore `json:"attendance_score" gorm:"type:varchar(1);default:'4'"`
	ActivityScore   ActivityScore   `json:"activity_score" gorm:"type:varchar(1);default:'3'"`
	TaskScore       TaskScore       `json:"task_score" gorm:"type:varchar(1);default:'2'"`

	LessonDate *DateOnly `json:"lesson_date" gorm:"type:date"`
	LessonTime *TimeOnly `json:"lesson_time" gorm:"type:time"`
	IsSent     bool      `json:"is_sent" gorm:"default:false"`
	ScheduleID *string   `json:"schedule_id" gorm:"type:varchar(150)"`
	TaskID     *string   `json:"task_id" gorm:"type:varchar(150)"`
	URLPDF     *string   `json:"url_pdf" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relasi
	StudentID *uint    `json:"student_id" gorm:"index"`
	Student   *Student `json:"student,omitempty" gorm:"foreignKey:StudentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// FeedbackRepository mendefinisikan operasi DB untuk Feedback
type FeedbackRepository interface {
	Create(ctx context.Context, feedback *Feedback) error
	GetByID(ctx context.Context, id uint) (*Feedback, error)
	GetAll(ctx context.Context) ([]Feedback, error)
	GetPaginated(ctx context.Context, params PaginationParams) ([]Feedback, int64, error)
	Update(ctx context.Context, feedback *Feedback) error
	Delete(ctx context.Context, id uint) error

	// Untuk fitur Feedback Seeder (update_or_create)
	UpsertSeeder(ctx context.Context, feedback *Feedback) (bool, error)

	// Untuk mengambil data feedback yang belum dikirim (is_sent=False)
	GetUnsentFeedbacks(ctx context.Context, studentID *uint, course *string, number *uint) ([]Feedback, error)

	// Untuk mengambil data feedback dengan filter fleksibel (misal untuk regenerasi PDF)
	GetFeedbacks(ctx context.Context, studentID *uint, course *string, number *uint, onlyUnsent bool) ([]Feedback, error)
}

// FeedbackUsecase mendefinisikan logika bisnis
type FeedbackUsecase interface {
	Create(ctx context.Context, feedback *Feedback) error
	GetByID(ctx context.Context, id uint) (*Feedback, error)
	GetAll(ctx context.Context) ([]Feedback, error)
	GetPaginated(ctx context.Context, params PaginationParams) (*PaginatedResult[Feedback], error)
	Update(ctx context.Context, id uint, req *Feedback) error
	Delete(ctx context.Context, id uint) error

	// Fitur Utama
	GenerateFeedback(ctx context.Context, groupID *uint, all bool) (map[string]int, error)
	GeneratePDFAsync(ctx context.Context, studentID *uint, course *string, number *uint, all bool) ([]map[string]interface{}, error)
	SendFeedbackPDF(ctx context.Context, feedbackID *uint) ([]map[string]interface{}, error)
}
