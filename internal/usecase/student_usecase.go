// File: internal/usecase/student_usecase.go
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
	"github.com/azharf99/algo-feedback/pkg/auth"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/formatter"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/azharf99/algo-feedback/pkg/pagination"
)

type studentUsecase struct {
	repo domain.StudentRepository
}

func (u *studentUsecase) getLang(ctx context.Context) string {
	return ctxutil.GetLanguage(ctx)
}

func NewStudentUsecase(repo domain.StudentRepository) domain.StudentUsecase {
	return &studentUsecase{
		repo: repo,
	}
}

// Create menambah siswa baru
func (u *studentUsecase) Create(ctx context.Context, req *domain.UpdateStudentRequest) error {
	lang := u.getLang(ctx)
	student := &domain.Student{
		Fullname:      req.Fullname,
		Surname:       req.Surname,
		Username:      req.Username,
		PhoneNumber:   req.PhoneNumber,
		ParentName:    req.ParentName,
		ParentContact: req.ParentContact,
		IsActive:      req.IsActive,
	}

	// Normalisasi nomor telepon
	if student.PhoneNumber != nil {
		normalized := formatter.NormalizePhoneNumber(*student.PhoneNumber)
		student.PhoneNumber = &normalized
	}
	if student.ParentContact != nil {
		normalized := formatter.NormalizePhoneNumber(*student.ParentContact)
		student.ParentContact = &normalized
	}

	if req.Password != "" {
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			return errors.New(i18n.T(lang, "error_password_process"))
		}
		student.Password = hashedPassword
	}
	return u.repo.Create(ctx, student)
}

// GetByID mengambil data siswa berdasarkan ID
func (u *studentUsecase) GetByID(ctx context.Context, id uint) (*domain.Student, error) {
	return u.repo.GetByID(ctx, id)
}

// GetAll mengambil semua data siswa
func (u *studentUsecase) GetAll(ctx context.Context) ([]domain.Student, error) {
	return u.repo.GetAll(ctx)
}

// GetPaginated mengambil data siswa dengan pagination
func (u *studentUsecase) GetPaginated(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.Student], error) {
	params = pagination.Normalize(params)
	students, total, err := u.repo.GetPaginated(ctx, params)
	if err != nil {
		return nil, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return &domain.PaginatedResult[domain.Student]{
		Data:       students,
		Page:       params.Page,
		Limit:      params.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// Update memperbarui data siswa
func (u *studentUsecase) Update(ctx context.Context, id uint, req *domain.UpdateStudentRequest) error {
	lang := u.getLang(ctx)
	// 1. Cek apakah siswa ada
	existingStudent, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New(i18n.T(lang, "error_student_not_found"))
	}

	// 2. Perbarui field yang diizinkan
	if req.Fullname != "" {
		existingStudent.Fullname = req.Fullname
	}
	if req.Surname != "" {
		existingStudent.Surname = req.Surname
	}
	if req.Username != "" {
		existingStudent.Username = req.Username
	}

	// Normalisasi nomor telepon sebelum update
	if req.PhoneNumber != nil {
		normalized := formatter.NormalizePhoneNumber(*req.PhoneNumber)
		existingStudent.PhoneNumber = &normalized
	}

	if req.ParentName != nil {
		existingStudent.ParentName = req.ParentName
	}

	if req.ParentContact != nil {
		normalized := formatter.NormalizePhoneNumber(*req.ParentContact)
		existingStudent.ParentContact = &normalized
	}

	existingStudent.IsActive = req.IsActive

	// Jika password dikirimkan (tidak kosong), perbarui password
	if req.Password != "" {
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			return errors.New(i18n.T(lang, "error_password_process"))
		}
		existingStudent.Password = hashedPassword
	}

	// 3. Simpan perubahan
	return u.repo.Update(ctx, existingStudent)
}

// Delete menghapus siswa berdasarkan ID
func (u *studentUsecase) Delete(ctx context.Context, id uint) error {
	lang := u.getLang(ctx)
	// Pastikan data ada sebelum dihapus
	_, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New(i18n.T(lang, "error_student_not_found"))
	}
	return u.repo.Delete(ctx, id)
}

func (u *studentUsecase) BulkDelete(ctx context.Context, ids []uint) error {
	return u.repo.BulkDelete(ctx, ids)
}

// ImportCSV memproses file CSV dan mengembalikannya ke Repository
func (u *studentUsecase) ImportCSV(ctx context.Context, fileReader io.Reader) (*domain.ImportResult, error) {
	result := &domain.ImportResult{
		Errors: make([]map[string]interface{}, 0),
	}
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

		idStr := record[headerMap["id"]]
		if idStr == "" {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "Missing 'id' field"})
			continue
		}

		idUint, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil || idUint == 0 {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": i18n.T(lang, "error_invalid_id")})
			continue
		}

		isActive := true
		if strings.ToLower(record[headerMap["is_active"]]) == "false" {
			isActive = false
		}

		phoneNumber := formatter.NormalizePhoneNumber(record[headerMap["phone_number"]])
		parentName := record[headerMap["parent_name"]]
		parentContact := formatter.NormalizePhoneNumber(record[headerMap["parent_contact"]])

		hashedPassword, err := auth.HashPassword(record[headerMap["password"]])
		if err != nil {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": i18n.T(lang, "error_password_process")})
			continue
		}

		student := &domain.Student{
			ID:            uint(idUint),
			Fullname:      record[headerMap["fullname"]],
			Surname:       record[headerMap["surname"]],
			Username:      record[headerMap["username"]],
			Password:      hashedPassword,
			PhoneNumber:   &phoneNumber,
			ParentName:    &parentName,
			ParentContact: &parentContact,
			IsActive:      isActive,
		}

		isCreated, err := u.repo.Upsert(ctx, student)
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
