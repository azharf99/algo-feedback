package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"
)

type RecaptchaResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

const recaptchaServerName = "https://www.google.com/recaptcha/api/siteverify"

// VerifyRecaptcha memverifikasi token ReCaptcha ke API Google
func VerifyRecaptcha(token string) (bool, error) {
	secret := os.Getenv("RECAPTCHA_SECRET")
	if secret == "" {
		// Jika secret tidak diatur, anggap valid (untuk development)
		// Tapi di production ini harus diatur
		return true, nil
	}

	resp, err := http.PostForm(recaptchaServerName, url.Values{
		"secret":   {secret},
		"response": {token},
	})
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result RecaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Success, nil
}
