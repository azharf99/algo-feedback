// File: internal/usecase/auth_usecase_test.go
package usecase

import (
	"context"
	"os"
	"testing"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// mockAuthUserRepository adalah in-memory fake domain.UserRepository, cukup untuk
// menguji alur Register -> Login tanpa database sungguhan.
type mockAuthUserRepository struct {
	usersByEmail map[string]*domain.User
	nextID       uint
}

func newMockAuthUserRepository() *mockAuthUserRepository {
	return &mockAuthUserRepository{usersByEmail: make(map[string]*domain.User)}
}

func (m *mockAuthUserRepository) Create(ctx context.Context, user *domain.User) error {
	m.nextID++
	user.ID = m.nextID
	stored := *user
	m.usersByEmail[user.Email] = &stored
	return nil
}

func (m *mockAuthUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if u, ok := m.usersByEmail[email]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthUserRepository) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	for _, u := range m.usersByEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthUserRepository) GetByResetToken(ctx context.Context, token string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthUserRepository) GetAll(ctx context.Context) ([]domain.User, error) {
	return nil, nil
}

func (m *mockAuthUserRepository) GetPaginated(ctx context.Context, params domain.PaginationParams) ([]domain.User, int64, error) {
	return nil, 0, nil
}

func (m *mockAuthUserRepository) Update(ctx context.Context, user *domain.User) error {
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *mockAuthUserRepository) Delete(ctx context.Context, id uint) error { return nil }

func (m *mockAuthUserRepository) BulkDelete(ctx context.Context, ids []uint) error { return nil }

// TestRegisterThenLogin adalah regression test untuk bug: handler Register (di
// internal/delivery/http/auth_handler.go) sempat meng-hash password SEBELUM memanggil
// AuthUsecase.Register, padahal Register di sini juga meng-hash-nya lagi — sehingga
// password ter-hash dua kali dan user yang baru mendaftar tidak akan pernah bisa login
// dengan password yang benar. AuthUsecase.Register harus selalu menerima password
// PLAINTEXT (sama seperti StudentUsecase/UserUsecase.Create) dan menjadi satu-satunya
// tempat yang meng-hash-nya.
func TestRegisterThenLogin(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	repo := newMockAuthUserRepository()
	uc := NewAuthUsecase(repo)
	ctx := context.Background()

	plainPassword := "SuperSecret123"
	newUser := &domain.User{
		Name:     "Budi Siswa",
		Email:    "budi@example.com",
		Password: plainPassword, // plaintext, exactly what the HTTP handler now passes through
		Role:     domain.RoleSiswa,
	}

	require.NoError(t, uc.Register(ctx, newUser))

	// Password di storage harus sudah ter-hash tepat satu kali (bukan plaintext, dan
	// bukan hash-dari-hash), sehingga bcrypt masih bisa memverifikasinya langsung.
	stored, err := repo.GetByEmail(ctx, "budi@example.com")
	require.NoError(t, err)
	assert.NotEqual(t, plainPassword, stored.Password, "password harus sudah di-hash")

	res, err := uc.Login(ctx, "budi@example.com", plainPassword)
	require.NoError(t, err, "login dengan password asli harus berhasil setelah register")
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)

	_, err = uc.Login(ctx, "budi@example.com", "wrong-password")
	assert.Error(t, err, "login dengan password salah tetap harus gagal")
}
