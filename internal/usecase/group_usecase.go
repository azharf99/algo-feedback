// File: internal/usecase/group_usecase.go
package usecase

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/formatter"
	"github.com/azharf99/algo-feedback/pkg/pagination"
)

// ... (NewGroupUsecase & operasi CRUD standar sama persis seperti sebelumnya) ...
type groupUsecase struct {
	repo        domain.GroupRepository
	lessonRepo  domain.LessonRepository
	sessionRepo domain.SessionRepository
}

func NewGroupUsecase(repo domain.GroupRepository, lessonRepo domain.LessonRepository, sessionRepo domain.SessionRepository) domain.GroupUsecase {
	return &groupUsecase{
		repo:        repo,
		lessonRepo:  lessonRepo,
		sessionRepo: sessionRepo,
	}
}
func (u *groupUsecase) Create(ctx context.Context, group *domain.Group, studentIDs []uint) error {
	if group.GroupPhone != nil {
		normalized := formatter.NormalizePhoneNumber(*group.GroupPhone)
		group.GroupPhone = &normalized
	}
	err := u.repo.Create(ctx, group, studentIDs)
	if err != nil {
		return err
	}
	// Seed sessions asynchronously or synchronously, let's do it synchronously to ensure it completes.
	return u.seedSessions(ctx, group)
}
func (u *groupUsecase) GetByID(ctx context.Context, id uint) (*domain.Group, error) {
	return u.repo.GetByID(ctx, id)
}
func (u *groupUsecase) GetAll(ctx context.Context) ([]domain.Group, error) { return u.repo.GetAll(ctx) }
func (u *groupUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.Group], error) {
	params = pagination.Normalize(params)
	groups, total, err := u.repo.GetPaginated(ctx, params)
	if err != nil {
		return nil, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return &domain.PaginatedResult[domain.Group]{
		Data: groups, Total: total, TotalPages: totalPages, Page: params.Page, Limit: params.Limit,
	}, nil
}
func (u *groupUsecase) Update(ctx context.Context, id uint, req *domain.Group, studentIDs []uint) error {
	req.ID = id
	if req.GroupPhone != nil {
		normalized := formatter.NormalizePhoneNumber(*req.GroupPhone)
		req.GroupPhone = &normalized
	}
	err := u.repo.Update(ctx, req, studentIDs)
	if err != nil {
		return err
	}
	return u.seedSessions(ctx, req)
}
func (u *groupUsecase) Delete(ctx context.Context, id uint) error { return u.repo.Delete(ctx, id) }

func (u *groupUsecase) ImportCSV(ctx context.Context, fileReader io.Reader) (*domain.ImportResult, error) {
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

		// Validasi ID Group
		idUint, err := strconv.ParseUint(record[headerMap["id"]], 10, 32)
		if err != nil || idUint == 0 {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "ID tidak valid"})
			continue
		}

		// Validasi Course ID (Wajib ada)
		courseID, err := strconv.ParseUint(record[headerMap["course_id"]], 10, 32)
		if err != nil || courseID == 0 {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "course_id tidak valid"})
			continue
		}

		// Tanggal dan Waktu First Lesson
		var firstLessonDate *domain.DateOnly
		if fd := record[headerMap["first_lesson_date"]]; fd != "" {
			if parsedDate, err := time.Parse("02/01/2006", fd); err == nil {
				firstLessonDate = &domain.DateOnly{Time: parsedDate}
			}
		}

		var firstLessonTime *domain.TimeOnly
		if ft := record[headerMap["first_lesson_time"]]; ft != "" {
			if parsedTime, err := time.Parse("15:04", ft); err == nil {
				firstLessonTime = &domain.TimeOnly{Time: parsedTime}
			}
		}

		desc := record[headerMap["description"]]
		groupPhone := record[headerMap["group_phone"]]
		if groupPhone != "" {
			normalized := formatter.NormalizePhoneNumber(groupPhone)
			groupPhone = normalized
		}
		meetLink := record[headerMap["meeting_link"]]
		recLink := record[headerMap["recordings_link"]]

		group := &domain.Group{
			ID:              uint(idUint),
			CourseID:        uint(courseID), // <-- Tambahan Course ID
			Name:            record[headerMap["name"]],
			Type:            record[headerMap["type"]],
			Description:     &desc,
			GroupPhone:      &groupPhone,
			MeetingLink:     &meetLink,
			RecordingsLink:  &recLink,
			FirstLessonDate: firstLessonDate,
			FirstLessonTime: firstLessonTime,
			IsActive:        strings.ToLower(record[headerMap["is_active"]]) != "false",
		}

		// Array Many-to-Many Siswa
		var studentIDs []uint
		if studentStr := record[headerMap["students"]]; studentStr != "" {
			parts := strings.Split(studentStr, ",")
			for _, p := range parts {
				sID, err := strconv.ParseUint(strings.TrimSpace(p), 10, 32)
				if err == nil {
					studentIDs = append(studentIDs, uint(sID))
				}
			}
		}

		isCreated, err := u.repo.Upsert(ctx, group, studentIDs)
		if err != nil {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": err.Error()})
			continue
		}

		if isCreated {
			result.Created++
		} else {
			result.Updated++
		}

		// Seed sessions after upsert
		u.seedSessions(ctx, group)
	}

	return result, nil
}

func (u *groupUsecase) seedSessions(ctx context.Context, group *domain.Group) error {
	if group.FirstLessonDate == nil {
		return nil // Nothing to schedule if there's no first lesson date
	}

	lessons, err := u.lessonRepo.GetByCourse(ctx, group.CourseID)
	if err != nil {
		return err
	}

	timeStart := domain.TimeOnly{}
	if group.FirstLessonTime != nil {
		timeStart = *group.FirstLessonTime
	}

	startDate := group.FirstLessonDate.Time

	for i, lesson := range lessons {
		// Calculate the date: FirstLessonDate + (i * 7 days)
		sessionDate := startDate.AddDate(0, 0, i*7)

		session := &domain.Session{
			GroupID:   group.ID,
			LessonID:  lesson.ID,
			DateStart: domain.DateOnly{Time: sessionDate},
			TimeStart: timeStart,
			IsDone:    false, // Default value for new sessions
		}

		_, err := u.sessionRepo.Upsert(ctx, session)
		if err != nil {
			// You might want to log this error, but we'll return it for now
			return fmt.Errorf("failed to upsert session for lesson %d: %w", lesson.ID, err)
		}
	}

	return nil
}
