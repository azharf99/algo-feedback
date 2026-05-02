// File: internal/domain/feedback_test.go
package domain_test

import (
	"testing"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestFeedbackModel_Creation(t *testing.T) {
	now := time.Now()
	studentID := uint(10)
	groupName := "Golang Masterclass"
	topic := "Membangun REST API"
	result := "Siswa memahami cara kerja Gin"
	level := "Advanced"
	course := "Backend Engineering"
	projectLink := "https://github.com/azharf99/algo-feedback"
	competency := "Memahami Goroutine"
	tutorFeedback := "Lanjutkan eksplorasi Clean Architecture"
	scheduleID := "SCH-12345"
	taskID := "TASK-67890"
	urlPdf := "https://storage.googleapis.com/pdf/feedback1.pdf"

	feedback := domain.Feedback{
		ID:            1,
		Number:        101,
		GroupName:     &groupName,
		Topic:         &topic,
		Result:        &result,
		Level:         &level,
		Course:        &course,
		ProjectLink:   &projectLink,
		Competency:    &competency,
		TutorFeedback: &tutorFeedback,
		// Memasukkan nilai menggunakan Custom Type Enum
		AttendanceScore: domain.AttendanceScoreAlways,
		ActivityScore:   domain.ActivityScoreActive,
		TaskScore:       domain.TaskScoreAll,
		LessonDate:      &domain.DateOnly{Time: now},
		LessonTime:      &domain.TimeOnly{Time: now},
		IsSent:          true,
		ScheduleID:      &scheduleID,
		TaskID:          &taskID,
		URLPDF:          &urlPdf,
		CreatedAt:       now,
		UpdatedAt:       now,
		StudentID:       &studentID,
		Student:         &domain.Student{ID: 10, Fullname: "John Doe"},
	}

	assert.Equal(t, uint(1), feedback.ID)
	assert.Equal(t, uint(101), feedback.Number)

	// Verifikasi memastikan tipe data dan nilainya cocok
	assert.Equal(t, domain.AttendanceScoreAlways, feedback.AttendanceScore)
	assert.Equal(t, domain.ActivityScoreActive, feedback.ActivityScore)
	assert.Equal(t, domain.TaskScoreAll, feedback.TaskScore)

	assert.True(t, feedback.IsSent)
	assert.Equal(t, now, feedback.LessonDate)
	assert.Equal(t, now, feedback.LessonTime)
	assert.Equal(t, now, feedback.CreatedAt)
	assert.Equal(t, now, feedback.UpdatedAt)

	assert.NotNil(t, feedback.GroupName)
	assert.Equal(t, "Golang Masterclass", *feedback.GroupName)
	assert.NotNil(t, feedback.Topic)
	assert.Equal(t, "Membangun REST API", *feedback.Topic)
	assert.NotNil(t, feedback.Result)
	assert.Equal(t, "Siswa memahami cara kerja Gin", *feedback.Result)
	assert.NotNil(t, feedback.Level)
	assert.Equal(t, "Advanced", *feedback.Level)
	assert.NotNil(t, feedback.Course)
	assert.Equal(t, "Backend Engineering", *feedback.Course)
	assert.NotNil(t, feedback.ProjectLink)
	assert.Equal(t, "https://github.com/azharf99/algo-feedback", *feedback.ProjectLink)
	assert.NotNil(t, feedback.Competency)
	assert.Equal(t, "Memahami Goroutine", *feedback.Competency)
	assert.NotNil(t, feedback.TutorFeedback)
	assert.Equal(t, "Lanjutkan eksplorasi Clean Architecture", *feedback.TutorFeedback)
	assert.NotNil(t, feedback.ScheduleID)
	assert.Equal(t, "SCH-12345", *feedback.ScheduleID)
	assert.NotNil(t, feedback.TaskID)
	assert.Equal(t, "TASK-67890", *feedback.TaskID)
	assert.NotNil(t, feedback.URLPDF)
	assert.Equal(t, "https://storage.googleapis.com/pdf/feedback1.pdf", *feedback.URLPDF)

	assert.NotNil(t, feedback.StudentID)
	assert.Equal(t, uint(10), *feedback.StudentID)
	assert.NotNil(t, feedback.Student)
	assert.Equal(t, "John Doe", feedback.Student.Fullname)
}

func TestFeedbackModel_EmptyPointers(t *testing.T) {
	feedback := domain.Feedback{
		ID:     2,
		Number: 102,
	}

	assert.Equal(t, uint(2), feedback.ID)
	assert.Equal(t, uint(102), feedback.Number)

	assert.Nil(t, feedback.GroupName)
	assert.Nil(t, feedback.StudentID)
	assert.Nil(t, feedback.Student)
	assert.Nil(t, feedback.URLPDF)
}
