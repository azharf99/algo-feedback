// File: internal/delivery/http/help_center_handler.go
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/internal/middleware"
	"github.com/azharf99/algo-feedback/pkg/attachment"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/azharf99/algo-feedback/pkg/ws"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// HelpCenterHandler menyediakan REST endpoint untuk Help Center serta jembatan ke
// WebSocket hub agar chat REST dan chat real-time selalu konsisten satu sama lain.
type HelpCenterHandler struct {
	usecase domain.HelpCenterUsecase
	hub     *ws.Hub
}

func (h *HelpCenterHandler) getLang(c *gin.Context) string {
	return ctxutil.GetLanguage(c.Request.Context())
}

// NewHelpCenterHandler mendaftarkan rute REST Help Center. Rute WebSocket (/help/ws)
// didaftarkan terpisah oleh caller (lihat cmd/api/main.go) karena butuh middleware
// autentikasi berbeda (token lewat query param, bukan header Authorization).
func NewHelpCenterHandler(r *gin.RouterGroup, us domain.HelpCenterUsecase, hub *ws.Hub) *HelpCenterHandler {
	handler := &HelpCenterHandler{
		usecase: us,
		hub:     hub,
	}

	help := r.Group("/help")
	{
		help.GET("/conversations", handler.ListConversations)
		help.GET("/conversations/me", handler.GetMyConversation)
		help.GET("/conversations/:id", handler.GetConversation)
		help.GET("/conversations/:id/messages", handler.GetMessages)
		help.POST("/messages", handler.SendMessage)
		// Rate limit lebih ketat khusus upload attachment (di atas rate limiter global),
		// mencegah penyalahgunaan storage server lewat spam upload file besar berulang.
		help.POST("/messages/attachment", middleware.RateLimitMiddleware(rate.Limit(10.0/60.0), 15), handler.UploadAttachment)
		help.GET("/messages/:id/attachment", handler.DownloadAttachment)
		help.PATCH("/conversations/:id/read", handler.MarkRead)
		help.PATCH("/conversations/:id/status", handler.UpdateStatus)
	}

	return handler
}

