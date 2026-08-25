// File: internal/domain/help_center.go
package domain

import (
	"context"
	"io"
	"time"
)

// HelpConversation adalah satu thread percakapan support antara seorang user (Tutor/Siswa)
// dengan tim Admin. Setiap user hanya memiliki satu conversation (uniqueIndex di UserID),
// mirip model live-chat Intercom, agar riwayat bantuan tetap dalam satu percakapan.
type HelpConversation struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"not null;uniqueIndex" json:"user_id"`
	User          *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Status        string     `gorm:"type:varchar(20);not null;default:'open'" json:"status"` // open | closed
	LastMessage   string     `gorm:"type:text" json:"last_message"`
	LastMessageAt *time.Time `json:"last_message_at"`
	// UnreadByUser/UnreadByAdmin dihitung terpisah karena kedua sisi membaca chat secara independen.
	UnreadByUser  int       `gorm:"not null;default:0" json:"unread_by_user"`
	UnreadByAdmin int       `gorm:"not null;default:0" json:"unread_by_admin"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// HelpMessage adalah satu pesan chat di dalam sebuah HelpConversation. Body boleh kosong
// jika pesan hanya berupa attachment (mis. foto bukti tanpa keterangan).
type HelpMessage struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ConversationID uint      `gorm:"not null;index" json:"conversation_id"`
	SenderID       uint      `gorm:"not null" json:"sender_id"`
	Sender         *User     `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	SenderRole     Role      `gorm:"type:varchar(20);not null" json:"sender_role"`
	Body           string    `gorm:"type:text" json:"body"`

	// Field attachment (opsional). AttachmentPath adalah path fisik di disk server dan
	// SENGAJA tidak pernah diserialisasi ke JSON (json:"-") — client hanya boleh mengakses
	// file lewat AttachmentURL (endpoint terautentikasi yang memvalidasi ulang kepemilikan
	// conversation), tidak pernah lewat path filesystem mentah.
	AttachmentPath     string `gorm:"type:varchar(500)" json:"-"`
	AttachmentName     string `gorm:"type:varchar(255)" json:"attachment_name,omitempty"`
	AttachmentMimeType string `gorm:"type:varchar(150)" json:"attachment_mime_type,omitempty"`
	AttachmentSize     int64  `json:"attachment_size,omitempty"`
	// AttachmentURL tidak disimpan di DB (gorm:"-") — diisi oleh handler sebelum
	// dikirim ke client, mengarah ke endpoint download terautentikasi.
	AttachmentURL string `gorm:"-" json:"attachment_url,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// HasAttachment menandakan apakah pesan ini membawa sebuah file lampiran.
func (m *HelpMessage) HasAttachment() bool {
	return m.AttachmentPath != ""
}

// HelpAttachmentInput adalah metadata attachment yang SUDAH divalidasi (lihat
// pkg/attachment.Validate) beserta isinya, dikirim dari handler ke usecase.
type HelpAttachmentInput struct {
	Content          io.Reader
	OriginalFilename string
	Extension        string // ekstensi penyimpanan tervalidasi, mis. ".png"
	MimeType         string // mime type tervalidasi, dipakai sebagai Content-Type saat disajikan
}

// Kontrak Repository untuk HelpConversation
type HelpConversationRepository interface {
	// GetOrCreateForUser mengambil conversation milik userID, membuat baru jika belum ada.
	GetOrCreateForUser(ctx context.Context, userID uint) (*HelpConversation, error)
	// GetByID mengambil conversation berdasarkan ID. Non-admin dibatasi hanya bisa
	// mengambil conversation miliknya sendiri lewat scopeByUser.
	GetByID(ctx context.Context, id uint) (*HelpConversation, error)
	ListAll(ctx context.Context, params PaginationParams) ([]HelpConversation, int64, error)
	Update(ctx context.Context, conv *HelpConversation) error
}

// Kontrak Repository untuk HelpMessage
type HelpMessageRepository interface {
	Create(ctx context.Context, msg *HelpMessage) error
	// ListByConversation mengembalikan pesan terbaru lebih dulu (page 1 = paling baru);
	// urutan kronologis untuk ditampilkan diatur di layer usecase.
	ListByConversation(ctx context.Context, conversationID uint, params PaginationParams) ([]HelpMessage, int64, error)
	// GetByID dipakai saat menyajikan (download) attachment sebuah pesan.
	GetByID(ctx context.Context, id uint) (*HelpMessage, error)
}

// Kontrak Usecase untuk Help Center
type HelpCenterUsecase interface {
	// GetMyConversation dipakai oleh Tutor/Siswa untuk mengambil (atau membuat) percakapan
	// bantuan miliknya sendiri.
	GetMyConversation(ctx context.Context) (*HelpConversation, error)
	// ListConversations khusus Admin, menampilkan seluruh percakapan yang masuk.
	ListConversations(ctx context.Context, params PaginationParams) (*PaginatedResult[HelpConversation], error)
	GetConversation(ctx context.Context, id uint) (*HelpConversation, error)
	GetMessages(ctx context.Context, conversationID uint, params PaginationParams) (*PaginatedResult[HelpMessage], error)
	// SendMessage mengirim pesan. Untuk non-Admin, conversationID diabaikan dan otomatis
	// diarahkan ke conversation miliknya sendiri (dibuat jika belum ada). Untuk Admin,
	// conversationID wajib diisi dan menunjuk ke percakapan user yang sedang dibalas.
	SendMessage(ctx context.Context, conversationID uint, body string) (*HelpMessage, *HelpConversation, error)
	// SendAttachment sama seperti SendMessage tapi menyertakan file lampiran. body boleh
	// kosong (caption opsional). attachment.Content dibaca dan disimpan ke disk server
	// dengan nama file yang di-generate ulang (bukan nama asli dari user).
	SendAttachment(ctx context.Context, conversationID uint, body string, attachment HelpAttachmentInput) (*HelpMessage, *HelpConversation, error)
	// GetMessageForDownload mengambil satu pesan untuk keperluan download attachment-nya,
	// sekaligus memvalidasi bahwa caller berhak mengakses conversation pesan tersebut
	// (pemilik conversation atau Admin) — dipanggil sebelum file di-stream ke client.
	GetMessageForDownload(ctx context.Context, messageID uint) (*HelpMessage, error)
	MarkRead(ctx context.Context, conversationID uint) error
	CloseConversation(ctx context.Context, id uint) error
	ReopenConversation(ctx context.Context, id uint) error
}
