// File: internal/delivery/http/help_center_handler.go
package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/azharf99/algo-feedback/pkg/ws"
	"github.com/gin-gonic/gin"
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

	h.broadcastNewMessage(conv, msg)

	c.JSON(http.StatusCreated, gin.H{"data": msg, "conversation": conv})
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

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type":"error","error":"internal error"}`)
	}
	return b
}