// ListConversations: GET /help/conversations (Admin only, ditegakkan di usecase)
func (h *HelpCenterHandler) ListConversations(c *gin.Context) {
	var params domain.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(h.getLang(c), "error_invalid_data")})
		return
	}

	result, err := h.usecase.ListConversations(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetMyConversation: GET /help/conversations/me
func (h *HelpCenterHandler) GetMyConversation(c *gin.Context) {
	conv, err := h.usecase.GetMyConversation(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": conv})
}

// GetConversation: GET /help/conversations/:id
func (h *HelpCenterHandler) GetConversation(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	conv, err := h.usecase.GetConversation(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": conv})
}

// GetMessages: GET /help/conversations/:id/messages?page=&limit=
func (h *HelpCenterHandler) GetMessages(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	var params domain.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	result, err := h.usecase.GetMessages(c.Request.Context(), uint(id), params)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	for i := range result.Data {
		withAttachmentURL(&result.Data[i])
	}
	c.JSON(http.StatusOK, result)
}

type sendHelpMessageInput struct {
	ConversationID uint   `json:"conversation_id"`
	Body           string `json:"body" binding:"required"`
}

// SendMessage: POST /help/messages
// Non-admin cukup mengirim {"body": "..."} — conversation_id diabaikan dan diarahkan
// otomatis ke percakapannya sendiri. Admin wajib menyertakan conversation_id target.
func (h *HelpCenterHandler) SendMessage(c *gin.Context) {
	lang := h.getLang(c)
	var req sendHelpMessageInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	body := strings.TrimSpace(req.Body)
	if body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	msg, conv, err := h.usecase.SendMessage(c.Request.Context(), req.ConversationID, body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	withAttachmentURL(msg)

	h.broadcastNewMessage(conv, msg)

	c.JSON(http.StatusCreated, gin.H{"data": msg, "conversation": conv})
}

// UploadAttachment: POST /help/messages/attachment (multipart/form-data)
// Fields: file (wajib), conversation_id (opsional — wajib untuk Admin), body (caption opsional).
//
// Lapisan keamanan yang diterapkan di sini (lihat juga pkg/attachment):
//  1. Content-Length request dibatasi lebih dulu (http.MaxBytesReader) sebelum Gin
//     mem-parsing multipart body, supaya upload oversized ditolak sedini mungkin alih-alih
//     dibaca penuh ke memori/disk dulu.
//  2. Ukuran file per-field juga dicek eksplisit dari metadata FileHeader.
//  3. Tipe file divalidasi dari ISI file (magic bytes), bukan dari Content-Type header atau
//     ekstensi nama file yang dikirim client — keduanya trivial dipalsukan.
//  4. Nama file di disk selalu di-generate ulang oleh server (lihat usecase.SendAttachment),
//     nama asli dari user hanya dipakai untuk tampilan setelah disanitasi.
func (h *HelpCenterHandler) UploadAttachment(c *gin.Context) {
	lang := h.getLang(c)

	// +1MB slack untuk field form lain (conversation_id, body) di luar isi file itu sendiri.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, attachment.MaxSize+(1<<20))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_attachment_required")})
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > attachment.MaxSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": i18n.T(lang, "error_attachment_too_large")})
		return
	}

	var conversationID uint
	if idStr := strings.TrimSpace(c.PostForm("conversation_id")); idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
			return
		}
		conversationID = uint(id)
	}
	body := strings.TrimSpace(c.PostForm("body"))

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_file_open")})
		return
	}
	defer file.Close()

	// Baca beberapa ratus byte pertama untuk sniffing magic bytes, lalu kembalikan cursor
	// ke awal supaya seluruh isi file (termasuk header yang sudah dibaca) ikut tersimpan.
	header := make([]byte, 512)
	n, _ := io.ReadFull(file, header)
	header = header[:n]

	result, err := attachment.Validate(fileHeader.Filename, header)
	if err != nil {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": i18n.T(lang, "error_attachment_type")})
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "msg_save_failed")})
		return
	}

	msg, conv, err := h.usecase.SendAttachment(c.Request.Context(), conversationID, body, domain.HelpAttachmentInput{
		Content:          file,
		OriginalFilename: fileHeader.Filename,
		Extension:        result.Extension,
		MimeType:         result.MimeType,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	withAttachmentURL(msg)

	h.broadcastNewMessage(conv, msg)

	c.JSON(http.StatusCreated, gin.H{"data": msg, "conversation": conv})
}

// DownloadAttachment: GET /help/messages/:id/attachment
// Menyajikan ulang file lampiran sebuah pesan. Otorisasi (pemilik conversation atau Admin)
// divalidasi ulang di usecase.GetMessageForDownload lewat message ID — jadi tidak cukup
// hanya menebak angka ID untuk membaca attachment milik percakapan orang lain.
func (h *HelpCenterHandler) DownloadAttachment(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	msg, err := h.usecase.GetMessageForDownload(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("X-Content-Type-Options", "nosniff")
	// Content-Type SELALU dari nilai yang divalidasi & disimpan server saat upload —
	// tidak pernah dari header/ekstensi yang diklaim client, supaya c.File() di bawah
	// tidak menebak-nebak tipe dari ekstensi file acak yang dipakai untuk penyimpanan.
	c.Header("Content-Type", msg.AttachmentMimeType)
	// Gambar ditampilkan inline (preview foto langsung di chat), tipe lain dipaksa
	// terunduh agar browser tidak pernah merender isinya langsung dari origin API.
	disposition := "attachment"
	if strings.HasPrefix(msg.AttachmentMimeType, "image/") {
		disposition = "inline"
	}
	c.Header("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, msg.AttachmentName))

	// msg.AttachmentPath berasal dari kolom DB milik pesan yang otorisasinya sudah
	// divalidasi di atas — bukan dari input request — sehingga aman dipakai langsung ke c.File().
	c.File(msg.AttachmentPath)
}

// MarkRead: PATCH /help/conversations/:id/read
func (h *HelpCenterHandler) MarkRead(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	if err := h.usecase.MarkRead(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.broadcastRead(c.Request.Context(), uint(id))
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_update_success")})
}

type updateHelpStatusInput struct {
	Status string `json:"status" binding:"required"`
}

// UpdateStatus: PATCH /help/conversations/:id/status  body: {"status": "open"|"closed"}
// Khusus Admin (ditegakkan di usecase).
func (h *HelpCenterHandler) UpdateStatus(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	var req updateHelpStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	var opErr error
	if strings.EqualFold(req.Status, "closed") {
		opErr = h.usecase.CloseConversation(c.Request.Context(), uint(id))
	} else {
		opErr = h.usecase.ReopenConversation(c.Request.Context(), uint(id))
	}
	if opErr != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": opErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_update_success")})
}

// ServeWS: GET /help/ws?token=<jwt> — meng-upgrade koneksi HTTP menjadi WebSocket untuk
// chat real-time. Autentikasi dilakukan oleh middleware.WSAuthMiddleware sebelum handler
// ini dipanggil (lihat pendaftaran rute di cmd/api/main.go).
func (h *HelpCenterHandler) ServeWS(c *gin.Context) {
	userID, err := ctxutil.GetUserID(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.T(h.getLang(c), "error_unauthorized")})
		return
	}
	role, _ := ctxutil.GetRole(c.Request.Context())

	conn, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(h.hub, conn, userID, role)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump(h.handleInbound)
}

