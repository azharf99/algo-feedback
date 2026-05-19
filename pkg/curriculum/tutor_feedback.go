// File: pkg/curriculum/tutor_feedback.go
package curriculum

import (
	"fmt"
	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
)

// GetFeedback generates localized feedback paragraphs based on scores
func GetFeedback(lang string, studentName string, attendanceScore domain.AttendanceScore, activityScore domain.ActivityScore, taskScore domain.TaskScore) string {
	var feedbacks []string

	if lang == "" {
		lang = "Indonesia"
	}

	// 1. Attendance Logic
	switch lang {
	case "Russian":
		switch attendanceScore {
		case "0":
			feedbacks = append(feedbacks, fmt.Sprintf("%s не присутствовал(а) ни на одном занятии в этом месяце. Мы хотим помочь %s вернуться к занятиям. Мы свяжемся с вами, чтобы обсудить решение.", studentName, studentName))
		case "1":
			feedbacks = append(feedbacks, fmt.Sprintf("%s присутствовал(а) только на 1 из 4 занятий в этом месяце. Мы обеспокоены тем, что это может повлиять на понимание материала. Давайте обсудим, как %s может посещать занятия более регулярно.", studentName, studentName))
		case "2":
			feedbacks = append(feedbacks, fmt.Sprintf("%s присутствовал(а) только на 2 из 4 занятий в этом месяце. Мы заметили, что нерегулярное посещение начинает сказываться на прогрессе. Будет лучше, если %s сможет посещать занятия более стабильно.", studentName, studentName))
		case "3":
			feedbacks = append(feedbacks, fmt.Sprintf("%s посетил(а) 3 из 4 занятий в этом месяце. Посещаемость хорошая, и несмотря на один пропуск, %s хорошо усваивает материал. Мы уверены, что более регулярное посещение принесет еще лучший результат!", studentName, studentName))
		case "4":
			feedbacks = append(feedbacks, fmt.Sprintf("%s всегда присутствует на занятиях и проявляет высокий интерес. Мы очень ценим такое стабильное посещение, так как это очень помогает %s в освоении материала.", studentName, studentName))
		}
	case "English":
		switch attendanceScore {
		case "0":
			feedbacks = append(feedbacks, fmt.Sprintf("%s was not present in any of the lessons this month. We want to help %s get back to class. We will contact you to discuss the best solution.", studentName, studentName))
		case "1":
			feedbacks = append(feedbacks, fmt.Sprintf("%s attended only 1 out of 4 lessons this month. We are concerned this might affect the understanding of the material. If possible, let's discuss how %s can attend lessons more regularly.", studentName, studentName))
		case "2":
			feedbacks = append(feedbacks, fmt.Sprintf("%s attended only 2 out of 4 lessons this month. We see that inconsistent attendance is starting to affect the learning progress. It would be better if %s could attend more regularly to avoid falling behind.", studentName, studentName))
		case "3":
			feedbacks = append(feedbacks, fmt.Sprintf("%s attended 3 out of 4 lessons this month. The attendance is quite good, and despite missing one session, %s still follows the material well. We are sure that more consistent attendance will maximize the learning!", studentName, studentName))
		case "4":
			feedbacks = append(feedbacks, fmt.Sprintf("%s is always present in every lesson and shows high enthusiasm. We really appreciate the consistent attendance, as it greatly helps %s in understanding the material provided.", studentName, studentName))
		}
	default: // Indonesia
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
	}

	// 2. Activity Logic
	switch lang {
	case "Russian":
		switch activityScore {
		case "0":
			feedbacks = append(feedbacks, fmt.Sprintf("На уроках %s проявляет слабую активность и чаще молчит. Мы хотим вдохновить %s быть увереннее, задавать вопросы и участвовать в дискуссиях. Мы верим, что у %s большой потенциал!", studentName, studentName, studentName))
		case "1":
			feedbacks = append(feedbacks, fmt.Sprintf("%s иногда участвует в уроке, но мы чувствуем, что %s может быть еще активнее. Мы будем мотивировать %s смелее выражать свои идеи и спрашивать, если что-то непонятно.", studentName, studentName, studentName))
		case "2":
			feedbacks = append(feedbacks, fmt.Sprintf("Мы видим, что %s достаточно активен(на) во время занятий. %s часто отвечает на вопросы и не стесняется участвовать. Это очень позитивный момент, и мы надеемся, что %s сохранит этот настрой!", studentName, studentName, studentName))
		case "3":
			feedbacks = append(feedbacks, fmt.Sprintf("%s очень активен(на) и всегда с энтузиазмом участвует в каждом занятии. %s часто делится креативными идеями и обсуждает их с одноклассниками. Такое вовлечение очень полезно для прогресса!", studentName, studentName))
		}
	case "English":
		switch activityScore {
		case "0":
			feedbacks = append(feedbacks, fmt.Sprintf("During lessons, %s appears to participate less and stays quiet most of the time. We want to encourage %s to be more confident in asking questions and joining class discussions. We believe %s has great potential!", studentName, studentName, studentName))
		case "1":
			feedbacks = append(feedbacks, fmt.Sprintf("%s sometimes participates in lessons, but we feel that %s could be even more active. We will continue to motivate %s to be braver in expressing ideas and asking questions when something is not understood.", studentName, studentName, studentName))
		case "2":
			feedbacks = append(feedbacks, fmt.Sprintf("We see that %s is quite active during class. %s often answers questions and does not hesitate to participate. This is very positive, and we hope %s can maintain this spirit!", studentName, studentName, studentName))
		case "3":
			feedbacks = append(feedbacks, fmt.Sprintf("%s is very active and always enthusiastic in every lesson. %s often shares creative ideas and discusses them with classmates. This incredible involvement will certainly have a very good impact on learning progress!", studentName, studentName))
		}
	default: // Indonesia
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
	}

	// 3. Task Logic
	switch lang {
	case "Russian":
		switch taskScore {
		case "0":
			feedbacks = append(feedbacks, fmt.Sprintf("Мы заметили, что %s не выполнил(а) задания в этом месяце. Если есть трудности или препятствия, мы готовы помочь %s, чтобы он(а) мог(ла) догнать группу.", studentName, studentName))
		case "1":
			feedbacks = append(feedbacks, fmt.Sprintf("%s успешно выполнил(а) большинство заданий, но есть области, требующие небольших улучшений. При дополнительной практике и внимании %s определенно сможет повысить качество работ и достичь лучших результатов.", studentName, studentName))
		case "2":
			feedbacks = append(feedbacks, fmt.Sprintf("%s успешно и очень качественно выполнил(а) все задания. Понимание материала ясное, и %s сдает все работы вовремя. Очень приятно видеть постоянный прогресс. Продолжай в том же духе!", studentName, studentName))
		}
	case "English":
		switch taskScore {
		case "0":
			feedbacks = append(feedbacks, fmt.Sprintf("We noticed that %s has not completed the tasks this month. If there are any difficulties or obstacles, we are very open to helping %s catch up.", studentName, studentName))
		case "1":
			feedbacks = append(feedbacks, fmt.Sprintf("%s managed to complete most tasks well, but there are some areas that require a bit of improvement. With extra practice and more attention, %s will surely be able to improve the quality of tasks and achieve even better results.", studentName, studentName))
		case "2":
			feedbacks = append(feedbacks, fmt.Sprintf("%s has successfully completed all tasks very well. The understanding of the material is very clear, and %s is able to complete each task on time. It's great to see continuous progress. Keep up the good work!", studentName, studentName))
		}
	default: // Indonesia
		switch taskScore {
		case "0":
			feedbacks = append(feedbacks, fmt.Sprintf("Kami perhatikan bahwa %s belum menyelesaikan tugas-tugas di bulan ini. Jika ada kesulitan atau hambatan dalam pengerjaan tugas, kami sangat terbuka untuk membantu %s agar bisa mengejar ketertinggalannya.", studentName, studentName))
		case "1":
			feedbacks = append(feedbacks, fmt.Sprintf("%s berhasil menyelesaikan sebagian besar tugas dengan baik, namun ada beberapa area yang memerlukan sedikit perbaikan. Dengan latihan tambahan dan perhatian lebih, %s pasti akan bisa meningkatkan kualitas tugas-tugasnya dan mencapai hasil yang lebih baik lagi.", studentName, studentName))
		case "2":
			feedbacks = append(feedbacks, fmt.Sprintf("%s telah berhasil menyelesaikan semua tugas dengan sangat baik. Pemahamannya terhadap materi sangat jelas, dan %s mampu menyelesaikan setiap tugas tepat waktu. Senang sekali melihat kemajuannya yang terus meningkat. Terus lanjutkan usaha ini, ya!", studentName, studentName))
		}
	}

	return strings.Join(feedbacks, "\n\n")
}

