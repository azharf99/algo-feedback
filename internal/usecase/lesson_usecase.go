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
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/azharf99/algo-feedback/pkg/pagination"
)

type lessonUsecase struct {
	repo           domain.LessonRepository
	sessionUsecase domain.SessionUsecase
}

func (u *lessonUsecase) getLang(ctx context.Context) string {
	return ctxutil.GetLanguage(ctx)
}

func NewLessonUsecase(repo domain.LessonRepository, sessionUsecase domain.SessionUsecase) domain.LessonUsecase {
	return &lessonUsecase{
		repo:           repo,
		sessionUsecase: sessionUsecase,
	}
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
	lang := u.getLang(ctx)
	if err != nil {
		return errors.New(i18n.T(lang, "error_lesson_not_found"))
	}

	competencyChanged := false
	if req.Competency != "" && existing.Competency != req.Competency {
		competencyChanged = true
		existing.Competency = req.Competency
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

	if req.IsProjectLesson != existing.IsProjectLesson {
		existing.IsProjectLesson = req.IsProjectLesson
	}

	err = u.repo.Update(ctx, existing)
	if err != nil {
		return err
	}

	// Cascading update to WhatsApp schedules
	if competencyChanged {
		go func(lessonID uint) {
			bgCtx := context.Background()
			sessions, err := u.sessionUsecase.GetByLesson(bgCtx, lessonID)
			if err != nil {
				return
			}
			for _, session := range sessions {
				if session.IsDone && session.ScheduledMessageID != nil {
					// We pass bgCtx. Note: userName from context will not be available in bgCtx!
					// But we can extract user_id from the original ctx and inject it to bgCtx.
					var updatedCtx = bgCtx
					if userID, ok := ctx.Value("user_id").(float64); ok {
						updatedCtx = context.WithValue(updatedCtx, "user_id", userID)
					} else if userID, ok := ctx.Value("user_id").(uint); ok {
						updatedCtx = context.WithValue(updatedCtx, "user_id", userID)
					}
					u.sessionUsecase.TriggerAfterSessionFeedback(updatedCtx, &session)
				}
			}
		}(existing.ID)
	}

	return nil
}
func (u *lessonUsecase) Delete(ctx context.Context, id uint) error {
	lang := u.getLang(ctx)
	_, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New(i18n.T(lang, "error_lesson_not_found"))
	}
	return u.repo.Delete(ctx, id)
}

func (u *lessonUsecase) BulkDelete(ctx context.Context, ids []uint) error {
	return u.repo.BulkDelete(ctx, ids)
}

func (u *lessonUsecase) ImportCSV(ctx context.Context, fileReader io.Reader) (*domain.ImportResult, error) {
	result := &domain.ImportResult{Errors: make([]map[string]interface{}, 0)}
	lang := u.getLang(ctx)

	reader := csv.NewReader(fileReader)
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T(lang, "error_csv_header"), err)
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
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": i18n.T(lang, "error_invalid_id")})
			continue
		}

		// Mengubah dari group_id menjadi course_id
		courseID, err := strconv.ParseUint(record[headerMap["course_id"]], 10, 32)
		if err != nil || courseID == 0 {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": i18n.T(lang, "error_course_not_found")})
			continue
		}

		num, _ := strconv.Atoi(record[headerMap["number"]])
		category := record[headerMap["category"]]
		desc := record[headerMap["description"]]

		lesson := &domain.Lesson{
			ID:              uint(idUint),
			CourseID:        uint(courseID),
			Title:           record[headerMap["title"]],
			Category:        &category,
			Module:          record[headerMap["module"]],
			Level:           record[headerMap["level"]],
			Competency:      record[headerMap["competency"]],
			IsProjectLesson: strings.ToLower(record[headerMap["is_project_lesson"]]) == "true",
			Number:          uint(num),
			Description:     &desc,
			IsActive:        strings.ToLower(record[headerMap["is_active"]]) != "false",
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

func (u *lessonUsecase) ImportCompetenciesCSV(ctx context.Context, fileReader io.Reader) (*domain.ImportResult, error) {
	result := &domain.ImportResult{Errors: make([]map[string]interface{}, 0)}
	lang := u.getLang(ctx)

	reader := csv.NewReader(fileReader)
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T(lang, "error_csv_header"), err)
	}

	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.TrimSpace(strings.ToLower(header))] = i
	}

	// Group competencies by CourseID
	courseCompetencies := make(map[uint][]string)

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

		idIdx, okID := headerMap["id"]
		compIdx, okComp := headerMap["competencies"]

		if !okID || !okComp {
			return nil, errors.New(i18n.T(lang, "error_csv_headers_missing"))
		}

		courseID, err := strconv.ParseUint(record[idIdx], 10, 32)
		if err != nil {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": i18n.T(lang, "error_invalid_course_id")})
			continue
		}

		compStr := record[compIdx]
		if compStr == "" {
			continue
		}

		courseCompetencies[uint(courseID)] = append(courseCompetencies[uint(courseID)], compStr)
	}

	// Process each CourseID
	for courseID, competencies := range courseCompetencies {
		lessons, err := u.repo.GetByCourse(ctx, courseID)
		if err != nil {
			result.Errors = append(result.Errors, map[string]interface{}{"course_id": courseID, "error": i18n.T(lang, "error_fetch_lesson_failed")})
			continue
		}

		if len(lessons) == 0 {
			result.Errors = append(result.Errors, map[string]interface{}{"course_id": courseID, "error": i18n.T(lang, "error_no_lesson_found")})
			continue
		}

		// Map competencies to lessons
		for i := 0; i < len(lessons) && i < len(competencies); i++ {
			lessons[i].Competency = competencies[i]
			err := u.repo.Update(ctx, &lessons[i])
			if err != nil {
				result.Errors = append(result.Errors, map[string]interface{}{"course_id": courseID, "lesson_id": lessons[i].ID, "error": err.Error()})
			} else {
				result.Updated++
			}
		}

		if len(competencies) != len(lessons) {
			result.Errors = append(result.Errors, map[string]interface{}{
				"course_id": courseID,
				"warning":   i18n.Tf(lang, "warn_count_mismatch", len(competencies), len(lessons)),
			})
		}
	}

	return result, nil
}
