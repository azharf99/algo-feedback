// File: pkg/curriculum/tutor_feedback.go
package curriculum

import (
	"fmt"
	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
)

// GetFeedback menerjemahkan get_feedback dari Python
// Menghasilkan satu paragraf utuh yang digabung dengan enter (\n\n)
func GetFeedback(studentName string, attendanceScore domain.AttendanceScore, activityScore domain.ActivityScore, taskScore domain.TaskScore) string {
	var feedbacks []string

	// 1. Logika Attendance (Kehadiran)
	switch attendanceScore {
	case "0":
		feedbacks = append(feedbacks, fmt.Sprintf("%s tidak hadir di seluruh sesi pelajaran bulan ini. Kami ingin membantu agar %s bisa kembali mengikuti pelajaran dengan lebih baik. Kami akan menghubungi Anda untuk membahas solusi yang tepat.", studentName, studentName))
	case "1":
		feedbacks = append(feedbacks, fmt.Sprintf("%s hadir hanya di 1 dari 4 sesi pelajaran bulan ini. Kami khawatir ini bisa mempengaruhi pemahaman materi yang diajarkan. Jika memungkinkan, mari kita diskusikan bagaimana agar %s bisa lebih rutin mengikuti pelajaran.", studentName, studentName))
	case "2":
		feedbacks = append(feedbacks, fmt.Sprintf("%s hanya hadir di 2 dari 4 sesi bulan ini. Kami melihat kehadiran yang tidak konsisten mulai mempengaruhi kemajuan belajar. Akan lebih baik jika %s bisa hadir lebih teratur agar tidak tertinggal materi.", studentName, studentName))
	case "3":
		feedbacks = append(feedbacks, fmt.Sprintf("%s mengikuti 3 dari 4 sesi pelajaran bulan ini. Kehadirannya cukup baik, dan meskipun ada satu sesi yang terlewat, %s tetap mengikuti materi dengan baik. Kami yakin kehadiran yang lebih konsisten akan membuat belajarnya lebih maksimal!", studentName, studentName))
	case "4":
		feedbacks = append(feedbacks, fmt.Sprintf("%s selalu hadir di setiap sesi pelajaran dan menunjukkan antusiasme yang tinggi. Kami sangat menghargai kehadirannya yang konsisten, karena hal ini sangat membantu %s dalam memahami setiap materi yang diberikan.", studentName, studentName))
	}

	// 2. Logika Activity (Keaktifan)
	switch activityScore {
	case "0":
		feedbacks = append(feedbacks, fmt.Sprintf("Selama pelajaran, %s terlihat kurang berpartisipasi dan lebih banyak diam. Kami ingin mendorong %s agar lebih percaya diri untuk bertanya dan ikut serta dalam diskusi kelas. Kami percaya %s memiliki potensi yang besar!", studentName, studentName, studentName))
	case "1":
		feedbacks = append(feedbacks, fmt.Sprintf("%s terkadang ikut berpartisipasi dalam pelajaran, namun kami merasa %s masih bisa lebih aktif lagi. Kami akan terus memotivasi %s agar lebih berani menyampaikan ide dan bertanya jika ada yang belum dipahami.", studentName, studentName, studentName))
	case "2":
		feedbacks = append(feedbacks, fmt.Sprintf("Kami melihat %s cukup aktif selama kelas berlangsung. %s sering menjawab pertanyaan dan tidak ragu untuk berpartisipasi. Ini adalah hal yang sangat positif, dan kami harap %s bisa terus mempertahankan semangat ini!", studentName, studentName, studentName))
	case "3":
		feedbacks = append(feedbacks, fmt.Sprintf("%s sangat aktif dan selalu bersemangat dalam setiap sesi pelajaran. %s sering membagikan ide-ide kreatif dan berdiskusi dengan teman-teman sekelasnya. Keterlibatan yang luar biasa ini tentu akan berdampak sangat baik bagi perkembangan belajarnya!", studentName, studentName))
	}

	// 3. Logika Task (Tugas)
	switch taskScore {
	case "0":
		feedbacks = append(feedbacks, fmt.Sprintf("Kami perhatikan bahwa %s belum menyelesaikan tugas-tugas di bulan ini. Jika ada kesulitan atau hambatan dalam pengerjaan tugas, kami sangat terbuka untuk membantu %s agar bisa mengejar ketertinggalannya.", studentName, studentName))
	case "1":
		feedbacks = append(feedbacks, fmt.Sprintf("%s berhasil menyelesaikan sebagian besar tugas dengan baik, namun ada beberapa area yang memerlukan sedikit perbaikan. Dengan latihan tambahan dan perhatian lebih, %s pasti akan bisa meningkatkan kualitas tugas-tugasnya dan mencapai hasil yang lebih baik lagi.", studentName, studentName))
	case "2":
		feedbacks = append(feedbacks, fmt.Sprintf("%s telah berhasil menyelesaikan semua tugas dengan sangat baik. Pemahamannya terhadap materi sangat jelas, dan %s mampu menyelesaikan setiap tugas tepat waktu. Senang sekali melihat kemajuannya yang terus meningkat. Terus lanjutkan usaha ini, ya!", studentName, studentName))
	}

	// Menggabungkan array string menjadi 1 teks utuh dengan jeda 2 baris
	return strings.Join(feedbacks, "\n\n")
}

// GetTutorFeedback menerjemahkan get_tutor_feedback dari Python
// (Jika sebelumnya kamu sudah memasukkannya di curriculum.go, kamu bisa menghapusnya dari sana dan pindahkan ke sini agar rapi)
func GetTutorIntro(studentName string) string {
	return fmt.Sprintf(`Halo, Ayah/Bunda dari %s! 👋

Saya Azhar Faturohman Ahidin, tutor %s di Sekolah Pemrograman Internasional Algonova.

Saya ingin berbagi kabar tentang perkembangan %s selama satu bulan terakhir. Kami telah menilai kemajuan %s berdasarkan keterampilan yang dipelajari di kelas, serta upaya yang telah ditunjukkan dalam menyelesaikan berbagai tugas. 😊 Hasil lengkapnya bisa Anda lihat pada lampiran yang sudah kami sediakan 📄.

Jika ada hal yang ingin ditanyakan mengenai hasil ini atau tentang perkembangan %s, saya siap membantu menjelaskan lebih lanjut. Terima kasih atas dukungan Anda dalam proses belajar %s, dan mari kita terus bekerja sama untuk mencapai hasil yang lebih baik ke depannya!`,
		studentName, studentName, studentName, studentName, studentName, studentName)
}
