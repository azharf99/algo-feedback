// File: internal/domain/lesson_test.go
package domain_test

import (
	"testing"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestLessonModel_Creation(t *testing.T) {
	category := "Programming"
	description := "Pengenalan Goroutine dan Channels di Golang"
	now := time.Now()

	lesson := domain.Lesson{
		ID:          1,
		CourseID:    10, // Sekarang menggunakan CourseID
		Title:       "Concurrency in Go",
		Category:    &category,
		Module:      "Backend Mastery",
		Level:       "Intermediate",
		Number:      5,
		Description: &description,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
		Course:      &domain.Course{ID: 10, Title: "Golang Expert Class"},
	}

	assert.Equal(t, uint(1), lesson.ID)
	assert.Equal(t, uint(10), lesson.CourseID)
	assert.Equal(t, "Concurrency in Go", lesson.Title)
	assert.Equal(t, "Programming", *lesson.Category)
	assert.Equal(t, "Pengenalan Goroutine dan Channels di Golang", *lesson.Description)
	assert.True(t, lesson.IsActive)
	assert.NotNil(t, lesson.Course)
	assert.Equal(t, "Golang Expert Class", lesson.Course.Title)
}

func TestLessonModel_EmptyPointers(t *testing.T) {
	lesson := domain.Lesson{
		ID:       2,
		Title:    "Basic Syntax",
		Module:   "Go Fundamental",
		Level:    "Beginner",
		Number:   1,
		CourseID: 5,
	}

	assert.Equal(t, uint(2), lesson.ID)
	assert.Equal(t, "Basic Syntax", lesson.Title)
	assert.Nil(t, lesson.Category)
	assert.Nil(t, lesson.Description)
}
