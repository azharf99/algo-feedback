// File: internal/usecase/lesson_usecase.go
package usecase

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
)

type lessonUsecase struct {
	repo domain.LessonRepository
}

func NewLessonUsecase(repo domain.LessonRepository) domain.LessonUsecase {
	return &lessonUsecase{repo: repo}
}

// ... (Metode CRUD Create, GetByID, GetAll, GetPaginated, Update, Delete tetap sama) ...
func (u *lessonUsecase) Create(ctx context.Context, lesson *domain.Lesson) error {
	return u.repo.Create(ctx, lesson)
}
func (u *lessonUsecase) GetByID(ctx context.Context, id uint) (*domain.Lesson, error) {
	return u.repo.GetByID(ctx, id)
}
func (u *lessonUsecase) GetAll(ctx context.Context) ([]domain.Lesson, error) {
	return u.repo.GetAll(ctx)
}

func (u *lessonUsecase) GetByCourse(ctx context.Context, courseID uint) ([]domain.Lesson, error) {
	return u.repo.GetByCourse(ctx, courseID)
}

func (u *lessonUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.Lesson], error) {
	params = pagination.Normalize(params)
	lessons, total, err := u.repo.GetPaginated(ctx, params)
	if err != nil {
		return nil, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return &domain.PaginatedResult[domain.Lesson]{
		Data:       lessons,
		Total:      total,
		TotalPages: totalPages,
		Page:       params.Page,
		Limit:      params.Limit,
	}, nil
}
func (u *lessonUsecase) Update(ctx context.Context, id uint, req *domain.Lesson) error {
	existing, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New("pelajaran tidak ditemukan")
	}

	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Level != "" {
		existing.Level = req.Level
	}
	if req.CourseID != 0 {
		existing.CourseID = req.CourseID
	}

	return u.repo.Update(ctx, existing)
}
func (u *lessonUsecase) Delete(ctx context.Context, id uint) error { return u.repo.Delete(ctx, id) }

func (u *lessonUsecase) ImportCSV(ctx context.Context, fileReader io.Reader) (*domain.ImportResult, error) {
	result := &domain.ImportResult{Errors: make([]map[string]interface{}, 0)}

	reader := csv.NewReader(fileReader)
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("gagal membaca header CSV: %w", err)
	}

	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.TrimSpace(header)] = i
	}

	rowNum := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": err.Error()})
			continue
		}
		rowNum++

		idUint, err := strconv.ParseUint(record[headerMap["id"]], 10, 32)
		if err != nil || idUint == 0 {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "ID tidak valid"})
			continue
		}

		// Mengubah dari group_id menjadi course_id
		courseID, err := strconv.ParseUint(record[headerMap["course_id"]], 10, 32)
		if err != nil || courseID == 0 {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "course_id tidak valid"})
			continue
		}

		num, _ := strconv.Atoi(record[headerMap["number"]])
		category := record[headerMap["category"]]
		desc := record[headerMap["description"]]

		lesson := &domain.Lesson{
			ID:          uint(idUint),
			CourseID:    uint(courseID),
			Title:       record[headerMap["title"]],
			Category:    &category,
			Module:      record[headerMap["module"]],
			Level:       record[headerMap["level"]],
			Number:      uint(num),
			Description: &desc,
			IsActive:    strings.ToLower(record[headerMap["is_active"]]) != "false",
		}

		// Panggil Upsert TANPA studentIDs
		isCreated, err := u.repo.Upsert(ctx, lesson)
		if err != nil {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": err.Error()})
			continue
		}

		if isCreated {
			result.Created++
		} else {
			result.Updated++
		}
	}

	return result, nil
}
