package curriculum_test

import (
	"strings"
	"testing"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/curriculum"
	"github.com/stretchr/testify/assert"
)

func TestGetFeedback(t *testing.T) {
	tests := []struct {
		name            string
		studentName     string
		attendanceScore domain.AttendanceScore
		activityScore   domain.ActivityScore
		taskScore       domain.TaskScore
		wantContains    []string
	}{
		{
			name:            "All Max Scores",
			studentName:     "Budi",
			attendanceScore: domain.AttendanceScoreAlways,
			activityScore:   domain.ActivityScoreActive,
			taskScore:       domain.TaskScoreAll,
			wantContains: []string{
				"selalu hadir di setiap sesi pelajaran",
				"sangat aktif dan selalu bersemangat",
				"berhasil menyelesaikan semua tugas dengan sangat baik",
			},
		},
		{
			name:            "All Min Scores",
			studentName:     "Andi",
			attendanceScore: domain.AttendanceScoreNone,
			activityScore:   domain.ActivityScoreInactive,
			taskScore:       domain.TaskScoreNone,
			wantContains: []string{
				"tidak hadir di seluruh sesi pelajaran bulan ini",
				"terlihat kurang berpartisipasi dan lebih banyak diam",
				"belum menyelesaikan tugas-tugas di bulan ini",
			},
		},
		{
			name:            "Mixed Scores 1",
			studentName:     "Siti",
			attendanceScore: domain.AttendanceScore1xMonth,
			activityScore:   domain.ActivityScoreLessActive,
			taskScore:       domain.TaskScorePartial,
			wantContains: []string{
				"hadir hanya di 1 dari 4 sesi pelajaran",
				"terkadang ikut berpartisipasi dalam pelajaran",
				"berhasil menyelesaikan sebagian besar tugas dengan baik",
			},
		},
		{
			name:            "Mixed Scores 2",
			studentName:     "Doni",
			attendanceScore: domain.AttendanceScore2xMonth,
			activityScore:   domain.ActivityScoreActiveEnough,
			taskScore:       domain.TaskScoreAll,
			wantContains: []string{
				"hanya hadir di 2 dari 4 sesi bulan ini",
				"cukup aktif selama kelas berlangsung",
				"berhasil menyelesaikan semua tugas dengan sangat baik",
			},
		},
		{
			name:            "Mixed Scores 3",
			studentName:     "Eka",
			attendanceScore: domain.AttendanceScore3xMonth,
			activityScore:   domain.ActivityScoreActive,
			taskScore:       domain.TaskScorePartial,
			wantContains: []string{
				"mengikuti 3 dari 4 sesi pelajaran bulan ini",
				"sangat aktif dan selalu bersemangat",
				"berhasil menyelesaikan sebagian besar tugas dengan baik",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := curriculum.GetFeedback(tt.studentName, tt.attendanceScore, tt.activityScore, tt.taskScore)

			// Verify that the feedback contains the expected phrases
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}

			// Verify that the student's name is in the feedback (often multiple times)
			assert.Contains(t, got, tt.studentName)

			// Verify it's joined by double newlines since there are 3 parts
			parts := strings.Split(got, "\n\n")
			assert.Equal(t, 3, len(parts), "Feedback should be composed of exactly 3 paragraphs")
		})
	}
}

func TestGetTutorIntro(t *testing.T) {
	studentName := "Fajar"
	got := curriculum.GetTutorIntro(studentName)

	assert.Contains(t, got, "Halo, Ayah/Bunda dari Fajar!")
	assert.Contains(t, got, "tutor Fajar di Sekolah Pemrograman")
	assert.Contains(t, got, "perkembangan Fajar selama satu bulan")
	assert.Contains(t, got, "kemajuan Fajar berdasarkan keterampilan")
	assert.Contains(t, got, "tentang perkembangan Fajar, saya siap")
	assert.Contains(t, got, "proses belajar Fajar, dan mari")
}
