// File: pkg/ws/hub.go
// Package ws menyediakan hub WebSocket generik untuk fitur real-time (saat ini dipakai
// oleh Help Center chat). Hub menyimpan seluruh koneksi aktif dan menyediakan cara
// mengirim pesan ke satu user tertentu atau broadcast ke semua Admin.
package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8192
)

// Upgrader mengangkat koneksi HTTP menjadi WebSocket. Origin tidak dicek di sini karena
// koneksi sudah diautentikasi lewat token JWT (lihat middleware.WSAuthMiddleware) sebelum
// handler ini dipanggil.
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// InboundMessage adalah payload JSON yang dikirim client -> server lewat WebSocket.
type InboundMessage struct {
	Type           string `json:"type"` // "message" | "read"
	ConversationID uint   `json:"conversation_id"`
	Body           string `json:"body"`
}

// Client merepresentasikan satu koneksi WebSocket milik seorang user (bisa multi-tab,
// setiap tab punya Client sendiri).
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	UserID uint
	Role   string
}

func NewClient(hub *Hub, conn *websocket.Conn, userID uint, role string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		UserID: userID,
		Role:   role,
	}
}

// Hub menyimpan seluruh koneksi WebSocket aktif dan mengatur pengiriman pesan real-time.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]bool)}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

// SendToUser mengirim payload ke semua koneksi aktif milik userID tertentu.
// Aman dipanggil meskipun user tersebut sedang tidak online (payload akan diabaikan).
func (h *Hub) SendToUser(userID uint, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.UserID == userID {
			h.trySend(c, payload)
		}
	}
}

// BroadcastToAdmins mengirim payload ke semua koneksi milik user berrole Admin,
// sehingga dashboard Help Center Admin ter-update secara real-time.
func (h *Hub) BroadcastToAdmins(payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.Role == "Admin" {
			h.trySend(c, payload)
		}
	}
}

// trySend mengirim non-blocking; jika buffer channel client penuh (client lambat/macet),
// pesan dilewati agar tidak memblokir goroutine hub.
func (h *Hub) trySend(c *Client, payload []byte) {
	select {
	case c.send <- payload:
	default:
	}
}

// WritePump mengirim pesan dari channel send ke koneksi WebSocket, serta mengirim ping
// berkala agar koneksi tetap hidup melewati proxy/load balancer.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump membaca pesan masuk dari client dan meneruskannya ke callback onMessage.
// Loop berhenti (dan Client di-unregister) begitu koneksi ditutup atau error.
func (c *Client) ReadPump(onMessage func(*Client, InboundMessage)) {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var inbound InboundMessage
		if err := json.Unmarshal(raw, &inbound); err != nil {
			continue // abaikan payload yang tidak valid, jangan putuskan koneksi
		}
		onMessage(c, inbound)
	}
}
