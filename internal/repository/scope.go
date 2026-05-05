// File: internal/repository/scope.go
package repository

import (
	"context"

	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"gorm.io/gorm"
)

// scopeByUser menambahkan filter user_id otomatis jika bukan Admin.
// Admin mendapatkan bypass (melihat semua data).
func scopeByUser(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if ctxutil.IsAdmin(ctx) {
			return db
		}
		userID, err := ctxutil.GetUserID(ctx)
		if err != nil {
			// Safety: kembalikan zero rows jika tidak ada user_id
			return db.Where("1 = 0")
		}
		return db.Where("user_id = ?", userID)
	}
}
