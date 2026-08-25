// File: internal/usecase/help_center_usecase.go
package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/azharf99/algo-feedback/pkg/pagination"
)

type helpCenterUsecase struct {
	convRepo domain.HelpConversationRepository
	msgRepo  domain.HelpMessageRepository
}

func NewHelpCenterUsecase(convRepo domain.HelpConversationRepository, msgRepo domain.HelpMessageRepository) domain.HelpCenterUsecase {
	return &helpCenterUsecase{
		convRepo: convRepo,
		msgRepo:  msgRepo,
	}
}

func (u *helpCenterUsecase) getLang(ctx context.Context) string {
	return ctxutil.GetLanguage(ctx)
}

// GetMyConversation mengambil (atau membuat) percakapan bantuan milik user yang login.
func (u *helpCenterUsecase) GetMyConversation(ctx context.Context) (*domain.HelpConversation, error) {
	userID, err := ctxutil.GetUserID(ctx)
	if err != nil {
		return nil, errors.New(i18n.T(u.getLang(ctx), "error_unauthorized"))
	}
	return u.convRepo.GetOrCreateForUser(ctx, userID)
}

// ListConversations khusus Admin: menampilkan seluruh percakapan yang masuk dari user.
func (u *helpCenterUsecase) ListConversations(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.HelpConversation], error) {
	lang := u.getLang(ctx)
	if !ctxutil.IsAdmin(ctx) {
		return nil, errors.New(i18n.T(lang, "error_unauthorized"))
	}

	norm := pagination.Normalize(params)
	items, total, err := u.convRepo.ListAll(ctx, norm)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(norm.Limit)))
	return &domain.PaginatedResult[domain.HelpConversation]{
		Data:       items,
		Page:       norm.Page,
		Limit:      norm.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// GetConversation mengambil satu conversation. Non-admin hanya bisa melihat miliknya sendiri
// (dijaga di layer repository lewat scopeByUser).
func (u *helpCenterUsecase) GetConversation(ctx context.Context, id uint) (*domain.HelpConversation, error) {
	conv, err := u.convRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.New(i18n.T(u.getLang(ctx), "error_conversation_not_found"))
	}
	return conv, nil
}

// GetMessages mengambil histori pesan sebuah conversation, diurutkan kronologis (lama -> baru).
func (u *helpCenterUsecase) GetMessages(ctx context.Context, conversationID uint, params domain.PaginationParams) (*domain.PaginatedResult[domain.HelpMessage], error) {
	lang := u.getLang(ctx)

	// Pastikan caller berhak mengakses conversation ini sebelum membaca pesannya.
	if _, err := u.convRepo.GetByID(ctx, conversationID); err != nil {
		return nil, errors.New(i18n.T(lang, "error_conversation_not_found"))
	}

	norm := pagination.Normalize(params)
	items, total, err := u.msgRepo.ListByConversation(ctx, conversationID, norm)
	if err != nil {
		return nil, err
	}

	// Repository mengembalikan pesan terbaru dulu (untuk pagination "load lebih lama"),
	// balik urutannya di sini supaya chat tampil kronologis (lama di atas, baru di bawah).
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	totalPages := int(math.Ceil(float64(total) / float64(norm.Limit)))
	return &domain.PaginatedResult[domain.HelpMessage]{
		Data:       items,
		Page:       norm.Page,
		Limit:      norm.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// SendMessage mengirim satu pesan chat baru dan memperbarui ringkasan conversation.
// Dipakai bersama oleh REST handler dan WebSocket hub agar behaviour selalu konsisten.
func (u *helpCenterUsecase) SendMessage(ctx context.Context, conversationID uint, body string) (*domain.HelpMessage, *domain.HelpConversation, error) {
	lang := u.getLang(ctx)

	userID, err := ctxutil.GetUserID(ctx)
	if err != nil {
		return nil, nil, errors.New(i18n.T(lang, "error_unauthorized"))
	}
	role, _ := ctxutil.GetRole(ctx)
	isAdmin := ctxutil.IsAdmin(ctx)

	var conv *domain.HelpConversation
	if isAdmin {
		if conversationID == 0 {
			return nil, nil, errors.New(i18n.T(lang, "error_conversation_not_found"))
		}
		conv, err = u.convRepo.GetByID(ctx, conversationID)
	} else {
		// Non-admin selalu mengirim ke percakapannya sendiri, dibuat otomatis jika baru pertama kali.
		conv, err = u.convRepo.GetOrCreateForUser(ctx, userID)
	}
	if err != nil {
		return nil, nil, errors.New(i18n.T(lang, "error_conversation_not_found"))
	}

	msg := &domain.HelpMessage{
		ConversationID: conv.ID,
		SenderID:       userID,
		SenderRole:     domain.Role(role),
		Body:           body,
	}
	if err := u.msgRepo.Create(ctx, msg); err != nil {
		return nil, nil, errors.New(i18n.T(lang, "msg_save_failed"))
	}
	msg.Sender = conv.User
	if isAdmin {
		// Sender bukan pemilik conversation (Admin), jangan tampilkan data user pemilik di msg.Sender.
		msg.Sender = nil
	}

	now := time.Now()
	conv.LastMessageAt = &now
	conv.LastMessage = body
	conv.Status = "open" // pesan baru otomatis membuka kembali percakapan yang sudah ditutup

	if isAdmin {
		conv.UnreadByUser++
	} else {
		conv.UnreadByAdmin++
	}
	if err := u.convRepo.Update(ctx, conv); err != nil {
		return nil, nil, err
	}

	return msg, conv, nil
}

// MarkRead mereset unread count pada sisi pemanggil (Admin atau pemilik conversation).
func (u *helpCenterUsecase) MarkRead(ctx context.Context, conversationID uint) error {
	conv, err := u.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return errors.New(i18n.T(u.getLang(ctx), "error_conversation_not_found"))
	}
	if ctxutil.IsAdmin(ctx) {
		conv.UnreadByAdmin = 0
	} else {
		conv.UnreadByUser = 0
	}
	return u.convRepo.Update(ctx, conv)
}

// CloseConversation menandai percakapan selesai ditangani. Khusus Admin.
func (u *helpCenterUsecase) CloseConversation(ctx context.Context, id uint) error {
	lang := u.getLang(ctx)
	if !ctxutil.IsAdmin(ctx) {
		return errors.New(i18n.T(lang, "error_unauthorized"))
	}
	conv, err := u.convRepo.GetByID(ctx, id)
	if err != nil {
		return errors.New(i18n.T(lang, "error_conversation_not_found"))
	}
	conv.Status = "closed"
	return u.convRepo.Update(ctx, conv)
}

// ReopenConversation membuka kembali percakapan yang sudah ditutup. Khusus Admin.
func (u *helpCenterUsecase) ReopenConversation(ctx context.Context, id uint) error {
	lang := u.getLang(ctx)
	if !ctxutil.IsAdmin(ctx) {
		return errors.New(i18n.T(lang, "error_unauthorized"))
	}
	conv, err := u.convRepo.GetByID(ctx, id)
	if err != nil {
		return errors.New(i18n.T(lang, "error_conversation_not_found"))
	}
	conv.Status = "open"
	return u.convRepo.Update(ctx, conv)
}
