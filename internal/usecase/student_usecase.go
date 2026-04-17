// File: internal/usecase/student_usecase.go
package usecase

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
)

type studentUsecase struct {
	repo domain.StudentRepository
}

func NewStudentUsecase(repo domain.StudentRepository) domain.StudentUsecase {
	return &studentUsecase{
		repo: repo,
	}
}

// Create menambah siswa baru
func (u *studentUsecase) Create(ctx context.Context, student *domain.Student) error {
	// TODO: Nanti di sini kita bisa menambahkan logika Hashing Password sebelum disimpan
	// misal: student.Password = hashPassword(student.Password)
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

// Update memperbarui data siswa
func (u *studentUsecase) Update(ctx context.Context, id uint, req *domain.Student) error {
	// 1. Cek apakah siswa ada
	existingStudent, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New("siswa tidak ditemukan")
	}

	// 2. Perbarui field yang diizinkan
	existingStudent.Fullname = req.Fullname
	existingStudent.Surname = req.Surname
	existingStudent.Username = req.Username
	existingStudent.PhoneNumber = req.PhoneNumber
	existingStudent.ParentName = req.ParentName
	existingStudent.ParentContact = req.ParentContact
	existingStudent.IsActive = req.IsActive

	// Jika password dikirimkan (tidak kosong), perbarui password
	if req.Password != "" {
		existingStudent.Password = req.Password // TODO: Hash password baru
	}

	// 3. Simpan perubahan
	return u.repo.Update(ctx, existingStudent)
}

// Delete menghapus siswa berdasarkan ID
func (u *studentUsecase) Delete(ctx context.Context, id uint) error {
	// Pastikan data ada sebelum dihapus
	_, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New("siswa tidak ditemukan")
	}
	return u.repo.Delete(ctx, id)
}

// ImportCSV memproses file CSV dan mengembalikannya ke Repository
func (u *studentUsecase) ImportCSV(ctx context.Context, fileReader io.Reader) (*domain.ImportResult, error) {
	result := &domain.ImportResult{
		Errors: make([]map[string]interface{}, 0),
	}

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

		idStr := record[headerMap["id"]]
		if idStr == "" {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "Missing 'id' field"})
			continue
		}

		idUint, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			result.Errors = append(result.Errors, map[string]interface{}{"row": rowNum, "error": "Format 'id' tidak valid"})
			continue
		}

		isActive := true
		if strings.ToLower(record[headerMap["is_active"]]) == "false" {
			isActive = false
		}

		phoneNumber := record[headerMap["phone_number"]]
		parentName := record[headerMap["parent_name"]]
		parentContact := record[headerMap["parent_contact"]]

		student := &domain.Student{
			ID:            uint(idUint),
			Fullname:      record[headerMap["fullname"]],
			Surname:       record[headerMap["surname"]],
			Username:      record[headerMap["username"]],
			Password:      record[headerMap["password"]],
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
