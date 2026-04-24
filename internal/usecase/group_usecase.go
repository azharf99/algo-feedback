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
	"github.com/azharf99/algo-feedback/pkg/pagination"
)

type groupUsecase struct {
	repo domain.GroupRepository
}

func NewGroupUsecase(repo domain.GroupRepository) domain.GroupUsecase {
	return &groupUsecase{repo: repo}
}

// ... (Untuk Create, GetByID, GetAll, Update, Delete kodenya sama persis seperti Student Usecase,
//      kamu bisa menambahkannya sendiri agar kita fokus ke logika ImportCSV) ...

func (u *groupUsecase) Create(ctx context.Context, group *domain.Group) error {
	return u.repo.Create(ctx, group)
}
func (u *groupUsecase) GetByID(ctx context.Context, id uint) (*domain.Group, error) {
	return u.repo.GetByID(ctx, id)
}
func (u *groupUsecase) GetAll(ctx context.Context) ([]domain.Group, error) { return u.repo.GetAll(ctx) }

// GetPaginated mengambil data grup dengan pagination
func (u *groupUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.Group], error) {
	params = pagination.Normalize(params)
	groups, total, err := u.repo.GetPaginated(ctx, params)
	if err != nil {
		return nil, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return &domain.PaginatedResult[domain.Group]{
		Data:       groups,
		Page:       params.Page,
		Limit:      params.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}
func (u *groupUsecase) Update(ctx context.Context, id uint, req *domain.Group) error {
	req.ID = id
	return u.repo.Update(ctx, req)
}
func (u *groupUsecase) Delete(ctx context.Context, id uint) error { return u.repo.Delete(ctx, id) }

// ImportCSV memproses CSV Grup
func (u *groupUsecase) ImportCSV(ctx context.Context, fileReader io.Reader) (*domain.ImportResult, error) {
	// Catatan: ImportResult sudah dideklarasikan di student_usecase.go, jadi bisa langsung dipakai
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

		// 1. Ambil & Validasi ID
		idUint, err := strconv.ParseUint(record[headerMap["id"]], 10, 32)
		if err != nil || idUint == 0 {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "ID tidak valid atau kosong"})
			continue
		}

		// 2. Parsing Tanggal & Waktu (Golang menggunakan layout referensi: Mon Jan 2 15:04:05 MST 2006)
		var firstLessonDate *time.Time
		if dateStr := record[headerMap["first_lesson_date"]]; dateStr != "" {
			parsedDate, err := time.Parse("02/01/2006", dateStr) // Format: dd/mm/yyyy
			if err == nil {
				firstLessonDate = &parsedDate
			}
		}

		var firstLessonTime *time.Time
		if timeStr := record[headerMap["first_lesson_time"]]; timeStr != "" {
			parsedTime, err := time.Parse("15:04:05", timeStr) // Coba format HH:MM:SS
			if err != nil {
				parsedTime, err = time.Parse("15:04", timeStr) // Coba format HH:MM jika gagal
			}
			if err == nil {
				firstLessonTime = &parsedTime
			}
		}

		// 3. Pointer String (Bisa Kosong)
		desc := record[headerMap["description"]]
		groupPhone := record[headerMap["group_phone"]]
		meetLink := record[headerMap["meeting_link"]]
		recLink := record[headerMap["recordings_link"]]

		group := &domain.Group{
			ID:              uint(idUint),
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

		// 4. Proses Array Many-to-Many Siswa
		var studentIDs []uint
		studentStr := record[headerMap["students"]] // Contoh isi: "1, 2, 5"
		if studentStr != "" {
			parts := strings.Split(studentStr, ",")
			for _, p := range parts {
				sID, err := strconv.ParseUint(strings.TrimSpace(p), 10, 32)
				if err == nil {
					studentIDs = append(studentIDs, uint(sID))
				}
			}
		}

		// 5. Simpan ke Database
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
	}

	return result, nil
}
