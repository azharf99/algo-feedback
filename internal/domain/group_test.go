// File: internal/domain/group_test.go
package domain_test

import (
	"testing"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestGroupModel_Creation(t *testing.T) {
	// Persiapan data (Arrange)
	description := "Grup belajar pemrograman Golang"
	groupPhone := "081234567890"
	meetingLink := "https://zoom.us/j/123456789"
	now := time.Now()

	// Tindakan: Membuat objek Group beserta relasi Student-nya (Act)
	group := domain.Group{
		ID:          1,
		Name:        "Golang Backend Batch 1",
		Description: &description,
		Type:        "Group",
		GroupPhone:  &groupPhone,
		MeetingLink: &meetingLink,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
		// Inisiasi relasi many-to-many secara langsung di struct
		Students: []domain.Student{
			{ID: 1, Fullname: "Budi Santoso", Username: "budisan"},
			{ID: 2, Fullname: "Siti Aminah", Username: "sitiamin"},
		},
	}

	// Verifikasi (Assert)
	// 1. Memastikan field wajib terisi dengan benar
	assert.Equal(t, uint(1), group.ID)
	assert.Equal(t, "Golang Backend Batch 1", group.Name)
	assert.Equal(t, "Group", group.Type)
	assert.Equal(t, "081234567890", *group.GroupPhone)
	assert.Equal(t, now, group.UpdatedAt)
	assert.Equal(t, now, group.CreatedAt)
	assert.True(t, group.IsActive)

	// 2. Memastikan field pointer terisi (tidak nil) dan nilainya sesuai
	assert.NotNil(t, group.Description)
	assert.Equal(t, "Grup belajar pemrograman Golang", *group.Description)

	assert.NotNil(t, group.MeetingLink)
	assert.Equal(t, "https://zoom.us/j/123456789", *group.MeetingLink)

	// 3. Memastikan field pointer yang tidak diisi akan bernilai nil (NULL)
	assert.Nil(t, group.RecordingsLink)
	assert.Nil(t, group.FirstLessonTime)

	// 4. Memastikan relasi Siswa (Many-to-Many) terbentuk
	assert.Len(t, group.Students, 2, "Harusnya ada 2 siswa di dalam grup ini")
	assert.Equal(t, "Budi Santoso", group.Students[0].Fullname)
}
