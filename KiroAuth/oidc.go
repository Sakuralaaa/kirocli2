package KiroAuth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RefreshInput struct {
	RefreshToken string
	ClientID     string
	ClientSecret string
	AuthMethod   string
	Region       string
}

func RefreshToken(input RefreshInput) (string, string, int64, error) {
	clientID := strings.TrimSpace(input.ClientID)
	clientSecret := input.ClientSecret
	refreshToken := strings.TrimSpace(input.RefreshToken)

	if input.AuthMethod == "social" {
		accessToken, newRefreshToken, expiresAt, err := refreshSocialToken(refreshToken)
		if err == nil {
			return accessToken, newRefreshToken, expiresAt, nil
		}
		if clientID != "" && clientSecret != "" {
			oidcAccess, oidcRefresh, oidcExpires, oidcErr := refreshOIDCToken(refreshToken, clientID, clientSecret, input.Region)
			if oidcErr == nil {
				return oidcAccess, oidcRefresh, oidcExpires, nil
			}
			return "", "", 0, fmt.Errorf("token refresh failed; please verify refreshToken/authMethod/client credentials")
		}
		// Missing OIDC client credentials: keep original social refresh error.
		return "", "", 0, err
	}
	accessToken, newRefreshToken, expiresAt, err := refreshOIDCToken(refreshToken, clientID, clientSecret, input.Region)
	if err == nil {
		return accessToken, newRefreshToken, expiresAt, nil
	}
	socialAccess, socialRefresh, socialExpires, socialErr := refreshSocialToken(refreshToken)
	if socialErr == nil {
		return socialAccess, socialRefresh, socialExpires, nil
	}
	return "", "", 0, fmt.Errorf("token refresh failed; please verify refreshToken/authMethod/client credentials")
}

func refreshOIDCToken(refreshToken, clientID, clientSecret, region string) (string, string, int64, error) {
	if clientID == "" || clientSecret == "" {
		return "", "", 0, fmt.Errorf("OIDC refresh requires clientId and clientSecret")
	}
	normalizedRegion, err := normalizeRegion(region)
	if err != nil {
		return "", "", 0, err
	}

	url := fmt.Sprintf("https://oidc.%s.amazonaws.com/token", normalizedRegion)
	payload := map[string]string{
		"clientId":     clientID,
		"clientSecret": clientSecret,
		"refreshToken": refreshToken,
		"grantType":    "refresh_token",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", 0, fmt.Errorf("refresh failed: %d %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int    `json:"expiresIn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", 0, err
	}
	expiresAt := time.Now().Unix() + int64(result.ExpiresIn)
	return result.AccessToken, result.RefreshToken, expiresAt, nil
}

func refreshSocialToken(refreshToken string) (string, string, int64, error) {
	url := "https://prod.us-east-1.auth.desktop.kiro.dev/refreshToken"
	payload := map[string]string{"refreshToken": refreshToken}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", 0, fmt.Errorf("refresh failed: %d %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int    `json:"expiresIn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", 0, err
	}
	expiresAt := time.Now().Unix() + int64(result.ExpiresIn)
	return result.AccessToken, result.RefreshToken, expiresAt, nil
}
