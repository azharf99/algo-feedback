// File: internal/repository/help_center_repository.go
package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/pagination"
	"gorm.io/gorm"
)

// helpConversationRepository adalah implementasi nyata dari domain.HelpConversationRepository
type helpConversationRepository struct {
	db *gorm.DB
}

func NewHelpConversationRepository(db *gorm.DB) domain.HelpConversationRepository {
	return &helpConversationRepository{db: db}
}

// GetOrCreateForUser: mengambil satu-satunya conversation milik userID, atau membuatnya
// jika ini pertama kalinya user tersebut menghubungi support.
func (r *helpConversationRepository) GetOrCreateForUser(ctx context.Context, userID uint) (*domain.HelpConversation, error) {
	var conv domain.HelpConversation
	err := r.db.WithContext(ctx).Preload("User").Where("user_id = ?", userID).First(&conv).Error
	if err == nil {
		return &conv, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	conv = domain.HelpConversation{UserID: userID, Status: "open"}
	if err := r.db.WithContext(ctx).Create(&conv).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Preload("User").First(&conv, conv.ID).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetByID: mengambil conversation berdasarkan ID. scopeByUser otomatis membatasi
// non-Admin hanya bisa mengambil conversation miliknya sendiri (user_id = caller).
func (r *helpConversationRepository) GetByID(ctx context.Context, id uint) (*domain.HelpConversation, error) {
	var conv domain.HelpConversation
	err := r.db.WithContext(ctx).Scopes(scopeByUser(ctx)).Preload("User").First(&conv, id).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// ListAll: daftar seluruh conversation, dipakai Admin untuk melihat semua thread masuk.
func (r *helpConversationRepository) ListAll(ctx context.Context, params domain.PaginationParams) ([]domain.HelpConversation, int64, error) {
	var convs []domain.HelpConversation
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.HelpConversation{}).Scopes(scopeByUser(ctx))

	status := strings.ToLower(strings.TrimSpace(params.Status))
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	if params.Search != "" {
		query = query.Joins("JOIN users ON users.id = help_conversations.user_id").
			Where("users.name ILIKE ? OR users.email ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("User").
		Scopes(pagination.Sort(params, "last_message_at DESC NULLS LAST, updated_at DESC"), pagination.Paginate(params)).
		Find(&convs).Error
	if err != nil {
		return nil, 0, err
	}
	return convs, total, nil
}

// Update: menyimpan perubahan status/unread count/last message ke conversation.
func (r *helpConversationRepository) Update(ctx context.Context, conv *domain.HelpConversation) error {
	return r.db.WithContext(ctx).Save(conv).Error
}

// helpMessageRepository adalah implementasi nyata dari domain.HelpMessageRepository
type helpMessageRepository struct {
	db *gorm.DB
}

func NewHelpMessageRepository(db *gorm.DB) domain.HelpMessageRepository {
	return &helpMessageRepository{db: db}
}

// Create: menyimpan satu pesan chat baru.
func (r *helpMessageRepository) Create(ctx context.Context, msg *domain.HelpMessage) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// ListByConversation: mengambil pesan dalam sebuah conversation, terbaru lebih dulu
// (page 1 = pesan paling baru) agar konsisten dengan pola "load more pesan lama" pada chat UI.
func (r *helpMessageRepository) ListByConversation(ctx context.Context, conversationID uint, params domain.PaginationParams) ([]domain.HelpMessage, int64, error) {
	var msgs []domain.HelpMessage
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.HelpMessage{}).Where("conversation_id = ?", conversationID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Sender").
		Scopes(pagination.Sort(params, "id DESC"), pagination.Paginate(params)).
		Find(&msgs).Error
	if err != nil {
		return nil, 0, err
	}
	return msgs, total, nil
}
