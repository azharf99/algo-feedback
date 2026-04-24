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
	CreateSchedule(dataList []map[string]interface{}) (map[string]interface{}, error)
	UpdateSchedule(dataList []map[string]interface{}, scheduleID string) (map[string]interface{}, error)
	UploadDocument(groupName, studentName, courseName string, feedbackNumber uint, filePath string) (map[string]interface{}, error)
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
	req.Header.Set("Authorization", w.config.ApiKey) // Wablas modern biasanya pakai Authorization: API_KEY atau X-API-Key
	// Jika user bilang X-API-Key secara spesifik:
	req.Header.Set("X-API-Key", w.config.ApiKey)
}

func (w *whatsappService) CreateSchedule(dataList []map[string]interface{}) (map[string]interface{}, error) {
	if len(dataList) == 0 {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v2/schedule", w.config.BaseURL)
	jsonData, _ := json.Marshal(dataList)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	w.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	return w.executeRequest(req)
}

func (w *whatsappService) UpdateSchedule(dataList []map[string]interface{}, scheduleID string) (map[string]interface{}, error) {
	if len(dataList) == 0 {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v2/schedule/%s", w.config.BaseURL, scheduleID)
	jsonData, _ := json.Marshal(dataList)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	w.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	return w.executeRequest(req)
}

// UploadDocument menggantikan fungsi upload_files_to_whatsapp di Python
func (w *whatsappService) UploadDocument(groupName, studentName, courseName string, feedbackNumber uint, filePath string) (map[string]interface{}, error) {
	// Buka file lokal
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka file lokal: %w", err)
	}
	defer file.Close()

	// Siapkan form-data (multipart)
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}
	writer.Close()

	// URL sesuai legacy Python: /api/upload/document
	url := fmt.Sprintf("%s/api/upload/document", w.config.BaseURL)
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}

	w.setAuthHeader(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return w.executeRequest(req)
}

// executeRequest menjalankan HTTP request dan mem-parsing JSON response
func (w *whatsappService) executeRequest(req *http.Request) (map[string]interface{}, error) {
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
