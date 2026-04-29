// File: internal/usecase/course_usecase.go
package usecase

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
)

type courseUsecase struct {
	repo domain.CourseRepository
}

func NewCourseUsecase(repo domain.CourseRepository) domain.CourseUsecase {
	return &courseUsecase{repo: repo}
}

func (u *courseUsecase) Create(ctx context.Context, course *domain.Course) error {
	return u.repo.Create(ctx, course)
}

func (u *courseUsecase) GetByID(ctx context.Context, id uint) (*domain.Course, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *courseUsecase) GetAll(ctx context.Context) ([]domain.Course, error) {
	return u.repo.GetAll(ctx)
}

func (u *courseUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.Course], error) {
	params = pagination.Normalize(params)
	courses, total, err := u.repo.GetPaginated(ctx, params)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return &domain.PaginatedResult[domain.Course]{
		Data:       courses,
		Total:      total,
		TotalPages: totalPages,
		Page:       params.Page,
		Limit:      params.Limit,
	}, nil
}

func (u *courseUsecase) Update(ctx context.Context, id uint, req *domain.Course) error {
	req.ID = id
	return u.repo.Update(ctx, req)
}

func (u *courseUsecase) Delete(ctx context.Context, id uint) error {
	return u.repo.Delete(ctx, id)
}

// ImportCSV memproses data blueprint kurikulum dari CSV
func (u *courseUsecase) ImportCSV(ctx context.Context, fileReader io.Reader) (*domain.ImportResult, error) {
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

		// 1. Validasi ID Course
		idUint, err := strconv.ParseUint(record[headerMap["id"]], 10, 32)
		if err != nil || idUint == 0 {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "ID tidak valid"})
			continue
		}

		// 2. Tangani Nilai Opsional (Pointer)
		desc := record[headerMap["description"]]

		// 3. Bangun Objek Course
		course := &domain.Course{
			ID:          uint(idUint),
			Title:       record[headerMap["title"]],
			Module:      record[headerMap["module"]],
			Description: &desc,
			IsActive:    strings.ToLower(record[headerMap["is_active"]]) != "false",
		}

		// 4. Eksekusi Upsert
		isCreated, err := u.repo.Upsert(ctx, course)
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
