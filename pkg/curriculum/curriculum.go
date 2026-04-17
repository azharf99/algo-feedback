// File: pkg/curriculum/curriculum.go
package curriculum

import "fmt"

// Levels Map (Menerjemahkan level.py)
var levels = map[string]string{
	"Artificial Intelligence, AI ENG": "Artificial Intelligence",
	"Visual programming INDONESIA":    "IT GENIUS level 3",
	"Frontend Development_ENG":        "Frontend Development",
	"Python Start 1st year IND":       "IT HERO level 6",
	// ... (Tambahkan sisa map levelmu di sini)
}

func GetCourseLevel(module string) string {
	if level, exists := levels[module]; exists {
		return level
	}
	return ""
}

// Topics Map (Menerjemahkan topic.py)
// Menggunakan map bersarang: map[Nama_Modul]map[Nomor_Bulan]String_Topik
var topics = map[string]map[int]string{
	"Python Start 1st year IND": {
		1: "Variabel dan Tipe Data",
		2: "Operator Kondisi dan Perulangan",
		3: "Fungsi dan bulan",
		// ...
	},
	// ...
}

var results = map[string]map[int]string{
	// Pindahkan data modules_result dari topic.py ke sini
}

var competencies = map[string]map[int]string{
	// Pindahkan data competency_result dari topic.py ke sini
}

func GetTopic(topicName string, number int) string {
	if mods, ok := topics[topicName]; ok {
		return mods[number]
	}
	return ""
}

func GetResult(topicName string, number int) string {
	if mods, ok := results[topicName]; ok {
		return mods[number]
	}
	return ""
}

func GetCompetency(topicName string, number int) string {
	if mods, ok := competencies[topicName]; ok {
		return mods[number]
	}
	return ""
}

// Feedback Tutor (Menerjemahkan tutor_feedback.py)
func GetTutorFeedback(studentName string) string {
	return fmt.Sprintf(`Halo, Ayah/Bunda dari %s! 👋

Saya Azhar Faturohman Ahidin, tutor %s di Sekolah Pemrograman Internasional Algonova.

Saya ingin berbagi kabar tentang perkembangan %s selama satu bulan terakhir. Kami telah menilai kemajuan %s berdasarkan keterampilan yang dipelajari di kelas, serta upaya yang telah ditunjukkan dalam menyelesaikan berbagai tugas. 😊 Hasil lengkapnya bisa Anda lihat pada lampiran yang sudah kami sediakan 📄.

Jika ada hal yang ingin ditanyakan mengenai hasil ini atau tentang perkembangan %s, saya siap membantu menjelaskan lebih lanjut. Terima kasih atas dukungan Anda dalam proses belajar %s, dan mari kita terus bekerja sama untuk mencapai hasil yang lebih baik ke depannya!`,
		studentName, studentName, studentName, studentName, studentName, studentName)
}
