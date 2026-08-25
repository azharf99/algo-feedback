// File: internal/usecase/help_center_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/attachment"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/azharf99/algo-feedback/pkg/pagination"
)

// attachmentStorageRoot adalah folder dasar penyimpanan attachment Help Center di disk
// server, mengikuti konvensi folder "mediafiles" yang sudah dipakai fitur PDF di aplikasi ini.
const attachmentStorageRoot = "mediafiles/help_attachments"

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

// resolveTargetConversation menentukan conversation tujuan pengiriman pesan, dipakai
// bersama oleh SendMessage dan SendAttachment agar aturan otorisasinya selalu identik:
// non-Admin hanya bisa mengirim ke percakapannya sendiri (dibuat otomatis jika belum ada),
// Admin wajib menunjuk conversationID yang valid.
func (u *helpCenterUsecase) resolveTargetConversation(ctx context.Context, conversationID uint) (*domain.HelpConversation, uint, string, bool, error) {
	lang := u.getLang(ctx)

	userID, err := ctxutil.GetUserID(ctx)
	if err != nil {
		return nil, 0, "", false, errors.New(i18n.T(lang, "error_unauthorized"))
	}
	role, _ := ctxutil.GetRole(ctx)
	isAdmin := ctxutil.IsAdmin(ctx)

	var conv *domain.HelpConversation
	if isAdmin {
		if conversationID == 0 {
			return nil, 0, "", false, errors.New(i18n.T(lang, "error_conversation_not_found"))
		}
		conv, err = u.convRepo.GetByID(ctx, conversationID)
	} else {
		// Non-admin selalu mengirim ke percakapannya sendiri, dibuat otomatis jika baru pertama kali.
		conv, err = u.convRepo.GetOrCreateForUser(ctx, userID)
	}
	if err != nil {
		return nil, 0, "", false, errors.New(i18n.T(lang, "error_conversation_not_found"))
	}

	return conv, userID, role, isAdmin, nil
}

// touchConversation memperbarui ringkasan conversation setelah sebuah pesan baru terkirim.
func (u *helpCenterUsecase) touchConversation(ctx context.Context, conv *domain.HelpConversation, preview string, isAdmin bool) error {
	now := time.Now()
	conv.LastMessageAt = &now
	conv.LastMessage = preview
	conv.Status = "open" // pesan baru otomatis membuka kembali percakapan yang sudah ditutup

	if isAdmin {
		conv.UnreadByUser++
	} else {
		conv.UnreadByAdmin++
	}
	return u.convRepo.Update(ctx, conv)
}

// SendMessage mengirim satu pesan chat teks baru dan memperbarui ringkasan conversation.
// Dipakai bersama oleh REST handler dan WebSocket hub agar behaviour selalu konsisten.
func (u *helpCenterUsecase) SendMessage(ctx context.Context, conversationID uint, body string) (*domain.HelpMessage, *domain.HelpConversation, error) {
	lang := u.getLang(ctx)

	conv, userID, role, isAdmin, err := u.resolveTargetConversation(ctx, conversationID)
	if err != nil {
		return nil, nil, err
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

	if err := u.touchConversation(ctx, conv, body, isAdmin); err != nil {
		return nil, nil, err
	}

	return msg, conv, nil
}

// SendAttachment mengirim satu pesan chat yang membawa file lampiran (mis. foto bukti
// atau dokumen). File SUDAH HARUS divalidasi (magic bytes) oleh handler lewat
// pkg/attachment.Validate sebelum method ini dipanggil — usecase ini hanya bertanggung
// jawab menyimpannya dengan aman:
//   - nama file di disk selalu di-generate ulang (RandomStorageName), TIDAK PERNAH memakai
//     nama asli dari user, sehingga path traversal/penimpaan file lain tidak mungkin terjadi;
//   - isi file dibatasi ulang dengan io.LimitReader sebagai lapisan pertahanan kedua,
//     independen dari pengecekan Content-Length di layer HTTP handler.
func (u *helpCenterUsecase) SendAttachment(ctx context.Context, conversationID uint, body string, in domain.HelpAttachmentInput) (*domain.HelpMessage, *domain.HelpConversation, error) {
	lang := u.getLang(ctx)

	conv, userID, role, isAdmin, err := u.resolveTargetConversation(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}

	dir := filepath.Join(attachmentStorageRoot, fmt.Sprintf("%d", conv.ID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, errors.New(i18n.T(lang, "msg_save_failed"))
	}

	storageName, err := attachment.RandomStorageName(in.Extension)
	if err != nil {
		return nil, nil, errors.New(i18n.T(lang, "msg_save_failed"))
	}
	fullPath := filepath.Join(dir, storageName)

	dst, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, nil, errors.New(i18n.T(lang, "msg_save_failed"))
	}
	defer dst.Close()

	// Lapisan pertahanan kedua terhadap ukuran file: independen dari pengecekan
	// Content-Length yang bisa dipalsukan/berbeda dari isi body sesungguhnya.
	written, copyErr := io.Copy(dst, io.LimitReader(in.Content, attachment.MaxSize+1))
	if copyErr == nil && written > attachment.MaxSize {
		os.Remove(fullPath)
		return nil, nil, errors.New(i18n.T(lang, "error_attachment_too_large"))
	}
	if copyErr != nil {
		os.Remove(fullPath)
		return nil, nil, errors.New(i18n.T(lang, "msg_save_failed"))
	}

	displayName := attachment.SanitizeDisplayName(in.OriginalFilename)
	msg := &domain.HelpMessage{
		ConversationID:     conv.ID,
		SenderID:           userID,
		SenderRole:         domain.Role(role),
		Body:               body,
		AttachmentPath:     fullPath,
		AttachmentName:     displayName,
		AttachmentMimeType: in.MimeType,
		AttachmentSize:     written,
	}
	if err := u.msgRepo.Create(ctx, msg); err != nil {
		os.Remove(fullPath)
		return nil, nil, errors.New(i18n.T(lang, "msg_save_failed"))
	}
	msg.Sender = conv.User
	if isAdmin {
		msg.Sender = nil
	}

	preview := body
	if preview == "" {
		preview = "📎 " + displayName
	}
	if err := u.touchConversation(ctx, conv, preview, isAdmin); err != nil {
		return nil, nil, err
	}

	return msg, conv, nil
}

// GetMessageForDownload mengambil satu pesan untuk keperluan penyajian ulang file
// lampirannya, sekaligus memvalidasi caller berhak mengakses conversation pemilik pesan
// tersebut (pemilik conversation atau Admin) lewat GetByID (dijaga scopeByUser di
// repository) — mencegah user membaca attachment milik percakapan orang lain hanya
// dengan menebak message ID.
func (u *helpCenterUsecase) GetMessageForDownload(ctx context.Context, messageID uint) (*domain.HelpMessage, error) {
	lang := u.getLang(ctx)

	msg, err := u.msgRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, errors.New(i18n.T(lang, "error_file_not_found"))
	}
	if !msg.HasAttachment() {
		return nil, errors.New(i18n.T(lang, "error_file_not_found"))
	}
	if _, err := u.convRepo.GetByID(ctx, msg.ConversationID); err != nil {
		return nil, errors.New(i18n.T(lang, "error_conversation_not_found"))
	}
	return msg, nil
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
