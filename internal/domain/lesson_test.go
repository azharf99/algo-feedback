// File: internal/domain/lesson_test.go
package domain_test

import (
	"testing"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestLessonModel_Creation(t *testing.T) {
	// 1. Arrange: Persiapan Data
	category := "Programming"
	description := "Pengenalan Goroutine dan Channels di Golang"
	meetingLink := "https://meet.google.com/abc-defg-hij"
	feedback := "Pelajaran berjalan sangat interaktif"
	now := time.Now()

	// 2. Act: Inisiasi Struct
	lesson := domain.Lesson{
		ID:          1,
		Title:       "Concurrency in Go",
		Category:    &category,
		Module:      "Backend Mastery",
		Level:       "Intermediate",
		Number:      5,
		Description: &description,
		DateStart:   now,
		TimeStart:   now,
		MeetingLink: &meetingLink,
		Feedback:    &feedback,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
		// Inisiasi relasi Foreign Key
		GroupID: 2,
		Group:   &domain.Group{ID: 2, Name: "Golang Backend Batch 2"},
		// Inisiasi relasi Many-to-Many
		StudentsAttended: []domain.Student{
			{ID: 1, Fullname: "Budi Santoso"},
		},
	}

	// 3. Assert: Verifikasi SEMUA field agar tidak ada warning "unused write"
	assert.Equal(t, uint(1), lesson.ID)
	assert.Equal(t, "Concurrency in Go", lesson.Title)
	assert.Equal(t, "Backend Mastery", lesson.Module)
	assert.Equal(t, "Intermediate", lesson.Level)
	assert.Equal(t, uint(5), lesson.Number)
	assert.True(t, lesson.IsActive)
	assert.Equal(t, now, lesson.CreatedAt)
	assert.Equal(t, now, lesson.UpdatedAt)
	assert.Equal(t, now, lesson.DateStart)
	assert.Equal(t, now, lesson.TimeStart)

	// Pengecekan Pointer
	assert.NotNil(t, lesson.Category)
	assert.Equal(t, "Programming", *lesson.Category)

	assert.NotNil(t, lesson.Description)
	assert.Equal(t, "Pengenalan Goroutine dan Channels di Golang", *lesson.Description)

	assert.NotNil(t, lesson.MeetingLink)
	assert.Equal(t, "https://meet.google.com/abc-defg-hij", *lesson.MeetingLink)

	assert.NotNil(t, lesson.Feedback)
	assert.Equal(t, "Pelajaran berjalan sangat interaktif", *lesson.Feedback)

	// Pengecekan Relasi Group
	assert.Equal(t, uint(2), lesson.GroupID)
	assert.NotNil(t, lesson.Group)
	assert.Equal(t, "Golang Backend Batch 2", lesson.Group.Name)

	// Pengecekan Relasi Students
	assert.Len(t, lesson.StudentsAttended, 1)
	assert.Equal(t, "Budi Santoso", lesson.StudentsAttended[0].Fullname)
}

func TestLessonModel_EmptyPointers(t *testing.T) {
	// Memastikan field yang nullable aman jika tidak diisi (nil)
	lesson := domain.Lesson{
		ID:      2,
		Title:   "Basic Syntax",
		Module:  "Go Fundamental",
		Level:   "Beginner",
		Number:  1,
		GroupID: 1,
	}

	// Assert required fields
	assert.Equal(t, uint(2), lesson.ID)
	assert.Equal(t, "Basic Syntax", lesson.Title)

	assert.Equal(t, "Go Fundamental", lesson.Module)
	assert.Equal(t, "Beginner", lesson.Level)
	assert.Equal(t, uint(1), lesson.Number)

	// Pengecekan Relasi Group
	assert.Equal(t, uint(1), lesson.GroupID)

	// Assert optional fields are nil or empty
	assert.Nil(t, lesson.Category)
	assert.Nil(t, lesson.Description)
	assert.Nil(t, lesson.MeetingLink)
	assert.Nil(t, lesson.Feedback)
	assert.Nil(t, lesson.Group)
	assert.Empty(t, lesson.StudentsAttended)
}