// handleInbound memproses pesan yang dikirim client lewat WebSocket dan meneruskannya
// ke usecase yang sama dipakai oleh REST endpoint, lalu broadcast hasilnya.
func (h *HelpCenterHandler) handleInbound(cl *ws.Client, inbound ws.InboundMessage) {
	ctx := ctxutil.WithRole(ctxutil.WithUserID(context.Background(), cl.UserID), cl.Role)

	switch inbound.Type {
	case "message":
		body := strings.TrimSpace(inbound.Body)
		if body == "" {
			return
		}
		msg, conv, err := h.usecase.SendMessage(ctx, inbound.ConversationID, body)
		if err != nil {
			h.hub.SendToUser(cl.UserID, mustMarshal(gin.H{"type": "error", "error": err.Error()}))
			return
		}
		withAttachmentURL(msg)
		h.broadcastNewMessage(conv, msg)

	case "read":
		if err := h.usecase.MarkRead(ctx, inbound.ConversationID); err != nil {
			return
		}
		h.broadcastRead(ctx, inbound.ConversationID)
	}
}

// broadcastNewMessage mengirim pesan baru ke pemilik conversation dan seluruh Admin
// yang sedang online, dipakai baik dari jalur REST maupun WebSocket.
func (h *HelpCenterHandler) broadcastNewMessage(conv *domain.HelpConversation, msg *domain.HelpMessage) {
	if h.hub == nil || conv == nil {
		return
	}
	payload := mustMarshal(gin.H{"type": "message", "conversation": conv, "data": msg})
	h.hub.SendToUser(conv.UserID, payload)
	h.hub.BroadcastToAdmins(payload)
}

// broadcastRead memberitahu sisi lain dari percakapan bahwa pesan sudah dibaca
// (dipakai untuk indikator "sudah dibaca" pada chat UI).
func (h *HelpCenterHandler) broadcastRead(ctx context.Context, conversationID uint) {
	if h.hub == nil {
		return
	}
	conv, err := h.usecase.GetConversation(ctx, conversationID)
	if err != nil {
		return
	}
	payload := mustMarshal(gin.H{"type": "read", "conversation": conv})
	h.hub.SendToUser(conv.UserID, payload)
	h.hub.BroadcastToAdmins(payload)
}

// withAttachmentURL mengisi AttachmentURL (field gorm:"-", tidak disimpan di DB) dengan
// path endpoint download terautentikasi, dipanggil di setiap titik pesan dikirim ke
// client (response REST maupun broadcast WebSocket) supaya frontend tidak pernah perlu
// tahu path file mentah di server (AttachmentPath sengaja json:"-").
func withAttachmentURL(msg *domain.HelpMessage) {
	if msg == nil || !msg.HasAttachment() {
		return
	}
	msg.AttachmentURL = fmt.Sprintf("/help/messages/%d/attachment", msg.ID)
}

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type":"error","error":"internal error"}`)
	}
	return b
}