// GetTutorIntro generates a localized tutor introduction
func GetTutorIntro(lang string, studentName string) string {
	if lang == "" {
		lang = "Indonesia"
	}

	switch lang {
	case "Russian":
		return fmt.Sprintf(`Здравствуйте, уважаемые родители %s! 👋

Меня зовут Азхар Фатурохман Ахидин, я преподаватель %s в международной школе программирования Algonova.

Я хотел бы поделиться новостями о прогрессе %s за последний месяц. Мы оценили успехи %s на основе навыков, изученных на занятиях, а также усилий, приложенных при выполнении различных заданий. 😊 Полные результаты вы можете увидеть в приложенном отчете 📄.

Если у вас есть вопросы по поводу этих результатов или прогресса %s, я буду рад помочь и объяснить подробнее. Спасибо за вашу поддержку в процессе обучения %s, и давайте продолжать работать вместе для достижения лучших результатов в будущем!`,
			studentName, studentName, studentName, studentName, studentName, studentName)
	case "English":
		return fmt.Sprintf(`Hello, parents of %s! 👋

I am Azhar Faturohman Ahidin, the tutor of %s at Algonova International Programming School.

I would like to share news about %s's progress over the past month. We have assessed %s's progress based on the skills learned in class, as well as the efforts shown in completing various tasks. 😊 You can see the full results in the attachment we have provided 📄.

If you have any questions regarding these results or about %s's progress, I am ready to help explain further. Thank you for your support in %s's learning process, and let's continue to work together to achieve even better results in the future!`,
			studentName, studentName, studentName, studentName, studentName, studentName)
	default: // Indonesia
		return fmt.Sprintf(`Halo, Ayah/Bunda dari %s! 👋

Saya Azhar Faturohman Ahidin, tutor %s di Sekolah Pemrograman Internasional Algonova.

Saya ingin berbagi kabar tentang perkembangan %s selama satu bulan terakhir. Kami telah menilai kemajuan %s berdasarkan keterampilan yang dipelajari di kelas, serta upaya yang telah ditunjukkan dalam menyelesaikan berbagai tugas. 😊 Hasil lengkapnya bisa Anda lihat pada lampiran yang sudah kami sediakan 📄.

Jika ada hal yang ingin ditanyakan mengenai hasil ini atau tentang perkembangan %s, saya siap membantu menjelaskan lebih lanjut. Terima kasih atas dukungan Anda dalam proses belajar %s, dan mari kita terus bekerja sama untuk mencapai hasil yang lebih baik ke depannya!`,
			studentName, studentName, studentName, studentName, studentName, studentName)
	}
}
