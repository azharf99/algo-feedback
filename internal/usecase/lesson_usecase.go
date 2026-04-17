// File: internal/usecase/lesson_usecase.go
package usecase

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
)

type lessonUsecase struct {
	repo domain.LessonRepository
}

func NewLessonUsecase(repo domain.LessonRepository) domain.LessonUsecase {
	return &lessonUsecase{repo: repo}
}

func (u *lessonUsecase) Create(ctx context.Context, lesson *domain.Lesson) error {
	return u.repo.Create(ctx, lesson)
}
func (u *lessonUsecase) GetByID(ctx context.Context, id uint) (*domain.Lesson, error) {
	return u.repo.GetByID(ctx, id)
}
func (u *lessonUsecase) GetAll(ctx context.Context) ([]domain.Lesson, error) {
	return u.repo.GetAll(ctx)
}
func (u *lessonUsecase) Update(ctx context.Context, id uint, req *domain.Lesson) error {
	req.ID = id
	return u.repo.Update(ctx, req)
}
func (u *lessonUsecase) Delete(ctx context.Context, id uint) error { return u.repo.Delete(ctx, id) }

func (u *lessonUsecase) ImportCSV(ctx context.Context, fileReader io.Reader) (interface{}, error) {
	result := &domain.ImportResult{Errors: make([]map[string]interface{}, 0)}
	reader := csv.NewReader(fileReader)

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("gagal membaca header CSV: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
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

		// Parsing ID Pelajaran
		idUint, _ := strconv.ParseUint(record[headerMap["id"]], 10, 32)
		if idUint == 0 {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "ID Lesson wajib diisi"})
			continue
		}

		// Parsing Group ID (Foreign Key)
		groupID, err := strconv.ParseUint(record[headerMap["group_id"]], 10, 32)
		if err != nil {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "Group ID tidak valid"})
			continue
		}

		// Parsing Date & Time
		var dateStart time.Time
		if ds := record[headerMap["date_start"]]; ds != "" {
			dateStart, _ = time.Parse("02/01/2006", ds) // Format dd/mm/yyyy
		}

		var timeStart time.Time
		if ts := record[headerMap["time_start"]]; ts != "" {
			timeStart, _ = time.Parse("15:04:05", ts)
		}

		num, _ := strconv.Atoi(record[headerMap["number"]])
		category := record[headerMap["category"]]
		desc := record[headerMap["description"]]
		meetLink := record[headerMap["meeting_link"]]
		fb := record[headerMap["feedback"]]

		lesson := &domain.Lesson{
			ID:          uint(idUint),
			Title:       record[headerMap["title"]],
			Category:    &category,
			Module:      record[headerMap["module"]],
			Level:       record[headerMap["level"]],
			Number:      uint(num),
			GroupID:     uint(groupID),
			Description: &desc,
			DateStart:   dateStart,
			TimeStart:   timeStart,
			MeetingLink: &meetLink,
			Feedback:    &fb,
			IsActive:    strings.ToLower(record[headerMap["is_active"]]) != "false",
		}

		// Parsing Many-to-Many Siswa yang Hadir
		var studentIDs []uint
		if sa := record[headerMap["students_attended"]]; sa != "" {
			parts := strings.Split(sa, ",")
			for _, p := range parts {
				sID, err := strconv.ParseUint(strings.TrimSpace(p), 10, 32)
				if err == nil {
					studentIDs = append(studentIDs, uint(sID))
				}
			}
		}

		isCreated, err := u.repo.Upsert(ctx, lesson, studentIDs)
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
