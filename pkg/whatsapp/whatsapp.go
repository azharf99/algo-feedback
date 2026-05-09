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
	"strconv"
)

// WhatsappConfig menyimpan konfigurasi API (Nanti diisi dari .env)
type WhatsappConfig struct {
	ApiKey  string
	BaseURL string
}

// WhatsappService mendefinisikan kontrak fungsi WhatsApp
type WhatsappService interface {
	ScheduleMedia(apiKey, deviceID, to, caption, filePath, runAt string, isGroup bool) (int, error)
	ScheduleMessage(apiKey, deviceID, to, message, runAt string, isGroup bool) (int, error)
	UpdateSchedule(apiKey, deviceID string, id int, to, message, runAt string, isGroup bool) error
	DeleteSchedule(apiKey string, id int) error
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
func (w *whatsappService) setAuthHeader(req *http.Request, apiKey string) {
	if apiKey == "" {
		apiKey = w.config.ApiKey
	}
	req.Header.Set("X-API-Key", apiKey)
}

// ScheduleMedia: POST /api/schedule/media
func (w *whatsappService) ScheduleMedia(apiKey, deviceID, to, caption, filePath, runAt string, isGroup bool) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("gagal membuka file: %w", err)
	}
	defer file.Close()

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	// Fallback device_id if empty
	if deviceID == "" {
		return 0, fmt.Errorf("deviceID is required")
	}

	isGroupStr := "false"
	if isGroup {
		isGroupStr = "true"
	}

	// Fields sesuai spesifikasi gateway baru
	_ = writer.WriteField("device_id", deviceID)
	_ = writer.WriteField("to", to)
	_ = writer.WriteField("is_group", isGroupStr)
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

	w.setAuthHeader(req, apiKey)
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

	fmt.Println("DEBUG: ScheduleMedia Sent")
	fmt.Println("  To:", to)
	fmt.Println("  IsGroup:", isGroup)
	fmt.Println("  Message Length:", len(caption))
	fmt.Println("  Schedule ID:", result.Data)
	fmt.Println("----------------------------")

	return result.Data, nil
}

// UpdateSchedule: PUT /api/schedule/update
func (w *whatsappService) UpdateSchedule(apiKey, deviceID string, id int, to, message, runAt string, isGroup bool) error {
	// Fallback device_id if empty
	if deviceID == "" {
		return fmt.Errorf("deviceID is required")
	}

	deviceIDInt, _ := strconv.Atoi(deviceID)

	payloadData := map[string]interface{}{
		"id":        id,
		"device_id": deviceIDInt,
		"to":        to,
		"message":   message,
		"run_at":    runAt,
		"is_group":  isGroup,
	}
	jsonData, _ := json.Marshal(payloadData)

	url := fmt.Sprintf("%s/api/schedule/update", w.config.BaseURL)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	w.setAuthHeader(req, apiKey)
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

	fmt.Println("DEBUG: UpdateSchedule Sent")
	fmt.Println("  To:", to)
	fmt.Println("  IsGroup:", isGroup)
	fmt.Println("  Message Length:", len(message))
	fmt.Println("  Schedule ID:", id)
	fmt.Println("----------------------------")

	return nil
}

// ScheduleMessage: POST /api/schedule/message
func (w *whatsappService) ScheduleMessage(apiKey, deviceID, to, message, runAt string, isGroup bool) (int, error) {
	// Fallback device_id if empty
	if deviceID == "" {
		return 0, fmt.Errorf("deviceID is required")
	}

	deviceIDInt, _ := strconv.Atoi(deviceID)

	payloadData := map[string]interface{}{
		"device_id": deviceIDInt,
		"to":        to,
		"message":   message,
		"run_at":    runAt,
		"is_group":  isGroup,
	}
	jsonData, _ := json.Marshal(payloadData)

	url := fmt.Sprintf("%s/api/schedule/message", w.config.BaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}

	w.setAuthHeader(req, apiKey)
	req.Header.Set("Content-Type", "application/json")

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

	fmt.Println("DEBUG: ScheduleMessage Sent")
	fmt.Println("  To:", to)
	fmt.Println("  IsGroup:", isGroup)
	fmt.Println("  Message Length:", len(message))
	fmt.Println("  Schedule ID:", result.Data)
	fmt.Println("----------------------------")

	return result.Data, nil
}

// DeleteSchedule: DELETE /api/schedule/delete?id=<id>
func (w *whatsappService) DeleteSchedule(apiKey string, id int) error {
	url := fmt.Sprintf("%s/api/schedule/delete?id=%d", w.config.BaseURL, id)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	w.setAuthHeader(req, apiKey)

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

	fmt.Println("DEBUG: DeleteSchedule Sent")
	fmt.Println("  Schedule ID:", id)
	fmt.Println("----------------------------")

	return nil
}
