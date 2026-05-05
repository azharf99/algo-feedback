// File: pkg/ctxutil/context.go
package ctxutil

import (
	"context"
	"errors"
)

// contextKey adalah tipe internal untuk menghindari collision di context.Value.
type contextKey string

const (
	userIDKey contextKey = "ctxutil_user_id"
	roleKey   contextKey = "ctxutil_role"
)

// ErrNoUserID dikembalikan jika user_id tidak ditemukan di context.
var ErrNoUserID = errors.New("ctxutil: user_id tidak ditemukan di context")

// WithUserID menyimpan user_id ke dalam context.
func WithUserID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// GetUserID mengambil user_id dari context.
func GetUserID(ctx context.Context) (uint, error) {
	v := ctx.Value(userIDKey)
	if v == nil {
		return 0, ErrNoUserID
	}
	id, ok := v.(uint)
	if !ok {
		return 0, ErrNoUserID
	}
	return id, nil
}

// WithRole menyimpan role ke dalam context.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

// GetRole mengambil role dari context.
func GetRole(ctx context.Context) (string, bool) {
	v := ctx.Value(roleKey)
	if v == nil {
		return "", false
	}
	role, ok := v.(string)
	return role, ok
}

// IsAdmin mengecek apakah user saat ini memiliki role Admin.
func IsAdmin(ctx context.Context) bool {
	role, ok := GetRole(ctx)
	if !ok {
		return false
	}
	return role == "Admin"
}
