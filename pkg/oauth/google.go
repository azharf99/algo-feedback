// File: pkg/oauth/google.go
package oauth

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

// GoogleOAuthConfig mengembalikan konfigurasi OAuth2 untuk Google.
// Redirect URL diambil dari env GOOGLE_REDIRECT_URL agar fleksibel
// antara development (localhost) dan production (domain).
func GoogleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// VerifyIDToken memverifikasi JWT ID Token (credential) yang dikirimkan oleh frontend (Google One Tap).
func VerifyIDToken(ctx context.Context, token string) (*GoogleUser, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	payload, err := idtoken.Validate(ctx, token, clientID)
	if err != nil {
		return nil, fmt.Errorf("invalid google id token: %v", err)
	}

	user := &GoogleUser{
		ID:    fmt.Sprintf("%v", payload.Claims["sub"]),
		Email: fmt.Sprintf("%v", payload.Claims["email"]),
		Name:  fmt.Sprintf("%v", payload.Claims["name"]),
	}

	if verified, ok := payload.Claims["email_verified"].(bool); ok {
		user.VerifiedEmail = verified
	}

	if picture, ok := payload.Claims["picture"].(string); ok {
		user.Picture = picture
	}

	return user, nil
}
