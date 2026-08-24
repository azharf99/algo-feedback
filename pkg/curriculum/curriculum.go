// File: pkg/curriculum/curriculum.go
package curriculum

import (
	"fmt"
	"strings"
)

// Levels Map (Menerjemahkan level.py)
var levels = map[string]string{
	"Coding Knight IND":                        "IT STAR level 1",
	"Coding Knight ENG":                        "IT STAR level 1",
	"Digital Literasi 2.0":                     "IT STAR level 2",
	"Digital Literasi 2.0 ENG":                 "IT STAR level 2",
	"Visual Programing IND":                    "IT GENIUS level 3",
	"Visual Programing ENG":                    "IT GENIUS level 3",
	"Graphic Design Junior IND":                "IT GENIUS level 4",
	"Graphic Design Junior ENG":                "IT GENIUS level 4",
	"Graphic Design Senior IND":                "IT GENIUS level 4",
	"Graphic Design Senior ENG":                "IT GENIUS level 4",
	"Desain Game 2.0":                          "IT GENIUS level 5",
	"Game Design 2.0 ENG":                      "IT GENIUS level 5",
	"Python Start 1st year IND":                "IT HERO level 6",
	"Python Start 1st year ENG":                "IT HERO level 6",
	"Python Start 2d year IND":                 "IT HERO level 7",
	"Python Start 2d year ENG":                 "IT HERO level 7",
	"Python Pro 1st year 2021-2022 ind":        "IT HERO level 8",
	"Python Pro_1_ENG":                         "IT HERO level 8",
	"Python Pro 2 IND":                         "IT HERO level 9",
	"Python Pro 2 ENG":                         "IT HERO level 9",
	"Building Websites ENG":                    "IT HERO level 9",
	"Artificial Intelligence, AI ENG":          "Artificial Intelligence",
	"Artificial Intelligence, AI IND":          "Artificial Intelligence",
	"Frontend Development_ENG":                 "Frontend Development",
	"Frontend Development_IND":                 "Frontend Development",
	"Fundamental Frontend Development 2.0 ENG": "Frontend Development",
	"Matematika Junior":                        "Algo Math",
	"Matematika Senior":                        "Algo Math",
	"Minecraft IND":                            "Minecraft",
	"Unity ENG":                                "Unity",
}

func GetCourseLevel(module string) string {
	if level, exists := levels[module]; exists {
		return level
	}
	return ""
}

// Topics Map (Menerjemahkan topic.py)
// Menggunakan map bersarang: map[Nama_Modul]map[Nomor_Bulan]String_Topik
// var topics = Topics
// var results = ModulesResult
// var competencies = CompetencyResult

func GetTopic(topicName string, number int, lang string) string {
	// Try to find the specific localized course name first
	key := topicName
	if lang == "English" && !strings.HasSuffix(key, "ENG") {
		// If requesting English but key doesn't have ENG, try to find ENG variant
		altKey := strings.TrimSuffix(key, " IND") + " ENG"
		if _, ok := Topics[altKey]; ok {
			key = altKey
		}
	} else if lang == "Indonesia" && !strings.HasSuffix(key, "IND") {
		altKey := strings.TrimSuffix(key, " ENG") + " IND"
		if _, ok := Topics[altKey]; ok {
			key = altKey
		}
	}

	if mods, ok := Topics[key]; ok {
		return mods[number]
	}
	return ""
}

func GetResult(topicName string, number int, lang string) string {
	key := topicName
	if lang == "English" && !strings.HasSuffix(key, "ENG") {
		altKey := strings.TrimSuffix(key, " IND") + " ENG"
		if _, ok := ModulesResult[altKey]; ok {
			key = altKey
		}
	} else if lang == "Indonesia" && !strings.HasSuffix(key, "IND") {
		altKey := strings.TrimSuffix(key, " ENG") + " IND"
		if _, ok := ModulesResult[altKey]; ok {
			key = altKey
		}
	}

	if mods, ok := ModulesResult[key]; ok {
		return mods[number]
	}
	return ""
}

func GetCompetency(topicName string, number int, lang string) string {
	key := topicName
	if lang == "English" && !strings.HasSuffix(key, "ENG") {
		altKey := strings.TrimSuffix(key, " IND") + " ENG"
		if _, ok := CompetencyResult[altKey]; ok {
			key = altKey
		}
	} else if lang == "Indonesia" && !strings.HasSuffix(key, "IND") {
		altKey := strings.TrimSuffix(key, " ENG") + " IND"
		if _, ok := CompetencyResult[altKey]; ok {
			key = altKey
		}
	}

	if mods, ok := CompetencyResult[key]; ok {
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
