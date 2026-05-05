// File: cmd/api/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	handler "github.com/azharf99/algo-feedback/internal/delivery/http"
	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/internal/middleware"
	"github.com/azharf99/algo-feedback/internal/repository"
	"github.com/azharf99/algo-feedback/internal/usecase"
	"github.com/azharf99/algo-feedback/pkg/auth"
	"github.com/azharf99/algo-feedback/pkg/pdfgen"
	"github.com/azharf99/algo-feedback/pkg/taskqueue"
	"github.com/azharf99/algo-feedback/pkg/whatsapp"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {

	// 1. Inisialisasi ENV dan Database
	_ = godotenv.Load()
	// Silakan ganti DSN sesuai dengan kredensial database lokalmu
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal koneksi ke database: ", err)
	}

	// 1.5. Pre-migration: Handle column user_id addition for existing data
	// Kita tambahkan kolom user_id secara manual sebagai nullable dulu jika belum ada,
	// agar AutoMigrate tidak error saat mencoba menambahkan kolom NOT NULL pada tabel yang sudah ada isinya.
	tables := []string{"students", "courses", "groups", "lessons", "sessions", "feedbacks"}
	for _, table := range tables {
		if db.Migrator().HasTable(table) && !db.Migrator().HasColumn(table, "user_id") {
			log.Printf("🛠️ MIGRATION: Menambahkan kolom user_id ke tabel %s...", table)
			if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN user_id bigint", table)).Error; err != nil {
				log.Printf("⚠️ MIGRATION: Gagal menambahkan kolom user_id ke %s: %v", table, err)
			}
			// Assign default user_id = 1 untuk data yang sudah ada (Legacy Data)
			// Kita asumsikan user 1 adalah Admin yang akan dibuat oleh seeder.
			db.Exec(fmt.Sprintf("UPDATE %s SET user_id = 1 WHERE user_id IS NULL", table))
		}
	}

	// Auto Migrate: Sekarang aman menjalankan AutoMigrate karena kolom sudah ada dan terisi.
	// GORM akan menyesuaikan tipe data dan constraint (NOT NULL, Index, dll).
	db.AutoMigrate(
		&domain.User{},
		&domain.Student{},
		&domain.Course{},
		&domain.Group{},
		&domain.Lesson{},
		&domain.Session{},
		&domain.Feedback{},
	)

	// 1.6. Seed Data & Final Cleanup
	auth.SeedAdmin(db)

	// Pastikan sekali lagi jika ada data yang masih NULL (misal tabel baru tapi ada anomali)
	var firstUser domain.User
	if err := db.Order("id asc").First(&firstUser).Error; err == nil {
		for _, table := range tables {
			db.Exec(fmt.Sprintf("UPDATE %s SET user_id = ? WHERE user_id IS NULL OR user_id = 0", table), firstUser.ID)
		}
		log.Println("✅ MIGRATION: Sinkronisasi data lama selesai.")
	}

	// 2. Setup Framework Gin
	r := gin.Default()

	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsEnv == "" {
		allowedOrigins = []string{"http://localhost:5173"} // Fallback aman
	} else {
		allowedOrigins = strings.Split(allowedOriginsEnv, ",")
	}

	// 3. Middlewares Dasar (Keamanan)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Next()
	})

	// 4. Inisialisasi Utilitas Pihak Ketiga (Third Party Services)
	// Pastikan folder templates/index.html sudah ada
	pdfService := pdfgen.NewPDFGenerator("templates")

	waConfig := whatsapp.WhatsappConfig{
		ApiKey:  os.Getenv("API_KEY"),
		BaseURL: os.Getenv("BASE_URL"),
	}
	if waConfig.ApiKey == "" {
		waConfig.ApiKey = os.Getenv("TOKEN") // Fallback
	}
	waService := whatsapp.NewWhatsappService(waConfig)

	// Inisialisasi Task Queue (Worker Pool) - 5 workers, 100 queue size
	pool := taskqueue.NewWorkerPool(5, 100)
	pool.Start()
	defer pool.Stop()

	// 5. Inisialisasi Layer (Dependency Injection)

	// --- Auth & User ---
	userRepo := repository.NewUserRepository(db)
	authUsecase := usecase.NewAuthUsecase(userRepo)
	userUsecase := usecase.NewUserUsecase(userRepo)

	// --- Student ---
	studentRepo := repository.NewStudentRepository(db)
	studentUsecase := usecase.NewStudentUsecase(studentRepo)

	// --- Modul Course ---
	courseRepo := repository.NewCourseRepository(db)
	courseUsecase := usecase.NewCourseUsecase(courseRepo)

	// --- Lesson ---
	lessonRepo := repository.NewLessonRepository(db)

	// --- Modul Session ---
	sessionRepo := repository.NewSessionRepository(db)
	sessionUsecase := usecase.NewSessionUsecase(sessionRepo, waService, userRepo)

	lessonUsecase := usecase.NewLessonUsecase(lessonRepo, sessionUsecase)

	// --- Group ---
	groupRepo := repository.NewGroupRepository(db)
	groupUsecase := usecase.NewGroupUsecase(groupRepo, lessonRepo, sessionRepo)

	// --- Feedback ---
	feedbackRepo := repository.NewFeedbackRepository(db)
	feedbackUsecase := usecase.NewFeedbackUsecase(
		feedbackRepo,
		groupRepo,   // <-- Masukkan Group Repo
		sessionRepo, // <-- Masukkan Session Repo
		pdfService,
		waService,
		pool, // <-- Masukkan Worker Pool
	)

	// 6. Routing API
	// Serve Static Files (Agar PDF bisa diakses via Browser)
	r.Static("/mediafiles", "./mediafiles")

	api := r.Group("/api")
	{
		// Endpoint Publik (Tanpa Login)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "Algonova Backend is Healthy"})
		})

		// Auth Handler (Register, Login, Refresh)
		handler.NewAuthHandler(api, authUsecase)

		// Grouping Endpoint yang Butuh Login (Protected)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// Student Module (Admin & Tutor)
			studentGroup := protected.Group("/")
			studentGroup.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleTutor))
			handler.NewStudentHandler(studentGroup, studentUsecase)

			// Group Module (Admin & Tutor)
			groupGroup := protected.Group("/")
			groupGroup.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleTutor))
			handler.NewGroupHandler(groupGroup, groupUsecase)

			// Course Module (Admin & Tutor)
			courseGroup := protected.Group("/")
			courseGroup.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleTutor))
			handler.NewCourseHandler(courseGroup, courseUsecase)

			// Lesson Module (Admin & Tutor)
			lessonGroup := protected.Group("/")
			lessonGroup.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleTutor))
			handler.NewLessonHandler(lessonGroup, lessonUsecase)

			// Session Module (Admin & Tutor)
			sessionGroup := protected.Group("/")
			sessionGroup.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleTutor))
			handler.NewSessionHandler(sessionGroup, sessionUsecase)

			// Feedback Module (Admin & Tutor)
			feedbackGroup := protected.Group("/")
			feedbackGroup.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleTutor))
			handler.NewFeedbackHandler(feedbackGroup, feedbackUsecase)

			// User Management Module (Admin Only)
			userGroup := protected.Group("/")
			userGroup.Use(middleware.RoleMiddleware(domain.RoleAdmin))
			handler.NewUserHandler(userGroup, userUsecase)
		}
	}

	// 7. Menjalankan Server
	port := ":8080"
	log.Printf("🚀 Algonova Feedback API berjalan di http://localhost%s", port)

	// Server Running
	if err := r.Run(port); err != nil {
		log.Fatal("Gagal menjalankan server: ", err)
	}
}
