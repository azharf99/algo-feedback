// File: pkg/whatsapp/whatsapp.go
package whatsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// WhatsappConfig menyimpan konfigurasi API (Nanti diisi dari .env)
type WhatsappConfig struct {
	ApiKey  string
	BaseURL string
}

// WhatsappService mendefinisikan kontrak fungsi WhatsApp
type WhatsappService interface {
	ScheduleMedia(to, caption, filePath, runAt string) (int, error)
	UpdateSchedule(id int, to, message, runAt string) error
}

type whatsappService struct {
	config WhatsappConfig
	client *http.Client
}

// NewWhatsappService membuat instance baru untuk layanan Whatsapp
func NewWhatsappService(cfg WhatsappConfig) WhatsappService {
	return &whatsappService{
		config: cfg,
		client: &http.Client{},
	}
}

// fungsi bantuan untuk menyematkan header otorisasi
func (w *whatsappService) setAuthHeader(req *http.Request) {
	req.Header.Set("X-API-Key", w.config.ApiKey)
}

// ScheduleMedia: POST /api/schedule/media
func (w *whatsappService) ScheduleMedia(to, caption, filePath, runAt string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("gagal membuka file: %w", err)
	}
	defer file.Close()

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	// Fields sesuai spesifikasi gateway baru
	_ = writer.WriteField("device_id", "3")
	_ = writer.WriteField("to", to)
	_ = writer.WriteField("is_group", "false")
	_ = writer.WriteField("caption", caption)
	_ = writer.WriteField("media_type", "document")
	_ = writer.WriteField("run_at", runAt)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(part, file)
	writer.Close()

	url := fmt.Sprintf("%s/api/schedule/media", w.config.BaseURL)
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return 0, err
	}

	w.setAuthHeader(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := w.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    int    `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result.Status != "success" {
		return 0, fmt.Errorf("gateway error: %s", result.Message)
	}

	return result.Data, nil
}

// UpdateSchedule: PUT /api/schedule/update
func (w *whatsappService) UpdateSchedule(id int, to, message, runAt string) error {
	payloadData := map[string]interface{}{
		"id":        id,
		"device_id": 3,
		"to":        to,
		"message":   message,
		"run_at":    runAt,
	}
	jsonData, _ := json.Marshal(payloadData)

	url := fmt.Sprintf("%s/api/schedule/update", w.config.BaseURL)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	w.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.Status != "success" {
		return fmt.Errorf("gateway error: %s", result.Message)
	}

	return nil
}
