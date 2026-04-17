package auth

import (
	"fmt"
	"os"

	"github.com/azharf99/algo-feedback/internal/domain"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedAdmin(db *gorm.DB) {
	var count int64
	db.Model(&domain.User{}).Where("email = ?", os.Getenv("ADMIN_USERNAME")).Count(&count)

	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte(os.Getenv("ADMIN_PASSWORD")), bcrypt.DefaultCost)
		adminUser := domain.User{
			Email:    os.Getenv("ADMIN_USERNAME"),
			Password: string(hash),
			Role:     "Admin",
		}
		if err := db.Create(&adminUser).Error; err != nil {
			fmt.Println("Gagal membuat akun admin:", err)
		} else {
			fmt.Printf("✅ SEEDER: Akun Admin berhasil dibuat (%s / %s)!", os.Getenv("ADMIN_USERNAME"), os.Getenv("ADMIN_PASSWORD"))
		}
	} else {
		fmt.Println("✅ SEEDER: Akun Admin sudah eksis.")
	}
}
